/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"reflect"
	"slices"
	"strings"

	"dario.cat/mergo"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/go-logr/logr"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/hash"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/version"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// SpokeReconciler reconciles a Spoke object
type SpokeReconciler struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	ConcurrentReconciles int
}

// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=spokes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=spokes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=spokes/finalizers,verbs=update

// Reconcile is the main reconcile loop for the Spoke resource.
func (r *SpokeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("request", req)
	ctx = log.IntoContext(ctx, logger)

	// Fetch the Spoke instance
	spoke := &v1beta1.Spoke{}
	err := r.Get(ctx, req.NamespacedName, spoke)
	if err != nil {
		if !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to fetch Spoke", "key", req)
		}
		return ret(ctx, ctrl.Result{}, client.IgnoreNotFound(err))
	}
	ctx = withOriginalSpoke(ctx, spoke)

	// Create a patch helper for this reconciliation
	patchHelper, err := patch.NewHelper(spoke, r.Client)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Ensure patch is applied at the end
	defer func() {
		if err := patchHelper.Patch(ctx, spoke); err != nil && !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to patch Spoke")
		}
	}()

	hubMeta, err := r.getHubMeta(ctx, spoke.Spec.HubRef)
	if err != nil {
		// notFound does not return an error
		logger.V(0).Info("Failed to get latest hub metadata", "error", err)
		spoke.Status.Phase = v1beta1.Unhealthy
	}

	// Add a finalizer if not already present, set defaults, and requeue
	if !slices.Contains(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer) {
		setDefaults(ctx, spoke, hubMeta)
		spoke.Finalizers = append(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer)
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	spokeKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, spoke.Spec.Kubeconfig, spoke.Namespace)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Handle deletion logic with finalizer
	if !spoke.DeletionTimestamp.IsZero() {
		if spoke.Status.Phase != v1beta1.Deleting {
			spoke.Status.Phase = v1beta1.Deleting
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}

		if slices.Contains(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer) {
			if err := r.cleanup(ctx, spoke, spokeKubeconfig, hubMeta); err != nil {
				spoke.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return ret(ctx, ctrl.Result{}, err)
			}
		}
		spoke.Finalizers = slices.DeleteFunc(spoke.Finalizers, func(s string) bool {
			return s == v1beta1.SpokeCleanupFinalizer
		})
		// end reconciliation
		return ret(ctx, ctrl.Result{}, nil)
	}

	// Initialize phase & conditions
	previousPhase := spoke.Status.Phase
	spoke.Status.Phase = v1beta1.SpokeJoining
	initConditions := []v1beta1.Condition{
		v1beta1.NewCondition(
			v1beta1.SpokeJoined, v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
		),
		v1beta1.NewCondition(
			v1beta1.CleanupFailed, v1beta1.CleanupFailed, metav1.ConditionFalse, metav1.ConditionFalse,
		),
		v1beta1.NewCondition(
			v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionFalse,
		),
	}
	spoke.SetConditions(false, initConditions...)

	if previousPhase == "" {
		// set initial phase/conditions and requeue
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	// Handle Spoke cluster: join and/or upgrade
	if err := r.handleSpoke(ctx, spoke, hubMeta, spokeKubeconfig); err != nil {
		logger.Error(err, "Failed to handle spoke operations")
		spoke.Status.Phase = v1beta1.Unhealthy
	}

	// Finalize phase
	for _, c := range spoke.Status.Conditions {
		if c.Status != c.WantStatus {
			logger.Info("WARNING: condition does not have the desired status", "type", c.Type, "reason", c.Reason, "message", c.Message, "status", c.Status, "wantStatus", c.WantStatus)
			spoke.Status.Phase = v1beta1.Unhealthy
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}
	}
	if spoke.Status.Phase == v1beta1.SpokeJoining {
		spoke.Status.Phase = v1beta1.SpokeRunning
	}

	return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
}

type spokeContextKey int

const (
	// originalSpokeKey is the key in the context that records the incoming original Spoke
	originalSpokeKey spokeContextKey = iota
)

func withOriginalSpoke(ctx context.Context, spoke *v1beta1.Spoke) context.Context {
	return context.WithValue(ctx, originalSpokeKey, spoke.DeepCopy())
}

