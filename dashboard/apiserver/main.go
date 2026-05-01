package main

import (
	"context"
	"log"
	"os"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/server"
)

func main() {
	debugMode := os.Getenv("DASHBOARD_DEBUG") == "true"
	ctx := context.Background()

	var ocmClient *client.OCMClient
	if os.Getenv("DASHBOARD_USE_MOCK") == "true" {
		log.Println("Mock mode enabled — skipping Kubernetes client creation")
	} else {
		ocmClient = client.CreateKubernetesClient()
	}

	r := server.SetupServer(ocmClient, ctx, debugMode)
	server.RunServer(r)
}
