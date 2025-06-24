package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// GetClusters handles retrieving all clusters
func GetClusters(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Use the OCM typed client to list ManagedClusters
	clusterList, err := ocmClient.ClusterClient.ClusterV1().ManagedClusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified Cluster format
	clusters := make([]models.Cluster, 0, len(clusterList.Items))
	for _, item := range clusterList.Items {
		// Create a cluster object from the ManagedCluster
		cluster := convertManagedClusterToCluster(item)
		clusters = append(clusters, cluster)
	}

	c.JSON(http.StatusOK, clusters)
}

// GetCluster handles retrieving a specific cluster by name
func GetCluster(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Use the OCM typed client to get a specific ManagedCluster
	managedCluster, err := ocmClient.ClusterClient.ClusterV1().ManagedClusters().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified Cluster format
	cluster := convertManagedClusterToCluster(*managedCluster)

	c.JSON(http.StatusOK, cluster)
}

// Helper function to convert a ManagedCluster to our simplified Cluster model
func convertManagedClusterToCluster(managedCluster clusterv1.ManagedCluster) models.Cluster {
	cluster := models.Cluster{
		ID:                string(managedCluster.ObjectMeta.UID),
		Name:              managedCluster.ObjectMeta.Name,
		Labels:            managedCluster.ObjectMeta.Labels,
		CreationTimestamp: managedCluster.ObjectMeta.CreationTimestamp.Format(time.RFC3339),
		Status:            "Unknown",
	}

	// Extract Kubernetes version
	if managedCluster.Status.Version.Kubernetes != "" {
		cluster.Version = managedCluster.Status.Version.Kubernetes
	}

	// Extract capacity and allocatable resources
	if len(managedCluster.Status.Capacity) > 0 {
		resourceMap := make(map[string]string)
		for k, v := range managedCluster.Status.Capacity {
			resourceMap[string(k)] = v.String()
		}
		cluster.Capacity = resourceMap
	}

	if len(managedCluster.Status.Allocatable) > 0 {
		resourceMap := make(map[string]string)
		for k, v := range managedCluster.Status.Allocatable {
			resourceMap[string(k)] = v.String()
		}
		cluster.Allocatable = resourceMap
	}

	// Convert cluster claims
	if len(managedCluster.Status.ClusterClaims) > 0 {
		claims := make([]models.ClusterClaim, 0, len(managedCluster.Status.ClusterClaims))
		for _, c := range managedCluster.Status.ClusterClaims {
			claims = append(claims, models.ClusterClaim{
				Name:  c.Name,
				Value: c.Value,
			})
		}
		cluster.ClusterClaims = claims
	}

	// Convert conditions
	if len(managedCluster.Status.Conditions) > 0 {
		conditions := make([]models.Condition, 0, len(managedCluster.Status.Conditions))
		for _, c := range managedCluster.Status.Conditions {
			conditions = append(conditions, models.Condition{
				Type:               string(c.Type),
				Status:             string(c.Status),
				LastTransitionTime: c.LastTransitionTime.Format(time.RFC3339),
				Reason:             c.Reason,
				Message:            c.Message,
			})

			// Update status based on ManagedClusterConditionAvailable condition
			if c.Type == clusterv1.ManagedClusterConditionAvailable && c.Status == metav1.ConditionTrue {
				cluster.Status = "Online"
			} else if c.Type == clusterv1.ManagedClusterConditionAvailable && c.Status != metav1.ConditionTrue {
				cluster.Status = "Offline"
			}
		}
		cluster.Conditions = conditions
	}

	// Add cluster client configs
	if len(managedCluster.Spec.ManagedClusterClientConfigs) > 0 {
		configs := make([]models.ManagedClusterClientConfig, 0, len(managedCluster.Spec.ManagedClusterClientConfigs))
		for _, cc := range managedCluster.Spec.ManagedClusterClientConfigs {
			configs = append(configs, models.ManagedClusterClientConfig{
				URL:      cc.URL,
				CABundle: string(cc.CABundle),
			})
		}
		cluster.ManagedClusterClientConfigs = configs
	}

	return cluster
}
