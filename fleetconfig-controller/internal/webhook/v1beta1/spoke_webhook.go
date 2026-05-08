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

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var spokelog = logf.Log.WithName("spoke-resource")

// SetupSpokeWebhookWithManager registers the webhook for Spoke in the manager.
func SetupSpokeWebhookWithManager(mgr ctrl.Manager, instanceType string) error {
	addonC, err := addonapi.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr, &v1beta1.Spoke{}).
		WithValidator(&SpokeCustomValidator{client: mgr.GetClient(), addonC: addonC, instanceType: instanceType}).
		Complete()
}

// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-fleetconfig-open-cluster-management-io-v1beta1-spoke,mutating=false,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=spokes,verbs=create;update,versions=v1beta1,name=vspoke-v1beta1.kb.io,admissionReviewVersions=v1

// SpokeCustomValidator struct is responsible for validating the Spoke resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SpokeCustomValidator struct {
	client       client.Client
	addonC       *addonapi.Clientset
	instanceType string
}

var _ admission.Validator[*v1beta1.Spoke] = &SpokeCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateCreate(ctx context.Context, spoke *v1beta1.Spoke) (admission.Warnings, error) {
	spokelog.Info("Validation for Spoke upon creation", "name", spoke.GetName())

	var allErrs field.ErrorList

	if spoke.Spec.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
		if spoke.Spec.Klusterlet.ManagedClusterKubeconfig.SecretReference == nil {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("klusterlet").Child("managedClusterKubeconfig").Child("secretReference"),
				spoke.Name, "managedClusterKubeconfig.secretReference is required in hosted mode"),
			)
		} else {
			if valid, msg := isKubeconfigValid(spoke.Spec.Klusterlet.ManagedClusterKubeconfig); !valid {
				allErrs = append(allErrs, field.Invalid(
					field.NewPath("spec").Child("klusterlet").Child("managedClusterKubeconfig").Child("secretReference"),
					spoke.Name, msg),
				)
			}
		}
	}
	if valid, msg := isKubeconfigValid(spoke.Spec.Kubeconfig); !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("kubeconfig"), spoke, msg),
		)
	}

	allErrs = append(allErrs, validateSpokeKlusterletAddon(spoke)...)

	hub := &v1beta1.Hub{}
	err := v.client.Get(ctx, types.NamespacedName{Name: spoke.Spec.HubRef.Name, Namespace: spoke.Spec.HubRef.Namespace}, hub)
	switch {
	case err == nil:
		allErrs = append(allErrs, validateSpokeRegistrationWithHub(ctx, v.client, spoke, hub)...)
	case kerrs.IsNotFound(err):
		// Hub-backed registration checks require the Hub object; optional skip until it exists.
	default:
		return nil, fmt.Errorf("get Hub %s/%s: %w", spoke.Spec.HubRef.Namespace, spoke.Spec.HubRef.Name, err)
	}

	warn, errs := v.validateAddons(ctx, v.client, spoke)
	allErrs = append(allErrs, errs...)

	if len(allErrs) > 0 {
		return warn, kerrs.NewInvalid(v1beta1.SpokeGroupKind, spoke.Name, allErrs)
	}
	return warn, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateUpdate(ctx context.Context, oldSpoke, newSpoke *v1beta1.Spoke) (admission.Warnings, error) {
	spokelog.Info("Validation for Spoke upon update", "name", newSpoke.GetName())

	err := allowSpokeUpdate(oldSpoke, newSpoke)
	if err != nil {
		return nil, err
	}

	var allErrs field.ErrorList

	valid, msg := isKubeconfigValid(newSpoke.Spec.Kubeconfig)
	if !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("kubeconfig"), newSpoke, msg),
		)
	}

	allErrs = append(allErrs, validateSpokeKlusterletAddon(newSpoke)...)

	hub := &v1beta1.Hub{}
	hubGetErr := v.client.Get(ctx, types.NamespacedName{Name: newSpoke.Spec.HubRef.Name, Namespace: newSpoke.Spec.HubRef.Namespace}, hub)
	switch {
	case hubGetErr == nil:
		allErrs = append(allErrs, validateSpokeRegistrationWithHub(ctx, v.client, newSpoke, hub)...)
	case kerrs.IsNotFound(hubGetErr):
	default:
		return nil, fmt.Errorf("get Hub %s/%s: %w", newSpoke.Spec.HubRef.Namespace, newSpoke.Spec.HubRef.Name, hubGetErr)
	}

	warn, valErrs := v.validateAddons(ctx, v.client, newSpoke)
	allErrs = append(allErrs, valErrs...)

	if len(allErrs) > 0 {
		return warn, kerrs.NewInvalid(v1beta1.SpokeGroupKind, newSpoke.Name, allErrs)
	}
	return warn, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateDelete(_ context.Context, spoke *v1beta1.Spoke) (admission.Warnings, error) {
	spokelog.Info("Validation for Spoke upon deletion", "name", spoke.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
