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

// Package v1beta1 contains the main reconciliation logic for fleetconfig-controller's v1beta1 resources.
package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	operatorapi "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	arg_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/hash"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/version"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// HubReconciler reconciles a Hub object
type HubReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs/finalizers,verbs=update

// Reconcile is the main reconcile loop for the Hub resource.
func (r *HubReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("request", req)
	ctx = log.IntoContext(ctx, logger)

	// Fetch the Hub instance
	hub := &v1beta1.Hub{}
	err := r.Get(ctx, req.NamespacedName, hub)
	if err != nil {
		if !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to fetch Hub", "key", req)
		}
		return ret(ctx, ctrl.Result{}, client.IgnoreNotFound(err))
	}
	ctx = withOriginalHub(ctx, hub)

	// Create a patch helper for this reconciliation
	patchHelper, err := patch.NewHelper(hub, r.Client)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Ensure patch is applied at the end
	defer func() {
		if err := patchHelper.Patch(ctx, hub); err != nil && !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to patch Hub")
		}
	}()

	// Add a finalizer and requeue if not already present
	if !slices.Contains(hub.Finalizers, v1beta1.HubCleanupFinalizer) {
		hub.Finalizers = append(hub.Finalizers, v1beta1.HubCleanupFinalizer)
		return ret(ctx, ctrl.Result{RequeueAfter: hubRequeuePreInit}, nil)
	}

	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, hub.Spec.Kubeconfig, hub.Namespace)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Handle deletion logic with finalizer
	if !hub.DeletionTimestamp.IsZero() {
		if hub.Status.Phase != v1beta1.Deleting {
			hub.Status.Phase = v1beta1.Deleting
			return ret(ctx, ctrl.Result{RequeueAfter: requeueDeleting}, nil)
		}

		if slices.Contains(hub.Finalizers, v1beta1.HubCleanupFinalizer) {
			requeue, err := r.cleanHub(ctx, hub, hubKubeconfig)
			if err != nil {
				hub.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return ret(ctx, ctrl.Result{}, err)
			}
			if requeue {
				return ret(ctx, ctrl.Result{RequeueAfter: requeueDeleting}, nil)
			}
		}

		// end reconciliation
		return ret(ctx, ctrl.Result{}, nil)
	}

	// Initialize phase & conditions
	previousPhase := hub.Status.Phase
	hub.Status.Phase = v1beta1.HubStarting
	initConditions := []v1beta1.Condition{
		v1beta1.NewCondition(
			v1beta1.HubInitialized, v1beta1.HubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
		),
		v1beta1.NewCondition(
			v1beta1.CleanupFailed, v1beta1.CleanupFailed, metav1.ConditionFalse, metav1.ConditionFalse,
		),
		v1beta1.NewCondition(
			v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionFalse,
		),
		v1beta1.NewCondition(
			v1beta1.HubUpgradeFailed, v1beta1.HubUpgradeFailed, metav1.ConditionFalse, metav1.ConditionFalse,
		),
	}
	hub.SetConditions(false, initConditions...)

	if previousPhase == "" {
		// set initial phase/conditions and requeue
		return ret(ctx, ctrl.Result{RequeueAfter: hubRequeuePreInit}, nil)
	}

	// Handle Hub cluster: initialization and/or upgrade
	if err := r.handleHub(ctx, hub, hubKubeconfig); err != nil {
		logger.Error(err, "Failed to handle hub operations")
		hub.Status.Phase = v1beta1.Unhealthy
	}
	hubInitializedCond := hub.GetCondition(v1beta1.HubInitialized)
	if hubInitializedCond == nil || hubInitializedCond.Status == metav1.ConditionFalse {
		return ret(ctx, ctrl.Result{RequeueAfter: hubRequeuePreInit}, nil)
	}

	// Finalize phase
	for _, c := range hub.Status.Conditions {
		if c.Status != c.WantStatus {
			logger.Info("WARNING: condition does not have the desired status", "type", c.Type, "reason", c.Reason, "message", c.Message, "status", c.Status, "wantStatus", c.WantStatus)
			hub.Status.Phase = v1beta1.Unhealthy
			return ret(ctx, ctrl.Result{RequeueAfter: hubRequeuePreInit}, nil)
		}
	}
	if hub.Status.Phase == v1beta1.HubStarting {
		hub.Status.Phase = v1beta1.HubRunning
	}

	return ret(ctx, ctrl.Result{RequeueAfter: hubRequeuePostInit}, nil)
}

