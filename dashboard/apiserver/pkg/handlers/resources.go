package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"open-cluster-management-io/lab/apiserver/pkg/client"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

// deriveResourceStatus computes a human-readable status from per-resource conditions
func deriveResourceStatus(conditions []models.Condition) string {
	if len(conditions) == 0 {
		return "Pending"
	}
	for _, c := range conditions {
		if c.Type == "Applied" && c.Status == "True" {
			return "Applied"
		}
	}
	for _, c := range conditions {
		if c.Type == "Available" {
			if c.Status == "True" {
				return "Available"
			}
			return "Failed"
		}
	}
	return "Failed"
}

// sortedSetKeys returns sorted unique keys from a set map
func sortedSetKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// GetManagedResources lists all individual Kubernetes resources extracted from ManifestWork specs
func GetManagedResources(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	kindFilter := c.Query("kind")
	clusterFilter := c.Query("cluster")
	nsFilter := c.Query("namespace")
	includeSpec := c.Query("includeSpec") == "true"

	// If cluster filter is specified, only list ManifestWorks from that namespace (= cluster)
	listNamespace := ""
	if clusterFilter != "" {
		listNamespace = clusterFilter
	}

	list, err := ocmClient.WorkClient.WorkV1().ManifestWorks(listNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resources := make([]models.ManagedResource, 0)
	kindsSet := map[string]struct{}{}
	clustersSet := map[string]struct{}{}
	namespacesSet := map[string]struct{}{}

	for _, mw := range list.Items {
		cluster := mw.GetNamespace()
		mwName := mw.GetName()

		// Build status and feedback lookups: ordinal -> conditions/feedback
		statusByOrdinal := map[int32][]models.Condition{}
		feedbackByOrdinal := map[int32]*models.StatusFeedbackResult{}
		for _, ms := range mw.Status.ResourceStatus.Manifests {
			var conds []models.Condition
			for _, cond := range ms.Conditions {
				conds = append(conds, models.Condition{
					Type:               string(cond.Type),
					Status:             string(cond.Status),
					LastTransitionTime: cond.LastTransitionTime.Format(time.RFC3339),
					Reason:             cond.Reason,
					Message:            cond.Message,
				})
			}
			statusByOrdinal[ms.ResourceMeta.Ordinal] = conds
			if len(ms.StatusFeedbacks.Values) > 0 {
				feedbackByOrdinal[ms.ResourceMeta.Ordinal] = convertStatusFeedback(ms.StatusFeedbacks)
			}
		}

		for i, manifest := range mw.Spec.Workload.Manifests {
			var rawObj map[string]interface{}
			if err := json.Unmarshal(manifest.Raw, &rawObj); err != nil {
				continue
			}

			kind, _ := rawObj["kind"].(string)
			apiVersion, _ := rawObj["apiVersion"].(string)
			var resName, resNamespace string
			if metadata, ok := rawObj["metadata"].(map[string]interface{}); ok {
				resName, _ = metadata["name"].(string)
				resNamespace, _ = metadata["namespace"].(string)
			}

			// Apply server-side filters
			if kindFilter != "" && !strings.EqualFold(kind, kindFilter) {
				continue
			}
			if nsFilter != "" && resNamespace != nsFilter {
				continue
			}

			conditions := statusByOrdinal[int32(i)]
			status := deriveResourceStatus(conditions)

			resource := models.ManagedResource{
				ID:               fmt.Sprintf("%s/%s/%d", cluster, mwName, i),
				Kind:             kind,
				APIVersion:       apiVersion,
				Name:             resName,
				Namespace:        resNamespace,
				Cluster:          cluster,
				ManifestWorkName: mwName,
				Ordinal:          i,
				Status:           status,
				Conditions:       conditions,
				StatusFeedback:   feedbackByOrdinal[int32(i)],
			}
			if includeSpec {
				resource.RawResource = rawObj
			}

			resources = append(resources, resource)

			if kind != "" {
				kindsSet[kind] = struct{}{}
			}
			if cluster != "" {
				clustersSet[cluster] = struct{}{}
			}
			if resNamespace != "" {
				namespacesSet[resNamespace] = struct{}{}
			}
		}
	}

	c.JSON(http.StatusOK, models.ManagedResourceList{
		Resources:      resources,
		AvailableKinds: sortedSetKeys(kindsSet),
		Clusters:       sortedSetKeys(clustersSet),
		Namespaces:     sortedSetKeys(namespacesSet),
	})
}

// GetManagedResource retrieves a single resource by cluster, manifestwork name, and ordinal (always includes spec)
func GetManagedResource(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	cluster := c.Param("cluster")
	manifestwork := c.Param("manifestwork")
	ordinalStr := c.Param("ordinal")

	ordinal, err := strconv.Atoi(ordinalStr)
	if err != nil || ordinal < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ordinal"})
		return
	}

	if ocmClient == nil || ocmClient.WorkClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	mw, err := ocmClient.WorkClient.WorkV1().ManifestWorks(cluster).Get(ctx, manifestwork, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if ordinal >= len(mw.Spec.Workload.Manifests) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ordinal out of range"})
		return
	}

	var rawObj map[string]interface{}
	if err := json.Unmarshal(mw.Spec.Workload.Manifests[ordinal].Raw, &rawObj); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse manifest"})
		return
	}

	kind, _ := rawObj["kind"].(string)
	apiVersion, _ := rawObj["apiVersion"].(string)
	var resName, resNamespace string
	if metadata, ok := rawObj["metadata"].(map[string]interface{}); ok {
		resName, _ = metadata["name"].(string)
		resNamespace, _ = metadata["namespace"].(string)
	}

	// Find matching status entry
	var conditions []models.Condition
	var feedback *models.StatusFeedbackResult
	for _, ms := range mw.Status.ResourceStatus.Manifests {
		if int(ms.ResourceMeta.Ordinal) == ordinal {
			for _, cond := range ms.Conditions {
				conditions = append(conditions, models.Condition{
					Type:               string(cond.Type),
					Status:             string(cond.Status),
					LastTransitionTime: cond.LastTransitionTime.Format(time.RFC3339),
					Reason:             cond.Reason,
					Message:            cond.Message,
				})
			}
			if len(ms.StatusFeedbacks.Values) > 0 {
				feedback = convertStatusFeedback(ms.StatusFeedbacks)
			}
			break
		}
	}

	c.JSON(http.StatusOK, models.ManagedResource{
		ID:               fmt.Sprintf("%s/%s/%d", cluster, manifestwork, ordinal),
		Kind:             kind,
		APIVersion:       apiVersion,
		Name:             resName,
		Namespace:        resNamespace,
		Cluster:          cluster,
		ManifestWorkName: manifestwork,
		Ordinal:          ordinal,
		Status:           deriveResourceStatus(conditions),
		Conditions:       conditions,
		StatusFeedback:   feedback,
		RawResource:      rawObj,
	})
}
