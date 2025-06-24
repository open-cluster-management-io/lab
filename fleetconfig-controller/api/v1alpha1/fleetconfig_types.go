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
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FleetConfigSpec defines the desired state of FleetConfig.
type FleetConfigSpec struct {
	Hub              Hub               `json:"hub"`
	Spokes           []Spoke           `json:"spokes"`
	RegistrationAuth *RegistrationAuth `json:"registrationAuth,omitempty"`
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
	// ClusterManager configuration.
	// +kubebuilder:default:={}
	ClusterManager *ClusterManager `json:"clusterManager,omitempty"`

	// If true, create open-cluster-management namespace, otherwise use existing one.
	// +kubebuilder:default:=true
	CreateNamespace bool `json:"createNamespace"`

	// If set, the hub will be reinitialized.
	Force bool `json:"force,omitempty"`

	// Kubeconfig details for the Hub cluster.
	Kubeconfig *Kubeconfig `json:"kubeconfig"`

	// Singleton control plane configuration. If provided, deploy a singleton control plane instead of clustermanager.
	// This is an alpha stage flag.
	SingletonControlPlane *SingletonControlPlane `json:"singleton,omitempty"`

	// APIServer is the API server URL for the Hub cluster. If provided, the hub will be joined
	// using this API server instead of the one in the obtained kubeconfig. This is useful when
	// using in-cluster kubeconfig when that kubeconfig would return an incorrect API server URL.
	APIServer *string `json:"apiServer,omitempty"`
}

// SingletonControlPlane is the configuration for a singleton control plane
type SingletonControlPlane struct {
	// The name of the singleton control plane.
	// +kubebuilder:default:="singleton-controlplane"
	Name string `json:"name"`

	// Helm configuration for the multicluster-controlplane Helm chart.
	// For now https://open-cluster-management.io/helm-charts/ocm/multicluster-controlplane is always used - no private registry support.
	// See: https://github.com/open-cluster-management-io/multicluster-controlplane/blob/main/charts/multicluster-controlplane/values.yaml
	Helm Helm `json:"helm"`
}

// Helm is the configuration for helm.
type Helm struct {
	// Raw, YAML-formatted Helm values.
	Values string `json:"values,omitempty"`

	// Comma-separated Helm values, e.g., key1=val1,key2=val2.
	Set []string `json:"set,omitempty"`

	// Comma-separated Helm JSON values, e.g., key1=jsonval1,key2=jsonval2.
	SetJSON []string `json:"setJson,omitempty"`

	// Comma-separated Helm literal STRING values.
	SetLiteral []string `json:"setLiteral,omitempty"`

	// Comma-separated Helm STRING values, e.g., key1=val1,key2=val2.
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
	FeatureGates string `json:"featureGates,omitempty"`

	// If set, the cluster manager operator will be purged and the open-cluster-management namespace deleted
	// when the FleetConfig CR is deleted.
	// +kubebuilder:default:=true
	PurgeOperator bool `json:"purgeOperator,omitempty"`

	// Resource specifications for all clustermanager-managed containers.
	Resources *ResourceSpec `json:"resources,omitempty"`

	// Version and image registry details for the cluster manager.
	// +kubebuilder:default:={}
	Source *OCMSource `json:"source,omitempty"`

	// If set, the bootstrap token will used instead of a service account token.
	UseBootstrapToken bool `json:"useBootstrapToken,omitempty"`
}

// OCMSource is the configuration for an OCM source.
type OCMSource struct {
	// The version of predefined compatible image versions (e.g. v0.6.0). Defaults to the latest released version.
	// You can also set "latest" to install the latest development version.
	// +kubebuilder:default:="default"
	BundleVersion string `json:"bundleVersion,omitempty"`

	// The name of the image registry serving OCM images, which will be used for all OCM components."
	// +kubebuilder:default:="quay.io/open-cluster-management"
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
	Context string `json:"context,omitempty"`
}

// SecretReference describes how to retrieve a kubeconfig stored as a secret
type SecretReference struct {
	// The name of the secret.
	Name string `json:"name"`

	// The namespace the secret is in.
	Namespace string `json:"namespace"`

	// The map key to access the kubeconfig.
	// Leave empty to use 'kubeconfig'.
	// +optional
	KubeconfigKey *string `json:"kubeconfigKey,omitempty"`
}

// Spoke provides specifications for joining and potentially upgrading spokes.
type Spoke struct {
	// The name of the spoke cluster.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Name string `json:"name"`

	// If true, create open-cluster-management namespace and agent namespace (open-cluster-management-agent for Default mode,
	// <klusterlet-name> for Hosted mode), otherwise use existing one.
	// +kubebuilder:default:=true
	CreateNamespace bool `json:"createNamespace"`

	// If true, sync the labels from klusterlet to all agent resources.
	SyncLabels bool `json:"syncLabels,omitempty"`

	// Kubeconfig details for the Spoke cluster.
	Kubeconfig *Kubeconfig `json:"kubeconfig"`

	// Hub cluster CA certificate, optional
	Ca string `json:"ca,omitempty"`

	// Proxy CA certificate, optional
	ProxyCa string `json:"proxyCa,omitempty"`

	// URL of a forward proxy server used by agents to connect to the Hub cluster.
	ProxyURL string `json:"proxyUrl,omitempty"`

	// Klusterlet configuration.
	// +kubebuilder:default:={}
	Klusterlet Klusterlet `json:"klusterlet,omitempty"`

	// ClusterARN is the ARN of the spoke cluster.
	// This field is optionally used for AWS IRSA registration authentication.
	ClusterARN string `json:"clusterARN,omitempty"`
}