type contextKey int

const (
	// originalHubKey is the key in the context that records the incoming original Hub
	originalHubKey contextKey = iota
)

func withOriginalHub(ctx context.Context, hub *v1beta1.Hub) context.Context {
	return context.WithValue(ctx, originalHubKey, hub.DeepCopy())
}

// cleanup cleans up a Hub and its associated resources.
func (r *HubReconciler) cleanHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanHub", "hub", hub.Name)

	// Check if there are any Spokes that need to be deleted
	spokeList := &v1beta1.SpokeList{}
	err := r.List(ctx, spokeList)
	if err != nil {
		return true, err
	}

	managedRemaining := 0
	for i := range spokeList.Items {
		s := &spokeList.Items[i]
		if !s.IsManagedBy(hub.ObjectMeta) {
			continue
		}
		if s.DeletionTimestamp.IsZero() {
			logger.Info("Marking Spoke for deletion", "spoke", s.Name)
			if err := r.Delete(ctx, s); err != nil && !kerrs.IsNotFound(err) {
				return true, fmt.Errorf("failed to delete spoke %s: %w", s.Name, err)
			}
		}
		// Count managed spokes until they're fully deleted
		managedRemaining++
	}
	if managedRemaining > 0 {
		logger.V(1).Info("Waiting for managed Spokes to be deleted before proceeding with Hub cleanup",
			"remainingSpokes", managedRemaining)
		return true, nil
	}

	logger.Info("All Spokes have been deleted, proceeding with Hub cleanup")

	operatorC, err := common.OperatorClient(hubKubeconfig)
	if err != nil {
		return true, fmt.Errorf("failed to create operator client for cleanup: %w", err)
	}
	_, err = operatorC.OperatorV1().ClusterManagers().Get(ctx, "cluster-manager", metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			logger.Info("ClusterManager not found; skip cleanup.", "hub", hub.Name)
			hub.Finalizers = slices.DeleteFunc(hub.Finalizers, func(s string) bool {
				return s == v1beta1.HubCleanupFinalizer
			})
			return false, nil
		}
		return true, fmt.Errorf("failed to get ClusterManager: %w", err)
	}

	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return true, fmt.Errorf("failed to create addon client for cleanup: %w", err)
	}

	hubCopy := hub.DeepCopy()
	hubCopy.Spec.AddOnConfigs = nil
	hubCopy.Spec.HubAddOns = nil
	_, err = handleAddonConfig(ctx, r.Client, addonC, hubCopy)
	if err != nil {
		return true, err
	}
	_, err = handleHubAddons(ctx, addonC, hubCopy)
	if err != nil {
		return true, err
	}

	purgeOperator := false
	if hub.Spec.ClusterManager != nil {
		purgeOperator = hub.Spec.ClusterManager.PurgeOperator
	}
	baseArgs := hub.BaseArgs()
	cleanArgs := make([]string, 0, 2+len(baseArgs))
	cleanArgs = append(cleanArgs,
		"clean",
		// name is omitted, as the default name, 'cluster-manager', is always used
		fmt.Sprintf("--purge-operator=%t", purgeOperator),
	)
	cleanArgs = append(cleanArgs, baseArgs...)

	logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(cleanArgs))
	cmd := exec.Command(clusteradm, cleanArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm clean' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return true, fmt.Errorf("failed to clean hub cluster: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("hub cleaned", "output", string(stdout))

	hub.Finalizers = slices.DeleteFunc(hub.Finalizers, func(s string) bool {
		return s == v1beta1.HubCleanupFinalizer
	})

	return false, nil
}

