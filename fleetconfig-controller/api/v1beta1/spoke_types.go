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
	"maps"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// SpokeSpec defines the desired state of Spoke
type SpokeSpec struct {
	// If true, create open-cluster-management namespace and agent namespace (open-cluster-management-agent for Default mode,
	// <klusterlet-name> for Hosted mode), otherwise use existing one.
	// +kubebuilder:default:=true
	// +optional
	CreateNamespace bool `json:"createNamespace,omitempty"`

	// HubRef is a reference to the Hub that this Spoke is managed by.
	// +required
	HubRef HubRef `json:"hubRef"`

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

	// Timeout is the timeout in seconds for all clusteradm operations, including init, accept, join, upgrade, etc.
	// If not set, defaults to the Hub's timeout.
	// +kubebuilder:default:=300
	// +optional
	Timeout int `json:"timeout,omitempty"`

	// LogVerbosity is the verbosity of the logs.
	// If not set, defaults to the Hub's logVerbosity.
	// +kubebuilder:validation:Enum=0;1;2;3;4;5;6;7;8;9;10
	// +kubebuilder:default:=0
	// +optional
	LogVerbosity int `json:"logVerbosity,omitempty"`

	// CleanupConfig is used to configure which resources should be automatically garbage collected during cleanup.
	// +kubebuilder:default:={}
	// +required
	CleanupConfig CleanupConfig `json:"cleanupConfig"`
}

// CleanupConfig is the configuration for cleaning up resources during Spoke cleanup.
type CleanupConfig struct {
	// If set, the agent will attempt to garbage collect its own namespace after the spoke cluster is unjoined.
	// +kubebuilder:default:=false
	// +optional
	PurgeAgentNamespace bool `json:"purgeAgentNamespace,omitempty"`

	// If set, the klusterlet operator will be purged and all open-cluster-management namespaces deleted
	// when the klusterlet is unjoined from its Hub cluster.
	// +kubebuilder:default:=true
	// +optional
	PurgeKlusterletOperator bool `json:"purgeKlusterletOperator,omitempty"`

	// If set, the kubeconfig secret will be automatically deleted after the agent has taken over managing the Spoke.
	// +kubebuilder:default:=false
	// +optional
	PurgeKubeconfigSecret bool `json:"purgeKubeconfigSecret,omitempty"`

	// If set, all ManifestWorks which were created using a Placement will be automatically descheduled from the Spoke cluster during deletion.
	// This includes AddOns installed using installStrategy.type=Placements. If an AddOn must stay running to reconcile deletion of other ManifestWorks,
	// it should tolerate the `fleetconfig.open-cluster-management.io/workload-cleanup` taint.
	// Manually created ManifestWorks will not be affected and must be manually cleaned up for Spoke deletion to proceed.
	// +kubebuilder:default:=false
	// +optional
	ForceClusterDrain bool `json:"forceClusterDrain,omitempty"`
}

// HubRef is the information required to get a Hub resource.
type HubRef struct {
	// Name is the name of the Hub that this Spoke is managed by.
	// +required
	Name string `json:"name"`

	// Namespace is namespace of the Hub that this Spoke is managed by.
	// +required
	Namespace string `json:"namespace"`
}

// IsManagedBy checks whether or not the Spoke is managed by a particular Hub.
func (s *Spoke) IsManagedBy(om metav1.ObjectMeta) bool {
	return s.Spec.HubRef.Name == om.Name && s.Spec.HubRef.Namespace == om.Namespace
}

// IsHubAsSpoke returns true if the cluster is a hub-as-spoke. Determined either by name `hub-as-spoke` or an InCluster kubeconfig
func (s *Spoke) IsHubAsSpoke() bool {
	return s.Name == ManagedClusterTypeHubAsSpoke || s.Spec.Kubeconfig.InCluster
}

// PivotComplete return true if the spoke's agent has successfully started managing day 2 operations.
func (s *Spoke) PivotComplete() bool {
	jc := s.GetCondition(SpokeJoined)
	if jc == nil || jc.Status != metav1.ConditionTrue {
		return false
	}
	pc := s.GetCondition(PivotComplete)
	return pc != nil && pc.Status == metav1.ConditionTrue
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

	// ValuesFrom is an optional reference to a ConfigMap containing values for the klusterlet Helm chart.
	// optional
	ValuesFrom *ConfigMapRef `json:"valuesFrom,omitempty"`

	// Values for the klusterlet Helm chart. Values defined here override values which are defined in ValuesFrom.
	// +optional
	Values *KlusterletChartConfig `json:"values,omitempty"`
}

// ConfigMapRef is a reference to data inside a ConfigMap, in the same namespace as the controller pod.
type ConfigMapRef struct {
	// Name is the name of the ConfigMap
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Key is the key under which the data is stored.
	// +required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
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

	// Annotations to apply to the add-on.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// DeploymentConfig provides additional configuration for the add-on deployment.
	// If specified, this will be used to create an AddOnDeploymentConfig resource.
	// +optional
	DeploymentConfig *addonv1alpha1.AddOnDeploymentConfigSpec `json:"deploymentConfig,omitempty"`
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
// +kubebuilder:resource:path=spokes
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

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

// BaseArgs returns the base arguments for all clusteradm commands.
func (s *Spoke) BaseArgs() []string {
	return []string{
		fmt.Sprintf("--timeout=%d", s.Spec.Timeout),
		fmt.Sprintf("--v=%d", s.Spec.LogVerbosity),
	}
}

// GetCondition returns the condition with the supplied type, if it exists.
func (s *SpokeStatus) GetCondition(cType string) *Condition {
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
func (s *SpokeStatus) SetConditions(cover bool, c ...Condition) {
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
func (s *Spoke) GetCondition(cType string) *Condition {
	return s.Status.GetCondition(cType)
}

// SetConditions sets the supplied conditions on a Spoke, replacing any existing conditions.
func (s *Spoke) SetConditions(cover bool, c ...Condition) {
	s.Status.SetConditions(cover, c...)
}

func init() {
	SchemeBuilder.Register(&Spoke{}, &SpokeList{})
}
