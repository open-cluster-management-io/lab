package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"open-cluster-management-io/lab/apiserver/pkg/client"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetupServer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		debugMode bool
		client    *client.OCMClient
	}{
		{
			name:      "debug mode enabled",
			debugMode: true,
			client:    nil,
		},
		{
			name:      "debug mode disabled",
			debugMode: false,
			client:    nil,
		},
		{
			name:      "with client",
			debugMode: false,
			client:    &client.OCMClient{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			router := SetupServer(tt.client, ctx, tt.debugMode)

			assert.NotNil(t, router)

			routes := router.Routes()
			assert.NotEmpty(t, routes)

			foundAPIRoutes := false
			for _, route := range routes {
				if route.Path == "/api/clusters" {
					foundAPIRoutes = true
					break
				}
			}
			assert.True(t, foundAPIRoutes, "API routes should be registered")
		})
	}
}

func TestRunServer(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{
			name: "default port",
			port: "",
		},
		{
			name: "custom port",
			port: "9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.port != "" {
				os.Setenv("PORT", tt.port)
				defer os.Unsetenv("PORT")
			}

			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "ok"})
			})

			assert.NotPanics(t, func() {
				go func() {
					defer func() {
						if r := recover(); r != nil {
							if r != "test exit" {
								panic(r)
							}
						}
					}()
					RunServer(router)
				}()
			})
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		bypassAuth     string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "bypass auth enabled",
			bypassAuth:     "true",
			authHeader:     "",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "bypass auth disabled with header",
			bypassAuth:     "false",
			authHeader:     "Bearer token",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "bypass auth disabled without header",
			bypassAuth:     "false",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DASHBOARD_BYPASS_AUTH", tt.bypassAuth)
			defer os.Unsetenv("DASHBOARD_BYPASS_AUTH")

			ctx := context.Background()
			router := SetupServer(nil, ctx, false)

			req, _ := http.NewRequest("GET", "/api/clusters", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCORSConfiguration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	router := SetupServer(nil, ctx, false)

	req, _ := http.NewRequest("OPTIONS", "/api/clusters", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestRootRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	router := SetupServer(nil, ctx, false)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/static/index.html", w.Header().Get("Location"))
}