// handleHub manages Hub cluster init and upgrade operations
func (r *HubReconciler) handleHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleHub", "hub", hub.Name)

	operatorC, err := common.OperatorClient(hubKubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return err
	}
	cm, err := getClusterManager(ctx, operatorC)
	if err != nil {
		return err
	}

	mergedClusterManagerValues, currClusterManagerHash, valuesHashChanged, err := r.clusterManagerChartState(ctx, hub, cm != nil)
	if err != nil {
		return err
	}

	if err := r.ensureHubClusterManagerInitialized(ctx, hub, hubKubeconfig, cm, mergedClusterManagerValues); err != nil {
		return err
	}

	hub.SetConditions(true, v1beta1.NewCondition(
		v1beta1.HubInitialized, v1beta1.HubInitialized, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	if err := r.reconcileHubAddonConditions(ctx, hub, addonC); err != nil {
		return err
	}

	if err := r.maybeUpgradeHubClusterManager(ctx, hub, hubKubeconfig, operatorC, mergedClusterManagerValues, valuesHashChanged); err != nil {
		return err
	}

	if hub.Spec.ClusterManager != nil {
		hub.Status.ClusterManagerHash = currClusterManagerHash
	}

	if err := r.reconcileHubGRPCStatus(ctx, hub, hubKubeconfig); err != nil {
		return err
	}

	return nil
}

func (r *HubReconciler) clusterManagerChartState(ctx context.Context, hub *v1beta1.Hub, clusterManagerExists bool) (
	merged *v1beta1.ClusterManagerChartConfig,
	currHash string,
	valuesHashChanged bool,
	err error,
) {
	if hub.Spec.ClusterManager == nil {
		return nil, "", false, nil
	}
	merged, err = r.mergeClusterManagerValues(ctx, hub)
	if err != nil {
		return nil, "", false, err
	}
	if merged == nil {
		merged = &v1beta1.ClusterManagerChartConfig{}
	}
	// Fold feature gates into the merged values so a featureGates change alters the
	// hash (triggering an upgrade) and is carried in --cluster-manager-values-file.
	applyHubFeatureGates(merged, hub.Spec.ClusterManager.FeatureGates)
	currHash, err = hash.ComputeHash(merged)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to compute hash of hub %s cluster-manager values: %w", hub.Name, err)
	}
	prevHash := hub.Status.ClusterManagerHash
	valuesHashChanged = clusterManagerValuesUpgradeNeeded(prevHash, currHash, clusterManagerExists, merged)
	return merged, currHash, valuesHashChanged, nil
}

// clusterManagerValuesUpgradeNeeded reports whether clusteradm upgrade clustermanager should run
// to apply merged Helm values. When prevHash is empty, an upgrade is still required if the hub
// already has a ClusterManager and non-empty merged values—otherwise the controller could persist
// a new hash without ever passing --cluster-manager-values-file.
func clusterManagerValuesUpgradeNeeded(prevHash, currHash string, clusterManagerExists bool, merged *v1beta1.ClusterManagerChartConfig) bool {
	hasNonEmptyMergedValues := merged != nil && !merged.IsEmpty()
	return currHash != prevHash && (prevHash != "" || (clusterManagerExists && hasNonEmptyMergedValues))
}

func clusterManagerStatusProblems(cm *operatorv1.ClusterManager) []string {
	if cm == nil || cm.Status.Conditions == nil {
		return nil
	}
	var msgs []string
	for _, c := range cm.Status.Conditions {
		switch {
		case c.Type == operatorv1.ConditionProgressing && c.Status == metav1.ConditionTrue:
			msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
		case c.Type == operatorv1.ConditionClusterManagerApplied && c.Status == metav1.ConditionFalse:
			msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
		case c.Type == operatorv1.ConditionHubRegistrationDegraded && c.Status == metav1.ConditionTrue:
			msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
		case c.Type == operatorv1.ConditionHubPlacementDegraded && c.Status == metav1.ConditionTrue:
			msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
		}
	}
	return msgs
}

