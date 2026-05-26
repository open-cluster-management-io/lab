package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workv1 "open-cluster-management.io/api/work/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// convertStatusFeedback converts OCM StatusFeedbackResult to our API model
func convertStatusFeedback(sf workv1.StatusFeedbackResult) *models.StatusFeedbackResult {
	result := &models.StatusFeedbackResult{}
	for _, fv := range sf.Values {
		val := models.FeedbackValue{
			Name: fv.Name,
			Value: models.FieldValue{
				Type:    string(fv.Value.Type),
				Integer: fv.Value.Integer,
				String:  fv.Value.String,
				Boolean: fv.Value.Boolean,
			},
		}
		result.Values = append(result.Values, val)
	}
	return result
}

// convertManifestWork converts a typed ManifestWork to our API model
func convertManifestWork(item workv1.ManifestWork) models.ManifestWork {
	mw := models.ManifestWork{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Namespace:         item.GetNamespace(),
		Labels:            item.GetLabels(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
	}

	// Process manifests
	if len(item.Spec.Workload.Manifests) > 0 {
		mw.Manifests = make([]models.Manifest, len(item.Spec.Workload.Manifests))
		for i, manifest := range item.Spec.Workload.Manifests {
			var rawObj map[string]interface{}
			if err := json.Unmarshal(manifest.Raw, &rawObj); err == nil {
				mw.Manifests[i] = models.Manifest{RawExtension: rawObj}
			}
		}
	}

	// Extract conditions
	for _, condition := range item.Status.Conditions {
		mw.Conditions = append(mw.Conditions, models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// Process resource status
	if len(item.Status.ResourceStatus.Manifests) > 0 {
		mw.ResourceStatus.Manifests = make([]models.ManifestCondition, len(item.Status.ResourceStatus.Manifests))
		for i, ms := range item.Status.ResourceStatus.Manifests {
			mc := models.ManifestCondition{
				ResourceMeta: models.ManifestResourceMeta{
					Ordinal:   ms.ResourceMeta.Ordinal,
					Group:     ms.ResourceMeta.Group,
					Version:   ms.ResourceMeta.Version,
					Kind:      ms.ResourceMeta.Kind,
					Resource:  ms.ResourceMeta.Resource,
					Name:      ms.ResourceMeta.Name,
					Namespace: ms.ResourceMeta.Namespace,
				},
			}
			for _, condition := range ms.Conditions {
				mc.Conditions = append(mc.Conditions, models.Condition{
					Type:               string(condition.Type),
					Status:             string(condition.Status),
					LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
					Reason:             condition.Reason,
					Message:            condition.Message,
				})
			}
			if len(ms.StatusFeedbacks.Values) > 0 {
				mc.StatusFeedback = convertStatusFeedback(ms.StatusFeedbacks)
			}
			mw.ResourceStatus.Manifests[i] = mc
		}
	}

	return mw
}

// GetAllManifestWorks retrieves all ManifestWorks across all namespaces
func GetAllManifestWorks(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	list, err := ocmClient.WorkClient.WorkV1().ManifestWorks("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]models.ManifestWork, 0, len(list.Items))
	for _, item := range list.Items {
		results = append(results, convertManifestWork(item))
	}

	c.JSON(http.StatusOK, results)
}

// GetManifestWorks retrieves all ManifestWorks for a specific namespace
func GetManifestWorks(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	list, err := ocmClient.WorkClient.WorkV1().ManifestWorks(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]models.ManifestWork, 0, len(list.Items))
	for _, item := range list.Items {
		results = append(results, convertManifestWork(item))
	}

	c.JSON(http.StatusOK, results)
}

// GetManifestWork retrieves a specific ManifestWork by name in a namespace
func GetManifestWork(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	item, err := ocmClient.WorkClient.WorkV1().ManifestWorks(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, convertManifestWork(*item))
}
