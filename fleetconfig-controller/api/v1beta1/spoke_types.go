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
	"maps"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// SpokeSpec defines the desired state of Spoke
type SpokeSpec struct {
	// The name of the spoke cluster.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +required
	Name string `json:"name"`

	// If true, create open-cluster-management namespace and agent namespace (open-cluster-management-agent for Default mode,
	// <klusterlet-name> for Hosted mode), otherwise use existing one.
	// +kubebuilder:default:=true
	// +optional
	CreateNamespace bool `json:"createNamespace,omitempty"`

	// If true, sync the labels from klusterlet to all agent resources.
	// +optional
	SyncLabels bool `json:"syncLabels,omitempty"`

	// Kubeconfig details for the Spoke cluster.
	// +required
	Kubeconfig Kubeconfig `json:"kubeconfig"`

	// Proxy CA certificate, optional
	// +optional
	ProxyCa string `json:"proxyCa,omitempty"`

	// URL of a forward proxy server used by agents to connect to the Hub cluster.
	// +optional
	ProxyURL string `json:"proxyUrl,omitempty"`

	// Klusterlet configuration.
	// +kubebuilder:default:={}
	// +optional
	Klusterlet Klusterlet `json:"klusterlet,omitzero"`

	// ClusterARN is the ARN of the spoke cluster.
	// This field is optionally used for AWS IRSA registration authentication.
	// +optional
	ClusterARN string `json:"clusterARN,omitempty"`

	// AddOns are the add-ons to enable for the spoke cluster.
	// +optional
	AddOns []AddOn `json:"addOns,omitempty"`
}

// Klusterlet is the configuration for a klusterlet.
type Klusterlet struct {
	// Annotations to apply to the spoke cluster. If not present, the 'agent.open-cluster-management.io/' prefix is added to each key.
	// Each annotation is added to klusterlet.spec.registrationConfiguration.clusterAnnotations on the spoke and subsequently to the ManagedCluster on the hub.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// A set of comma-separated pairs of the form 'key1=value1,key2=value2' that describe feature gates for alpha/experimental features.
	// Options are:
	//  - AddonManagement (ALPHA - default=true)
	//  - AllAlpha (ALPHA - default=false)
	//  - AllBeta (BETA - default=false)
	//  - ClusterClaim (ALPHA - default=true)
	//  - ExecutorValidatingCaches (ALPHA - default=false)
	//  - RawFeedbackJsonString (ALPHA - default=false)
	//  - V1beta1CSRAPICompatibility (ALPHA - default=false)
	// +kubebuilder:default:="AddonManagement=true,ClusterClaim=true"
	// +optional
	FeatureGates string `json:"featureGates,omitempty"`

	// Deployent mode for klusterlet
	// +kubebuilder:validation:Enum=Default;Hosted
	// +kubebuilder:default:="Default"
	// +optional
	Mode string `json:"mode,omitempty"`

	// If set, the klusterlet operator will be purged and all open-cluster-management namespaces deleted
	// when the klusterlet is unjoined from its Hub cluster.
	// +kubebuilder:default:=true
	// +optional
	PurgeOperator bool `json:"purgeOperator,omitempty"`

	// If true, the installed klusterlet agent will start the cluster registration process by looking for the
	// internal endpoint from the public cluster-info in the Hub cluster instead of using hubApiServer.
	// +optional
	ForceInternalEndpointLookup bool `json:"forceInternalEndpointLookup,omitempty"`

	// External managed cluster kubeconfig, required if using hosted mode.
	// +optional
	ManagedClusterKubeconfig Kubeconfig `json:"managedClusterKubeconfig,omitzero"`

	// If true, the klusterlet accesses the managed cluster using the internal endpoint from the public
	// cluster-info in the managed cluster instead of using managedClusterKubeconfig.
	// +optional
	ForceInternalEndpointLookupManaged bool `json:"forceInternalEndpointLookupManaged,omitempty"`

	// Resource specifications for all klusterlet-managed containers.
	// +kubebuilder:default:={}
	// +optional
	Resources ResourceSpec `json:"resources,omitzero"`

	// If true, deploy klusterlet in singleton mode, with registration and work agents running in a single pod.
	// This is an alpha stage flag.
	// +optional
	Singleton bool `json:"singleton,omitempty"`

	// Version and image registry details for the klusterlet.
	// +kubebuilder:default:={}
	// +optional
	Source OCMSource `json:"source,omitzero"`

	// Values for the klusterlet Helm chart.
	// +optional
	Values *KlusterletChartConfig `json:"values,omitempty"`
}

