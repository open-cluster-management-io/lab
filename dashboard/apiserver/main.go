package main

import (
	"context"
	"os"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/server"
)

func main() {
	// Check if debug mode is enabled
	debugMode := os.Getenv("DASHBOARD_DEBUG") == "true"

	// Create a context
	ctx := context.Background()

	// Initialize Kubernetes client
	ocmClient := client.CreateKubernetesClient()

	// Set up and run the server
	r := server.SetupServer(ocmClient, ctx, debugMode)
	server.RunServer(r)
}
