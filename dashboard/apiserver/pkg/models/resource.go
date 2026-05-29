package models

// ManagedResource represents a single Kubernetes resource extracted from ManifestWork specs
type ManagedResource struct {
	ID               string                 `json:"id"`                    // "<cluster>/<mwName>/<ordinal>"
	Kind             string                 `json:"kind"`
	APIVersion       string                 `json:"apiVersion"`
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace,omitempty"`
	Cluster          string                 `json:"cluster"`
	ManifestWorkName string                 `json:"manifestWorkName"`
	Ordinal          int                    `json:"ordinal"`
	Status           string                 `json:"status"`                // "Applied", "Available", "Pending", "Failed"
	Conditions       []Condition            `json:"conditions,omitempty"`
	StatusFeedback   *StatusFeedbackResult  `json:"statusFeedback,omitempty"`
	RawResource      map[string]interface{} `json:"rawResource,omitempty"`
}

// ManagedResourceList wraps the resource list with filter metadata for populating dropdowns
type ManagedResourceList struct {
	Resources      []ManagedResource `json:"resources"`
	AvailableKinds []string          `json:"availableKinds"`
	Clusters       []string          `json:"clusters"`
	Namespaces     []string          `json:"namespaces"`
}
