package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// writeK8sError maps Kubernetes API errors to appropriate HTTP status codes
// and avoids leaking raw error strings to clients.
func writeK8sError(c *gin.Context, err error, notFoundMsg, defaultMsg string) {
	switch {
	case apierrors.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": notFoundMsg})
	case apierrors.IsForbidden(err):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": defaultMsg})
	}
}
