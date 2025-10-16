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
	"os/exec"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	arg_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
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
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, hub.Spec.Kubeconfig, hub.Namespace)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Handle deletion logic with finalizer
	if !hub.DeletionTimestamp.IsZero() {
		if hub.Status.Phase != v1beta1.Deleting {
			hub.Status.Phase = v1beta1.Deleting
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}

		if slices.Contains(hub.Finalizers, v1beta1.HubCleanupFinalizer) {
			if err := r.cleanHub(ctx, hub, hubKubeconfig); err != nil {
				hub.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return ret(ctx, ctrl.Result{}, err)
			}
		}
		hub.Finalizers = slices.DeleteFunc(hub.Finalizers, func(s string) bool {
			return s == v1beta1.HubCleanupFinalizer
		})
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
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	// Handle Hub cluster: initialization and/or upgrade
	if err := r.handleHub(ctx, hub, hubKubeconfig); err != nil {
		logger.Error(err, "Failed to handle hub operations")
		hub.Status.Phase = v1beta1.Unhealthy
	}
	hubInitializedCond := hub.GetCondition(v1beta1.HubInitialized)
	if hubInitializedCond == nil || hubInitializedCond.Status == metav1.ConditionFalse {
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	// Finalize phase
	for _, c := range hub.Status.Conditions {
		if c.Status != c.WantStatus {
			logger.Info("WARNING: condition does not have the desired status", "type", c.Type, "reason", c.Reason, "message", c.Message, "status", c.Status, "wantStatus", c.WantStatus)
			hub.Status.Phase = v1beta1.Unhealthy
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}
	}
	if hub.Status.Phase == v1beta1.HubStarting {
		hub.Status.Phase = v1beta1.HubRunning
	}

	return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
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
func (r *HubReconciler) cleanHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanHub", "hub", hub.Name)

	// Check if there are any Spokes that need to be deleted
	spokeList := &v1beta1.SpokeList{}
	err := r.List(ctx, spokeList)
	if err != nil {
		return err
	}

	spokes := spokeList.Items
	if len(spokes) > 0 {
		// Mark all Spokes for deletion if they haven't been deleted yet
		for i := range spokes {
			spoke := &spokes[i]
			if spoke.DeletionTimestamp.IsZero() {
				if !spoke.IsManagedBy(hub.ObjectMeta) {
					continue
				}
				logger.Info("Marking Spoke for deletion", "spoke", spoke.Name)
				if err := r.Delete(ctx, spoke); err != nil && !kerrs.IsNotFound(err) {
					return fmt.Errorf("failed to delete spoke %s: %w", spoke.Name, err)
				}
			}
		}

		logger.V(1).Info("Waiting for all Spokes to be deleted before proceeding with Hub cleanup",
			"remainingSpokes", len(spokes))
		// Return a retriable error to requeue and check again later
		return fmt.Errorf("waiting for background spoke deletion. Remaining: %d spokes", len(spokes))
	}

	logger.Info("All Spokes have been deleted, proceeding with Hub cleanup")

	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create addon client for cleanup: %w", err)
	}

	hubCopy := hub.DeepCopy()
	hubCopy.Spec.AddOnConfigs = nil
	hubCopy.Spec.HubAddOns = nil
	_, err = handleAddonConfig(ctx, r.Client, addonC, hubCopy)
	if err != nil {
		return err
	}
	_, err = handleHubAddons(ctx, addonC, hubCopy)
	if err != nil {
		return err
	}

	purgeOperator := false
	if hub.Spec.ClusterManager != nil {
		purgeOperator = hub.Spec.ClusterManager.PurgeOperator
	}
	cleanArgs := []string{
		"clean",
		// name is omitted, as the default name, 'cluster-manager', is always used
		fmt.Sprintf("--purge-operator=%t", purgeOperator),
	}
	cleanArgs = append(cleanArgs, hub.BaseArgs()...)

	logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(cleanArgs))
	cmd := exec.Command(clusteradm, cleanArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm clean' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to clean hub cluster: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("hub cleaned", "output", string(stdout))

	return nil

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

	// if a clustermanager already exists, we don't need to init the hub
	if cm != nil && cm.Status.Conditions != nil {
		msgs := make([]string, 0)
		for _, c := range cm.Status.Conditions {
			if c.Type == operatorv1.ConditionProgressing && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionClusterManagerApplied && c.Status == metav1.ConditionFalse {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionHubRegistrationDegraded && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionHubPlacementDegraded && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
		}
		if len(msgs) > 0 {
			msg := strings.TrimSuffix(strings.Join(msgs, "; "), "; ")
			msg = fmt.Sprintf("hub pending/degraded: %s", msg)
			hub.SetConditions(true, v1beta1.NewCondition(
				msg, v1beta1.HubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return errors.New(msg)
		}
	} else {
		if err := r.initializeHub(ctx, hub, hubKubeconfig); err != nil {
			return err
		}
	}

	hub.SetConditions(true, v1beta1.NewCondition(
		v1beta1.HubInitialized, v1beta1.HubInitialized, metav1.ConditionTrue, metav1.ConditionTrue,
	))

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

	// attempt an upgrade whenever the clustermanager's bundleVersion changes
	if hub.Spec.ClusterManager != nil {
		upgrade, err := r.hubNeedsUpgrade(ctx, hub, operatorC)
		if err != nil {
			hub.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.HubUpgradeFailed, metav1.ConditionTrue, metav1.ConditionFalse,
			))
			return fmt.Errorf("failed to check if hub needs upgrade: %w", err)
		}
		if upgrade {
			err = r.upgradeHub(ctx, hub)
			if err != nil {
				hub.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.HubUpgradeFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return fmt.Errorf("failed to upgrade hub: %w", err)
			}
		}
		hub.SetConditions(true, v1beta1.NewCondition(
			v1beta1.HubUpgradeFailed, v1beta1.HubUpgradeFailed, metav1.ConditionFalse, metav1.ConditionFalse,
		))
	}

	return nil
}

// initializeHub initializes the Hub cluster via 'clusteradm init'
func (r *HubReconciler) initializeHub(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte) error {
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
		initArgs = append(initArgs, "--feature-gates", hub.Spec.ClusterManager.FeatureGates)
		initArgs = append(initArgs, fmt.Sprintf("--use-bootstrap-token=%t", hub.Spec.ClusterManager.UseBootstrapToken))
		// source args
		initArgs = append(initArgs, "--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion)
		initArgs = append(initArgs, "--image-registry", hub.Spec.ClusterManager.Source.Registry)
		// resources args
		initArgs = append(initArgs, arg_utils.PrepareResources(hub.Spec.ClusterManager.Resources)...)
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

// upgradeHub upgrades the Hub cluster's clustermanager to the specified version
func (r *HubReconciler) upgradeHub(ctx context.Context, hub *v1beta1.Hub) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeHub", "hub", hub.Name)

	upgradeArgs := append([]string{
		"upgrade", "clustermanager",
		"--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion,
		"--image-registry", hub.Spec.ClusterManager.Source.Registry,
		"--wait=true",
	}, hub.BaseArgs()...)

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
