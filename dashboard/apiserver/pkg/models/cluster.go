package models

// ClusterClaim represents a claim from the managed cluster
type ClusterClaim struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Taint represents a taint on the managed cluster
type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

// ClusterStatus represents a simplified cluster status
type ClusterStatus struct {
	Available  bool        `json:"available"`
	Joined     bool        `json:"joined"`
	Conditions []Condition `json:"conditions,omitempty"`
}

// ManagedClusterClientConfig represents the client configuration for a managed cluster
type ManagedClusterClientConfig struct {
	URL      string `json:"url"`
	CABundle string `json:"caBundle,omitempty"`
}

// Cluster represents a simplified OCM ManagedCluster
type Cluster struct {
	ID                          string                       `json:"id"`
	Name                        string                       `json:"name"`
	Status                      string                       `json:"status"` // "Online", "Offline", etc.
	Version                     string                       `json:"version,omitempty"`
	Labels                      map[string]string            `json:"labels,omitempty"`
	Conditions                  []Condition                  `json:"conditions,omitempty"`
	HubAccepted                 bool                         `json:"hubAccepted"`
	Capacity                    map[string]string            `json:"capacity,omitempty"`
	Allocatable                 map[string]string            `json:"allocatable,omitempty"`
	ClusterClaims               []ClusterClaim               `json:"clusterClaims,omitempty"`
	Taints                      []Taint                      `json:"taints,omitempty"`
	ManagedClusterClientConfigs []ManagedClusterClientConfig `json:"managedClusterClientConfigs,omitempty"`
	CreationTimestamp           string                       `json:"creationTimestamp,omitempty"`
}

// LabelSelector represents a Kubernetes label selector
type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// ClusterSelector represents the selector for clusters in a ManagedClusterSet
type ClusterSelector struct {
	SelectorType  string         `json:"selectorType"`
	LabelSelector *LabelSelector `json:"labelSelector,omitempty"`
}


