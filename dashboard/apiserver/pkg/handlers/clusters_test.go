package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
)

func TestGetClusters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ctx := context.Background()

			GetClusters(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusInternalServerError {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestGetCluster(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		clusterName    string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			clusterName:    "test-cluster",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "name", Value: tt.clusterName}}

			ctx := context.Background()

			GetCluster(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusInternalServerError {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestConvertManagedClusterToCluster(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		managedCluster  clusterv1.ManagedCluster
		expectedStatus  string
		expectedVersion string
	}{
		{
			name: "cluster with available condition",
			managedCluster: clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
					UID:  types.UID("test-uid"),
					Labels: map[string]string{
						"env": "test",
					},
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: clusterv1.ManagedClusterStatus{
					Version: clusterv1.ManagedClusterVersion{
						Kubernetes: "v1.20.0",
					},
					Capacity: clusterv1.ResourceList{
						"cpu":    resource.MustParse("4"),
						"memory": resource.MustParse("8Gi"),
					},
					Allocatable: clusterv1.ResourceList{
						"cpu":    resource.MustParse("3.5"),
						"memory": resource.MustParse("7Gi"),
					},
					Conditions: []metav1.Condition{
						{
							Type:               string(clusterv1.ManagedClusterConditionAvailable),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Time{Time: now},
							Reason:             "Available",
							Message:            "Cluster is available",
						},
					},
					ClusterClaims: []clusterv1.ManagedClusterClaim{
						{
							Name:  "platform.open-cluster-management.io",
							Value: "AWS",
						},
					},
				},
				Spec: clusterv1.ManagedClusterSpec{
					ManagedClusterClientConfigs: []clusterv1.ClientConfig{
						{
							URL:      "https://test-cluster:6443",
							CABundle: []byte("test-ca-bundle"),
						},
					},
				},
			},
			expectedStatus:  "Online",
			expectedVersion: "v1.20.0",
		},
		{
			name: "cluster with unavailable condition",
			managedCluster: clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-cluster-2",
					UID:               types.UID("test-uid-2"),
					CreationTimestamp: metav1.Time{Time: now},
				},
				Status: clusterv1.ManagedClusterStatus{
					Conditions: []metav1.Condition{
						{
							Type:               string(clusterv1.ManagedClusterConditionAvailable),
							Status:             metav1.ConditionFalse,
							LastTransitionTime: metav1.Time{Time: now},
							Reason:             "Unavailable",
							Message:            "Cluster is not available",
						},
					},
				},
			},
			expectedStatus:  "Offline",
			expectedVersion: "",
		},
		{
			name: "cluster without conditions",
			managedCluster: clusterv1.ManagedCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-cluster-3",
					UID:               types.UID("test-uid-3"),
					CreationTimestamp: metav1.Time{Time: now},
				},
			},
			expectedStatus:  "Unknown",
			expectedVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := convertManagedClusterToCluster(tt.managedCluster)

			assert.Equal(t, string(tt.managedCluster.UID), cluster.ID)
			assert.Equal(t, tt.managedCluster.Name, cluster.Name)
			assert.Equal(t, tt.expectedStatus, cluster.Status)
			assert.Equal(t, tt.expectedVersion, cluster.Version)
			assert.Equal(t, tt.managedCluster.Labels, cluster.Labels)
			assert.Equal(t, tt.managedCluster.CreationTimestamp.Format(time.RFC3339), cluster.CreationTimestamp)

			if len(tt.managedCluster.Status.Capacity) > 0 {
				assert.NotEmpty(t, cluster.Capacity)
				assert.Equal(t, "4", cluster.Capacity["cpu"])
			}

			if len(tt.managedCluster.Status.Allocatable) > 0 {
				assert.NotEmpty(t, cluster.Allocatable)
				assert.Equal(t, "3500m", cluster.Allocatable["cpu"])
			}

			if len(tt.managedCluster.Status.ClusterClaims) > 0 {
				assert.Len(t, cluster.ClusterClaims, len(tt.managedCluster.Status.ClusterClaims))
				assert.Equal(t, "platform.open-cluster-management.io", cluster.ClusterClaims[0].Name)
				assert.Equal(t, "AWS", cluster.ClusterClaims[0].Value)
			}

			if len(tt.managedCluster.Status.Conditions) > 0 {
				assert.Len(t, cluster.Conditions, len(tt.managedCluster.Status.Conditions))
			}

			if len(tt.managedCluster.Spec.ManagedClusterClientConfigs) > 0 {
				assert.Len(t, cluster.ManagedClusterClientConfigs, len(tt.managedCluster.Spec.ManagedClusterClientConfigs))
				assert.Equal(t, "https://test-cluster:6443", cluster.ManagedClusterClientConfigs[0].URL)
				assert.Equal(t, "test-ca-bundle", cluster.ManagedClusterClientConfigs[0].CABundle)
			}
		})
	}
}
