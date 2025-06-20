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
)

// GetClusterAddons handles retrieving all addons for a specific cluster
func GetClusterAddons(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	clusterName := c.Param("name")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.AddonClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// List real managed cluster addons for the specific namespace (cluster name)
	list, err := ocmClient.AddonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to our simplified ManagedClusterAddon format
	addons := make([]models.ManagedClusterAddon, 0, len(list.Items))
	for _, item := range list.Items {
		// Extract the basic metadata
		addon := models.ManagedClusterAddon{
			ID:                string(item.GetUID()),
			Name:              item.GetName(),
			Namespace:         item.GetNamespace(),
			CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
		}

		// Extract installNamespace from spec
		if item.Spec.InstallNamespace != "" {
			addon.InstallNamespace = item.Spec.InstallNamespace
		}

		// Extract conditions from status
		for _, condition := range item.Status.Conditions {
			addon.Conditions = append(addon.Conditions, models.Condition{
				Type:               string(condition.Type),
				Status:             string(condition.Status),
				Reason:             condition.Reason,
				Message:            condition.Message,
				LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
			})
		}

		// Extract registrations from status
		for _, registration := range item.Status.Registrations {
			reg := models.AddonRegistration{
				SignerName: registration.SignerName,
			}

			if registration.Subject.User != "" {
				reg.Subject.User = registration.Subject.User
			}

			if len(registration.Subject.Groups) > 0 {
				reg.Subject.Groups = registration.Subject.Groups
			}

			addon.Registrations = append(addon.Registrations, reg)
		}

		// Extract supportedConfigs from status
		for _, config := range item.Status.SupportedConfigs {
			addon.SupportedConfigs = append(addon.SupportedConfigs, models.AddonSupportedConfig{
				Group:    config.Group,
				Resource: config.Resource,
			})
		}

		addons = append(addons, addon)
	}

	c.JSON(http.StatusOK, addons)
}

// GetClusterAddon handles retrieving a specific addon for a specific cluster
func GetClusterAddon(c *gin.Context, ocmClient *client.OCMClient, ctx context.Context) {
	clusterName := c.Param("name")
	addonName := c.Param("addonName")

	// Ensure we have a client before proceeding
	if ocmClient == nil || ocmClient.AddonClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OCM client not initialized"})
		return
	}

	// Get the real managed cluster addon
	item, err := ocmClient.AddonClient.AddonV1alpha1().ManagedClusterAddOns(clusterName).Get(ctx, addonName, metav1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Addon %s not found for cluster %s", addonName, clusterName)})
		return
	}

	// Extract the basic metadata
	addon := models.ManagedClusterAddon{
		ID:                string(item.GetUID()),
		Name:              item.GetName(),
		Namespace:         item.GetNamespace(),
		CreationTimestamp: item.GetCreationTimestamp().Format(time.RFC3339),
	}

	// Extract installNamespace from spec
	if item.Spec.InstallNamespace != "" {
		addon.InstallNamespace = item.Spec.InstallNamespace
	}

	// Extract conditions from status
	for _, condition := range item.Status.Conditions {
		addon.Conditions = append(addon.Conditions, models.Condition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: condition.LastTransitionTime.Format(time.RFC3339),
		})
	}

	// Extract registrations from status
	for _, registration := range item.Status.Registrations {
		reg := models.AddonRegistration{
			SignerName: registration.SignerName,
		}

		if registration.Subject.User != "" {
			reg.Subject.User = registration.Subject.User
		}

		if len(registration.Subject.Groups) > 0 {
			reg.Subject.Groups = registration.Subject.Groups
		}

		addon.Registrations = append(addon.Registrations, reg)
	}

	// Extract supportedConfigs from status
	for _, config := range item.Status.SupportedConfigs {
		addon.SupportedConfigs = append(addon.SupportedConfigs, models.AddonSupportedConfig{
			Group:    config.Group,
			Resource: config.Resource,
		})
	}

	c.JSON(http.StatusOK, addon)
}
