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
	"fmt"
	"maps"
	"reflect"
	"sort"
	"time"

	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FleetConfigSpec defines the desired state of FleetConfig.
type FleetConfigSpec struct {
	// +required
	Hub Hub `json:"hub"`

	// +required
	Spokes []Spoke `json:"spokes"`

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

// FleetConfigStatus defines the observed state of FleetConfig.
type FleetConfigStatus struct {
	Phase        string        `json:"phase,omitempty"`
	Conditions   []Condition   `json:"conditions,omitempty"`
	JoinedSpokes []JoinedSpoke `json:"joinedSpokes,omitempty"`
}

// ToComparable returns a deep copy of the FleetConfigStatus that's suitable for semantic comparison.
func (s *FleetConfigStatus) ToComparable(_ ...Condition) *FleetConfigStatus {
	comparable := s.DeepCopy()
	for i := range comparable.Conditions {
		comparable.Conditions[i].LastTransitionTime = metav1.Time{}
	}
	return comparable
}

// GetCondition returns the condition with the supplied type, if it exists.
func (s *FleetConfigStatus) GetCondition(cType string) *Condition {
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
func (s *FleetConfigStatus) SetConditions(cover bool, c ...Condition) {
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

// Equal returns true if the status is identical to the supplied status,
// ignoring the LastTransitionTimes and order of statuses.
func (s *FleetConfigStatus) Equal(other *FleetConfigStatus) bool {
	if s == nil || other == nil {
		return s == nil && other == nil
	}

	if len(other.Conditions) != len(s.Conditions) {
		return false
	}

	sc := make([]Condition, len(s.Conditions))
	copy(sc, s.Conditions)

	oc := make([]Condition, len(other.Conditions))
	copy(oc, other.Conditions)

	// We should not have more than one condition of each type
	sort.Slice(sc, func(i, j int) bool { return sc[i].Type < sc[j].Type })
	sort.Slice(oc, func(i, j int) bool { return oc[i].Type < oc[j].Type })

	for i := range sc {
		if !sc[i].Equal(oc[i]) {
			return false
		}
	}

	return true
}

// NewCondition returns a new v1alpha1.Condition.
func NewCondition(msg, cType string, status, wantStatus metav1.ConditionStatus) Condition {
	return Condition{
		Condition: metav1.Condition{
			Status:             status,
			Message:            msg,
			Reason:             ReconcileSuccess,
			Type:               cType,
			LastTransitionTime: metav1.Time{Time: time.Now()},
		},
		WantStatus: wantStatus,
	}
}

// Condition describes the state of a FleetConfig.
type Condition struct {
	metav1.Condition `json:",inline"`
	WantStatus       metav1.ConditionStatus `json:"wantStatus"`
}

// Equal returns true if the condition is identical to the supplied condition, ignoring the LastTransitionTime.
func (c Condition) Equal(other Condition) bool {
	return c.Type == other.Type && c.Status == other.Status && c.WantStatus == other.WantStatus &&
		c.Reason == other.Reason && c.Message == other.Message
}

// Hub provides specifications for an OCM hub cluster.
type Hub struct {
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

// Kubeconfig is the configuration for a kubeconfig.
type Kubeconfig struct {
	// A reference to an existing secret containing a kubeconfig.
	// Must be provided for remote clusters.
	// For same-cluster, must be provided unless InCluster is set to true.
	// +optional
	SecretReference *SecretReference `json:"secretReference,omitempty"`

	// If set, the kubeconfig will be read from the cluster.
	// Only applicable for same-cluster operations.
	// Defaults to false.
	// +optional
	InCluster bool `json:"inCluster,omitempty"`

	// The context to use in the kubeconfig file.
	// +optional
	Context string `json:"context,omitempty"`
}

// SecretReference describes how to retrieve a kubeconfig stored as a secret
type SecretReference struct {
	// The name of the secret.
	// +required
	Name string `json:"name"`

	// The namespace the secret is in.
	// +required
	Namespace string `json:"namespace"`

	// The map key to access the kubeconfig. Defaults to 'kubeconfig'.
	// +kubebuilder:default:="kubeconfig"
	// +optional
	KubeconfigKey string `json:"kubeconfigKey,omitempty"`
}

// ISpoke is an interface that both Spoke and JoinedSpoke implement.
// +kubebuilder:object:generate=false
type ISpoke interface {
	GetName() string
	GetKubeconfig() Kubeconfig
	GetPurgeKlusterletOperator() bool
}

var _ ISpoke = &Spoke{}
var _ ISpoke = &JoinedSpoke{}

// Spoke provides specifications for joining and potentially upgrading spokes.
type Spoke struct {
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

// GetName returns the name of the spoke cluster.
func (s *Spoke) GetName() string {
	return s.Name
}

// GetKubeconfig returns the kubeconfig for the spoke cluster.
func (s *Spoke) GetKubeconfig() Kubeconfig {
	return s.Kubeconfig
}

// GetPurgeKlusterletOperator returns the purge klusterlet operator flag.
func (s *Spoke) GetPurgeKlusterletOperator() bool {
	return s.Klusterlet.PurgeOperator
}

// JoinType returns a status condition type indicating that a particular Spoke cluster has joined the Hub.
func (s *Spoke) JoinType() string {
	baseMsg := "spoke-cluster-%s-joined"
	return fmt.Sprintf(baseMsg, s.conditionName(maxConditionNameLen(baseMsg)))
}

// AddonEnableType returns a status condition type indicating that a particular Spoke cluster's addons have been disabled.
func (s *Spoke) AddonEnableType() string {
	baseMsg := "spoke-cluster-%s-addons-enabled"
	return fmt.Sprintf(baseMsg, s.conditionName(maxConditionNameLen(baseMsg)))
}

func (s *Spoke) conditionName(c int) string {
	name := s.Name
	if len(name) > c {
		name = name[:c] // account for extra chars in the condition type (max. total of 63)
	}
	return name
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

// JoinedSpoke represents a spoke that has been joined to a hub.
type JoinedSpoke struct {
	// The name of the spoke cluster.
	Name string `json:"name"`

	// Kubeconfig details for the Spoke cluster.
	Kubeconfig Kubeconfig `json:"kubeconfig"`

	// If set, the klusterlet operator will be purged and all open-cluster-management namespaces deleted
	// when the klusterlet is unjoined from its Hub cluster.
	// +kubebuilder:default:=true
	// +optional
	PurgeKlusterletOperator bool `json:"purgeKlusterletOperator,omitempty"`

	// EnabledAddons is the list of addons that are currently enabled for the cluster.
	// +kubebuilder:default:={}
	// +optional
	EnabledAddons []string `json:"enabledAddons,omitempty"`
}

// GetName returns the name of the joined spoke cluster.
func (j *JoinedSpoke) GetName() string {
	return j.Name
}

// GetKubeconfig returns the kubeconfig for the joined spoke cluster.
func (j *JoinedSpoke) GetKubeconfig() Kubeconfig {
	return j.Kubeconfig
}

// GetPurgeKlusterletOperator returns the purge klusterlet operator flag for the joined spoke cluster.
func (j *JoinedSpoke) GetPurgeKlusterletOperator() bool {
	return j.PurgeKlusterletOperator
}

// UnjoinType returns a status condition type indicating that a particular Spoke cluster has been removed from the Hub.
func (j *JoinedSpoke) UnjoinType() string {
	baseMsg := "spoke-cluster-%s-unjoined"
	return fmt.Sprintf(baseMsg, j.conditionName(maxConditionNameLen(baseMsg)))
}

// AddonDisableType returns a status condition type indicating that a particular Spoke cluster's addons have been disabled.
func (j *JoinedSpoke) AddonDisableType() string {
	baseMsg := "spoke-cluster-%s-addons-disabled"
	return fmt.Sprintf(baseMsg, j.conditionName(maxConditionNameLen(baseMsg)))
}

func (j *JoinedSpoke) conditionName(c int) string {
	name := j.Name
	if len(name) > c {
		name = name[:c] // account for extra chars in the condition type (max. total of 63)
	}
	return name
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

// ResourceSpec defines resource limits and requests for all managed clusters.
type ResourceSpec struct {
	// The resource limits of all the containers managed by the Cluster Manager or Klusterlet operators.
	// +optional
	Limits *ResourceValues `json:"limits,omitempty"`

	// The resource requests of all the containers managed by the Cluster Manager or Klusterlet operators.
	// +optional
	Requests *ResourceValues `json:"requests,omitempty"`

	// The resource QoS class of all the containers managed by the Cluster Manager or Klusterlet operators.
	// One of Default, BestEffort or ResourceRequirement.
	// +kubebuilder:validation:Enum=Default;BestEffort;ResourceRequirement
	// +kubebuilder:default:="Default"
	// +optional
	QosClass string `json:"qosClass,omitempty"`
}

// ResourceValues detail container resource constraints.
type ResourceValues struct {
	// The number of CPU units to request, e.g., '800m'.
	// +optional
	CPU string `json:"cpu,omitempty"`

	// The amount of memory to request, e.g., '8Gi'.
	// +optional
	Memory string `json:"memory,omitempty"`
}

// String returns a string representation of the resource values.
func (r *ResourceValues) String() string {
	if r.CPU != "" && r.Memory != "" {
		return fmt.Sprintf("cpu=%s,memory=%s", r.CPU, r.Memory)
	} else if r.CPU != "" {
		return fmt.Sprintf("cpu=%s", r.CPU)
	} else if r.Memory != "" {
		return fmt.Sprintf("memory=%s", r.Memory)
	}
	return ""
}

// RegistrationAuth provides specifications for registration authentication.
type RegistrationAuth struct {
	// The registration authentication driver to use.
	// Options are:
	//  - csr: Use the default CSR-based registration authentication.
	//  - awsirsa: Use AWS IAM Role for Service Accounts (IRSA) registration authentication.
	// The set of valid options is open for extension.
	// +kubebuilder:validation:Enum=csr;awsirsa
	// +kubebuilder:default:="csr"
	// +optional
	Driver string `json:"driver,omitempty"`

	// The Hub cluster ARN for awsirsa registration authentication. Required when Type is awsirsa, otherwise ignored.
	// +optional
	HubClusterARN string `json:"hubClusterARN,omitempty"`

	// List of AWS EKS ARN patterns so any EKS clusters with these patterns will be auto accepted to join with hub cluster.
	// Example pattern: "arn:aws:eks:us-west-2:123456789013:cluster/.*"
	// +optional
	AutoApprovedARNPatterns []string `json:"autoApprovedARNPatterns,omitempty"`
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

// FleetConfig is the Schema for the fleetconfigs API.
type FleetConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   FleetConfigSpec   `json:"spec,omitzero"`
	Status FleetConfigStatus `json:"status,omitempty"`
}

// BaseArgs returns the base arguments for all clusteradm commands.
func (m *FleetConfig) BaseArgs() []string {
	return []string{
		fmt.Sprintf("--timeout=%d", m.Spec.Timeout),
		fmt.Sprintf("--v=%d", m.Spec.LogVerbosity),
	}
}

// GetCondition gets the condition with the supplied type, if it exists.
func (m *FleetConfig) GetCondition(cType string) *Condition {
	return m.Status.GetCondition(cType)
}

// SetConditions sets the supplied conditions on a FleetConfig, replacing any existing conditions.
func (m *FleetConfig) SetConditions(cover bool, c ...Condition) {
	m.Status.SetConditions(cover, c...)
}

// +kubebuilder:object:root=true

// FleetConfigList contains a list of FleetConfig.
type FleetConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []FleetConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FleetConfig{}, &FleetConfigList{})
}

func maxConditionNameLen(base string) int {
	maxLen := 316 // a metav1.Condition.Type type can be at most 316 chars long
	return maxLen - (len(base) - 2)
}

// IsUnjoined returns true if a spoke cluster was successfully joined and then successfully unjoined, and the unjoined timestamp is after the joined timestamp.
// Returns false in all other cases.
func (m *FleetConfig) IsUnjoined(spoke Spoke, joinedSpoke JoinedSpoke) bool {
	joinedC := m.GetCondition(spoke.JoinType())
	if joinedC == nil {
		return false
	}
	unjoinedC := m.GetCondition(joinedSpoke.UnjoinType())
	if unjoinedC == nil {
		return false
	}
	if unjoinedC.Status != metav1.ConditionTrue {
		return false
	}
	// at this point, unjoined is known to be true. if joined is not true, cluster has not been successfully rejoined
	if joinedC.Status != metav1.ConditionTrue {
		return true
	}
	// if both exist and are true, compare timestamps
	return unjoinedC.LastTransitionTime.After(joinedC.LastTransitionTime.Time)
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
