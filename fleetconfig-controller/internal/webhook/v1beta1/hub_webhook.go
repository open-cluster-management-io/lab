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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"open-cluster-management.io/api/client/addon/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// nolint:unused
// log is for logging in this package.
var hublog = logf.Log.WithName("hub-resource")

// SetupHubWebhookWithManager registers the webhook for Hub in the manager.
func SetupHubWebhookWithManager(mgr ctrl.Manager) error {
	kubeconfig, err := kube.RawFromInClusterRestConfig()
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(kubeconfig)
	if err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr).For(&v1beta1.Hub{}).
		WithValidator(&HubCustomValidator{client: mgr.GetClient(), addonC: addonC}).
		Complete()
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-fleetconfig-open-cluster-management-io-v1beta1-hub,mutating=false,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=hubs,verbs=create;update,versions=v1beta1,name=vhub-v1beta1.kb.io,admissionReviewVersions=v1

// HubCustomValidator struct is responsible for validating the Hub resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type HubCustomValidator struct {
	client client.Client
	addonC *versioned.Clientset
}

var _ webhook.CustomValidator = &HubCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	hub, ok := obj.(*v1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object but got %T", obj)
	}
	hublog.Info("Validation for Hub upon creation", "name", hub.GetName())

	var allErrs field.ErrorList

	if valid, msg := isKubeconfigValid(hub.Spec.Kubeconfig); !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), hub.Spec.Kubeconfig, msg),
		)
	}
	if hub.Spec.ClusterManager == nil && hub.Spec.SingletonControlPlane == nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), hub.Spec, "either hub.clusterManager or hub.singletonControlPlane must be specified"),
		)
	}

	if hub.Spec.ClusterManager != nil && hub.Spec.SingletonControlPlane != nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), hub.Spec, "only one of hub.clusterManager or hub.singletonControlPlane may be specified"),
		)
	}
	allErrs = append(allErrs, validateHubAddons(ctx, v.client, nil, hub, v.addonC)...)

	if len(allErrs) > 0 {
		return nil, errors.NewInvalid(v1beta1.HubGroupKind, hub.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	hub, ok := newObj.(*v1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object for the newObj but got %T", newObj)
	}
	oldHub, ok := oldObj.(*v1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object for the oldObj but got %T", oldObj)
	}
	hublog.Info("Validation for Hub upon update", "name", hub.GetName())

	var allErrs field.ErrorList

	err := allowHubUpdate(oldHub, hub)
	if err != nil {
		return nil, err
	}

	if valid, msg := isKubeconfigValid(hub.Spec.Kubeconfig); !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), hub.Spec.Kubeconfig, msg),
		)
	}
	allErrs = append(allErrs, validateHubAddons(ctx, v.client, oldHub, hub, v.addonC)...)

	if len(allErrs) > 0 {
		return nil, errors.NewInvalid(v1beta1.HubGroupKind, hub.Name, allErrs)
	}
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	hub, ok := obj.(*v1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object but got %T", obj)
	}
	hublog.Info("Validation for Hub upon deletion", "name", hub.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
