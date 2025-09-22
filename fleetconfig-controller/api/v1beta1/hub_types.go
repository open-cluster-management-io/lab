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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HubSpec defines the desired state of Hub
type HubSpec struct {
	// APIServer is the API server URL for the Hub cluster. If provided, spokes clusters will
	// join the hub using this API server instead of the one in the bootstrap kubeconfig.
	// Spoke clusters with ForceInternalEndpointLookup set to true will ignore this field.
	// +optional
	APIServer string `json:"apiServer,omitempty"`

	// Hub cluster CA certificate, optional
	// +optional
	Ca string `json:"ca,omitempty"`

	// ClusterManager configuration.
	// +optional
	ClusterManager *ClusterManager `json:"clusterManager,omitempty"`

	// If true, create open-cluster-management namespace, otherwise use existing one.
	// +kubebuilder:default:=true
	// +optional
	CreateNamespace bool `json:"createNamespace,omitempty"`

	// If set, the hub will be reinitialized.
	// +optional
	Force bool `json:"force,omitempty"`

	// Kubeconfig details for the Hub cluster.
	// +required
	Kubeconfig Kubeconfig `json:"kubeconfig"`

	// Singleton control plane configuration. If provided, deploy a singleton control plane instead of clustermanager.
	// This is an alpha stage flag.
	// +optional
	SingletonControlPlane *SingletonControlPlane `json:"singleton,omitempty"`

	// +kubebuilder:default:={}
	// +optional
	RegistrationAuth RegistrationAuth `json:"registrationAuth,omitzero"`

	// +optional
	AddOnConfigs []AddOnConfig `json:"addOnConfigs,omitempty"`

	// +optional
	HubAddOns []HubAddOn `json:"hubAddOns,omitempty"`

	// Timeout is the timeout in seconds for all clusteradm operations, including init, accept, join, upgrade, etc.
	// +kubebuilder:default:=300
	// +optional
	Timeout int `json:"timeout,omitempty"`

	// LogVerbosity is the verbosity of the logs.
	// +kubebuilder:validation:Enum=0;1;2;3;4;5;6;7;8;9;10
	// +kubebuilder:default:=0
	// +optional
	LogVerbosity int `json:"logVerbosity,omitempty"`
}

// SingletonControlPlane is the configuration for a singleton control plane
type SingletonControlPlane struct {
	// The name of the singleton control plane.
	// +kubebuilder:default:="singleton-controlplane"
	// +optional
	Name string `json:"name,omitempty"`

	// Helm configuration for the multicluster-controlplane Helm chart.
	// For now https://open-cluster-management.io/helm-charts/ocm/multicluster-controlplane is always used - no private registry support.
	// See: https://github.com/open-cluster-management-io/multicluster-controlplane/blob/main/charts/multicluster-controlplane/values.yaml
	// +optional
	Helm *Helm `json:"helm,omitempty"`
}

// Helm is the configuration for helm.
type Helm struct {
	// Raw, YAML-formatted Helm values.
	// +optional
	Values string `json:"values,omitempty"`

	// Comma-separated Helm values, e.g., key1=val1,key2=val2.
	// +optional
	Set []string `json:"set,omitempty"`

	// Comma-separated Helm JSON values, e.g., key1=jsonval1,key2=jsonval2.
	// +optional
	SetJSON []string `json:"setJson,omitempty"`

	// Comma-separated Helm literal STRING values.
	// +optional
	SetLiteral []string `json:"setLiteral,omitempty"`

	// Comma-separated Helm STRING values, e.g., key1=val1,key2=val2.
	// +optional
	SetString []string `json:"setString,omitempty"`
}

// ClusterManager is the configuration for a cluster manager.
type ClusterManager struct {
	// A set of comma-separated pairs of the form 'key1=value1,key2=value2' that describe feature gates for alpha/experimental features.
	// Options are:
	//  - AddonManagement (ALPHA - default=true)
	//  - AllAlpha (ALPHA - default=false)
	//  - AllBeta (BETA - default=false)
	//  - CloudEventsDrivers (ALPHA - default=false)
	//  - DefaultClusterSet (ALPHA - default=false)
	//  - ManagedClusterAutoApproval (ALPHA - default=false)
	//  - ManifestWorkReplicaSet (ALPHA - default=false)
	//  - NilExecutorValidating (ALPHA - default=false)
	//  - ResourceCleanup (BETA - default=true)
	//  - V1beta1CSRAPICompatibility (ALPHA - default=false)
	// +kubebuilder:default:="AddonManagement=true"
	// +optional
	FeatureGates string `json:"featureGates,omitempty"`

	// If set, the cluster manager operator will be purged and the open-cluster-management namespace deleted
	// when the FleetConfig CR is deleted.
	// +kubebuilder:default:=true
	// +optional
	PurgeOperator bool `json:"purgeOperator,omitempty"`

	// Resource specifications for all clustermanager-managed containers.
	// +kubebuilder:default:={}
	// +optional
	Resources ResourceSpec `json:"resources,omitzero"`

	// Version and image registry details for the cluster manager.
	// +kubebuilder:default:={}
	// +optional
	Source OCMSource `json:"source,omitzero"`

	// If set, the bootstrap token will used instead of a service account token.
	// +optional
	UseBootstrapToken bool `json:"useBootstrapToken,omitempty"`
}

