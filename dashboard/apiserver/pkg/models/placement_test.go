package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntPtr(t *testing.T) {
	value := int32(42)
	ptr := IntPtr(value)

	assert.NotNil(t, ptr)
	assert.Equal(t, value, *ptr)
}

func TestMatchExpressionModel(t *testing.T) {
	expr := MatchExpression{
		Key:      "region",
		Operator: "In",
		Values:   []string{"us-east-1", "us-west-2"},
	}

	assert.Equal(t, "region", expr.Key)
	assert.Equal(t, "In", expr.Operator)
	assert.Len(t, expr.Values, 2)
	assert.Contains(t, expr.Values, "us-east-1")
}

func TestLabelSelectorWithExpressionsModel(t *testing.T) {
	selector := LabelSelectorWithExpressions{
		MatchLabels: map[string]string{
			"env": "prod",
		},
		MatchExpressions: []MatchExpression{
			{
				Key:      "region",
				Operator: "In",
				Values:   []string{"us-east-1"},
			},
		},
	}

	assert.Equal(t, "prod", selector.MatchLabels["env"])
	assert.Len(t, selector.MatchExpressions, 1)
	assert.Equal(t, "region", selector.MatchExpressions[0].Key)
}

func TestClaimSelectorWithExpressionsModel(t *testing.T) {
	selector := ClaimSelectorWithExpressions{
		MatchExpressions: []MatchExpression{
			{
				Key:      "platform",
				Operator: "In",
				Values:   []string{"AWS"},
			},
		},
	}

	assert.Len(t, selector.MatchExpressions, 1)
	assert.Equal(t, "platform", selector.MatchExpressions[0].Key)
}

func TestRequiredClusterSelectorModel(t *testing.T) {
	selector := RequiredClusterSelector{
		LabelSelector: &LabelSelectorWithExpressions{
			MatchLabels: map[string]string{
				"env": "prod",
			},
		},
		ClaimSelector: &ClaimSelectorWithExpressions{
			MatchExpressions: []MatchExpression{
				{
					Key:      "platform",
					Operator: "In",
					Values:   []string{"AWS"},
				},
			},
		},
	}

	assert.NotNil(t, selector.LabelSelector)
	assert.NotNil(t, selector.ClaimSelector)
	assert.Equal(t, "prod", selector.LabelSelector.MatchLabels["env"])
	assert.Len(t, selector.ClaimSelector.MatchExpressions, 1)
}

func TestPlacementModel(t *testing.T) {
	numberOfClusters := int32(3)
	placement := Placement{
		ID:               "test-id",
		Name:             "test-placement",
		Namespace:        "test-namespace",
		ClusterSets:      []string{"clusterset1"},
		NumberOfClusters: &numberOfClusters,
		Predicates: []Predicate{
			{
				RequiredClusterSelector: &RequiredClusterSelector{
					LabelSelector: &LabelSelectorWithExpressions{
						MatchLabels: map[string]string{
							"env": "prod",
						},
					},
				},
			},
		},
		NumberOfSelectedClusters: 2,
		Satisfied:                true,
	}

	assert.Equal(t, "test-id", placement.ID)
	assert.Equal(t, "test-placement", placement.Name)
	assert.Equal(t, "test-namespace", placement.Namespace)
	assert.Len(t, placement.ClusterSets, 1)
	assert.Equal(t, "clusterset1", placement.ClusterSets[0])
	assert.NotNil(t, placement.NumberOfClusters)
	assert.Equal(t, int32(3), *placement.NumberOfClusters)
	assert.Len(t, placement.Predicates, 1)
	assert.Equal(t, int32(2), placement.NumberOfSelectedClusters)
	assert.True(t, placement.Satisfied)
}

func TestPlacementDecisionModel(t *testing.T) {
	decision := PlacementDecision{
		ID:        "test-id",
		Name:      "test-decision",
		Namespace: "test-namespace",
		Decisions: []ClusterDecision{
			{
				ClusterName: "cluster1",
				Reason:      "Selected",
			},
		},
	}

	assert.Equal(t, "test-id", decision.ID)
	assert.Equal(t, "test-decision", decision.Name)
	assert.Equal(t, "test-namespace", decision.Namespace)
	assert.Len(t, decision.Decisions, 1)
	assert.Equal(t, "cluster1", decision.Decisions[0].ClusterName)
	assert.Equal(t, "Selected", decision.Decisions[0].Reason)
}

func TestClusterDecisionModel(t *testing.T) {
	decision := ClusterDecision{
		ClusterName: "test-cluster",
		Reason:      "Available",
	}

	assert.Equal(t, "test-cluster", decision.ClusterName)
	assert.Equal(t, "Available", decision.Reason)
}

func TestDecisionGroupStatusModel(t *testing.T) {
	group := DecisionGroupStatus{
		DecisionGroupIndex: 0,
		DecisionGroupName:  "group1",
		Decisions:          []string{"cluster1", "cluster2"},
		ClusterCount:       2,
	}

	assert.Equal(t, int32(0), group.DecisionGroupIndex)
	assert.Equal(t, "group1", group.DecisionGroupName)
	assert.Len(t, group.Decisions, 2)
	assert.Equal(t, int32(2), group.ClusterCount)
}
