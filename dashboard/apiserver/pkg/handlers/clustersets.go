package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// GetClusterSets handles retrieving all cluster sets
func GetClusterSets(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Normal processing - list real managed cluster sets using OCM client
	list, err := ocmClient.ClusterClient.ClusterV1beta2().ManagedClusterSets().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ClusterSet format
	clusterSets := make([]models.ClusterSet, 0, len(list.Items))
	for _, item := range list.Items {
		// Extract the basic metadata
		clusterSet := models.ClusterSet{
			ID:                string(item.GetUID()),
			Name:              item.GetName(),
			Labels:            item.GetLabels(),
			CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
		}

		// Extract spec info
		clusterSet.Spec.ClusterSelector.SelectorType = string(item.Spec.ClusterSelector.SelectorType)

		if item.Spec.ClusterSelector.LabelSelector != nil {
			clusterSet.Spec.ClusterSelector.LabelSelector = &models.LabelSelector{
				MatchLabels: item.Spec.ClusterSelector.LabelSelector.MatchLabels,
			}
		}

		// Extract status info
		for _, condition := range item.Status.Conditions {
			clusterSet.Status.Conditions = append(clusterSet.Status.Conditions, models.Condition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}

		clusterSets = append(clusterSets, clusterSet)
	}

	c.JSON(http.StatusOK, clusterSets)
}

// GetClusterSet handles retrieving a specific cluster set
func GetClusterSet(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Get the cluster set by name using OCM client
	item, err := ocmClient.ClusterClient.ClusterV1beta2().ManagedClusterSets().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ClusterSet format
	clusterSet := models.ClusterSet{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Labels:            item.GetLabels(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
	}

	// Extract spec info
	clusterSet.Spec.ClusterSelector.SelectorType = string(item.Spec.ClusterSelector.SelectorType)

	if item.Spec.ClusterSelector.LabelSelector != nil {
		clusterSet.Spec.ClusterSelector.LabelSelector = &models.LabelSelector{
			MatchLabels: item.Spec.ClusterSelector.LabelSelector.MatchLabels,
		}
	}

	// Extract status info
	for _, condition := range item.Status.Conditions {
		clusterSet.Status.Conditions = append(clusterSet.Status.Conditions, models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	c.JSON(http.StatusOK, clusterSet)
}
