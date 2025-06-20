package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/handlers"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// validateToken validates a Bearer token using Kubernetes TokenReview API
func validateToken(token string, ocmClient *client.OCMClient, ctx context.Context) bool {
	if ocmClient == nil || ocmClient.KubernetesClient == nil {
		log.Println("OCM client or Kubernetes client is nil")
		return false
	}

	// Create TokenReview request
	tokenReview := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token: token,
		},
	}

	// Send TokenReview to Kubernetes API
	result, err := ocmClient.KubernetesClient.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
	if err != nil {
		log.Printf("TokenReview API call failed: %v", err)
		return false
	}

	// Check if token is authenticated
	if !result.Status.Authenticated {
		log.Printf("Token not authenticated: %s", result.Status.Error)
		return false
	}

	log.Printf("Token authenticated for user: %s", result.Status.User.Username)
	return true
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SetupServer initializes the HTTP server with all required routes
func SetupServer(ocmClient *client.OCMClient, ctx context.Context, debugMode bool) *gin.Engine {
	// Check if debug mode is enabled
	if debugMode {
		log.Println("Debug mode enabled")
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Set up Gin router
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API routes
	api := r.Group("/api")
	{
		// Enhanced authorization middleware with TokenReview validation
		authMiddleware := func(c *gin.Context) {
			// Check if authentication is bypassed
			if os.Getenv("DASHBOARD_BYPASS_AUTH") == "true" {
				log.Println("Authentication bypassed (DASHBOARD_BYPASS_AUTH=true)")
				c.Next()
				return
			}

			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				log.Println("Authorization header missing")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
				c.Abort()
				return
			}

			// Extract token from "Bearer <token>" format
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				log.Println("Invalid authorization header format")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format. Expected: Bearer <token>"})
				c.Abort()
				return
			}

			token := tokenParts[1]

			// Validate token using Kubernetes TokenReview API
			if !validateToken(token, ocmClient, ctx) {
				log.Printf("Token validation failed for token: %s...", token[:min(len(token), 10)])
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
				c.Abort()
				return
			}

			log.Println("Token validation successful")
			c.Next()
		}

		// Register cluster routes
		api.GET("/clusters", authMiddleware, func(c *gin.Context) {
			handlers.GetClusters(c, ocmClient, ctx)
		})

		api.GET("/clusters/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetCluster(c, ocmClient, ctx)
		})

		// Register cluster addon routes
		api.GET("/clusters/:name/addons", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterAddons(c, ocmClient, ctx)
		})

		api.GET("/clusters/:name/addons/:addonName", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterAddon(c, ocmClient, ctx)
		})

		// Register clusterset routes
		api.GET("/clustersets", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterSets(c, ocmClient, ctx)
		})

		api.GET("/clustersets/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterSet(c, ocmClient, ctx)
		})

		// Register clustersetbinding routes
		api.GET("/clustersetbindings", authMiddleware, func(c *gin.Context) {
			handlers.GetAllClusterSetBindings(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/clustersetbindings", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterSetBindings(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/clustersetbindings/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetClusterSetBinding(c, ocmClient, ctx)
		})

		// Register manifestwork routes
		api.GET("/namespaces/:namespace/manifestworks", authMiddleware, func(c *gin.Context) {
			handlers.GetManifestWorks(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/manifestworks/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetManifestWork(c, ocmClient, ctx)
		})

		// Register placement routes
		api.GET("/placements", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacements(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placements", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacementsByNamespace(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placements/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacement(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placements/:name/decisions", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacementDecisions(c, ocmClient, ctx)
		})

		// Register placementdecision routes
		api.GET("/placementdecisions", authMiddleware, func(c *gin.Context) {
			handlers.GetAllPlacementDecisions(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placementdecisions", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacementDecisionsByNamespace(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placementdecisions/:name", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacementDecision(c, ocmClient, ctx)
		})

		api.GET("/namespaces/:namespace/placements/:name/placementdecisions", authMiddleware, func(c *gin.Context) {
			handlers.GetPlacementDecisionsByPlacement(c, ocmClient, ctx)
		})

		// Register streaming routes
		api.GET("/stream/clusters", authMiddleware, func(c *gin.Context) {
			handlers.StreamClusters(c, ocmClient.Interface, ctx)
		})
	}

	// Add health check endpoint (no authentication required)
	r.GET("/health", func(c *gin.Context) {
		// Simple health check - you can add more sophisticated checks here
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Alternative health check endpoint following Kubernetes conventions
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API status endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OCM Dashboard API Server",
			"version": "v0.0.1",
			"status":  "running",
			"endpoints": gin.H{
				"health":  "/health",
				"healthz": "/healthz",
				"api":     "/api/*",
			},
		})
	})

	return r
}

// RunServer starts the HTTP server on the specified port
func RunServer(r *gin.Engine) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	r.Run(":" + port)
}