func setDefaults(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta) {
	logger := log.FromContext(ctx)
	if hubMeta.hub == nil {
		logger.V(0).Info("hub not found, skip overriding default timeout and log verbosity")
		return
	}
	if spoke.Spec.Timeout == 300 {
		spoke.Spec.Timeout = hubMeta.hub.Spec.Timeout
	}
	if spoke.Spec.LogVerbosity == 0 {
		spoke.Spec.LogVerbosity = hubMeta.hub.Spec.LogVerbosity
	}
}

// cleanup cleans up a Spoke and its associated resources.
func (r *SpokeReconciler) cleanup(ctx context.Context, spoke *v1beta1.Spoke, spokeKubeconfig []byte, hubMeta hubMeta) error {
	logger := log.FromContext(ctx)

	clusterC, err := common.ClusterClient(hubMeta.kubeconfig)
	if err != nil {
		return err
	}
	workC, err := common.WorkClient(hubMeta.kubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubMeta.kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create addon client for cleanup: %w", err)
	}

	// skip clean up if the ManagedCluster resource is not found or if any manifestWorks exist
	managedCluster, err := clusterC.ClusterV1().ManagedClusters().Get(ctx, spoke.Name, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		logger.Info("ManagedCluster resource not found; nothing to do")
		return nil
	} else if err != nil {
		return fmt.Errorf("unexpected error listing managedClusters: %w", err)
	}
	manifestWorks, err := workC.WorkV1().ManifestWorks(managedCluster.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list manifestWorks for managedCluster %s: %w", managedCluster.Name, err)
	}

	// check that the number of manifestWorks is the same as the number of addons enabled for that spoke
	if len(manifestWorks.Items) > 0 && !allOwnersAddOns(manifestWorks.Items) {
		msg := fmt.Sprintf("Found manifestWorks for ManagedCluster %s; cannot unjoin spoke cluster while it has active ManifestWorks", managedCluster.Name)
		logger.Info(msg)
		return errors.New(msg)
	}

	// remove addons only after confirming that the cluster can be unjoined - this avoids leaving dangling resources that may rely on the addon
	spokeCopy := spoke.DeepCopy()
	spokeCopy.Spec.AddOns = nil
	if _, err := handleSpokeAddons(ctx, addonC, spokeCopy); err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionFalse,
		))
		return err
	}

	if len(spoke.Status.EnabledAddons) > 0 {
		// Wait for addon manifestWorks to be fully cleaned up before proceeding with unjoin
		if err := waitForAddonManifestWorksCleanup(ctx, workC, spoke.Name, addonCleanupTimeout); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionFalse,
			))
			return fmt.Errorf("addon manifestWorks cleanup failed: %w", err)
		}
		spoke.SetConditions(true, v1beta1.NewCondition(
			v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionFalse,
		))
	}

	if err := r.unjoinSpoke(ctx, spoke, spokeKubeconfig); err != nil {
		return err
	}

	// remove CSR
	csrList := &certificatesv1.CertificateSigningRequestList{}
	if err := r.List(ctx, csrList, client.HasLabels{"open-cluster-management.io/cluster-name"}); err != nil {
		return err
	}
	for _, c := range csrList.Items {
		trimmedName := csrSuffixPattern.ReplaceAllString(c.Name, "")
		if trimmedName == spoke.Name {
			if err := r.Delete(ctx, &c); err != nil {
				return err
			}
		}
	}

	// remove ManagedCluster
	if err = clusterC.ClusterV1().ManagedClusters().Delete(ctx, spoke.Name, metav1.DeleteOptions{}); err != nil {
		return client.IgnoreNotFound(err)
	}

	// remove Namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spoke.Name}}
	if err := r.Delete(ctx, ns); err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// handleSpoke manages Spoke cluster join and upgrade operations