// JoinType returns a status condition type indicating that a particular Spoke cluster has joined the Hub.
func (s *Spoke) JoinType() string {
	return fmt.Sprintf("spoke-cluster-%s-joined", s.conditionName())
}

func (s *Spoke) conditionName() string {
	name := s.Name
	if len(name) > 42 {
		name = name[:42] // account for extra 21 chars in the condition type (max. total of 63)
	}
	return name
}

// JoinedSpoke represents a spoke that has been joined to a hub.
type JoinedSpoke struct {
	// The name of the spoke cluster.
	Name string `json:"name"`

	// Kubeconfig details for the Spoke cluster.
	Kubeconfig *Kubeconfig `json:"kubeconfig"`

	// If set, the klusterlet operator will be purged and all open-cluster-management namespaces deleted
	// when the klusterlet is unjoined from its Hub cluster.
	// +kubebuilder:default:=true
	PurgeKlusterletOperator bool `json:"purgeKlusterletOperator,omitempty"`
}

// UnjoinType returns a status condition type indicating that a particular Spoke cluster has been removed from the Hub.
func (j *JoinedSpoke) UnjoinType() string {
	return fmt.Sprintf("spoke-cluster-%s-unjoined", j.conditionName())
}

func (j *JoinedSpoke) conditionName() string {
	name := j.Name
	if len(name) > 40 {
		name = name[:40] // account for extra 23 chars in the condition type (max. total of 63)
	}
	return name
}

// Klusterlet is the configuration for a klusterlet.
type Klusterlet struct {
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
	FeatureGates string `json:"featureGates,omitempty"`

	// Deployent mode for klusterlet
	// +kubebuilder:validation:Enum=Default;Hosted
	// +kubebuilder:default:="Default"
	Mode string `json:"mode,omitempty"`

	// If set, the klusterlet operator will be purged and all open-cluster-management namespaces deleted
	// when the klusterlet is unjoined from its Hub cluster.
	// +kubebuilder:default:=true
	PurgeOperator bool `json:"purgeOperator,omitempty"`

	// If true, the installed klusterlet agent will start the cluster registration process by looking for the
	// internal endpoint from the public cluster-info in the Hub cluster instead of using hubApiServer.
	ForceInternalEndpointLookup bool `json:"forceInternalEndpointLookup,omitempty"`

	// External managed cluster kubeconfig, required if using hosted mode.
	ManagedClusterKubeconfig *Kubeconfig `json:"managedClusterKubeconfig,omitempty"`

	// If true, the klusterlet accesses the managed cluster using the internal endpoint from the public
	// cluster-info in the managed cluster instead of using managedClusterKubeconfig.
	ForceInternalEndpointLookupManaged bool `json:"forceInternalEndpointLookupManaged,omitempty"`

	// Resource specifications for all klusterlet-managed containers.
	Resources *ResourceSpec `json:"resources,omitempty"`

	// If true, deploy klusterlet in singleton mode, with registration and work agents running in a single pod.
	// This is an alpha stage flag.
	Singleton bool `json:"singleton,omitempty"`

	// Version and image registry details for the klusterlet.
	// +kubebuilder:default:={}
	Source *OCMSource `json:"source,omitempty"`
}

// ResourceSpec defines resource limits and requests for all managed clusters.
type ResourceSpec struct {
	// The resource limits of all the containers managed by the Cluster Manager or Klusterlet operators.
	Limits ResourceValues `json:"limits,omitempty"`

	// The resource requests of all the containers managed by the Cluster Manager or Klusterlet operators.
	Requests ResourceValues `json:"requests,omitempty"`

	// The resource QoS class of all the containers managed by the Cluster Manager or Klusterlet operators.
	// One of Default, BestEffort or ResourceRequirement.
	// +kubebuilder:validation:Enum=Default;BestEffort;ResourceRequirement
	// +kubebuilder:default:="Default"
	QosClass string `json:"qosClass"`
}

// ResourceValues detail container resource constraints.
type ResourceValues struct {
	// The number of CPU units to request, e.g., '800m'.
	CPU string `json:"cpu,omitempty"`

	// The amount of memory to request, e.g., '8Gi'.
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
	Driver string `json:"driver"`

	// The Hub cluster ARN for awsirsa registration authentication. Required when Type is awsirsa, otherwise ignored.
	HubClusterARN string `json:"hubClusterARN,omitempty"`

	// List of AWS EKS ARN patterns so any EKS clusters with these patterns will be auto accepted to join with hub cluster.
	// Example pattern: "arn:aws:eks:us-west-2:123456789013:cluster/.*"
	AutoApprovedARNPatterns []string `json:"autoApprovedARNPatterns,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=".metadata.creationTimestamp"

// FleetConfig is the Schema for the fleetconfigs API.
type FleetConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetConfigSpec   `json:"spec,omitempty"`
	Status FleetConfigStatus `json:"status,omitempty"`
}

// GetDriver returns the registration auth type, defaults to csr.
func (ra *RegistrationAuth) GetDriver() string {
	if ra == nil {
		// default registration auth type
		return CSRRegistrationDriver
	}
	return ra.Driver
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
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FleetConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FleetConfig{}, &FleetConfigList{})
}
