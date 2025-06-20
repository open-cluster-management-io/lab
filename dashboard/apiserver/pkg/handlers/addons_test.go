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

func TestGetClusterAddons(t *testing.T) {
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
		{
			name:           "client with nil addon client",
			clusterName:    "test-cluster",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "name", Value: tt.clusterName}}

			ctx := context.Background()

			GetClusterAddons(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetClusterAddon(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		clusterName    string
		addonName      string
		client         *client.OCMClient
		expectedStatus int
	}{
		{
			name:           "nil client",
			clusterName:    "test-cluster",
			addonName:      "test-addon",
			client:         nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "client with nil addon client",
			clusterName:    "test-cluster",
			addonName:      "test-addon",
			client:         &client.OCMClient{},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "name", Value: tt.clusterName},
				{Key: "addonName", Value: tt.addonName},
			}

			ctx := context.Background()

			GetClusterAddon(c, tt.client, ctx)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