func (r *SpokeReconciler) handleSpoke(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleSpoke", "spoke", spoke.Name)

	hub := hubMeta.hub
	hubKubeconfig := hubMeta.kubeconfig

	clusterClient, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create addon client: %w", err)
	}

	// check if the spoke has already been joined to the hub
	managedCluster, err := common.GetManagedCluster(ctx, clusterClient, spoke.Name)
	if err != nil {
		logger.Error(err, "failed to get managedCluster", "spoke", spoke.Name)
		return err
	}

	klusterletValues, err := r.mergeKlusterletValues(ctx, spoke)
	if err != nil {
		return err
	}

	// attempt to join the spoke cluster if it hasn't already been joined
	if managedCluster == nil {
		if err := r.joinSpoke(ctx, spoke, hubMeta, klusterletValues, spokeKubeconfig); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}

		// Accept the cluster join request
		if err := acceptCluster(ctx, spoke, false); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}

		managedCluster, err = common.GetManagedCluster(ctx, clusterClient, spoke.Name)
		if err != nil {
			logger.Error(err, "failed to get managedCluster after join", "spoke", spoke.Name)
			return err
		}
	}

	// check managed clusters joined condition
	jc := r.getJoinedCondition(managedCluster)
	if jc == nil {
		logger.V(0).Info("waiting for spoke cluster to join", "name", spoke.Name)
		msg := fmt.Sprintf("ManagedClusterJoined condition not found in ManagedCluster for spoke cluster %s", spoke.Name)
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		// Re-accept all join requests for the spoke cluster
		if err := acceptCluster(ctx, spoke, true); err != nil {
			logger.Error(err, "failed to accept spoke cluster join request(s)", "spoke", spoke.Name)
		}
		return nil
	}

	logger.V(0).Info("found join condition", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
	if jc.Status != metav1.ConditionTrue {
		msg := fmt.Sprintf("failed to join spoke cluster %s: %s", spoke.Name, jc.Message)
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return errors.New(msg)
	}

	// spoke cluster has joined successfully
	spoke.SetConditions(true, v1beta1.NewCondition(
		"Joined", v1beta1.SpokeJoined, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	// Label the spoke ManagedCluster if in hub-as-spoke mode.
	// This allows the 'spoke' ManagedClusterSet to omit the hub-as-spoke cluster from its list
	// of spoke clusters.
	if managedCluster != nil && spoke.Spec.Kubeconfig.InCluster {
		if managedCluster.Labels == nil {
			managedCluster.Labels = make(map[string]string)
		}
		managedCluster.Labels[v1beta1.LabelManagedClusterType] = v1beta1.ManagedClusterTypeHubAsSpoke
		if err := common.UpdateManagedCluster(ctx, clusterClient, managedCluster); err != nil {
			return err
		}
		logger.V(0).Info("labeled ManagedCluster as hub-as-spoke", "name", spoke.Name)
	}

	// attempt an upgrade whenever the klusterlet's bundleVersion or values change
	currKlusterletHash, err := hash.ComputeHash(klusterletValues)
	if err != nil {
		return fmt.Errorf("failed to compute hash of spoke %s klusterlet values: %w", spoke.Name, err)
	}
	if hub != nil && hub.Spec.ClusterManager.Source.BundleVersion != "" {
		upgrade, err := r.spokeNeedsUpgrade(ctx, spoke, currKlusterletHash, hub.Spec.ClusterManager.Source, spokeKubeconfig)
		if err != nil {
			return fmt.Errorf("failed to check if spoke cluster needs upgrade: %w", err)
		}

		if upgrade {
			if err := r.upgradeSpoke(ctx, spoke, klusterletValues, hub.Spec.ClusterManager.Source, spokeKubeconfig); err != nil {
				return fmt.Errorf("failed to upgrade spoke cluster %s: %w", spoke.Name, err)
			}
		}
	}

	enabledAddons, err := handleSpokeAddons(ctx, addonC, spoke)
	if err != nil {
		msg := fmt.Sprintf("failed to enable addons for spoke cluster %s: %s", spoke.Name, err.Error())
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	// Update status with enabled addons and klusterlet hash
	spoke.Status.EnabledAddons = enabledAddons
	spoke.Status.KlusterletHash = currKlusterletHash

	return nil
}

type tokenMeta struct {
	Token        string `json:"hub-token"`
	HubAPIServer string `json:"hub-apiserver"`
}

type hubMeta struct {
	hub        *v1beta1.Hub
	kubeconfig []byte
}

// joinSpoke joins a Spoke cluster to the Hub cluster
func (r *SpokeReconciler) joinSpoke(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta, klusterletValues *v1beta1.KlusterletChartConfig, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("joinSpoke", "spoke", spoke.Name)

	hub := hubMeta.hub

	if hub == nil {
		return errors.New("hub not found")
	}
	// dont start join until the hub is ready
	hubInitCond := hubMeta.hub.GetCondition(v1beta1.HubInitialized)
	if hubInitCond == nil || hubInitCond.Status != metav1.ConditionTrue {
		return errors.New("hub does not have initialized condition")
	}

	tokenMeta, err := getToken(ctx, hubMeta)
	if err != nil {
		return fmt.Errorf("failed to get join token: %w", err)
	}

	joinArgs := append([]string{
		"join",
		"--cluster-name", spoke.Name,
		fmt.Sprintf("--create-namespace=%t", spoke.Spec.CreateNamespace),
		fmt.Sprintf("--enable-sync-labels=%t", spoke.Spec.SyncLabels),
		"--hub-token", tokenMeta.Token,
		"--wait=true",
		// klusterlet args
		"--mode", spoke.Spec.Klusterlet.Mode,
		"--feature-gates", spoke.Spec.Klusterlet.FeatureGates,
		fmt.Sprintf("--force-internal-endpoint-lookup=%t", spoke.Spec.Klusterlet.ForceInternalEndpointLookup),
		fmt.Sprintf("--singleton=%t", spoke.Spec.Klusterlet.Singleton),
		// source args
		"--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion,
		"--image-registry", hub.Spec.ClusterManager.Source.Registry,
	}, spoke.BaseArgs()...)

	for k, v := range spoke.Spec.Klusterlet.Annotations {
		joinArgs = append(joinArgs, fmt.Sprintf("--klusterlet-annotation=%s=%s", k, v))
	}

	// resources args
	joinArgs = append(joinArgs, args.PrepareResources(spoke.Spec.Klusterlet.Resources)...)

	// Use hub API server from spec if provided and not forced to use internal endpoint,
	// otherwise fall back to the hub API server from the tokenMeta
	if hub.Spec.APIServer != "" && !spoke.Spec.Klusterlet.ForceInternalEndpointLookup {
		joinArgs = append(joinArgs, "--hub-apiserver", hub.Spec.APIServer)
	} else if tokenMeta.HubAPIServer != "" {
		joinArgs = append(joinArgs, "--hub-apiserver", tokenMeta.HubAPIServer)
	}

	if hub.Spec.Ca != "" {
		caFile, caCleanup, err := file.TmpFile([]byte(hub.Spec.Ca), "ca")
		if caCleanup != nil {
			defer caCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write hub CA to disk: %w", err)
		}
		joinArgs = append([]string{fmt.Sprintf("--ca-file=%s", caFile)}, joinArgs...)
	}

	ra := hub.Spec.RegistrationAuth
	if ra.Driver == v1alpha1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-auth=%s", ra.Driver),
		}
		if ra.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", ra.HubClusterARN))
		}
		if spoke.Spec.ClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--managed-cluster-arn=%s", spoke.Spec.ClusterARN))
		}

		joinArgs = append(joinArgs, raArgs...)
	}

	if spoke.Spec.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
		joinArgs = append(joinArgs,
			fmt.Sprintf("--force-internal-endpoint-lookup-managed=%t", spoke.Spec.Klusterlet.ForceInternalEndpointLookupManaged),
		)
		raw, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, spoke.Spec.Klusterlet.ManagedClusterKubeconfig, spoke.Namespace)
		if err != nil {
			return err
		}
		mgdKcfg, mgdKcfgCleanup, err := file.TmpFile(raw, "kubeconfig")
		if mgdKcfgCleanup != nil {
			defer mgdKcfgCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write managedClusterKubeconfig to disk: %w", err)
		}
		joinArgs = append(joinArgs, "--managed-cluster-kubeconfig", mgdKcfg)
	}

	if spoke.Spec.ProxyCa != "" {
		proxyCaFile, proxyCaCleanup, err := file.TmpFile([]byte(spoke.Spec.ProxyCa), "proxy-ca")
		if proxyCaCleanup != nil {
			defer proxyCaCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write proxy CA to disk: %w", err)
		}
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-ca-file=%s", proxyCaFile))
	}
	if spoke.Spec.ProxyURL != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-url=%s", spoke.Spec.ProxyURL))
	}

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(klusterletValues)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	joinArgs = append(joinArgs, valuesArgs...)

	joinArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, joinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm join", "args", joinArgs)

	cmd := exec.Command(clusteradm, joinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm join' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("clusteradm join command failed for spoke %s: %v, output: %s", spoke.Name, err, string(out))
	}
	logger.V(1).Info("successfully requested spoke cluster join", "output", string(stdout))

	return nil
}