// ensureHubClusterManagerInitialized runs clusteradm init when no ClusterManager exists yet; otherwise checks for pending/degraded status.
func (r *HubReconciler) ensureHubClusterManagerInitialized(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte, cm *operatorv1.ClusterManager, merged *v1beta1.ClusterManagerChartConfig) error {
	if cm != nil {
		if len(cm.Status.Conditions) == 0 {
			msg := "waiting for ClusterManager status conditions"
			hub.SetConditions(true, v1beta1.NewCondition(
				msg, v1beta1.HubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return errors.New(msg)
		}
		if msgs := clusterManagerStatusProblems(cm); len(msgs) > 0 {
			msg := fmt.Sprintf("hub pending/degraded: %s", strings.TrimSuffix(strings.Join(msgs, "; "), "; "))
			hub.SetConditions(true, v1beta1.NewCondition(
				msg, v1beta1.HubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return errors.New(msg)
		}
		return nil
	}
	return r.initializeHub(ctx, hub, hubKubeconfig, merged)
}

func (r *HubReconciler) reconcileHubAddonConditions(ctx context.Context, hub *v1beta1.Hub, addonC *addonapi.Clientset) error {
	addonConfigChanged, err := handleAddonConfig(ctx, r.Client, addonC, hub)
	if err != nil && addonConfigChanged {
		hub.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	hubAddonChanged, err := handleHubAddons(ctx, addonC, hub)
	if err != nil && hubAddonChanged {
		hub.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	if addonConfigChanged || hubAddonChanged {
		hub.SetConditions(true, v1beta1.NewCondition(
			v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionTrue,
		))
	}
	return nil
}

func (r *HubReconciler) maybeUpgradeHubClusterManager(
	ctx context.Context,
	hub *v1beta1.Hub,
	hubKubeconfig []byte,
	operatorC *operatorapi.Clientset,
	merged *v1beta1.ClusterManagerChartConfig,
	valuesHashChanged bool,
) error {
	if hub.Spec.ClusterManager == nil {
		return nil
	}
	upgrade, err := r.hubNeedsUpgrade(ctx, hub, operatorC)
	if err != nil {
		hub.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.HubUpgradeFailed, metav1.ConditionTrue, metav1.ConditionFalse,
		))
		return fmt.Errorf("failed to check if hub needs upgrade: %w", err)
	}
	if upgrade || valuesHashChanged {
		if err := r.upgradeHub(ctx, hub, hubKubeconfig, merged); err != nil {
			hub.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.HubUpgradeFailed, metav1.ConditionTrue, metav1.ConditionFalse,
			))
			return fmt.Errorf("failed to upgrade hub: %w", err)
		}
	}

	hub.SetConditions(true, v1beta1.NewCondition(
		v1beta1.HubUpgradeFailed, v1beta1.HubUpgradeFailed, metav1.ConditionFalse, metav1.ConditionFalse,
	))

	return nil
}

// initializeHub initializes the Hub cluster via 'clusteradm init'
func (r *HubReconciler) initializeHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte, clusterManagerValues *v1beta1.ClusterManagerChartConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("initHub", "hub", hub.Name)

	initArgs := append([]string{
		"init",
		fmt.Sprintf("--create-namespace=%t", hub.Spec.CreateNamespace),
		fmt.Sprintf("--force=%t", hub.Spec.Force),
		"--wait=true",
	}, hub.BaseArgs()...)

	if hub.Spec.RegistrationAuth.Driver == v1beta1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-drivers=%s", hub.Spec.RegistrationAuth.Driver),
		}
		if hub.Spec.RegistrationAuth.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", hub.Spec.RegistrationAuth.HubClusterARN))
		}
		if len(hub.Spec.RegistrationAuth.AutoApprovedARNPatterns) > 0 {
			raArgs = append(raArgs, fmt.Sprintf("--auto-approved-arn-patterns=%s", strings.Join(hub.Spec.RegistrationAuth.AutoApprovedARNPatterns, ",")))
		}
		initArgs = append(initArgs, raArgs...)
	}

	if hub.Spec.RegistrationAuth.GRPCInitEnabled() {
		ra := hub.Spec.RegistrationAuth
		initArgs = append(initArgs, fmt.Sprintf("--registration-drivers=%s,%s", v1beta1.CSRRegistrationDriver, v1beta1.GRPCRegistrationDriver))
		et := v1beta1.GRPCEndpointTypeHostname
		if ra.GRPC != nil && ra.GRPC.EndpointType != "" {
			et = ra.GRPC.EndpointType
		}
		initArgs = append(initArgs, fmt.Sprintf("--grpc-endpoint-type=%s", et))
		if ra.GRPC != nil && ra.GRPC.Server != "" {
			initArgs = append(initArgs, "--grpc-server", ra.GRPC.Server)
		}
		if ra.GRPC != nil && len(ra.GRPC.AutoApprovedIdentities) > 0 {
			initArgs = append(initArgs, fmt.Sprintf("--auto-approved-grpc-identities=%s", strings.Join(ra.GRPC.AutoApprovedIdentities, ",")))
		}
	}

	if hub.Spec.SingletonControlPlane != nil {
		initArgs = append(initArgs, "--singleton=true")
		initArgs = append(initArgs, "--singleton-name", hub.Spec.SingletonControlPlane.Name)
		if hub.Spec.SingletonControlPlane.Helm != nil {
			if hub.Spec.SingletonControlPlane.Helm.Values != "" {
				values, cleanupValues, err := file.TmpFile([]byte(hub.Spec.SingletonControlPlane.Helm.Values), "values")
				if cleanupValues != nil {
					defer cleanupValues()
				}
				if err != nil {
					return err
				}
				initArgs = append(initArgs, "--values", values)
			}
			for _, s := range hub.Spec.SingletonControlPlane.Helm.Set {
				initArgs = append(initArgs, "--set", s)
			}
			for _, s := range hub.Spec.SingletonControlPlane.Helm.SetJSON {
				initArgs = append(initArgs, "--set-json", s)
			}
			for _, s := range hub.Spec.SingletonControlPlane.Helm.SetLiteral {
				initArgs = append(initArgs, "--set-literal", s)
			}
			for _, s := range hub.Spec.SingletonControlPlane.Helm.SetString {
				initArgs = append(initArgs, "--set-string", s)
			}
		}
	} else if hub.Spec.ClusterManager != nil {
		// clustermanager args
		// Omit --feature-gates entirely when no gates are specified.
		if hub.Spec.ClusterManager.FeatureGates != "" {
			initArgs = append(initArgs, "--feature-gates", hub.Spec.ClusterManager.FeatureGates)
		}
		initArgs = append(initArgs, fmt.Sprintf("--use-bootstrap-token=%t", hub.Spec.ClusterManager.UseBootstrapToken))
		// source args
		initArgs = append(initArgs, "--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion)
		initArgs = append(initArgs, "--image-registry", hub.Spec.ClusterManager.Source.Registry)
		// resources args
		initArgs = append(initArgs, arg_utils.PrepareResources(hub.Spec.ClusterManager.Resources)...)
		valuesArgs, valuesCleanup, err := prepareClusterManagerChartValuesArgs(clusterManagerValues)
		if valuesCleanup != nil {
			defer valuesCleanup()
		}
		if err != nil {
			return err
		}
		initArgs = append(initArgs, valuesArgs...)
	} else {
		// one of clusterManager or singletonControlPlane must be specified, per validating webhook, but handle the edge case anyway
		return fmt.Errorf("unknown hub type, must specify either hub.clusterManager or hub.singletonControlPlane")
	}

	initArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, hubKubeconfig, hub.Spec.Kubeconfig.Context, initArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm init", "args", arg_utils.SanitizeArgs(initArgs))

	cmd := exec.Command(clusteradm, initArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm init' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to init hub: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("hub initialized", "output", string(arg_utils.SanitizeOutput(stdout)))

	return nil
}

// hubNeedsUpgrade checks if the clustermanager on the Hub cluster has the desired bundle version
func (r *HubReconciler) hubNeedsUpgrade(ctx context.Context, hub *v1beta1.Hub, operatorC *operatorapi.Clientset) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("hubNeedsUpgrade", "hub", hub.Name)

	if hub.Spec.ClusterManager.Source.BundleVersion == v1beta1.BundleVersionDefault {
		logger.V(0).Info("clustermanager bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if hub.Spec.ClusterManager.Source.BundleVersion == v1beta1.BundleVersionLatest {
		logger.V(0).Info("clustermanager bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	cm, err := getClusterManager(ctx, operatorC)
	if err != nil {
		return false, err
	}

	// identify lowest bundleVersion referenced in the clustermanager spec
	bundleSpecs := make([]string, 0)
	if cm.Spec.AddOnManagerImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.AddOnManagerImagePullSpec)
	}
	if cm.Spec.PlacementImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.PlacementImagePullSpec)
	}
	if cm.Spec.RegistrationImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.RegistrationImagePullSpec)
	}
	if cm.Spec.WorkImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.WorkImagePullSpec)
	}
	// bundle version changed
	activeBundleVersion, err := version.LowestBundleVersion(ctx, bundleSpecs)
	if err != nil {
		return false, fmt.Errorf("failed to detect bundleVersion from clustermanager spec: %w", err)
	}
	desiredBundleVersion, err := version.Normalize(hub.Spec.ClusterManager.Source.BundleVersion)
	if err != nil {
		return false, err
	}
	versionChanged := activeBundleVersion != desiredBundleVersion

	// bundle source changed
	activeBundleSource, err := version.GetBundleSource(bundleSpecs)
	if err != nil {
		return false, fmt.Errorf("failed to get bundle source: %w", err)
	}
	desiredBundleSource := hub.Spec.ClusterManager.Source.Registry
	sourceChanged := activeBundleSource != desiredBundleSource

	logger.V(0).Info("found clustermanager bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", desiredBundleVersion,
		"activeBundleSource", activeBundleSource,
		"desiredBundleSource", desiredBundleSource,
	)

	return versionChanged || sourceChanged, nil
}

