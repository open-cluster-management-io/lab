package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/dynamic"

	"open-cluster-management-io/lab/apiserver/pkg/models"
)

func TestStreamClusters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		dynamicClient  dynamic.Interface
		expectedStatus int
	}{
		{
			name:           "nil client",
			dynamicClient:  nil,
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ctx := context.Background()

			StreamClusters(c, tt.dynamicClient, ctx)

			if tt.expectedStatus == 500 {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestConvertToCluster(t *testing.T) {
	tests := []struct {
		name        string
		item        interface{}
		expectError bool
		expected    models.Cluster
	}{
		{
			name: "valid cluster object",
			item: map[string]interface{}{
				"metadata": map[string]interface{}{
					"uid":               "test-uid",
					"name":              "test-cluster",
					"creationTimestamp": "2023-01-01T00:00:00Z",
					"labels": map[string]interface{}{
						"env": "test",
					},
				},
				"status": map[string]interface{}{
					"version": map[string]interface{}{
						"kubernetes": "v1.20.0",
					},
					"conditions": []interface{}{
						map[string]interface{}{
							"type":               "ManagedClusterConditionAvailable",
							"status":             "True",
							"reason":             "Available",
							"message":            "Cluster is available",
							"lastTransitionTime": "2023-01-01T00:00:00Z",
						},
					},
				},
			},
			expectError: false,
			expected: models.Cluster{
				ID:                "test-uid",
				Name:              "test-cluster",
				Status:            "Online",
				Version:           "v1.20.0",
				CreationTimestamp: "2023-01-01T00:00:00Z",
				Labels: map[string]string{
					"env": "test",
				},
				Conditions: []models.Condition{
					{
						Type:               "ManagedClusterConditionAvailable",
						Status:             "True",
						Reason:             "Available",
						Message:            "Cluster is available",
						LastTransitionTime: "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			name: "cluster with offline status",
			item: map[string]interface{}{
				"metadata": map[string]interface{}{
					"uid":               "test-uid-2",
					"name":              "test-cluster-2",
					"creationTimestamp": "2023-01-01T00:00:00Z",
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":               "ManagedClusterConditionAvailable",
							"status":             "False",
							"reason":             "Unavailable",
							"message":            "Cluster is not available",
							"lastTransitionTime": "2023-01-01T00:00:00Z",
						},
					},
				},
			},
			expectError: false,
			expected: models.Cluster{
				ID:                "test-uid-2",
				Name:              "test-cluster-2",
				Status:            "Offline",
				CreationTimestamp: "2023-01-01T00:00:00Z",
				Conditions: []models.Condition{
					{
						Type:               "ManagedClusterConditionAvailable",
						Status:             "False",
						Reason:             "Unavailable",
						Message:            "Cluster is not available",
						LastTransitionTime: "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			name: "invalid object without metadata",
			item: map[string]interface{}{
				"spec": map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name:        "invalid object type",
			item:        "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster, err := convertToCluster(tt.item)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, cluster.ID)
				assert.Equal(t, tt.expected.Name, cluster.Name)
				assert.Equal(t, tt.expected.Status, cluster.Status)
				assert.Equal(t, tt.expected.Version, cluster.Version)
				assert.Equal(t, tt.expected.CreationTimestamp, cluster.CreationTimestamp)
				assert.Equal(t, tt.expected.Labels, cluster.Labels)

				if len(tt.expected.Conditions) > 0 {
					assert.Len(t, cluster.Conditions, len(tt.expected.Conditions))
					assert.Equal(t, tt.expected.Conditions[0].Type, cluster.Conditions[0].Type)
					assert.Equal(t, tt.expected.Conditions[0].Status, cluster.Conditions[0].Status)
				}
			}
		})
	}
}

func TestConvertToClusterWithMarshalError(t *testing.T) {
	invalidItem := make(chan int)

	_, err := convertToCluster(invalidItem)
	assert.Error(t, err)
}

func TestConvertToClusterWithUnmarshalError(t *testing.T) {
	item := map[string]interface{}{
		"metadata": "invalid-metadata-type",
	}

	cluster, err := convertToCluster(item)
	assert.Error(t, err)
	assert.Equal(t, models.Cluster{}, cluster)
}
