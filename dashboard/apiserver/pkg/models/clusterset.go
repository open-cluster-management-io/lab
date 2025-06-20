package models

// ClusterSetSpec represents the spec of a ManagedClusterSet
type ClusterSetSpec struct {
	ClusterSelector ClusterSelector `json:"clusterSelector"`
}

// ClusterSetStatus represents the status of a ManagedClusterSet
type ClusterSetStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
}

// ClusterSet represents a simplified OCM ManagedClusterSet
type ClusterSet struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Labels            map[string]string `json:"labels,omitempty"`
	Spec              ClusterSetSpec    `json:"spec,omitempty"`
	Status            ClusterSetStatus  `json:"status,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty"`
}
