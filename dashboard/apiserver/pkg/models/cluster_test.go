package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClusterModel(t *testing.T) {
	cluster := Cluster{
		ID:     "test-id",
		Name:   "test-cluster",
		Status: "Online",
		Labels: map[string]string{
			"env": "test",
		},
		Conditions: []Condition{
			{
				Type:   "Available",
				Status: "True",
			},
		},
		ClusterClaims: []ClusterClaim{
			{
				Name:  "platform",
				Value: "AWS",
			},
		},
		Taints: []Taint{
			{
				Key:    "test-key",
				Value:  "test-value",
				Effect: "NoSchedule",
			},
		},
		ManagedClusterClientConfigs: []ManagedClusterClientConfig{
			{
				URL:      "https://test-cluster:6443",
				CABundle: "test-ca-bundle",
			},
		},
	}

	assert.Equal(t, "test-id", cluster.ID)
	assert.Equal(t, "test-cluster", cluster.Name)
	assert.Equal(t, "Online", cluster.Status)
	assert.Equal(t, "test", cluster.Labels["env"])
	assert.Len(t, cluster.Conditions, 1)
	assert.Equal(t, "Available", cluster.Conditions[0].Type)
	assert.Len(t, cluster.ClusterClaims, 1)
	assert.Equal(t, "platform", cluster.ClusterClaims[0].Name)
	assert.Len(t, cluster.Taints, 1)
	assert.Equal(t, "test-key", cluster.Taints[0].Key)
	assert.Len(t, cluster.ManagedClusterClientConfigs, 1)
	assert.Equal(t, "https://test-cluster:6443", cluster.ManagedClusterClientConfigs[0].URL)
}

func TestClusterClaimModel(t *testing.T) {
	claim := ClusterClaim{
		Name:  "test-claim",
		Value: "test-value",
	}

	assert.Equal(t, "test-claim", claim.Name)
	assert.Equal(t, "test-value", claim.Value)
}

func TestTaintModel(t *testing.T) {
	taint := Taint{
		Key:    "test-key",
		Value:  "test-value",
		Effect: "NoSchedule",
	}

	assert.Equal(t, "test-key", taint.Key)
	assert.Equal(t, "test-value", taint.Value)
	assert.Equal(t, "NoSchedule", taint.Effect)
}

func TestClusterStatusModel(t *testing.T) {
	status := ClusterStatus{
		Available: true,
		Joined:    true,
		Conditions: []Condition{
			{
				Type:   "Available",
				Status: "True",
			},
		},
	}

	assert.True(t, status.Available)
	assert.True(t, status.Joined)
	assert.Len(t, status.Conditions, 1)
}

func TestManagedClusterClientConfigModel(t *testing.T) {
	config := ManagedClusterClientConfig{
		URL:      "https://test-cluster:6443",
		CABundle: "test-ca-bundle",
	}

	assert.Equal(t, "https://test-cluster:6443", config.URL)
	assert.Equal(t, "test-ca-bundle", config.CABundle)
}

func TestLabelSelectorModel(t *testing.T) {
	selector := LabelSelector{
		MatchLabels: map[string]string{
			"env": "prod",
		},
	}

	assert.Equal(t, "prod", selector.MatchLabels["env"])
}

func TestClusterSelectorModel(t *testing.T) {
	selector := ClusterSelector{
		SelectorType: "LabelSelector",
		LabelSelector: &LabelSelector{
			MatchLabels: map[string]string{
				"env": "prod",
			},
		},
	}

	assert.Equal(t, "LabelSelector", selector.SelectorType)
	assert.NotNil(t, selector.LabelSelector)
	assert.Equal(t, "prod", selector.LabelSelector.MatchLabels["env"])
}
