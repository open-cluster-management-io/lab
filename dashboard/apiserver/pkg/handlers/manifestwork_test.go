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
