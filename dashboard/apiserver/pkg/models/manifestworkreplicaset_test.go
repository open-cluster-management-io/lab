package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestWorkReplicaSetModel(t *testing.T) {
	mwrs := ManifestWorkReplicaSet{
		ID:        "mwrs-1",
		Name:      "deploy-nginx",
		Namespace: "default",
		Labels:    map[string]string{"app": "nginx"},
		PlacementRefs: []LocalPlacementReference{
			{Name: "all-clusters", RolloutStrategyType: "All"},
		},
		Conditions: []Condition{
			{Type: "PlacementVerified", Status: "True"},
			{Type: "ManifestworkApplied", Status: "True"},
		},
		Summary: ManifestWorkReplicaSetSummary{
			Total:       3,
			Available:   2,
			Progressing: 1,
			Degraded:    0,
			Applied:     3,
		},
		PlacementsSummary: []MWRSPlacementSummary{
			{
				Name:                    "all-clusters",
				AvailableDecisionGroups: "2/3",
				Summary: ManifestWorkReplicaSetSummary{
					Total:     3,
					Available: 2,
					Applied:   3,
				},
			},
		},
		CreationTimestamp: "2025-01-01T00:00:00Z",
		ManifestCount:    2,
	}

	assert.Equal(t, "mwrs-1", mwrs.ID)
	assert.Equal(t, "deploy-nginx", mwrs.Name)
	assert.Equal(t, "default", mwrs.Namespace)
	assert.Equal(t, "nginx", mwrs.Labels["app"])
	assert.Len(t, mwrs.PlacementRefs, 1)
	assert.Equal(t, "all-clusters", mwrs.PlacementRefs[0].Name)
	assert.Equal(t, "All", mwrs.PlacementRefs[0].RolloutStrategyType)
	assert.Len(t, mwrs.Conditions, 2)
	assert.Equal(t, 3, mwrs.Summary.Total)
	assert.Equal(t, 2, mwrs.Summary.Available)
	assert.Equal(t, 1, mwrs.Summary.Progressing)
	assert.Len(t, mwrs.PlacementsSummary, 1)
	assert.Equal(t, "2/3", mwrs.PlacementsSummary[0].AvailableDecisionGroups)
	assert.Equal(t, 2, mwrs.ManifestCount)
	assert.Equal(t, "2025-01-01T00:00:00Z", mwrs.CreationTimestamp)
}

func TestLocalPlacementReferenceModel(t *testing.T) {
	ref := LocalPlacementReference{
		Name:                "prod-clusters",
		RolloutStrategyType: "Progressive",
	}

	assert.Equal(t, "prod-clusters", ref.Name)
	assert.Equal(t, "Progressive", ref.RolloutStrategyType)
}

func TestManifestWorkReplicaSetSummaryModel(t *testing.T) {
	summary := ManifestWorkReplicaSetSummary{
		Total:       10,
		Available:   7,
		Progressing: 2,
		Degraded:    1,
		Applied:     9,
	}

	assert.Equal(t, 10, summary.Total)
	assert.Equal(t, 7, summary.Available)
	assert.Equal(t, 2, summary.Progressing)
	assert.Equal(t, 1, summary.Degraded)
	assert.Equal(t, 9, summary.Applied)
}

func TestMWRSPlacementSummaryModel(t *testing.T) {
	ps := MWRSPlacementSummary{
		Name:                    "staging-clusters",
		AvailableDecisionGroups: "3/5",
		Summary: ManifestWorkReplicaSetSummary{
			Total:     5,
			Available: 3,
			Applied:   5,
		},
	}

	assert.Equal(t, "staging-clusters", ps.Name)
	assert.Equal(t, "3/5", ps.AvailableDecisionGroups)
	assert.Equal(t, 5, ps.Summary.Total)
	assert.Equal(t, 3, ps.Summary.Available)
}

func TestManifestWorkReplicaSetEmptyOptionalFields(t *testing.T) {
	mwrs := ManifestWorkReplicaSet{
		ID:        "mwrs-empty",
		Name:      "empty-mwrs",
		Namespace: "ns",
	}

	assert.Nil(t, mwrs.Labels)
	assert.Nil(t, mwrs.PlacementRefs)
	assert.Nil(t, mwrs.Conditions)
	assert.Nil(t, mwrs.PlacementsSummary)
	assert.Equal(t, 0, mwrs.ManifestCount)
	assert.Equal(t, "", mwrs.CreationTimestamp)
	assert.Equal(t, 0, mwrs.Summary.Total)
}