// KlusterletChartConfig is a wrapper around the external chart.KlusterletChartConfig
// to provide the required DeepCopy methods for code generation.
type KlusterletChartConfig struct {
	chart.KlusterletChartConfig `json:",inline"`
}

// DeepCopy returns a deep copy of the KlusterletChartConfig.
func (k *KlusterletChartConfig) DeepCopy() *KlusterletChartConfig {
	if k == nil {
		return nil
	}
	out := new(KlusterletChartConfig)
	k.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (k *KlusterletChartConfig) DeepCopyInto(out *KlusterletChartConfig) {
	*out = *k

	out.KlusterletChartConfig = k.KlusterletChartConfig

	if k.NodeSelector != nil {
		k, out := &k.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*k))
		maps.Copy(*out, *k)
	}
	if k.Tolerations != nil {
		k, out := &k.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*k))
		for i := range *k {
			(*k)[i].DeepCopyInto(&(*out)[i])
		}
	}

	k.Affinity.DeepCopyInto(&out.Affinity)
	k.Resources.DeepCopyInto(&out.Resources)
	k.PodSecurityContext.DeepCopyInto(&out.PodSecurityContext)
	k.SecurityContext.DeepCopyInto(&out.SecurityContext)

	out.Images = k.Images
	out.Klusterlet = k.Klusterlet

	if k.MultiHubBootstrapHubKubeConfigs != nil {
		k, out := &k.MultiHubBootstrapHubKubeConfigs, &out.MultiHubBootstrapHubKubeConfigs
		*out = make([]chart.BootStrapKubeConfig, len(*k))
		copy(*out, *k)
	}
}

// IsEmpty checks if the KlusterletChartConfig is empty/default/zero-valued
func (k *KlusterletChartConfig) IsEmpty() bool {
	return reflect.DeepEqual(*k, KlusterletChartConfig{})
}

// AddOn enables add-on installation on the cluster.
type AddOn struct {
	// The name of the add-on being enabled. Must match one of the AddOnConfigs or HubAddOns names.
	// +required
	ConfigName string `json:"configName"`

	// The namespace to install the add-on in. If left empty, installs into the "open-cluster-management-addon" namespace.
	// +optional
	InstallNamespace string `json:"installNamespace,omitempty"`

	// Annotations to apply to the add-on.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// SpokeStatus defines the observed state of Spoke.
type SpokeStatus struct {
	// Phase is the current phase of the Spoke reconcile.
	Phase string `json:"phase,omitempty"`

	// Conditions are the current conditions of the Spoke.
	Conditions []Condition `json:"conditions,omitempty"`

	// EnabledAddons is the list of addons that are currently enabled on the Spoke.
	// +kubebuilder:default:={}
	// +optional
	EnabledAddons []string `json:"enabledAddons,omitempty"`

	// KlusterletHash is a hash of the Spoke's .spec.klusterlet.values.
	// +kubebuilder:default:=""
	// +optional
	KlusterletHash string `json:"klusterletHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=spokes,scope=Cluster

// Spoke is the Schema for the spokes API
type Spoke struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Spoke
	// +required
	Spec SpokeSpec `json:"spec"`

	// status defines the observed state of Spoke
	// +optional
	Status SpokeStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// SpokeList contains a list of Spoke
type SpokeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Spoke `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Spoke{}, &SpokeList{})
}
