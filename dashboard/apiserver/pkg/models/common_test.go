package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionModel(t *testing.T) {
	condition := Condition{
		Type:               "Available",
		Status:             "True",
		LastTransitionTime: "2023-01-01T00:00:00Z",
		Reason:             "ClusterAvailable",
		Message:            "Cluster is available",
	}

	assert.Equal(t, "Available", condition.Type)
	assert.Equal(t, "True", condition.Status)
	assert.Equal(t, "2023-01-01T00:00:00Z", condition.LastTransitionTime)
	assert.Equal(t, "ClusterAvailable", condition.Reason)
	assert.Equal(t, "Cluster is available", condition.Message)
}

func TestConditionWithMinimalFields(t *testing.T) {
	condition := Condition{
		Type:   "Ready",
		Status: "False",
	}

	assert.Equal(t, "Ready", condition.Type)
	assert.Equal(t, "False", condition.Status)
	assert.Empty(t, condition.LastTransitionTime)
	assert.Empty(t, condition.Reason)
	assert.Empty(t, condition.Message)
}
