package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"

	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

// GetPlacements handles retrieving all placements across namespaces
func GetPlacements(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Use the OCM cluster client to list placements
	placementList, err := ocmClient.ClusterClient.ClusterV1beta1().Placements("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified Placement format
	placements := make([]models.Placement, 0, len(placementList.Items))
	for _, placement := range placementList.Items {
		placementModel := convertPlacementToModel(placement)
		placements = append(placements, placementModel)
	}

	c.JSON(http.StatusOK, placements)
}

// GetPlacementsByNamespace handles retrieving placements for a specific namespace
func GetPlacementsByNamespace(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Use the OCM cluster client to list placements in the specified namespace
	placementList, err := ocmClient.ClusterClient.ClusterV1beta1().Placements(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified Placement format
	placements := make([]models.Placement, 0, len(placementList.Items))
	for _, placement := range placementList.Items {
		placementModel := convertPlacementToModel(placement)
		placements = append(placements, placementModel)
	}

	c.JSON(http.StatusOK, placements)
}

// GetPlacement handles retrieving a specific placement
func GetPlacement(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Get the specific placement using the OCM cluster client
	placement, err := ocmClient.ClusterClient.ClusterV1beta1().Placements(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified Placement format
	placementModel := convertPlacementToModel(*placement)

	c.JSON(http.StatusOK, placementModel)
}

// GetPlacementDecisions handles retrieving decisions for a placement
func GetPlacementDecisions(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	placementName := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.ClusterClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// List placement decisions for this placement
	// We need to use a label selector to find decisions related to this placement
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("cluster.open-cluster-management.io/placement=%s", placementName),
	}

	list, err := ocmClient.ClusterClient.ClusterV1beta1().PlacementDecisions(namespace).List(ctx, listOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified PlacementDecision format
	placementDecisions := make([]models.PlacementDecision, 0, len(list.Items))
	for _, item := range list.Items {
		placementDecsion := models.PlacementDecision{
			ID:        string(item.GetUID()),
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
		}

		decisions := make([]models.ClusterDecision, 0, len(item.Status.Decisions))

		for _, decision := range item.Status.Decisions {
			clusterDecision := models.ClusterDecision{
				ClusterName: decision.ClusterName,
				Reason:      decision.Reason,
			}
			decisions = append(decisions, clusterDecision)
		}

		placementDecsion.Decisions = decisions

		placementDecisions = append(placementDecisions, placementDecsion)
	}

	c.JSON(http.StatusOK, placementDecisions)
}

// Helper function to convert a Placement resource to our model
func convertPlacementToModel(placement clusterv1beta1.Placement) models.Placement {
	p := models.Placement{
		ID:                string(placement.UID),
		Name:              placement.Name,
		Namespace:         placement.Namespace,
		CreationTimestamp: placement.CreationTimestamp.Format(time.RFC3339),
	}

	// Extract ClusterSets
	if len(placement.Spec.ClusterSets) > 0 {
		p.ClusterSets = placement.Spec.ClusterSets
	}

	// Extract NumberOfClusters
	if placement.Spec.NumberOfClusters != nil {
		p.NumberOfClusters = models.IntPtr(int32(*placement.Spec.NumberOfClusters))
	}

	// Extract Predicates
	if len(placement.Spec.Predicates) > 0 {
		for _, predicate := range placement.Spec.Predicates {
			modelPredicate := models.Predicate{}

			// Check if the RequiredClusterSelector is set with a LabelSelector
			if predicate.RequiredClusterSelector.LabelSelector.MatchLabels != nil ||
				len(predicate.RequiredClusterSelector.LabelSelector.MatchExpressions) > 0 {

				modelPredicate.RequiredClusterSelector = &models.RequiredClusterSelector{}
				modelLabelSelector := &models.LabelSelectorWithExpressions{
					MatchLabels: predicate.RequiredClusterSelector.LabelSelector.MatchLabels,
				}

				// Process label selector expressions if they exist
				if len(predicate.RequiredClusterSelector.LabelSelector.MatchExpressions) > 0 {
					for _, expr := range predicate.RequiredClusterSelector.LabelSelector.MatchExpressions {
						modelExpr := models.MatchExpression{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						}
						modelLabelSelector.MatchExpressions = append(modelLabelSelector.MatchExpressions, modelExpr)
					}
				}

				modelPredicate.RequiredClusterSelector.LabelSelector = modelLabelSelector
			}

			// Check if the RequiredClusterSelector is set with a ClaimSelector
			if len(predicate.RequiredClusterSelector.ClaimSelector.MatchExpressions) > 0 {
				if modelPredicate.RequiredClusterSelector == nil {
					modelPredicate.RequiredClusterSelector = &models.RequiredClusterSelector{}
				}

				modelClaimSelector := &models.ClaimSelectorWithExpressions{}

				// Process claim selector expressions
				for _, expr := range predicate.RequiredClusterSelector.ClaimSelector.MatchExpressions {
					modelExpr := models.MatchExpression{
						Key:      expr.Key,
						Operator: string(expr.Operator),
						Values:   expr.Values,
					}
					modelClaimSelector.MatchExpressions = append(modelClaimSelector.MatchExpressions, modelExpr)
				}

				modelPredicate.RequiredClusterSelector.ClaimSelector = modelClaimSelector
			}

			p.Predicates = append(p.Predicates, modelPredicate)
		}
	}

	// Extract status
	p.NumberOfSelectedClusters = int32(placement.Status.NumberOfSelectedClusters)

	// Check satisfaction
	p.Satisfied = false // default to false
	for _, condition := range placement.Status.Conditions {
		// Add conditions
		modelCondition := models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
		p.Conditions = append(p.Conditions, modelCondition)

		// Check if placement is satisfied
		if condition.Type == clusterv1beta1.PlacementConditionSatisfied && condition.Status == metav1.ConditionTrue {
			p.Satisfied = true
		}
	}

	// Extract decision groups
	for i, group := range placement.Status.DecisionGroups {
		decisionGroup := models.DecisionGroupStatus{
			DecisionGroupIndex: int32(i),
			DecisionGroupName:  group.DecisionGroupName,
			Decisions:          group.Decisions,
			ClusterCount:       int32(group.ClustersCount),
		}
		p.DecisionGroups = append(p.DecisionGroups, decisionGroup)
	}

	return p
}
