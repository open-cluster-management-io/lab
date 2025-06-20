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

func TestGetClusterSets(t *testing.T) {
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

			GetClusterSets(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetClusterSet(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		clusterSetName string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			clusterSetName: "test-clusterset",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "name", Value: tt.clusterSetName}}

			ctx := context.Background()

			GetClusterSet(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
