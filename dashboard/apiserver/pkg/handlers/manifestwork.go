package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// GetManifestWorks retrieves all ManifestWorks for a specific namespace
func GetManifestWorks(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Get the manifest works for the specified namespace
	list, err := ocmClient.WorkClient.WorkV1().ManifestWorks(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ManifestWork models
	manifestWorks := make([]models.ManifestWork, 0, len(list.Items))
	for _, item := range list.Items {
		manifestWork := models.ManifestWork{
			ID:                string(item.GetUID()),
			Name:              item.GetName(),
			Namespace:         item.GetNamespace(),
			Labels:            item.GetLabels(),
			CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
		}

		// Process manifests
		if len(item.Spec.Workload.Manifests) > 0 {
			manifestWork.Manifests = make([]models.Manifest, len(item.Spec.Workload.Manifests))
			for i, manifest := range item.Spec.Workload.Manifests {
				// Convert raw bytes to map[string]interface{}
				var rawObj map[string]interface{}
				if err := json.Unmarshal(manifest.Raw, &rawObj); err == nil {
					manifestWork.Manifests[i] = models.Manifest{
						RawExtension: rawObj,
					}
				}
			}
		}

		// Extract conditions
		for _, condition := range item.Status.Conditions {
			manifestWork.Conditions = append(manifestWork.Conditions, models.Condition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
				Reason:             condition.Reason,
				Message:            condition.Message,
			})
		}

		// Process resource status
		if len(item.Status.ResourceStatus.Manifests) > 0 {
			manifestWork.ResourceStatus.Manifests = make([]models.ManifestCondition, len(item.Status.ResourceStatus.Manifests))
			for i, manifestStatus := range item.Status.ResourceStatus.Manifests {
				manifestCondition := models.ManifestCondition{
					ResourceMeta: models.ManifestResourceMeta{
						Ordinal:   manifestStatus.ResourceMeta.Ordinal,
						Group:     manifestStatus.ResourceMeta.Group,
						Version:   manifestStatus.ResourceMeta.Version,
						Kind:      manifestStatus.ResourceMeta.Kind,
						Resource:  manifestStatus.ResourceMeta.Resource,
						Name:      manifestStatus.ResourceMeta.Name,
						Namespace: manifestStatus.ResourceMeta.Namespace,
					},
				}

				// Process conditions for this manifest
				for _, condition := range manifestStatus.Conditions {
					manifestCondition.Conditions = append(manifestCondition.Conditions, models.Condition{
						Type:               string(condition.Type),
						Status:             string(condition.Status),
						LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
						Reason:             condition.Reason,
						Message:            condition.Message,
					})
				}

				manifestWork.ResourceStatus.Manifests[i] = manifestCondition
			}
		}

		manifestWorks = append(manifestWorks, manifestWork)
	}

	c.JSON(http.StatusOK, manifestWorks)
}

// GetManifestWork retrieves a specific ManifestWork by name in a namespace
func GetManifestWork(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	namespace := c.Param("namespace")
	name := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kubernetes client not initialized"})
		return
	}

	// Get the manifest work by name
	item, err := ocmClient.WorkClient.WorkV1().ManifestWorks(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ManifestWork model
	manifestWork := models.ManifestWork{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Namespace:         item.GetNamespace(),
		Labels:            item.GetLabels(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
	}

	// Process manifests
	if len(item.Spec.Workload.Manifests) > 0 {
		manifestWork.Manifests = make([]models.Manifest, len(item.Spec.Workload.Manifests))
		for i, manifest := range item.Spec.Workload.Manifests {
			// Convert raw bytes to map[string]interface{}
			var rawObj map[string]interface{}
			if err := json.Unmarshal(manifest.Raw, &rawObj); err == nil {
				manifestWork.Manifests[i] = models.Manifest{
					RawExtension: rawObj,
				}
			}
		}
	}

	// Extract conditions
	for _, condition := range item.Status.Conditions {
		manifestWork.Conditions = append(manifestWork.Conditions, models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// Process resource status
	if len(item.Status.ResourceStatus.Manifests) > 0 {
		manifestWork.ResourceStatus.Manifests = make([]models.ManifestCondition, len(item.Status.ResourceStatus.Manifests))
		for i, manifestStatus := range item.Status.ResourceStatus.Manifests {
			manifestCondition := models.ManifestCondition{
				ResourceMeta: models.ManifestResourceMeta{
					Ordinal:   manifestStatus.ResourceMeta.Ordinal,
					Group:     manifestStatus.ResourceMeta.Group,
					Version:   manifestStatus.ResourceMeta.Version,
					Kind:      manifestStatus.ResourceMeta.Kind,
					Resource:  manifestStatus.ResourceMeta.Resource,
					Name:      manifestStatus.ResourceMeta.Name,
					Namespace: manifestStatus.ResourceMeta.Namespace,
				},
			}

			// Process conditions for this manifest
			for _, condition := range manifestStatus.Conditions {
				manifestCondition.Conditions = append(manifestCondition.Conditions, models.Condition{
					Type:               string(condition.Type),
					Status:             string(condition.Status),
					LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
					Reason:             condition.Reason,
					Message:            condition.Message,
				})
			}

			manifestWork.ResourceStatus.Manifests[i] = manifestCondition
		}
	}

	c.JSON(http.StatusOK, manifestWork)
}
