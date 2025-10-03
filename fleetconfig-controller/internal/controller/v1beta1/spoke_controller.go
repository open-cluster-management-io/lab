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
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

	"github.com/go-logr/logr"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/watch"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// SpokeReconciler reconciles a Spoke object
type SpokeReconciler struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	ConcurrentReconciles int
	InstanceType         string
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

	switch r.InstanceType {
	case v1beta1.InstanceTypeManager:
		if !slices.Contains(spoke.Finalizers, v1beta1.HubCleanupFinalizer) {
			setDefaults(ctx, spoke, hubMeta)
			spoke.Finalizers = append(
				spoke.Finalizers,
				v1beta1.HubCleanupPreflightFinalizer, // removed by the hub to signal to the spoke that preflight is completed
				v1beta1.HubCleanupFinalizer,          // removed by the hub after post-unjoin cleanup is finished
			)
			if spoke.IsHubAsSpoke() {
				spoke.Finalizers = append(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer)
			}
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}
	case v1beta1.InstanceTypeUnified:
		if !slices.Contains(spoke.Finalizers, v1beta1.HubCleanupFinalizer) {
			setDefaults(ctx, spoke, hubMeta)
			spoke.Finalizers = append(
				spoke.Finalizers,
				v1beta1.HubCleanupPreflightFinalizer, // removed by the hub to signal to the spoke that preflight is completed
				v1beta1.SpokeCleanupFinalizer,        // removed by the hub after successful unjoin
				v1beta1.HubCleanupFinalizer,          // removed by the hub after post-unjoin cleanup is finished
			)
		}
	case v1beta1.InstanceTypeAgent:
		if !slices.Contains(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer) && spoke.DeletionTimestamp.IsZero() {
			spoke.Finalizers = append(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer) // removed by the spoke to signal to the hub that unjoin succeeded
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}
	default:
		// this is guarded against when the manager is initialized. should never reach this point
		panic(fmt.Sprintf("unknown instance type %s. Must be one of %v", r.InstanceType, v1beta1.SupportedInstanceTypes))
	}

	// Handle deletion logic with finalizer
	if !spoke.DeletionTimestamp.IsZero() {
		if spoke.Status.Phase != v1beta1.Deleting {
			spoke.Status.Phase = v1beta1.Deleting
			return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
		}

		// HubCleanupFinalizer is the last finalizer to be removed
		if slices.Contains(spoke.Finalizers, v1beta1.HubCleanupFinalizer) {
			if err := r.cleanup(ctx, spoke, hubMeta.kubeconfig); err != nil {
				spoke.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
				))
				return ret(ctx, ctrl.Result{}, err)
			}
		}
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
		v1beta1.NewCondition(
			v1beta1.PivotComplete, v1beta1.PivotComplete, metav1.ConditionFalse, metav1.ConditionTrue,
		),
		v1beta1.NewCondition(
			v1beta1.KlusterletSynced, v1beta1.KlusterletSynced, metav1.ConditionFalse, metav1.ConditionFalse,
		),
	}
	spoke.SetConditions(false, initConditions...)

	if previousPhase == "" {
		// set initial phase/conditions and requeue
		return ret(ctx, ctrl.Result{RequeueAfter: requeue}, nil)
	}

	// Handle Spoke cluster: join and/or upgrade
	if err := r.handleSpoke(ctx, spoke, hubMeta); err != nil {
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

// SetupWithManagerForHub sets up the controller with the Manager to run on a Hub cluster.
func (r *SpokeReconciler) SetupWithManagerForHub(mgr ctrl.Manager) error {
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
					return sharedFieldsChanged(oldHub.Spec.DeepCopy(), newHub.Spec.DeepCopy()) ||
						!reflect.DeepEqual(oldHub.Status, newHub.Status)
				},
				GenericFunc: func(_ event.GenericEvent) bool {
					return false
				},
			}),
		).
		Named("spoke").
		Complete(r)
}

// SetupWithManagerForSpoke sets up the controller with the Manager to run on a Spoke cluster.
func (r *SpokeReconciler) SetupWithManagerForSpoke(mgr ctrl.Manager) error {
	spokeName := os.Getenv(v1beta1.SpokeNameEnvVar) // we know this is set, because the mgr setup would have failed otherwise

	// set up a watcher that is independent of the reconcile loop, so that we can delete the controller AMW after the Spoke is fully deleted
	watcher := watch.NewOrDie(watch.Config{
		Client:    mgr.GetClient(),
		Log:       r.Log.WithName(v1beta1.AgentCleanupWatcherName),
		Interval:  requeue,
		Timeout:   10 * time.Second,
		Name:      v1beta1.AgentCleanupWatcherName,
		Condition: spokeDeletedCondition,
		Handler:   agentSelfDestruct,
	})

	err := mgr.Add(watcher)
	if err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Spoke{},
			builder.WithPredicates(predicate.NewPredicateFuncs(
				func(object client.Object) bool {
					return object.GetName() == spokeName // no need to include namespace in the predicate, since the manager cache is configured to only watch 1 namespace
				},
			)),
		).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.ConcurrentReconciles,
		}).
		Named("spoke").
		Complete(r)
}

func spokeDeletedCondition(ctx context.Context, c client.Client) (bool, error) {
	spokeName := os.Getenv(v1beta1.SpokeNameEnvVar)
	spokeNamespace := os.Getenv(v1beta1.SpokeNamespaceEnvVar)
	spoke := &v1beta1.Spoke{}
	err := c.Get(ctx, types.NamespacedName{
		Name:      spokeName,
		Namespace: spokeNamespace,
	}, spoke)

	// condition is met when resource is NOT found
	return kerrs.IsNotFound(err), client.IgnoreNotFound(err)
}

func agentSelfDestruct(ctx context.Context, _ client.Client) error {
	spokeKubeconfig, err := kube.RawFromInClusterRestConfig()
	if err != nil {
		return err
	}
	workClient, err := common.WorkClient(spokeKubeconfig)
	if err != nil {
		return err
	}
	restCfg, err := kube.RestConfigFromKubeconfig(spokeKubeconfig)
	if err != nil {
		return err
	}
	spokeClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		return err
	}

	purgeNamespaceStr := os.Getenv(v1beta1.PurgeAgentNamespaceEnvVar)
	purgeNamespace, err := strconv.ParseBool(purgeNamespaceStr)
	if err != nil {
		// fall back to orphaning the namespace which is less destructive
		purgeNamespace = false
	}

	if purgeNamespace {
		agentNamespace := os.Getenv(v1beta1.ControllerNamespaceEnvVar) // manager.go enforces that this is not ""
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: agentNamespace}}
		err := spokeClient.Delete(ctx, ns)
		if err != nil && !kerrs.IsNotFound(err) {
			return err
		}
	}
	return workClient.WorkV1().AppliedManifestWorks().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{})
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
