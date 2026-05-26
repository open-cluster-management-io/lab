package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	workv1 "open-cluster-management.io/api/work/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
)

func TestGetManifestWorks(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "namespace", Value: tt.namespace}}

			ctx := context.Background()

			GetManifestWorks(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetManifestWork(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		namespace      string
		manifestName   string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			namespace:      "test-namespace",
			manifestName:   "test-manifest",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "namespace", Value: tt.namespace},
				{Key: "name", Value: tt.manifestName},
			}

			ctx := context.Background()

			GetManifestWork(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestConvertStatusFeedback(t *testing.T) {
	tests := []struct {
		name           string
		input          workv1.StatusFeedbackResult
		expectedLen    int
		expectedFirst  string
		expectedType   string
		expectedInt    *int64
		expectedString *string
	}{
		{
			name: "integer feedback value",
			input: workv1.StatusFeedbackResult{
				Values: []workv1.FeedbackValue{
					{
						Name: "ReadyReplicas",
						Value: workv1.FieldValue{
							Type:    workv1.Integer,
							Integer: ptr.To(int64(2)),
						},
					},
				},
			},
			expectedLen:   1,
			expectedFirst: "ReadyReplicas",
			expectedType:  "Integer",
			expectedInt:   ptr.To(int64(2)),
		},
		{
			name: "string feedback value",
			input: workv1.StatusFeedbackResult{
				Values: []workv1.FeedbackValue{
					{
						Name: "Available",
						Value: workv1.FieldValue{
							Type:   workv1.String,
							String: ptr.To("True"),
						},
					},
				},
			},
			expectedLen:    1,
			expectedFirst:  "Available",
			expectedType:   "String",
			expectedString: ptr.To("True"),
		},
		{
			name: "multiple feedback values",
			input: workv1.StatusFeedbackResult{
				Values: []workv1.FeedbackValue{
					{
						Name:  "ReadyReplicas",
						Value: workv1.FieldValue{Type: workv1.Integer, Integer: ptr.To(int64(1))},
					},
					{
						Name:  "Replicas",
						Value: workv1.FieldValue{Type: workv1.Integer, Integer: ptr.To(int64(3))},
					},
					{
						Name:  "clusterIP",
						Value: workv1.FieldValue{Type: workv1.String, String: ptr.To("10.96.0.1")},
					},
				},
			},
			expectedLen:   3,
			expectedFirst: "ReadyReplicas",
			expectedType:  "Integer",
			expectedInt:   ptr.To(int64(1)),
		},
		{
			name:        "empty feedback",
			input:       workv1.StatusFeedbackResult{},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertStatusFeedback(tt.input)
			assert.NotNil(t, result)
			assert.Len(t, result.Values, tt.expectedLen)

			if tt.expectedLen > 0 {
				first := result.Values[0]
				assert.Equal(t, tt.expectedFirst, first.Name)
				assert.Equal(t, tt.expectedType, first.Value.Type)
				if tt.expectedInt != nil {
					assert.Equal(t, *tt.expectedInt, *first.Value.Integer)
				}
				if tt.expectedString != nil {
					assert.Equal(t, *tt.expectedString, *first.Value.String)
				}
			}
		})
	}
}

