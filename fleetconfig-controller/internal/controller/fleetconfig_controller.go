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

package controller

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-logr/logr"
	certificatesv1 "k8s.io/api/certificates/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

const (
	clusteradm = "clusteradm"
	requeue    = 30 * time.Second
)

type contextKey int

const (
	// originalFleetConfigKey is the key in the context that records the incoming original FleetConfig
	originalFleetConfigKey contextKey = iota
)

func withOriginalFleetConfig(ctx context.Context, mc *v1alpha1.FleetConfig) context.Context {
	return context.WithValue(ctx, originalFleetConfigKey, mc.DeepCopy())
}

// FleetConfigReconciler reconciles a FleetConfig object
type FleetConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles a FleetConfig object
func (r *FleetConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("request", req)
	ctx = log.IntoContext(ctx, logger)

	// Fetch the FleetConfig instance
	mc := &v1alpha1.FleetConfig{}
	err := r.Get(ctx, req.NamespacedName, mc)
	if err != nil {
		if !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to fetch FleetConfig", "key", req)
		}
		return ret(ctx, ctrl.Result{}, client.IgnoreNotFound(err))
	}
	ctx = withOriginalFleetConfig(ctx, mc)

	// Create a patch helper for this reconciliation
	patchHelper, err := patch.NewHelper(mc, r.Client)
	if err != nil {
		return ret(ctx, ctrl.Result{}, err)
	}

	// Ensure patch is applied at the end
	defer func() {
		if err := patchHelper.Patch(ctx, mc); err != nil && !kerrs.IsNotFound(err) {
			logger.Error(err, "failed to patch FleetConfig")
		}
	}()

	// Add a finalizer and requeue if not already present
	if !slices.Contains(mc.Finalizers, v1alpha1.FleetConfigFinalizer) {
		mc.Finalizers = append(mc.Finalizers, v1alpha1.FleetConfigFinalizer)
		return ret(ctx, ctrl.Result{Requeue: true}, nil)
	}

	// Handle deletion logic with finalizer
	if !mc.DeletionTimestamp.IsZero() {
		if mc.Status.Phase != v1alpha1.FleetConfigDeleting {
			mc.Status.Phase = v1alpha1.FleetConfigDeleting
			return ret(ctx, ctrl.Result{Requeue: true}, nil)
		}

		if slices.Contains(mc.Finalizers, v1alpha1.FleetConfigFinalizer) {
			if err := r.cleanup(ctx, mc); err != nil {
				mc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), v1alpha1.FleetConfigCleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return ret(ctx, ctrl.Result{}, err)
			}
		}
		// end reconciliation
		return ret(ctx, ctrl.Result{}, nil)
	}

	// Initialize phase & conditions
	previousPhase := mc.Status.Phase
	mc.Status.Phase = v1alpha1.FleetConfigStarting
	initConditions := []v1alpha1.Condition{
		v1alpha1.NewCondition(
			v1alpha1.FleetConfigHubInitialized, v1alpha1.FleetConfigHubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
		),
		v1alpha1.NewCondition(
			v1alpha1.FleetConfigCleanupFailed, v1alpha1.FleetConfigCleanupFailed, metav1.ConditionFalse, metav1.ConditionFalse,
		),
	}
	for _, s := range mc.Spec.Spokes {
		initConditions = append(
			initConditions, v1alpha1.NewCondition("", s.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue))
	}
	mc.SetConditions(false, initConditions...)

	if previousPhase == "" {
		// set initial phase/conditions and requeue
		return ret(ctx, ctrl.Result{Requeue: true}, nil)
	}

	// Handle Hub cluster: initialization and/or upgrade
	hubInitializedCond := mc.GetCondition(v1alpha1.FleetConfigHubInitialized)
	if err := handleHub(ctx, r.Client, mc); err != nil {
		logger.Error(err, "Failed to handle hub operations")
		mc.Status.Phase = v1alpha1.FleetConfigUnhealthy
	}
	if hubInitializedCond.Status == metav1.ConditionFalse {
		return ret(ctx, ctrl.Result{Requeue: true}, nil)
	}

	// Handle Spoke clusters: join and/or upgrade
	if err := handleSpokes(ctx, r.Client, mc); err != nil {
		logger.Error(err, "Failed to handle spoke operations")
		mc.Status.Phase = v1alpha1.FleetConfigUnhealthy
	}

	// Finalize phase
	for _, c := range mc.Status.Conditions {
		if c.Status != c.WantStatus {
			logger.Info("WARNING: condition does not have the desired status", "type", c.Type, "reason", c.Reason, "message", c.Message, "status", c.Status, "wantStatus", c.WantStatus)
			mc.Status.Phase = v1alpha1.FleetConfigUnhealthy
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}
	}
	if mc.Status.Phase == v1alpha1.FleetConfigStarting {
		mc.Status.Phase = v1alpha1.FleetConfigRunning
	}

	return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
}

