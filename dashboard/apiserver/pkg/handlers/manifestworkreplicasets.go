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
	workv1alpha1 "open-cluster-management.io/api/work/v1alpha1"
)

// GetAllManifestWorkReplicaSets retrieves all ManifestWorkReplicaSets across all namespaces
func GetAllManifestWorkReplicaSets(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	list, err := ocmClient.WorkClient.WorkV1alpha1().ManifestWorkReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]models.ManifestWorkReplicaSet, 0, len(list.Items))
	for _, item := range list.Items {
		results = append(results, convertMWRS(&item))
	}

	c.JSON(http.StatusOK, results)
}

// GetManifestWorkReplicaSets retrieves all ManifestWorkReplicaSets for a specific namespace
func GetManifestWorkReplicaSets(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	list, err := ocmClient.WorkClient.WorkV1alpha1().ManifestWorkReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]models.ManifestWorkReplicaSet, 0, len(list.Items))
	for _, item := range list.Items {
		results = append(results, convertMWRS(&item))
	}

	c.JSON(http.StatusOK, results)
}

// GetManifestWorkReplicaSet retrieves a specific ManifestWorkReplicaSet by name in a namespace
func GetManifestWorkReplicaSet(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	item, err := ocmClient.WorkClient.WorkV1alpha1().ManifestWorkReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, convertMWRS(item))
}

// GetManifestWorksByReplicaSet lists all ManifestWorks created by a specific ManifestWorkReplicaSet
func GetManifestWorksByReplicaSet(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// ManifestWorks created by an MWRS are labeled: work.open-cluster-management.io/manifestworkreplicaset=<namespace>.<name>
	labelSelector := fmt.Sprintf("work.open-cluster-management.io/manifestworkreplicaset=%s.%s", namespace, name)
	list, err := ocmClient.WorkClient.WorkV1().ManifestWorks("").List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	manifestWorks := make([]models.ManifestWork, 0, len(list.Items))
	for _, item := range list.Items {
		manifestWorks = append(manifestWorks, convertManifestWork(item))
	}

	c.JSON(http.StatusOK, manifestWorks)
}

// convertMWRS converts a typed ManifestWorkReplicaSet to our API model
func convertMWRS(item *workv1alpha1.ManifestWorkReplicaSet) models.ManifestWorkReplicaSet {
	mwrs := models.ManifestWorkReplicaSet{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Namespace:         item.GetNamespace(),
		Labels:            item.GetLabels(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
		ManifestCount:     len(item.Spec.ManifestWorkTemplate.Workload.Manifests),
		Summary: models.ManifestWorkReplicaSetSummary{
			Total:       item.Status.Summary.Total,
			Available:   item.Status.Summary.Available,
			Progressing: item.Status.Summary.Progressing,
			Degraded:    item.Status.Summary.Degraded,
			Applied:     item.Status.Summary.Applied,
		},
	}

	// Placement refs
	for _, ref := range item.Spec.PlacementRefs {
		mwrs.PlacementRefs = append(mwrs.PlacementRefs, models.LocalPlacementReference{
			Name:                ref.Name,
			RolloutStrategyType: string(ref.RolloutStrategy.Type),
		})
	}

	// Conditions
	for _, condition := range item.Status.Conditions {
		mwrs.Conditions = append(mwrs.Conditions, models.Condition{
			Type:               condition.Type,
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// Placement summaries
	for _, ps := range item.Status.PlacementsSummary {
		mwrs.PlacementsSummary = append(mwrs.PlacementsSummary, models.MWRSPlacementSummary{
			Name:                    ps.Name,
			AvailableDecisionGroups: ps.AvailableDecisionGroups,
			Summary: models.ManifestWorkReplicaSetSummary{
				Total:       ps.Summary.Total,
				Available:   ps.Summary.Available,
				Progressing: ps.Summary.Progressing,
				Degraded:    ps.Summary.Degraded,
				Applied:     ps.Summary.Applied,
			},
		})
	}

	return mwrs
}
