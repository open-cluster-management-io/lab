package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedClusterSetBindingModel(t *testing.T) {
	binding := ManagedClusterSetBinding{
		ID:        "test-id",
		Name:      "test-binding",
		Namespace: "test-namespace",
		Spec: ManagedClusterSetBindingSpec{
			ClusterSet: "test-clusterset",
		},
		Status: ManagedClusterSetBindingStatus{
			Conditions: []Condition{
				{
					Type:   "Bound",
					Status: "True",
					Reason: "ClusterSetBound",
				},
			},
		},
		CreationTimestamp: "2023-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-id", binding.ID)
	assert.Equal(t, "test-binding", binding.Name)
	assert.Equal(t, "test-namespace", binding.Namespace)
	assert.Equal(t, "test-clusterset", binding.Spec.ClusterSet)
	assert.Equal(t, "2023-01-01T00:00:00Z", binding.CreationTimestamp)
	assert.Len(t, binding.Status.Conditions, 1)
	assert.Equal(t, "Bound", binding.Status.Conditions[0].Type)
	assert.Equal(t, "True", binding.Status.Conditions[0].Status)
}

func TestManagedClusterSetBindingWithMinimalFields(t *testing.T) {
	binding := ManagedClusterSetBinding{
		ID:        "minimal-id",
		Name:      "minimal-binding",
		Namespace: "default",
		Spec: ManagedClusterSetBindingSpec{
			ClusterSet: "default-clusterset",
		},
	}

	assert.Equal(t, "minimal-id", binding.ID)
	assert.Equal(t, "minimal-binding", binding.Name)
	assert.Equal(t, "default", binding.Namespace)
	assert.Equal(t, "default-clusterset", binding.Spec.ClusterSet)
	assert.Empty(t, binding.CreationTimestamp)
	assert.Empty(t, binding.Status.Conditions)
}