// AddOnConfig is the configuration of a custom AddOn that can be installed on a cluster.
type AddOnConfig struct {
	// The name of the add-on.
	// +required
	Name string `json:"name"`

	// The add-on version. Optional, defaults to "v0.0.1"
	// +kubebuilder:default:="v0.0.1"
	// +optional
	Version string `json:"version,omitempty"`

	// The rolebinding to the clusterrole in the cluster namespace for the addon agent
	// +optional
	ClusterRoleBinding string `json:"clusterRoleBinding,omitempty"`

	// Enable the agent to register to the hub cluster. Optional, defaults to false.
	// +kubebuilder:default:=false
	// +optional
	HubRegistration bool `json:"hubRegistration,omitempty"`

	// Whether to overwrite the add-on if it already exists. Optional, defaults to false.
	// +kubebuilder:default:=false
	// +optional
	Overwrite bool `json:"overwrite,omitempty"`
}

// HubAddOn is the configuration for enabling a built-in AddOn.
type HubAddOn struct {
	// Name is the name of the HubAddOn.
	// +kubebuilder:validation:Enum=argocd;governance-policy-framework
	// +required
	Name string `json:"name"`

	// The namespace to install the add-on in. If left empty, installs into the "open-cluster-management-addon" namespace.
	// +optional
	InstallNamespace string `json:"installNamespace,omitempty"`

	// Whether or not the selected namespace should be created. If left empty, defaults to false.
	// +kubebuilder:default:=false
	// +optional
	CreateNamespace bool `json:"createNamespace,omitempty"`
}

// OCMSource is the configuration for an OCM source.
type OCMSource struct {
	// The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version.
	// You can also set "latest" to install the latest development version.
	// +kubebuilder:default:="default"
	// +optional
	BundleVersion string `json:"bundleVersion,omitempty"`

	// The name of the image registry serving OCM images, which will be used for all OCM components."
	// +kubebuilder:default:="quay.io/open-cluster-management"
	// +optional
	Registry string `json:"registry,omitempty"`
}

// HubStatus defines the observed state of Hub.
type HubStatus struct {
	// Phase is the current phase of the Hub reconcile.
	Phase string `json:"phase,omitempty"`

	// Conditions are the current conditions of the Hub.
	Conditions []Condition `json:"conditions,omitempty"`

	InstalledHubAddOns []InstalledHubAddOn `json:"installedHubAddOns,omitempty"`
}

// InstalledHubAddOn tracks metadata for each hubAddon that is successfully installed on the hub.
type InstalledHubAddOn struct {
	// BundleVersion is the bundle version used when installing the addon.
	BundleVersion string `json:"bundleVersion"`

	// Name is the name of the addon.
	Name string `json:"name"`

	// Namespace is the namespace that the addon was installed into.
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

// Hub is the Schema for the hubs API
type Hub struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Hub
	// +required
	Spec HubSpec `json:"spec"`

	// status defines the observed state of Hub
	// +optional
	Status HubStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// HubList contains a list of Hub
type HubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hub `json:"items"`
}

// BaseArgs returns the base arguments for all clusteradm commands.
func (h *Hub) BaseArgs() []string {
	return []string{
		fmt.Sprintf("--timeout=%d", h.Spec.Timeout),
		fmt.Sprintf("--v=%d", h.Spec.LogVerbosity),
	}
}

// GetCondition returns the condition with the supplied type, if it exists.
func (s *HubStatus) GetCondition(cType string) *Condition {
	for _, c := range s.Conditions {
		if c.Type == cType {
			return &c
		}
	}
	return nil
}

// SetConditions sets the supplied conditions, adding net-new conditions and
// replacing any existing conditions of the same type. This is a no-op if all
// supplied conditions are identical (ignoring the last transition time) to
// those already set. If cover is false, existing conditions are not replaced.
func (s *HubStatus) SetConditions(cover bool, c ...Condition) {
	for _, new := range c {
		exists := false
		for i, existing := range s.Conditions {
			if existing.Type != new.Type {
				continue
			}
			if existing.Equal(new) {
				exists = true
				continue
			}
			exists = true
			if cover {
				s.Conditions[i] = new
			}
		}
		if !exists {
			s.Conditions = append(s.Conditions, new)
		}
	}
}

// GetCondition gets the condition with the supplied type, if it exists.
func (h *Hub) GetCondition(cType string) *Condition {
	return h.Status.GetCondition(cType)
}

// SetConditions sets the supplied conditions on a Hub, replacing any existing conditions.
func (h *Hub) SetConditions(cover bool, c ...Condition) {
	h.Status.SetConditions(cover, c...)
}

func init() {
	SchemeBuilder.Register(&Hub{}, &HubList{})
}