// acceptCluster accepts a Spoke cluster's join request
func acceptCluster(ctx context.Context, spoke *v1beta1.Spoke, skipApproveCheck bool) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("acceptCluster", "spoke", spoke.Name)

	acceptArgs := append([]string{
		"accept", "--cluster", spoke.Name,
	}, spoke.BaseArgs()...)

	logger.V(1).Info("clusteradm accept", "args", acceptArgs)

	// TODO: handle other args:
	// --requesters=[]:
	//     Common Names of agents to be approved.

	if skipApproveCheck {
		acceptArgs = append(acceptArgs, "--skip-approve-check")
	}

	cmd := exec.Command(clusteradm, acceptArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm accept' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to accept spoke cluster join request: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("spoke cluster join request accepted", "output", string(stdout))

	return nil
}

// getJoinedCondition gets the joined condition from a managed cluster
func (r *SpokeReconciler) getJoinedCondition(managedCluster *clusterv1.ManagedCluster) *metav1.Condition {
	if managedCluster == nil || managedCluster.Status.Conditions == nil {
		return nil
	}

	for _, c := range managedCluster.Status.Conditions {
		if c.Type == "ManagedClusterJoined" {
			return &c
		}
	}

	return nil
}

// spokeNeedsUpgrade checks if the klusterlet on a Spoke cluster requires an upgrade
func (r *SpokeReconciler) spokeNeedsUpgrade(ctx context.Context, spoke *v1beta1.Spoke, currKlusterletHash string, source v1beta1.OCMSource, spokeKubeconfig []byte) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("spokeNeedsUpgrade", "spokeClusterName", spoke.Name)

	hashChanged := spoke.Status.KlusterletHash != currKlusterletHash
	logger.V(2).Info("comparing klusterlet values hash",
		"spoke", spoke.Name,
		"prevHash", spoke.Status.KlusterletHash,
		"currHash", currKlusterletHash,
	)
	if hashChanged {
		return true, nil
	}

	if source.BundleVersion == "default" {
		logger.V(0).Info("klusterlet bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if source.BundleVersion == "latest" {
		logger.V(0).Info("klusterlet bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	operatorC, err := common.OperatorClient(spokeKubeconfig)
	if err != nil {
		return false, err
	}

	k, err := operatorC.OperatorV1().Klusterlets().Get(ctx, "klusterlet", metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get klusterlet: %w", err)
	}

	// identify lowest bundleVersion referenced in the klusterlet spec
	bundleSpecs := make([]string, 0)
	if k.Spec.ImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.ImagePullSpec)
	}
	if k.Spec.RegistrationImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.RegistrationImagePullSpec)
	}
	if k.Spec.WorkImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.WorkImagePullSpec)
	}
	activeBundleVersion, err := version.LowestBundleVersion(ctx, bundleSpecs)
	if err != nil {
		return false, fmt.Errorf("failed to detect bundleVersion from klusterlet spec: %w", err)
	}
	desiredBundleVersion, err := version.Normalize(source.BundleVersion)
	if err != nil {
		return false, err
	}

	logger.V(0).Info("found klusterlet bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", desiredBundleVersion,
	)
	return activeBundleVersion != desiredBundleVersion, nil
}

