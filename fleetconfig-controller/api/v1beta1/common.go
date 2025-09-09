package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

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

// IsInCluster returns true if the kubeconfig should be loaded from the in-cluster configuration.
func (k Kubeconfig) IsInCluster() bool {
	return k.InCluster
}

// GetSecretReference returns the SecretReference used to locate the kubeconfig secret.
func (k Kubeconfig) GetSecretReference() kube.SecretReference {
	return k.SecretReference
}

// GetContext returns the context to use from the kubeconfig file.
func (k Kubeconfig) GetContext() string {
	return k.Context
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

// GetNamespacedName returns the NamespacedName for the SecretReference.
func (s *SecretReference) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      s.Name,
		Namespace: s.Namespace,
	}
}

// GetKubeconfigKey returns the key used to access the kubeconfig in the secret.
func (s *SecretReference) GetKubeconfigKey() string {
	return s.KubeconfigKey
}

var _ kube.Kubeconfig = &Kubeconfig{}
var _ kube.SecretReference = &SecretReference{}

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

// GetQosClass returns the resource QoS class for all containers managed by the Cluster Manager or Klusterlet operators.
func (s ResourceSpec) GetQosClass() string {
	return s.QosClass
}

// GetLimits returns the resource limits for all containers managed by the Cluster Manager or Klusterlet operators.
func (s ResourceSpec) GetLimits() common.ResourceValues {
	return s.Limits
}

// GetRequests returns the resource requests for all containers managed by the Cluster Manager or Klusterlet operators.
func (s ResourceSpec) GetRequests() common.ResourceValues {
	return s.Requests
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

var _ common.ResourceSpec = &ResourceSpec{}
var _ common.ResourceValues = &ResourceValues{}

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