// upgradeHub upgrades the Hub cluster's clustermanager (version/image and/or chart values via --cluster-manager-values-file).
func (r *HubReconciler) upgradeHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte, clusterManagerValues *v1beta1.ClusterManagerChartConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeHub", "hub", hub.Name)

	upgradeArgs := append([]string{
		"upgrade", "clustermanager",
		"--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion,
		"--image-registry", hub.Spec.ClusterManager.Source.Registry,
		"--wait=true",
	}, hub.BaseArgs()...)

	valuesArgs, valuesCleanup, err := prepareClusterManagerChartValuesArgs(clusterManagerValues)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	upgradeArgs = append(upgradeArgs, valuesArgs...)

	upgradeArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, hubKubeconfig, hub.Spec.Kubeconfig.Context, upgradeArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm upgrade clustermanager", "args", arg_utils.SanitizeArgs(upgradeArgs))

	cmd := exec.Command(clusteradm, upgradeArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm upgrade clustermanager' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf(
			"failed to upgrade hub clustermanager to %s: %v, output: %s",
			hub.Spec.ClusterManager.Source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("clustermanager upgraded", "output", string(stdout))

	return nil
}

// mergeClusterManagerValues merges chart values from a ConfigMap in the Hub namespace and from the Hub spec. Spec takes precedence.
// If valuesFrom is set, the referenced ConfigMap and data key must exist or an error is returned.
func (r *HubReconciler) mergeClusterManagerValues(ctx context.Context, hub *v1beta1.Hub) (*v1beta1.ClusterManagerChartConfig, error) {
	logger := log.FromContext(ctx)
	cmSpec := hub.Spec.ClusterManager
	if cmSpec == nil {
		return nil, nil
	}
	if cmSpec.ValuesFrom == nil && cmSpec.Values == nil {
		logger.V(3).Info("no cluster-manager values or valuesFrom provided", "hub", hub.Name)
		return nil, nil
	}

	var fromInterface = map[string]any{}
	var specInterface = map[string]any{}

	if cmSpec.ValuesFrom != nil {
		configMap := &corev1.ConfigMap{}
		nn := types.NamespacedName{Name: cmSpec.ValuesFrom.Name, Namespace: hub.Namespace}
		err := r.Get(ctx, nn, configMap)
		if err != nil {
			if kerrs.IsNotFound(err) {
				return nil, fmt.Errorf(
					"cluster-manager valuesFrom references missing ConfigMap %s/%s for hub %s",
					hub.Namespace, cmSpec.ValuesFrom.Name, hub.Name,
				)
			}
			return nil, fmt.Errorf("failed to retrieve cluster-manager values ConfigMap %s: %w", nn, err)
		}
		fromValues, ok := configMap.Data[cmSpec.ValuesFrom.Key]
		if !ok {
			return nil, fmt.Errorf(
				"cluster-manager valuesFrom key %q not found in ConfigMap %s/%s for hub %s",
				cmSpec.ValuesFrom.Key, hub.Namespace, cmSpec.ValuesFrom.Name, hub.Name,
			)
		}
		if err := yaml.Unmarshal([]byte(fromValues), &fromInterface); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML values from ConfigMap %s key %s: %w", nn, cmSpec.ValuesFrom.Key, err)
		}
	}

	if cmSpec.Values != nil {
		specBytes, err := yaml.Marshal(cmSpec.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster-manager values from hub spec for hub %s: %w", hub.Name, err)
		}
		if err := yaml.Unmarshal(specBytes, &specInterface); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cluster-manager values from hub spec for hub %s: %w", hub.Name, err)
		}
	}

	mergedMap := map[string]any{}
	maps.Copy(mergedMap, fromInterface)

	if err := mergo.Map(&mergedMap, specInterface, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merge cluster-manager values failed for hub %s: %w", hub.Name, err)
	}

	mergedBytes, err := yaml.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged cluster-manager values for hub %s: %w", hub.Name, err)
	}

	merged := &v1beta1.ClusterManagerChartConfig{}
	if err := yaml.Unmarshal(mergedBytes, merged); err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged values into ClusterManagerChartConfig for hub %s: %w", hub.Name, err)
	}

	return merged, nil
}

