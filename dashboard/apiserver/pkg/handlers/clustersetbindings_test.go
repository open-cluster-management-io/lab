package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"open-cluster-management-io/lab/apiserver/pkg/client"
)

func TestGetAllClusterSetBindings(t *testing.T) {
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

			GetAllClusterSetBindings(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetClusterSetBindings(t *testing.T) {
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

			GetClusterSetBindings(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetClusterSetBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		namespace      string
		bindingName    string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			namespace:      "test-namespace",
			bindingName:    "test-binding",
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
				{Key: "name", Value: tt.bindingName},
			}

			ctx := context.Background()

			GetClusterSetBinding(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