// upgradeSpoke upgrades the Spoke cluster's klusterlet
func (r *SpokeReconciler) upgradeSpoke(ctx context.Context, spoke *v1beta1.Spoke, klusterletValues *v1beta1.KlusterletChartConfig, source v1beta1.OCMSource, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeSpoke", "spoke", spoke.Name)

	upgradeArgs := append([]string{
		"upgrade", "klusterlet",
		"--bundle-version", source.BundleVersion,
		"--image-registry", source.Registry,
		"--wait=true",
	}, spoke.BaseArgs()...)

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(klusterletValues)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	upgradeArgs = append(upgradeArgs, valuesArgs...)

	upgradeArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, upgradeArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm upgrade klusterlet", "args", upgradeArgs)

	cmd := exec.Command(clusteradm, upgradeArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm upgrade klusterlet' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf(
			"failed to upgrade klusterlet on spoke cluster %s to %s: %v, output: %s",
			spoke.Name, source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("klusterlet upgraded", "output", string(stdout))

	return nil
}

// unjoinSpoke unjoins a spoke from the hub
func (r *SpokeReconciler) unjoinSpoke(ctx context.Context, spoke *v1beta1.Spoke, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("unjoinSpoke", "spoke", spoke.Name)

	unjoinArgs := append([]string{
		"unjoin",
		"--cluster-name", spoke.GetName(),
		fmt.Sprintf("--purge-operator=%t", spoke.Spec.Klusterlet.PurgeOperator),
	}, spoke.BaseArgs()...)

	unjoinArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, unjoinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %w", spoke.GetName(), err)
	}

	logger.V(1).Info("clusteradm unjoin", "args", unjoinArgs)

	cmd := exec.Command(clusteradm, unjoinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm unjoin' to complete for spoke %s...", spoke.GetName()))
	out := append(stdout, stderr...)
	if err != nil || strings.Contains(string(out), amwExistsError) {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %v, output: %s", spoke.GetName(), err, string(out))
	}
	logger.V(1).Info("spoke cluster unjoined", "output", string(stdout))

	return nil
}

// getToken gets a join token from the Hub cluster via 'clusteradm get token'
func getToken(ctx context.Context, hubMeta hubMeta) (*tokenMeta, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("getToken")

	tokenArgs := append([]string{
		"get", "token", "--output=json",
	}, hubMeta.hub.BaseArgs()...)

	if hubMeta.hub.Spec.ClusterManager != nil {
		tokenArgs = append(tokenArgs, fmt.Sprintf("--use-bootstrap-token=%t", hubMeta.hub.Spec.ClusterManager.UseBootstrapToken))
	}
	tokenArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, hubMeta.kubeconfig, hubMeta.hub.Spec.Kubeconfig.Context, tokenArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to prepare kubeconfig: %w", err)
	}

	logger.V(1).Info("clusteradm get token", "args", tokenArgs)

	cmd := exec.Command(clusteradm, tokenArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm get token' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return nil, fmt.Errorf("failed to get join token: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("got join token", "output", string(stdout))

	tokenMeta := &tokenMeta{}
	if err := json.Unmarshal(stdout, &tokenMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal join token: %w", err)
	}
	return tokenMeta, nil
}

func (r *SpokeReconciler) getHubMeta(ctx context.Context, hubRef v1beta1.HubRef) (hubMeta, error) {
	hub := &v1beta1.Hub{}
	hubMeta := hubMeta{}
	nn := types.NamespacedName{Name: hubRef.Name, Namespace: hubRef.Namespace}

	// get Hub using local client
	err := r.Get(ctx, nn, hub)
	if err != nil {
		return hubMeta, client.IgnoreNotFound(err)
	}
	hubMeta.hub = hub
	// if found, load the hub's kubeconfig
	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, hub.Spec.Kubeconfig, hub.Namespace)
	if err != nil {
		return hubMeta, err
	}
	hubMeta.kubeconfig = hubKubeconfig
	return hubMeta, nil
}

