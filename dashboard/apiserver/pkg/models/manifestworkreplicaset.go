package models

// ManifestWorkReplicaSet represents a simplified version of the OCM ManifestWorkReplicaSet resource
type ManifestWorkReplicaSet struct {
	ID                string                        `json:"id"`
	Name              string                        `json:"name"`
	Namespace         string                        `json:"namespace"`
	Labels            map[string]string             `json:"labels,omitempty"`
	PlacementRefs     []LocalPlacementReference     `json:"placementRefs,omitempty"`
	Conditions        []Condition                   `json:"conditions,omitempty"`
	Summary           ManifestWorkReplicaSetSummary `json:"summary"`
	PlacementsSummary []MWRSPlacementSummary        `json:"placementsSummary,omitempty"`
	CreationTimestamp string                        `json:"creationTimestamp,omitempty"`
	ManifestCount     int                           `json:"manifestCount"`
}

// LocalPlacementReference represents a reference to a Placement resource
type LocalPlacementReference struct {
	Name                string `json:"name"`
	RolloutStrategyType string `json:"rolloutStrategyType,omitempty"`
}

// ManifestWorkReplicaSetSummary represents the summary of ManifestWork states
type ManifestWorkReplicaSetSummary struct {
	Total       int `json:"total"`
	Available   int `json:"available"`
	Progressing int `json:"progressing"`
	Degraded    int `json:"degraded"`
	Applied     int `json:"applied"`
}

// MWRSPlacementSummary represents per-placement summary
type MWRSPlacementSummary struct {
	Name                    string                        `json:"name"`
	AvailableDecisionGroups string                        `json:"availableDecisionGroups"`
	Summary                 ManifestWorkReplicaSetSummary `json:"summary"`
}
