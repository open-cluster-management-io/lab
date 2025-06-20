package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlacementDecisionModelDetailed(t *testing.T) {
	decision := PlacementDecision{
		ID:        "test-id",
		Name:      "test-decision",
		Namespace: "test-namespace",
		Decisions: []ClusterDecision{
			{
				ClusterName: "cluster1",
				Reason:      "Selected",
			},
			{
				ClusterName: "cluster2",
				Reason:      "Available",
			},
		},
	}

	assert.Equal(t, "test-id", decision.ID)
	assert.Equal(t, "test-decision", decision.Name)
	assert.Equal(t, "test-namespace", decision.Namespace)
	assert.Len(t, decision.Decisions, 2)
	assert.Equal(t, "cluster1", decision.Decisions[0].ClusterName)
	assert.Equal(t, "Selected", decision.Decisions[0].Reason)
	assert.Equal(t, "cluster2", decision.Decisions[1].ClusterName)
	assert.Equal(t, "Available", decision.Decisions[1].Reason)
}

func TestPlacementDecisionWithEmptyDecisions(t *testing.T) {
	decision := PlacementDecision{
		ID:        "empty-id",
		Name:      "empty-decision",
		Namespace: "default",
		Decisions: []ClusterDecision{},
	}

	assert.Equal(t, "empty-id", decision.ID)
	assert.Equal(t, "empty-decision", decision.Name)
	assert.Equal(t, "default", decision.Namespace)
	assert.Empty(t, decision.Decisions)
}

func TestClusterDecisionModelDetailed(t *testing.T) {
	decision := ClusterDecision{
		ClusterName: "production-cluster",
		Reason:      "MatchesRequirements",
	}

	assert.Equal(t, "production-cluster", decision.ClusterName)
	assert.Equal(t, "MatchesRequirements", decision.Reason)
}
