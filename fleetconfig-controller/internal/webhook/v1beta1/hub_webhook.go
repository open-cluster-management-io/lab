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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

var hublog = logf.Log.WithName("hub-resource")

// SetupHubWebhookWithManager registers the webhook for Hub in the manager.
func SetupHubWebhookWithManager(mgr ctrl.Manager) error {
	addonC, err := addonapi.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr, &v1beta1.Hub{}).
		WithValidator(&HubCustomValidator{client: mgr.GetClient(), addonC: addonC}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-fleetconfig-open-cluster-management-io-v1beta1-hub,mutating=false,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=hubs,verbs=create;update,versions=v1beta1,name=vhub-v1beta1.kb.io,admissionReviewVersions=v1

// HubCustomValidator struct is responsible for validating the Hub resource
// when it is created, updated, or deleted.
type HubCustomValidator struct {
	client client.Client
	addonC *addonapi.Clientset
}

var _ admission.Validator[*v1beta1.Hub] = &HubCustomValidator{}

// ValidateCreate implements admission.Validator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateCreate(ctx context.Context, hub *v1beta1.Hub) (admission.Warnings, error) {
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
	allErrs = append(allErrs, validateHubRegistrationAuth(hub)...)
	allErrs = append(allErrs, validateHubAddons(ctx, v.client, nil, hub, v.addonC)...)

	if len(allErrs) > 0 {
		return nil, errors.NewInvalid(v1beta1.HubGroupKind, hub.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements admission.Validator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateUpdate(ctx context.Context, oldHub, hub *v1beta1.Hub) (admission.Warnings, error) {
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
	allErrs = append(allErrs, validateHubRegistrationAuth(hub)...)
	allErrs = append(allErrs, validateHubAddons(ctx, v.client, oldHub, hub, v.addonC)...)

	if len(allErrs) > 0 {
		return nil, errors.NewInvalid(v1beta1.HubGroupKind, hub.Name, allErrs)
	}
	return nil, nil
}

// ValidateDelete implements admission.Validator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateDelete(_ context.Context, hub *v1beta1.Hub) (admission.Warnings, error) {
	hublog.Info("Validation for Hub upon deletion", "name", hub.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