func (r *SpokeReconciler) mergeKlusterletValues(ctx context.Context, spoke *v1beta1.Spoke) (*v1beta1.KlusterletChartConfig, error) {
	logger := log.FromContext(ctx)

	if spoke.Spec.Klusterlet.ValuesFrom == nil && spoke.Spec.Klusterlet.Values == nil {
		logger.V(3).Info("no values or valuesFrom provided. Using default klusterlet chart values", "spoke", spoke.Name)
		return nil, nil
	}

	var fromInterface = map[string]any{}
	var specInterface = map[string]any{}

	if spoke.Spec.Klusterlet.ValuesFrom != nil {
		cm := &corev1.ConfigMap{}
		nn := types.NamespacedName{Name: spoke.Spec.Klusterlet.ValuesFrom.Name, Namespace: spoke.Namespace}
		err := r.Get(ctx, nn, cm)
		if err != nil {
			if kerrs.IsNotFound(err) {
				// cm not found, return spec's values
				logger.V(1).Info("warning: Klusterlet values ConfigMap not found", "spoke", spoke.Name, "configMap", nn)
				return spoke.Spec.Klusterlet.Values, nil
			}
			return nil, fmt.Errorf("failed to retrieve Klusterlet values ConfigMap %s: %w", nn, err)
		}
		fromValues, ok := cm.Data[spoke.Spec.Klusterlet.ValuesFrom.Key]
		if !ok {
			logger.V(1).Info("warning: Klusterlet values ConfigMap not found", "spoke", spoke.Name, "configMap", nn, "key", spoke.Spec.Klusterlet.ValuesFrom.Key)
			return spoke.Spec.Klusterlet.Values, nil
		}
		fromBytes := []byte(fromValues)
		err = yaml.Unmarshal(fromBytes, &fromInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML values from ConfigMap %s key %s: %w", nn, spoke.Spec.Klusterlet.ValuesFrom.Key, err)
		}
	}

	if spoke.Spec.Klusterlet.Values != nil {
		specBytes, err := yaml.Marshal(spoke.Spec.Klusterlet.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Klusterlet values from spoke spec for spoke %s: %w", spoke.Name, err)
		}
		err = yaml.Unmarshal(specBytes, &specInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Klusterlet values from spoke spec for spoke %s: %w", spoke.Name, err)
		}
	}

	mergedMap := map[string]any{}
	maps.Copy(mergedMap, fromInterface)

	// Merge spec on top but ignore zero-values from spec
	if err := mergo.Map(&mergedMap, specInterface, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merge failed for spoke %s: %w", spoke.Name, err)
	}

	mergedBytes, err := yaml.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged Klusterlet values for spoke %s: %w", spoke.Name, err)
	}

	merged := &v1beta1.KlusterletChartConfig{}
	err = yaml.Unmarshal(mergedBytes, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged values into KlusterletChartConfig for spoke %s: %w", spoke.Name, err)
	}

	return merged, nil

}

