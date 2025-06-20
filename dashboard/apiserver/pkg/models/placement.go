package models

// MatchExpression represents a label/claim expression for a Placement
type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// LabelSelectorWithExpressions represents a label selector with expressions
type LabelSelectorWithExpressions struct {
	MatchLabels      map[string]string `json:"matchLabels,omitempty"`
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
}

// ClaimSelectorWithExpressions represents a claim selector with expressions
type ClaimSelectorWithExpressions struct {
	MatchExpressions []MatchExpression `json:"matchExpressions,omitempty"`
}

// CelSelectorWithExpressions represents a CEL selector with expressions
type CelSelectorWithExpressions struct {
	CelExpressions []string `json:"celExpressions,omitempty"`
}

// RequiredClusterSelector represents required cluster selector
type RequiredClusterSelector struct {
	LabelSelector *LabelSelectorWithExpressions `json:"labelSelector,omitempty"`
	ClaimSelector *ClaimSelectorWithExpressions `json:"claimSelector,omitempty"`
	CelSelector   *CelSelectorWithExpressions   `json:"celSelector,omitempty"`
}

// Predicate represents a placement predicate
type Predicate struct {
	RequiredClusterSelector *RequiredClusterSelector `json:"requiredClusterSelector,omitempty"`
}

// AddOnScore represents addon score configuration
type AddOnScore struct {
	ResourceName string `json:"resourceName"`
	ScoreName    string `json:"scoreName"`
}

// ScoreCoordinate represents score coordinate
type ScoreCoordinate struct {
	Type    string      `json:"type,omitempty"`
	BuiltIn string      `json:"builtIn,omitempty"`
	AddOn   *AddOnScore `json:"addOn,omitempty"`
}

// PrioritizerConfig represents the configuration of a prioritizer
type PrioritizerConfig struct {
	ScoreCoordinate *ScoreCoordinate `json:"scoreCoordinate,omitempty"`
	Weight          int32            `json:"weight,omitempty"`
}

// PrioritizerPolicy represents prioritizer policy
type PrioritizerPolicy struct {
	Mode           string              `json:"mode,omitempty"`
	Configurations []PrioritizerConfig `json:"configurations,omitempty"`
}

// GroupClusterSelector represents a selector for a group of clusters
type GroupClusterSelector struct {
	LabelSelector *LabelSelectorWithExpressions `json:"labelSelector,omitempty"`
}

// DecisionGroup represents a group in decision strategy
type DecisionGroup struct {
	GroupName            string               `json:"groupName,omitempty"`
	GroupClusterSelector GroupClusterSelector `json:"groupClusterSelector,omitempty"`
}

// GroupStrategy represents group strategy for decisions
type GroupStrategy struct {
	DecisionGroups           []DecisionGroup `json:"decisionGroups,omitempty"`
	ClustersPerDecisionGroup string          `json:"clustersPerDecisionGroup,omitempty"`
}

// DecisionStrategy represents strategy for placement decisions
type DecisionStrategy struct {
	GroupStrategy GroupStrategy `json:"groupStrategy,omitempty"`
}

// PlacementToleration represents a toleration for placement
type PlacementToleration struct {
	Key               string `json:"key,omitempty"`
	Operator          string `json:"operator,omitempty"`
	Value             string `json:"value,omitempty"`
	Effect            string `json:"effect,omitempty"`
	TolerationSeconds *int64 `json:"tolerationSeconds,omitempty"`
}

// DecisionGroupStatus represents status of a decision group
type DecisionGroupStatus struct {
	DecisionGroupIndex int32    `json:"decisionGroupIndex"`
	DecisionGroupName  string   `json:"decisionGroupName,omitempty"`
	Decisions          []string `json:"decisions,omitempty"`
	ClusterCount       int32    `json:"clusterCount"`
}

// Placement represents a simplified OCM Placement
type Placement struct {
	ID                       string                `json:"id"`
	Name                     string                `json:"name"`
	Namespace                string                `json:"namespace"`
	CreationTimestamp        string                `json:"creationTimestamp,omitempty"`
	ClusterSets              []string              `json:"clusterSets,omitempty"`
	NumberOfClusters         *int32                `json:"numberOfClusters,omitempty"`
	Predicates               []Predicate           `json:"predicates,omitempty"`
	PrioritizerPolicy        *PrioritizerPolicy    `json:"prioritizerPolicy,omitempty"`
	Tolerations              []PlacementToleration `json:"tolerations,omitempty"`
	DecisionStrategy         *DecisionStrategy     `json:"decisionStrategy,omitempty"`
	NumberOfSelectedClusters int32                 `json:"numberOfSelectedClusters"`
	DecisionGroups           []DecisionGroupStatus `json:"decisionGroups,omitempty"`
	Conditions               []Condition           `json:"conditions,omitempty"`
	Satisfied                bool                  `json:"satisfied"`
	ReasonMessage            string                `json:"reasonMessage,omitempty"`
}

// ClusterDecision represents a single cluster decision
type ClusterDecision struct {
	ClusterName string `json:"clusterName"`
	Reason      string `json:"reason"`
}

// PlacementDecision represents a simplified OCM PlacementDecision
type PlacementDecision struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Decisions []ClusterDecision `json:"decisions,omitempty"`
}

// Helper function to create a pointer to an int32
func IntPtr(i int32) *int32 {
	return &i
}
