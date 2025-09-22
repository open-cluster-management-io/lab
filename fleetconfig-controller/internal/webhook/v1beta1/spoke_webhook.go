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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"open-cluster-management.io/api/client/addon/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
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
var spokelog = logf.Log.WithName("spoke-resource")

// SetupSpokeWebhookWithManager registers the webhook for Spoke in the manager.
func SetupSpokeWebhookWithManager(mgr ctrl.Manager) error {
	kubeconfig, err := kube.RawFromInClusterRestConfig()
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(kubeconfig)
	if err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr).For(&v1beta1.Spoke{}).
		WithValidator(&SpokeCustomValidator{client: mgr.GetClient(), addonC: addonC}).
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
	client client.Client
	addonC *versioned.Clientset
}

var _ webhook.CustomValidator = &SpokeCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	spoke, ok := obj.(*v1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object but got %T", obj)
	}
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

	warn, errs := validateAddons(ctx, v.client, spoke, v.addonC)
	allErrs = append(allErrs, errs...)

	if len(allErrs) > 0 {
		return warn, kerrs.NewInvalid(v1beta1.SpokeGroupKind, spoke.Name, allErrs)
	}
	return warn, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	spoke, ok := newObj.(*v1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object for the newObj but got %T", newObj)
	}
	oldSpoke, ok := oldObj.(*v1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object for the oldObj but got %T", oldObj)
	}
	spokelog.Info("Validation for Spoke upon update", "name", spoke.GetName())

	err := allowSpokeUpdate(oldSpoke, spoke)
	if err != nil {
		return nil, err
	}

	var allErrs field.ErrorList

	valid, msg := isKubeconfigValid(spoke.Spec.Kubeconfig)
	if !valid {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec").Child("kubeconfig"), spoke, msg),
		)
	}

	warn, valErrs := validateAddons(ctx, v.client, spoke, v.addonC)
	allErrs = append(allErrs, valErrs...)

	if len(allErrs) > 0 {
		return warn, kerrs.NewInvalid(v1beta1.SpokeGroupKind, spoke.Name, allErrs)
	}
	return warn, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	spoke, ok := obj.(*v1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object but got %T", obj)
	}
	spokelog.Info("Validation for Spoke upon deletion", "name", spoke.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