// prepareKlusterletValuesFile creates a temporary file with klusterlet values and returns
// args to append and a cleanup function. Returns empty slice if values are empty.
func prepareKlusterletValuesFile(values *v1beta1.KlusterletChartConfig) ([]string, func(), error) {
	if values == nil {
		return nil, nil, nil
	}

	if values.IsEmpty() {
		return nil, nil, nil
	}
	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal klusterlet values to YAML: %w", err)
	}
	valuesFile, valuesCleanup, err := file.TmpFile(valuesYAML, "klusterlet-values")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write klusterlet values to disk: %w", err)
	}
	return []string{"--klusterlet-values-file", valuesFile}, valuesCleanup, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpokeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Spoke{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.ConcurrentReconciles,
		}).
		// watch for Hub updates, to immediately propagate any updates to RegistrationAuth, OCMSource
		Watches(
			&v1beta1.Hub{},
			handler.EnqueueRequestsFromMapFunc(r.mapHubEventToSpoke),
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc: func(_ event.DeleteEvent) bool {
					return false
				},
				CreateFunc: func(_ event.CreateEvent) bool {
					return false
				},
				// only return true if old and new hub specs shared fields are different
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldHub, ok := e.ObjectOld.(*v1beta1.Hub)
					if !ok {
						return false
					}
					newHub, ok := e.ObjectNew.(*v1beta1.Hub)
					if !ok {
						return false
					}
					return sharedFieldsChanged(oldHub.Spec.DeepCopy(), newHub.Spec.DeepCopy())
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return false
				},
			}),
		).
		Named("spoke").
		Complete(r)
}

// sharedFieldsChanged checks whether the spec fields that are shared between Hub and Spokes were updated,
// to prevent unnecessary reconciles of Spokes
func sharedFieldsChanged(oldSpec, newSpec *v1beta1.HubSpec) bool {
	return !reflect.DeepEqual(oldSpec.RegistrationAuth, newSpec.RegistrationAuth) ||
		!reflect.DeepEqual(oldSpec.ClusterManager.Source, newSpec.ClusterManager.Source)
}

func (r *SpokeReconciler) mapHubEventToSpoke(ctx context.Context, obj client.Object) []reconcile.Request {
	hub, ok := obj.(*v1beta1.Hub)
	if !ok {
		r.Log.V(1).Info("failed to enqueue spoke requests", "expected", "hub", "got", fmt.Sprintf("%T", obj))
		return nil
	}
	spokeList := &v1beta1.SpokeList{}
	err := r.List(ctx, spokeList)
	if err != nil {
		r.Log.Error(err, "failed to List spokes")
		return nil
	}
	req := make([]reconcile.Request, 0)
	for _, s := range spokeList.Items {
		if !s.IsManagedBy(hub.ObjectMeta) {
			continue
		}
		req = append(req, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
		})
	}
	return req
}
