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

// Package v1beta1 contains API Schema definitions for the  v1beta1 API group.
// +kubebuilder:object:generate=true
// +groupName=fleetconfig.open-cluster-management.io
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const group = "fleetconfig.open-cluster-management.io"

var (

	// HubGroupKind is the group kind for the Hub API
	HubGroupKind = schema.GroupKind{Group: group, Kind: "Hub"}

	// SpokeGroupKind is the group kind for the Spoke API
	SpokeGroupKind = schema.GroupKind{Group: group, Kind: "Spoke"}

	// GroupVersion is group version used to register these objects.
	GroupVersion = schema.GroupVersion{Group: group, Version: "v1beta1"}

	// SchemeBuilder registers this group-version's types with a *runtime.Scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Hub{},
		&HubList{},
		&Spoke{},
		&SpokeList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
