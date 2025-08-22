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

// TODO - remove once spoke webhooks are implemented.
//
//nolint:all // Required because of `dupl` between this file and spoke_webhook.go
package v1beta1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	fleetconfigopenclustermanagementiov1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// nolint:unused
// log is for logging in this package.
var spokelog = logf.Log.WithName("spoke-resource")

// SetupSpokeWebhookWithManager registers the webhook for Spoke in the manager.
func SetupSpokeWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&fleetconfigopenclustermanagementiov1beta1.Spoke{}).
		WithValidator(&SpokeCustomValidator{}).
		WithDefaulter(&SpokeCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-fleetconfig-open-cluster-management-io-v1beta1-spoke,mutating=true,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=spokes,verbs=create;update,versions=v1beta1,name=mspoke-v1beta1.kb.io,admissionReviewVersions=v1

// SpokeCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Spoke when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type SpokeCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &SpokeCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Spoke.
func (d *SpokeCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	spoke, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Spoke)

	if !ok {
		return fmt.Errorf("expected an Spoke object but got %T", obj)
	}
	spokelog.Info("Defaulting for Spoke", "name", spoke.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-fleetconfig-open-cluster-management-io-v1beta1-spoke,mutating=false,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=spokes,verbs=create;update,versions=v1beta1,name=vspoke-v1beta1.kb.io,admissionReviewVersions=v1

// SpokeCustomValidator struct is responsible for validating the Spoke resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type SpokeCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &SpokeCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	spoke, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object but got %T", obj)
	}
	spokelog.Info("Validation for Spoke upon creation", "name", spoke.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	spoke, ok := newObj.(*fleetconfigopenclustermanagementiov1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object for the newObj but got %T", newObj)
	}
	spokelog.Info("Validation for Spoke upon update", "name", spoke.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Spoke.
func (v *SpokeCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	spoke, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Spoke)
	if !ok {
		return nil, fmt.Errorf("expected a Spoke object but got %T", obj)
	}
	spokelog.Info("Validation for Spoke upon deletion", "name", spoke.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
