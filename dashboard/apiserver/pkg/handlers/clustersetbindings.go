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

// GetAllClusterSetBindings retrieves all ManagedClusterSetBindings across all namespaces
func GetAllClusterSetBindings(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Since we don't have a direct way to list resources across all namespaces with the OCM client,
	// we'll use an empty string to indicate "all namespaces" if the API supports it
	list, err := ocmClient.ClusterClient.ClusterV1beta2().ManagedClusterSetBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		// If listing across all namespaces is not supported, we'll handle the error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to list clustersetbindings across all namespaces: " + err.Error()})
		return
	}

	// Convert to our simplified ClusterSetBinding models
	allBindings := make([]models.ManagedClusterSetBinding, 0, len(list.Items))

	for _, item := range list.Items {
		binding := models.ManagedClusterSetBinding{
			ID:                string(item.GetUID()),
			Name:              item.GetName(),
			Namespace:         item.GetNamespace(),
			CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
			Spec: models.ManagedClusterSetBindingSpec{
				ClusterSet: item.Spec.ClusterSet,
			},
		}

		// Extract status info (conditions)
		for _, condition := range item.Status.Conditions {
			binding.Status.Conditions = append(binding.Status.Conditions, models.Condition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}

		allBindings = append(allBindings, binding)
	}

	c.JSON(http.StatusOK, allBindings)
}

// GetClusterSetBindings retrieves all ManagedClusterSetBindings for a specific namespace
func GetClusterSetBindings(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Get the cluster set bindings for the specified namespace
	list, err := ocmClient.ClusterClient.ClusterV1beta2().ManagedClusterSetBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ClusterSetBinding models
	clusterSetBindings := make([]models.ManagedClusterSetBinding, 0, len(list.Items))
	for _, item := range list.Items {
		clusterSetBinding := models.ManagedClusterSetBinding{
			ID:                string(item.GetUID()),
			Name:              item.GetName(),
			Namespace:         item.GetNamespace(),
			CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
			Spec: models.ManagedClusterSetBindingSpec{
				ClusterSet: item.Spec.ClusterSet,
			},
		}

		// Extract status info (conditions)
		for _, condition := range item.Status.Conditions {
			clusterSetBinding.Status.Conditions = append(clusterSetBinding.Status.Conditions, models.Condition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}

		clusterSetBindings = append(clusterSetBindings, clusterSetBinding)
	}

	c.JSON(http.StatusOK, clusterSetBindings)
}

// GetClusterSetBinding retrieves a specific ManagedClusterSetBinding by name in a namespace
func GetClusterSetBinding(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Get the cluster set binding by name
	item, err := ocmClient.ClusterClient.ClusterV1beta2().ManagedClusterSetBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ClusterSetBinding model
	clusterSetBinding := models.ManagedClusterSetBinding{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Namespace:         item.GetNamespace(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
		Spec: models.ManagedClusterSetBindingSpec{
			ClusterSet: item.Spec.ClusterSet,
		},
	}

	// Extract status info (conditions)
	for _, condition := range item.Status.Conditions {
		clusterSetBinding.Status.Conditions = append(clusterSetBinding.Status.Conditions, models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	c.JSON(http.StatusOK, clusterSetBinding)
}
