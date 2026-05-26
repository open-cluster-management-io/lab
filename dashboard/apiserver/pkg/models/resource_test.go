package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedResourceModel(t *testing.T) {
	intVal := int64(2)
	resource := ManagedResource{
		ID:               "cluster1/deploy-nginx/0",
		Kind:             "Deployment",
		APIVersion:       "apps/v1",
		Name:             "nginx",
		Namespace:        "default",
		Cluster:          "cluster1",
		ManifestWorkName: "deploy-nginx",
		Ordinal:          0,
		Status:           "Applied",
		Conditions: []Condition{
			{Type: "Applied", Status: "True", Reason: "AppliedManifestComplete"},
		},
		StatusFeedback: &StatusFeedbackResult{
			Values: []FeedbackValue{
				{
					Name: "ReadyReplicas",
					Value: FieldValue{
						Type:    "Integer",
						Integer: &intVal,
					},
				},
			},
		},
		RawResource: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
		},
	}

	assert.Equal(t, "cluster1/deploy-nginx/0", resource.ID)
	assert.Equal(t, "Deployment", resource.Kind)
	assert.Equal(t, "apps/v1", resource.APIVersion)
	assert.Equal(t, "nginx", resource.Name)
	assert.Equal(t, "default", resource.Namespace)
	assert.Equal(t, "cluster1", resource.Cluster)
	assert.Equal(t, "deploy-nginx", resource.ManifestWorkName)
	assert.Equal(t, 0, resource.Ordinal)
	assert.Equal(t, "Applied", resource.Status)
	assert.Len(t, resource.Conditions, 1)
	assert.NotNil(t, resource.StatusFeedback)
	assert.Len(t, resource.StatusFeedback.Values, 1)
	assert.Equal(t, int64(2), *resource.StatusFeedback.Values[0].Value.Integer)
	assert.NotNil(t, resource.RawResource)
}

func TestManagedResourceWithoutOptionalFields(t *testing.T) {
	resource := ManagedResource{
		ID:               "cluster1/deploy-nginx/0",
		Kind:             "ConfigMap",
		APIVersion:       "v1",
		Name:             "settings",
		Cluster:          "cluster1",
		ManifestWorkName: "deploy-nginx",
		Ordinal:          1,
		Status:           "Pending",
	}

	assert.Equal(t, "", resource.Namespace)
	assert.Nil(t, resource.Conditions)
	assert.Nil(t, resource.StatusFeedback)
	assert.Nil(t, resource.RawResource)
}

func TestManagedResourceListModel(t *testing.T) {
	list := ManagedResourceList{
		Resources: []ManagedResource{
			{ID: "c1/mw/0", Kind: "Deployment", Cluster: "cluster1"},
			{ID: "c1/mw/1", Kind: "Service", Cluster: "cluster1"},
			{ID: "c2/mw/0", Kind: "Deployment", Cluster: "cluster2"},
		},
		AvailableKinds: []string{"Deployment", "Service"},
		Clusters:       []string{"cluster1", "cluster2"},
		Namespaces:     []string{"default", "monitoring"},
	}

	assert.Len(t, list.Resources, 3)
	assert.Equal(t, []string{"Deployment", "Service"}, list.AvailableKinds)
	assert.Equal(t, []string{"cluster1", "cluster2"}, list.Clusters)
	assert.Equal(t, []string{"default", "monitoring"}, list.Namespaces)
}

func TestManagedResourceListEmpty(t *testing.T) {
	list := ManagedResourceList{
		Resources: []ManagedResource{},
	}

	assert.Empty(t, list.Resources)
	assert.Nil(t, list.AvailableKinds)
	assert.Nil(t, list.Clusters)
	assert.Nil(t, list.Namespaces)
}

func TestManagedResourceIDFormat(t *testing.T) {
	tests := []struct {
		cluster      string
		manifestWork string
		ordinal      int
		expectedID   string
	}{
		{"cluster1", "deploy-nginx", 0, "cluster1/deploy-nginx/0"},
		{"cluster2", "monitoring-stack", 3, "cluster2/monitoring-stack/3"},
	}

	for _, tt := range tests {
		resource := ManagedResource{
			ID:               tt.expectedID,
			Cluster:          tt.cluster,
			ManifestWorkName: tt.manifestWork,
			Ordinal:          tt.ordinal,
		}
		assert.Equal(t, tt.expectedID, resource.ID)
	}
}