func TestConvertManifestWorkWithStatusFeedback(t *testing.T) {
	now := metav1.Now()
	configMapJSON, _ := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      "test",
			"namespace": "default",
		},
	})

	mw := workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-mw",
			Namespace:         "cluster1",
			UID:               types.UID("uid-123"),
			CreationTimestamp: now,
			Labels: map[string]string{
				"work.open-cluster-management.io/manifestworkreplicaset": "default.deploy-nginx",
			},
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					{RawExtension: runtime.RawExtension{Raw: configMapJSON}},
				},
			},
		},
		Status: workv1.ManifestWorkStatus{
			Conditions: []metav1.Condition{
				{
					Type:               workv1.WorkApplied,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "AppliedManifestComplete",
				},
			},
			ResourceStatus: workv1.ManifestResourceStatus{
				Manifests: []workv1.ManifestCondition{
					{
						ResourceMeta: workv1.ManifestResourceMeta{
							Ordinal:   0,
							Group:     "",
							Version:   "v1",
							Kind:      "ConfigMap",
							Resource:  "configmaps",
							Name:      "test",
							Namespace: "default",
						},
						Conditions: []metav1.Condition{
							{
								Type:               "Applied",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: now,
							},
						},
						StatusFeedbacks: workv1.StatusFeedbackResult{
							Values: []workv1.FeedbackValue{
								{
									Name: "ReadyReplicas",
									Value: workv1.FieldValue{
										Type:    workv1.Integer,
										Integer: ptr.To(int64(2)),
									},
								},
								{
									Name: "Available",
									Value: workv1.FieldValue{
										Type:   workv1.String,
										String: ptr.To("True"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := convertManifestWork(mw)

	assert.Equal(t, "test-mw", result.Name)
	assert.Equal(t, "cluster1", result.Namespace)
	assert.Equal(t, "uid-123", result.ID)
	assert.Equal(t, "default.deploy-nginx", result.Labels["work.open-cluster-management.io/manifestworkreplicaset"])

	// Conditions
	assert.Len(t, result.Conditions, 1)
	assert.Equal(t, "Applied", result.Conditions[0].Type)
	assert.Equal(t, "True", result.Conditions[0].Status)

	// Resource status with StatusFeedback
	assert.Len(t, result.ResourceStatus.Manifests, 1)
	mc := result.ResourceStatus.Manifests[0]
	assert.Equal(t, "ConfigMap", mc.ResourceMeta.Kind)
	assert.Equal(t, "test", mc.ResourceMeta.Name)
	assert.Len(t, mc.Conditions, 1)

	assert.NotNil(t, mc.StatusFeedback)
	assert.Len(t, mc.StatusFeedback.Values, 2)
	assert.Equal(t, "ReadyReplicas", mc.StatusFeedback.Values[0].Name)
	assert.Equal(t, "Integer", mc.StatusFeedback.Values[0].Value.Type)
	assert.Equal(t, int64(2), *mc.StatusFeedback.Values[0].Value.Integer)
	assert.Equal(t, "Available", mc.StatusFeedback.Values[1].Name)
	assert.Equal(t, "True", *mc.StatusFeedback.Values[1].Value.String)
}

func TestConvertManifestWorkWithoutStatusFeedback(t *testing.T) {
	now := metav1.Now()
	configMapJSON, _ := json.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]any{"name": "test"},
	})

	mw := workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-mw",
			Namespace:         "cluster1",
			UID:               types.UID("uid-456"),
			CreationTimestamp: now,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					{RawExtension: runtime.RawExtension{Raw: configMapJSON}},
				},
			},
		},
		Status: workv1.ManifestWorkStatus{
			ResourceStatus: workv1.ManifestResourceStatus{
				Manifests: []workv1.ManifestCondition{
					{
						ResourceMeta: workv1.ManifestResourceMeta{
							Ordinal: 0,
							Kind:    "ConfigMap",
							Name:    "test",
						},
						Conditions: []metav1.Condition{
							{
								Type:               "Applied",
								Status:             metav1.ConditionTrue,
								LastTransitionTime: now,
							},
						},
						// No StatusFeedbacks
					},
				},
			},
		},
	}

	result := convertManifestWork(mw)
	assert.Len(t, result.ResourceStatus.Manifests, 1)
	assert.Nil(t, result.ResourceStatus.Manifests[0].StatusFeedback)
}

func TestConvertManifestWorkTimestamps(t *testing.T) {
	ts := metav1.NewTime(time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC))

	mw := workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ts-test",
			Namespace:         "cluster1",
			UID:               types.UID("uid-ts"),
			CreationTimestamp: ts,
		},
	}

	result := convertManifestWork(mw)
	assert.Equal(t, "2025-06-15T10:30:00Z", result.CreationTimestamp)
}
