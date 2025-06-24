package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"

	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

// GetAllPlacementDecisions handles retrieving all placement decisions across namespaces
func GetAllPlacementDecisions(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Use the OCM cluster client to list placement decisions
	pdList, err := ocmClient.ClusterClient.ClusterV1beta1().PlacementDecisions("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified PlacementDecision format
	placementDecisions := make([]models.PlacementDecision, 0, len(pdList.Items))
	for _, pd := range pdList.Items {
		placementDecision := convertPlacementDecisionToModel(&pd)
		placementDecisions = append(placementDecisions, placementDecision)
	}

	c.JSON(http.StatusOK, placementDecisions)
}

// GetPlacementDecisionsByNamespace handles retrieving placement decisions in a specific namespace
func GetPlacementDecisionsByNamespace(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Use the OCM cluster client to list placement decisions in the namespace
	pdList, err := ocmClient.ClusterClient.ClusterV1beta1().PlacementDecisions(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified PlacementDecision format
	placementDecisions := make([]models.PlacementDecision, 0, len(pdList.Items))
	for _, pd := range pdList.Items {
		placementDecision := convertPlacementDecisionToModel(&pd)
		placementDecisions = append(placementDecisions, placementDecision)
	}

	c.JSON(http.StatusOK, placementDecisions)
}

// GetPlacementDecision handles retrieving a specific placement decision
func GetPlacementDecision(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Get the specific placement decision using the OCM cluster client
	pd, err := ocmClient.ClusterClient.ClusterV1beta1().PlacementDecisions(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified PlacementDecision format
	placementDecision := convertPlacementDecisionToModel(pd)

	c.JSON(http.StatusOK, placementDecision)
}

// GetPlacementDecisionsByPlacement handles retrieving placement decisions for a specific placement
func GetPlacementDecisionsByPlacement(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// List placement decisions for this placement
	// We need to use a label selector to find decisions related to this placement
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("cluster.open-cluster-management.io/placement=%s", name),
	}

	pdList, err := ocmClient.ClusterClient.ClusterV1beta1().PlacementDecisions(namespace).List(ctx, listOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified PlacementDecision format
	placementDecisions := make([]models.PlacementDecision, 0, len(pdList.Items))
	for _, pd := range pdList.Items {
		placementDecision := convertPlacementDecisionToModel(&pd)
		placementDecisions = append(placementDecisions, placementDecision)
	}

	c.JSON(http.StatusOK, placementDecisions)
}

// Helper function to convert a PlacementDecision resource to our model
func convertPlacementDecisionToModel(pd *clusterv1beta1.PlacementDecision) models.PlacementDecision {
	placementDecision := models.PlacementDecision{
		ID:        string(pd.GetUID()),
		Name:      pd.GetName(),
		Namespace: pd.GetNamespace(),
	}

	// Extract decisions from the PlacementDecision status
	decisions := make([]models.ClusterDecision, 0, len(pd.Status.Decisions))
	for _, decision := range pd.Status.Decisions {
		clusterDecision := models.ClusterDecision{
			ClusterName: decision.ClusterName,
			Reason:      decision.Reason,
		}
		decisions = append(decisions, clusterDecision)
	}

	placementDecision.Decisions = decisions

	return placementDecision
}
