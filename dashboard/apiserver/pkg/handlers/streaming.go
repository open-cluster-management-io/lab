package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// StreamClusters handles streaming cluster updates via SSE
func StreamClusters(c *gin.Context, dynamicClient dynamic.Interface, ctx context.Context) {
	// Ensure we have a client before proceeding
	if dynamicClient == nil {
		c.JSON(500, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Flush()

	// Create a watch for ManagedClusters
	watcher, err := dynamicClient.Resource(client.ManagedClusterResource).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		c.Writer.Write([]byte(fmt.Sprintf("event: error\ndata: %s\n\n", err.Error())))
		c.Writer.Flush()
		return
	}
	defer watcher.Stop()

	// Send initial list of clusters
	initialList, err := dynamicClient.Resource(client.ManagedClusterResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.Writer.Write([]byte(fmt.Sprintf("event: error\ndata: %s\n\n", err.Error())))
		c.Writer.Flush()
		return
	}

	// Convert to our simplified Cluster format
	clusters := make([]models.Cluster, 0, len(initialList.Items))
	for _, item := range initialList.Items {
		cluster, err := convertToCluster(item)
		if err != nil {
			continue
		}
		clusters = append(clusters, cluster)
	}

	// Send initial data
	data, err := json.Marshal(clusters)
	if err != nil {
		c.Writer.Write([]byte(fmt.Sprintf("event: error\ndata: %s\n\n", err.Error())))
		c.Writer.Flush()
		return
	}
	c.Writer.Write([]byte(fmt.Sprintf("event: clusters\ndata: %s\n\n", data)))
	c.Writer.Flush()

	// Listen for watch events
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				// Channel closed, end streaming
				return
			}

			// Process the event
			switch event.Type {
			case watch.Added, watch.Modified, watch.Deleted:
				// Get updated list to ensure we have full state
				updatedList, err := dynamicClient.Resource(client.ManagedClusterResource).List(ctx, metav1.ListOptions{})
				if err != nil {
					continue
				}

				// Convert again to our simplified format
				updatedClusters := make([]models.Cluster, 0, len(updatedList.Items))
				for _, item := range updatedList.Items {
					cluster, err := convertToCluster(item)
					if err != nil {
						continue
					}
					updatedClusters = append(updatedClusters, cluster)
				}

				// Send updated data
				data, err := json.Marshal(updatedClusters)
				if err != nil {
					continue
				}
				c.Writer.Write([]byte(fmt.Sprintf("event: clusters\ndata: %s\n\n", data)))
				c.Writer.Flush()
			case watch.Error:
				c.Writer.Write([]byte(fmt.Sprintf("event: error\ndata: Watch error occurred\n\n")))
				c.Writer.Flush()
			}
		case <-time.After(30 * time.Second):
			// Send a keepalive ping every 30 seconds
			c.Writer.Write([]byte(": ping\n\n"))
			c.Writer.Flush()
		}
	}
}

// Helper function to convert unstructured item to Cluster
func convertToCluster(item interface{}) (models.Cluster, error) {
	// Re-use the logic from GetClusters but without context - keep it simple for streaming
	unstructuredMap, err := json.Marshal(item)
	if err != nil {
		return models.Cluster{}, err
	}

	// Convert to our simplified format
	var obj map[string]interface{}
	if err := json.Unmarshal(unstructuredMap, &obj); err != nil {
		return models.Cluster{}, err
	}

	// Extract metadata
	metadata, ok := obj["metadata"].(map[string]interface{})
	if !ok {
		return models.Cluster{}, fmt.Errorf("metadata not found or not a map")
	}

	cluster := models.Cluster{
		ID:                fmt.Sprintf("%v", metadata["uid"]),
		Name:              fmt.Sprintf("%v", metadata["name"]),
		Status:            "Unknown",
		CreationTimestamp: fmt.Sprintf("%v", metadata["creationTimestamp"]),
	}

	// Extract labels
	if labels, ok := metadata["labels"].(map[string]interface{}); ok {
		cluster.Labels = make(map[string]string)
		for k, v := range labels {
			cluster.Labels[k] = fmt.Sprintf("%v", v)
		}
	}

	// Extract version from status
	if status, ok := obj["status"].(map[string]interface{}); ok {
		if version, ok := status["version"].(map[string]interface{}); ok {
			if k8sVersion, ok := version["kubernetes"].(string); ok {
				cluster.Version = k8sVersion
			}
		}

		// Extract conditions
		if conditions, ok := status["conditions"].([]interface{}); ok {
			for _, c := range conditions {
				condMap, ok := c.(map[string]interface{})
				if !ok {
					continue
				}

				condition := models.Condition{
					Type:               fmt.Sprintf("%v", condMap["type"]),
					Status:             fmt.Sprintf("%v", condMap["status"]),
					Reason:             fmt.Sprintf("%v", condMap["reason"]),
					Message:            fmt.Sprintf("%v", condMap["message"]),
					LastTransitionTime: fmt.Sprintf("%v", condMap["lastTransitionTime"]),
				}

				cluster.Conditions = append(cluster.Conditions, condition)

				// Update status based on the ManagedClusterConditionAvailable condition
				if condition.Type == "ManagedClusterConditionAvailable" && condition.Status == "True" {
					cluster.Status = "Online"
				} else if condition.Type == "ManagedClusterConditionAvailable" && condition.Status != "True" {
					cluster.Status = "Offline"
				}
			}
		}
	}

	return cluster, nil
}
