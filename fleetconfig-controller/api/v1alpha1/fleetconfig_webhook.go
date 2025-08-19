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

package v1alpha1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var log = logf.Log.WithName("fleetconfig-resource")

// SetupFleetConfigWebhookWithManager registers the webhook for FleetConfig in the manager.
func SetupFleetConfigWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&FleetConfig{}).
		WithDefaulter(&FleetConfigCustomDefaulter{}).
		WithValidator(&FleetConfigCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-fleetconfig-open-cluster-management-io-v1alpha1-fleetconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=fleetconfigs,verbs=create;update,versions=v1alpha1,name=mfleetconfig-v1alpha1.open-cluster-management.io,admissionReviewVersions=v1

// FleetConfigCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind FleetConfig when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type FleetConfigCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &FleetConfigCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind FleetConfig.
func (d *FleetConfigCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	fc, ok := obj.(*FleetConfig)

	if !ok {
		return fmt.Errorf("expected an FleetConfig object but got %T", obj)
	}
	log.Info("Defaulting for FleetConfig", "name", fc.GetName())

	return nil
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-fleetconfig-open-cluster-management-io-v1alpha1-fleetconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=fleetconfigs,verbs=create;update;delete,versions=v1alpha1,name=vfleetconfig-v1alpha1.open-cluster-management.io,admissionReviewVersions=v1
// +kubebuilder:object:generate=false

// FleetConfigCustomValidator struct is responsible for validating the FleetConfig resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type FleetConfigCustomValidator struct {
	client client.Client
}

var _ webhook.CustomValidator = &FleetConfigCustomValidator{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *FleetConfigCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	fc, ok := obj.(*FleetConfig)
	if !ok {
		return nil, fmt.Errorf("expected a FleetConfig object but got %T", obj)
	}
	log.Info("Validation for FleetConfig upon creation", "name", fc.GetName())

	var (
		allErrs  field.ErrorList
		warnings admission.Warnings
	)

	// hub
	if valid, msg := isKubeconfigValid(fc.Spec.Hub.Kubeconfig); !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), fc.Spec.Hub.Kubeconfig, msg),
		)
	}
	if fc.Spec.Hub.ClusterManager == nil && fc.Spec.Hub.SingletonControlPlane == nil {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("hub"), fc.Spec.Hub, "either hub.clusterManager or hub.singletonControlPlane must be specified"),
		)
	}

	// spokes
	for i, spoke := range fc.Spec.Spokes {
		if spoke.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
			if spoke.Klusterlet.ManagedClusterKubeconfig.SecretReference == nil {
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("spokes").Index(i), fc.Spec.Spokes, "managedClusterKubeconfig.secretReference is required in hosted mode"),
				)
			} else {
				if valid, msg := isKubeconfigValid(spoke.Klusterlet.ManagedClusterKubeconfig); !valid {
					allErrs = append(allErrs, field.Invalid(
						field.NewPath("spokes").Index(i), fc.Spec.Spokes, msg),
					)
				}
			}
		}
		if valid, msg := isKubeconfigValid(spoke.Kubeconfig); !valid {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spokes").Index(i), fc.Spec.Spokes, msg),
			)
		}
	}

	allErrs = append(allErrs, validateAddonConfigs(ctx, v.client, nil, fc)...)
	allErrs = append(allErrs, validateAddons(fc)...)
	allErrs = append(allErrs, validateHubAddons(ctx, nil, fc)...)
	if len(allErrs) > 0 {
		return warnings, errors.NewInvalid(GroupKind, fc.Name, allErrs)
	}

	return warnings, nil
}

func isKubeconfigValid(kubeconfig Kubeconfig) (bool, string) {
	if kubeconfig.SecretReference == nil && !kubeconfig.InCluster {
		return false, "either secretReference or inCluster must be specified for the kubeconfig"
	}
	if kubeconfig.SecretReference != nil && kubeconfig.InCluster {
		return false, "either secretReference or inCluster can be specified for the kubeconfig, not both"
	}
	return true, ""
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *FleetConfigCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	fc, ok := newObj.(*FleetConfig)
	if !ok {
		return nil, fmt.Errorf("expected a FleetConfig object for the newObj but got %T", newObj)
	}
	oldFc, ok := oldObj.(*FleetConfig)
	if !ok {
		return nil, fmt.Errorf("expected a FleetConfig object for the oldObj but got %T", oldObj)
	}
	log.Info("starting validation for FleetConfig update", "name", fc.GetName())

	err := allowFleetConfigUpdate(fc, oldFc)
	if err != nil {
		return nil, err
	}

	errs := validateAddonConfigs(ctx, v.client, oldFc, fc)
	errs = append(errs, validateAddons(fc)...)
	errs = append(errs, validateHubAddons(ctx, oldFc, fc)...)
	if len(errs) > 0 {
		return nil, errors.NewInvalid(GroupKind, fc.Name, errs)
	}

	log.Info("validation for FleetConfig update allowed", "name", fc.GetName())
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *FleetConfigCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	fc, ok := obj.(*FleetConfig)
	if !ok {
		return nil, fmt.Errorf("expected a FleetConfig object but got %T", obj)
	}
	log.Info("Validation for FleetConfig upon deletion", "name", fc.GetName())

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
