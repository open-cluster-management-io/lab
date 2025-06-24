package models

// ManagedClusterSetBindingSpec represents the spec of a ManagedClusterSetBinding
type ManagedClusterSetBindingSpec struct {
	// ClusterSet is the name of the ManagedClusterSet bound to the namespace
	ClusterSet string `json:"clusterSet"`
}

// ManagedClusterSetBindingStatus represents the status of a ManagedClusterSetBinding
type ManagedClusterSetBindingStatus struct {
	// Conditions contains the different condition statuses for this ManagedClusterSetBinding
	Conditions []Condition `json:"conditions,omitempty"`
}

// ManagedClusterSetBinding represents a binding between a ManagedClusterSet and a namespace
// This allows resources in that namespace to use the ManagedClusterSet for placement
type ManagedClusterSetBinding struct {
	// ID is a unique identifier for the ManagedClusterSetBinding
	ID string `json:"id"`

	// Name is the name of the ManagedClusterSetBinding
	Name string `json:"name"`

	// Namespace is the namespace where this binding exists
	Namespace string `json:"namespace"`

	// Spec contains the binding specification
	Spec ManagedClusterSetBindingSpec `json:"spec"`

	// Status contains the binding status
	Status ManagedClusterSetBindingStatus `json:"status,omitempty"`

	// CreationTimestamp is the creation time of the binding
	CreationTimestamp string `json:"creationTimestamp,omitempty"`
}
