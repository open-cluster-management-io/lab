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

// TODO - remove once hub webhooks are implemented.
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
var hublog = logf.Log.WithName("hub-resource")

// SetupHubWebhookWithManager registers the webhook for Hub in the manager.
func SetupHubWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&fleetconfigopenclustermanagementiov1beta1.Hub{}).
		WithValidator(&HubCustomValidator{}).
		WithDefaulter(&HubCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-fleetconfig-open-cluster-management-io-v1beta1-hub,mutating=true,failurePolicy=fail,sideEffects=None,groups=fleetconfig.open-cluster-management.io,resources=hubs,verbs=create;update,versions=v1beta1,name=mhub-v1beta1.kb.io,admissionReviewVersions=v1

// HubCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Hub when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type HubCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &HubCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Hub.
func (d *HubCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	hub, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Hub)

	if !ok {
		return fmt.Errorf("expected an Hub object but got %T", obj)
	}
	hublog.Info("Defaulting for Hub", "name", hub.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
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
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &HubCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	hub, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object but got %T", obj)
	}
	hublog.Info("Validation for Hub upon creation", "name", hub.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	hub, ok := newObj.(*fleetconfigopenclustermanagementiov1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object for the newObj but got %T", newObj)
	}
	hublog.Info("Validation for Hub upon update", "name", hub.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Hub.
func (v *HubCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	hub, ok := obj.(*fleetconfigopenclustermanagementiov1beta1.Hub)
	if !ok {
		return nil, fmt.Errorf("expected a Hub object but got %T", obj)
	}
	hublog.Info("Validation for Hub upon deletion", "name", hub.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
