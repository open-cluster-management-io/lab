package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

func TestGetPlacements(t *testing.T) {
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
		{
			name:           "client with nil cluster client",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			ctx := context.Background()

			GetPlacements(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetPlacementsByNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		namespace      string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			namespace:      "test-namespace",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "client with nil cluster client",
			namespace:      "test-namespace",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "namespace", Value: tt.namespace}}

			ctx := context.Background()

			GetPlacementsByNamespace(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetPlacement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		namespace      string
		placementName  string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			namespace:      "test-namespace",
			placementName:  "test-placement",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "client with nil cluster client",
			namespace:      "test-namespace",
			placementName:  "test-placement",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "namespace", Value: tt.namespace},
				{Key: "name", Value: tt.placementName},
			}

			ctx := context.Background()

			GetPlacement(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetPlacementDecisions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		namespace      string
		placementName  string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			namespace:      "test-namespace",
			placementName:  "test-placement",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "client with nil cluster client",
			namespace:      "test-namespace",
			placementName:  "test-placement",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "namespace", Value: tt.namespace},
				{Key: "name", Value: tt.placementName},
			}

			ctx := context.Background()

			GetPlacementDecisions(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestConvertPlacementToModel(t *testing.T) {
	now := time.Now()
	numberOfClusters := int32(3)

	tests := []struct {
		name      string
		placement clusterv1beta1.Placement
		expected  models.Placement
	}{
		{
			name: "placement with all fields",
			placement: clusterv1beta1.Placement{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-placement",
					Namespace:         "test-namespace",
					UID:               types.UID("test-uid"),
					CreationTimestamp: metav1.Time{Time: now},
				},
				Spec: clusterv1beta1.PlacementSpec{
					ClusterSets:      []string{"clusterset1", "clusterset2"},
					NumberOfClusters: &numberOfClusters,
					Predicates: []clusterv1beta1.ClusterPredicate{
						{
							RequiredClusterSelector: clusterv1beta1.ClusterSelector{
								LabelSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{
										"env": "prod",
									},
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "region",
											Operator: metav1.LabelSelectorOpIn,
											Values:   []string{"us-east-1", "us-west-2"},
										},
									},
								},
							},
						},
					},
				},
				Status: clusterv1beta1.PlacementStatus{
					NumberOfSelectedClusters: 2,
					Conditions: []metav1.Condition{
						{
							Type:               string(clusterv1beta1.PlacementConditionSatisfied),
							Status:             metav1.ConditionTrue,
							LastTransitionTime: metav1.Time{Time: now},
							Reason:             "Satisfied",
							Message:            "Placement is satisfied",
						},
					},
				},
			},
			expected: models.Placement{
				ID:                       "test-uid",
				Name:                     "test-placement",
				Namespace:                "test-namespace",
				CreationTimestamp:        now.Format(time.RFC3339),
				ClusterSets:              []string{"clusterset1", "clusterset2"},
				NumberOfClusters:         &numberOfClusters,
				NumberOfSelectedClusters: 2,
				Satisfied:                true,
			},
		},
		{
			name: "placement with minimal fields",
			placement: clusterv1beta1.Placement{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "minimal-placement",
					Namespace:         "default",
					UID:               types.UID("minimal-uid"),
					CreationTimestamp: metav1.Time{Time: now},
				},
			},
			expected: models.Placement{
				ID:                       "minimal-uid",
				Name:                     "minimal-placement",
				Namespace:                "default",
				CreationTimestamp:        now.Format(time.RFC3339),
				NumberOfSelectedClusters: 0,
				Satisfied:                false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPlacementToModel(tt.placement)

			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Namespace, result.Namespace)
			assert.Equal(t, tt.expected.CreationTimestamp, result.CreationTimestamp)
			assert.Equal(t, tt.expected.ClusterSets, result.ClusterSets)
			assert.Equal(t, tt.expected.NumberOfClusters, result.NumberOfClusters)
			assert.Equal(t, tt.expected.NumberOfSelectedClusters, result.NumberOfSelectedClusters)
			assert.Equal(t, tt.expected.Satisfied, result.Satisfied)

			if len(tt.placement.Spec.Predicates) > 0 {
				assert.NotEmpty(t, result.Predicates)
				if len(tt.placement.Spec.Predicates[0].RequiredClusterSelector.LabelSelector.MatchLabels) > 0 {
					assert.NotNil(t, result.Predicates[0].RequiredClusterSelector)
					assert.NotNil(t, result.Predicates[0].RequiredClusterSelector.LabelSelector)
				}
			}
		})
	}
}