func ret(ctx context.Context, res ctrl.Result, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if res.RequeueAfter > 0 {
		logger.Info("requeueing", "after", res.RequeueAfter)
	}
	if err != nil {
		logger.Info("requeueing due to error")
	}
	if res.RequeueAfter == 0 && err == nil {
		logger.Info("reconciliation complete; no requeue or error")
	}
	return res, err
}

// cleanup cleans up a FleetConfig and its associated resources.
func (r *FleetConfigReconciler) cleanup(ctx context.Context, mc *v1alpha1.FleetConfig) error {
	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, mc.Spec.Hub.Kubeconfig)
	if err != nil {
		return err
	}

	enabledFeatureGates := common.ExtractFeatureGates(mc)

	doCleanup, err := cleanupPreflight(ctx, hubKubeconfig, enabledFeatureGates)
	if err != nil {
		return err
	}
	if doCleanup {
		if err := cleanupSpokes(ctx, r.Client, mc); err != nil {
			return err
		}
		if err := cleanHub(ctx, r.Client, hubKubeconfig, mc); err != nil {
			return err
		}
		if err := r.DeleteAllOf(ctx, &certificatesv1.CertificateSigningRequest{},
			client.HasLabels{"open-cluster-management.io/cluster-name"},
		); err != nil {
			return err
		}
	}
	mc.Finalizers = slices.DeleteFunc(mc.Finalizers, func(s string) bool {
		return s == v1alpha1.FleetConfigFinalizer
	})
	return nil
}

// cleanupPreflight performs preflight checks before attempting FleetConfig cleanup.
func cleanupPreflight(ctx context.Context, hubKubeconfig []byte, enabledFeatureGates map[featuregate.Feature]bool) (bool, error) {
	logger := log.FromContext(ctx)

	clusterC, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return false, err
	}
	workC, err := common.WorkClient(hubKubeconfig)
	if err != nil {
		return false, err
	}

	resourceCleanupEnabled := enabledFeatureGates[ocmfeature.ResourceCleanup]

	// skip clean up if the ManagedCluster resource is not found or if any manifestWorks exist
	managedClusters, err := clusterC.ClusterV1().ManagedClusters().List(ctx, metav1.ListOptions{})
	if kerrs.IsNotFound(err) {
		logger.Info("ManagedCluster resource not found; nothing to do")
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("unexpected error listing managedClusters: %w", err)
	}
	for _, managedCluster := range managedClusters.Items {
		manifestWorks, err := workC.WorkV1().ManifestWorks(managedCluster.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to list manifestWorks for managedCluster %s: %w", managedCluster.Name, err)
		}
		// If resourceCleanup is not enabled and there are manifestWorks, return false with an error message
		if len(manifestWorks.Items) > 0 && !resourceCleanupEnabled {
			msg := fmt.Sprintf("Found manifestWorks for ManagedCluster %s; cannot clean hub while any ManagedClusters have active ManifestWorks", managedCluster.Name)
			logger.Info(msg)
			return false, errors.New(msg)
		}
	}

	return true, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FleetConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FleetConfig{}).
		Complete(r)
}
