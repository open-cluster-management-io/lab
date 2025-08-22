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

// Package v1beta1 contains the main reconciliation logic for fleetconfig-controller's v1alpha1 resources.
package v1beta1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// HubReconciler reconciles a Hub object
type HubReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=fleetconfig.open-cluster-management.io,resources=hubs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Hub object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *HubReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HubReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Hub{}).
		Named("hub").
		Complete(r)
}