// prepareClusterManagerChartValuesArgs writes chart values to a temp file and returns
// --cluster-manager-values-file <path> for clusteradm init and clusteradm upgrade clustermanager.
func prepareClusterManagerChartValuesArgs(values *v1beta1.ClusterManagerChartConfig) ([]string, func(), error) {
	if values == nil {
		return nil, nil, nil
	}
	if values.IsEmpty() {
		return nil, nil, nil
	}
	valuesYAML, err := marshalClusterManagerValues(values)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal cluster-manager values to YAML: %w", err)
	}
	valuesFile, valuesCleanup, err := file.TmpFile(valuesYAML, "clustermanager-values")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write cluster-manager values to disk: %w", err)
	}
	return []string{"--cluster-manager-values-file", valuesFile}, valuesCleanup, nil
}

// marshalClusterManagerValues serializes the cluster-manager chart values, dropping
// fields that would otherwise be written as invalid zero values and clobber settings
// derived from clusteradm flags (on init) or carried forward from the live
// ClusterManager CR (on upgrade, where --cluster-manager-values-file is merged last).
//
// ClusterManagerConfig.ResourceRequirement is a non-pointer struct whose Type field
// has no omitempty, so it always marshals as `resourceRequirement: {type: ""}`. An
// empty QoS type is rejected by the ClusterManager webhook, so when it is unset the
// whole resourceRequirement block is pruned, leaving --resource-qos-class / the CR
// value intact.
func marshalClusterManagerValues(values *v1beta1.ClusterManagerChartConfig) ([]byte, error) {
	raw, err := yaml.Marshal(values)
	if err != nil {
		return nil, err
	}
	asMap := map[string]any{}
	if err := yaml.Unmarshal(raw, &asMap); err != nil {
		return nil, err
	}
	pruneEmptyResourceRequirement(asMap)
	return yaml.Marshal(asMap)
}

