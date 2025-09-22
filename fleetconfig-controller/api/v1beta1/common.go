package v1beta1

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
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

// SecretReference describes how to retrieve a kubeconfig stored as a secret in the same namespace as the resource.
type SecretReference struct {
	// The name of the secret.
	// +required
	Name string `json:"name"`

	// The map key to access the kubeconfig. Defaults to 'kubeconfig'.
	// +kubebuilder:default:="kubeconfig"
	// +optional
	KubeconfigKey string `json:"kubeconfigKey,omitempty"`
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

// GetRequests returns the resource requests.
func (r ResourceSpec) GetRequests() args.ResourceValues {
	if r.Requests == nil {
		return &ResourceValues{}
	}
	return r.Requests
}

// GetLimits returns the resource limits.
func (r ResourceSpec) GetLimits() args.ResourceValues {
	if r.Limits == nil {
		return &ResourceValues{}
	}
	return r.Limits
}

// GetQosClass returns the QoS class.
func (r ResourceSpec) GetQosClass() string {
	return r.QosClass
}

// Ensure ResourceSpec implements args.ResourceSpec interface
var _ args.ResourceSpec = (*ResourceSpec)(nil)

// Ensure ResourceValues implements args.ResourceValues interface
var _ args.ResourceValues = (*ResourceValues)(nil)

// NewCondition returns a new v1beta1.Condition.
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
