package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.Default()

	// Determine static files directory
	staticDir := "/app/dist" // Default for Docker
	if _, err := os.Stat("../dist"); err == nil {
		// Local development: check if dist exists in parent directory
		staticDir = "../dist"
	} else if _, err := os.Stat("./dist"); err == nil {
		// Alternative: check if dist exists in current directory
		staticDir = "./dist"
	}

	fmt.Printf("Using static directory: %s\n", staticDir)

	// Setup API proxy to forward API requests to the API container
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost:8080" // Default for same-pod communication
	}

	apiURL, err := url.Parse("http://" + apiHost)
	if err != nil {
		fmt.Printf("Error parsing API URL: %v\n", err)
		apiURL, _ = url.Parse("http://localhost:8080")
	}

	proxy := httputil.NewSingleHostReverseProxy(apiURL)

	// Modify proxy to handle headers properly
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Allow CORS
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		resp.Header.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		return nil
	}

	// API proxy routes - forward all /api/* requests to API container
	r.Any("/api/*path", func(c *gin.Context) {
		fmt.Printf("Proxying API request: %s %s\n", c.Request.Method, c.Request.URL.Path)
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "ocm-dashboard-frontend",
		})
	})

	// Handle favicon with GET instead of StaticFile to avoid NoRoute conflicts
	faviconPath := filepath.Join(staticDir, "favicons", "favicon.ico")
	r.GET("/favicon.ico", func(c *gin.Context) {
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(http.StatusNotFound)
		}
	})

	// Serve other static files
	r.Static("/assets", filepath.Join(staticDir, "assets"))
	r.Static("/favicons", filepath.Join(staticDir, "favicons"))
	r.Static("/images", filepath.Join(staticDir, "images"))
	r.StaticFile("/manifest.json", filepath.Join(staticDir, "manifest.json"))

	// Serve index.html for all other routes (SPA routing)
	r.NoRoute(func(c *gin.Context) {
		// Check if the requested file exists in dist directory
		requestedPath := filepath.Join(staticDir, c.Request.URL.Path)
		if _, err := os.Stat(requestedPath); err == nil {
			c.File(requestedPath)
			return
		}

		// If file doesn't exist, serve index.html (for client-side routing)
		c.File(filepath.Join(staticDir, "index.html"))
	})

	// Start server on port 3000
	fmt.Println("Starting server on :3000")
	r.Run(":3000")
}