// pruneEmptyResourceRequirement removes clusterManager.resourceRequirement from the
// values map when its QoS type is empty. See marshalClusterManagerValues.
func pruneEmptyResourceRequirement(values map[string]any) {
	clusterManager, ok := values["clusterManager"].(map[string]any)
	if !ok {
		return
	}
	resourceRequirement, ok := clusterManager["resourceRequirement"].(map[string]any)
	if !ok {
		return
	}
	if qosType, _ := resourceRequirement["type"].(string); qosType == "" {
		delete(clusterManager, "resourceRequirement")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Hub{}).
		// watch for deleted Spokes to prevent idly waiting after all spokes have been GCd during Hub deletion
		Watches(
			&v1beta1.Spoke{},
			handler.EnqueueRequestsFromMapFunc(r.mapSpokeEventToHub),
			builder.WithPredicates(
				predicate.Funcs{
					DeleteFunc: func(_ event.DeleteEvent) bool {
						return true
					},
					CreateFunc: func(_ event.CreateEvent) bool {
						return false
					},
					UpdateFunc: func(_ event.UpdateEvent) bool {
						return false
					},
					GenericFunc: func(_ event.GenericEvent) bool {
						return false
					},
				},
			),
		).
		Named("hub").
		Complete(r)
}

func (r *HubReconciler) mapSpokeEventToHub(_ context.Context, obj client.Object) []reconcile.Request {
	spoke, ok := obj.(*v1beta1.Spoke)
	if !ok {
		r.Log.V(1).Info("failed to enqueue hub requests", "expected", "spoke", "got", fmt.Sprintf("%T", obj))
		return nil
	}
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      spoke.Spec.HubRef.Name,
				Namespace: spoke.Spec.HubRef.Namespace,
			},
		},
	}
}
