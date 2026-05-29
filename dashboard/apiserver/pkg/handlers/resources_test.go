package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"open-cluster-management-io/lab/apiserver/pkg/models"
)

func TestDeriveResourceStatus(t *testing.T) {
	tests := []struct {
		name       string
		conditions []models.Condition
		expected   string
	}{
		{
			name:       "empty conditions returns Pending",
			conditions: []models.Condition{},
			expected:   "Pending",
		},
		{
			name:       "nil conditions returns Pending",
			conditions: nil,
			expected:   "Pending",
		},
		{
			name: "Applied True returns Applied",
			conditions: []models.Condition{
				{Type: "Applied", Status: "True"},
			},
			expected: "Applied",
		},
		{
			name: "Applied False returns Failed",
			conditions: []models.Condition{
				{Type: "Applied", Status: "False"},
			},
			expected: "Failed",
		},
		{
			name: "Available True returns Available",
			conditions: []models.Condition{
				{Type: "Available", Status: "True"},
			},
			expected: "Available",
		},
		{
			name: "Available False returns Failed",
			conditions: []models.Condition{
				{Type: "Available", Status: "False"},
			},
			expected: "Failed",
		},
		{
			name: "Applied takes precedence over Available",
			conditions: []models.Condition{
				{Type: "Applied", Status: "True"},
				{Type: "Available", Status: "False"},
			},
			expected: "Applied",
		},
		{
			name: "unrecognized condition type returns Failed",
			conditions: []models.Condition{
				{Type: "Other", Status: "True"},
			},
			expected: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveResourceStatus(tt.conditions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSortedSetKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]struct{}
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]struct{}{},
			expected: []string{},
		},
		{
			name: "single key",
			input: map[string]struct{}{
				"Deployment": {},
			},
			expected: []string{"Deployment"},
		},
		{
			name: "multiple keys sorted",
			input: map[string]struct{}{
				"Service":    {},
				"Deployment": {},
				"ConfigMap":  {},
			},
			expected: []string{"ConfigMap", "Deployment", "Service"},
		},
		{
			name: "excludes empty string",
			input: map[string]struct{}{
				"":          {},
				"ConfigMap": {},
				"Service":   {},
			},
			expected: []string{"ConfigMap", "Service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortedSetKeys(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- Handler nil-client tests ---

func TestGetManagedResourcesNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx := context.Background()

	GetManagedResources(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetManagedResourceNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "cluster", Value: "cluster1"},
		{Key: "manifestwork", Value: "deploy-nginx"},
		{Key: "ordinal", Value: "0"},
	}
	ctx := context.Background()

	GetManagedResource(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetManagedResourceBadOrdinal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		ordinal string
	}{
		{name: "non-numeric", ordinal: "abc"},
		{name: "negative", ordinal: "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "cluster", Value: "cluster1"},
				{Key: "manifestwork", Value: "deploy-nginx"},
				{Key: "ordinal", Value: tt.ordinal},
			}
			ctx := context.Background()

			GetManagedResource(c, nil, ctx)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}
