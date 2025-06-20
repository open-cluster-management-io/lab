package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterSetSpecModel(t *testing.T) {
	spec := ClusterSetSpec{
		ClusterSelector: ClusterSelector{
			SelectorType: "LabelSelector",
			LabelSelector: &LabelSelector{
				MatchLabels: map[string]string{
					"env": "prod",
				},
			},
		},
	}

	assert.Equal(t, "LabelSelector", spec.ClusterSelector.SelectorType)
	assert.NotNil(t, spec.ClusterSelector.LabelSelector)
	assert.Equal(t, "prod", spec.ClusterSelector.LabelSelector.MatchLabels["env"])
}

func TestClusterSetStatusModel(t *testing.T) {
	status := ClusterSetStatus{
		Conditions: []Condition{
			{
				Type:   "ClusterSetEmpty",
				Status: "False",
				Reason: "ClustersSelected",
			},
		},
	}

	assert.Len(t, status.Conditions, 1)
	assert.Equal(t, "ClusterSetEmpty", status.Conditions[0].Type)
	assert.Equal(t, "False", status.Conditions[0].Status)
	assert.Equal(t, "ClustersSelected", status.Conditions[0].Reason)
}

func TestClusterSetModel(t *testing.T) {
	clusterSet := ClusterSet{
		ID:   "test-id",
		Name: "test-clusterset",
		Labels: map[string]string{
			"env": "test",
		},
		Spec: ClusterSetSpec{
			ClusterSelector: ClusterSelector{
				SelectorType: "LabelSelector",
			},
		},
		Status: ClusterSetStatus{
			Conditions: []Condition{
				{
					Type:   "ClusterSetEmpty",
					Status: "False",
				},
			},
		},
		CreationTimestamp: "2023-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-id", clusterSet.ID)
	assert.Equal(t, "test-clusterset", clusterSet.Name)
	assert.Equal(t, "test", clusterSet.Labels["env"])
	assert.Equal(t, "LabelSelector", clusterSet.Spec.ClusterSelector.SelectorType)
	assert.Len(t, clusterSet.Status.Conditions, 1)
	assert.Equal(t, "2023-01-01T00:00:00Z", clusterSet.CreationTimestamp)
}

func TestClusterSetWithMinimalFields(t *testing.T) {
	clusterSet := ClusterSet{
		ID:   "minimal-id",
		Name: "minimal-clusterset",
	}

	assert.Equal(t, "minimal-id", clusterSet.ID)
	assert.Equal(t, "minimal-clusterset", clusterSet.Name)
	assert.Nil(t, clusterSet.Labels)
	assert.Empty(t, clusterSet.CreationTimestamp)
}
