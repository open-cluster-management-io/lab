// Package args provides helpers for formatting clusteradm args.
package args

import (
	"context"
	"reflect"
	"slices"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
)

// Redacted is used to mask sensitive values
const Redacted = "REDACTED"

var sensitiveKeys = []string{
	"join-token",
	"jointoken",
	"hub-token",
	"hubtoken",
}

// PrepareKubeconfig parses a kubeconfig spec and returns updated clusteradm args.
// The '--kubeconfig' flag is added and a cleanup function is returned to remove the temp kubeconfig file.
func PrepareKubeconfig(ctx context.Context, rawKubeconfig []byte, context string, args []string) ([]string, func(), error) {
	logger := log.FromContext(ctx)

	kubeconfigPath, cleanup, err := file.TmpFile(rawKubeconfig, "kubeconfig")
	if err != nil {
		return args, cleanup, err
	}
	if context != "" {
		args = append(args, "--context", context)
	}

	logger.V(1).Info("Using kubeconfig", "path", kubeconfigPath)
	args = append(args, "--kubeconfig", kubeconfigPath)
	return args, cleanup, nil
}

// PrepareResources returns resource-related flags
func PrepareResources(resources ResourceSpec) []string {
	qos := resources.GetQosClass()
	if qos == "" {
		qos = "Default"
	}
	flags := []string{"--resource-qos-class", qos}
	if req := resources.GetRequests(); req != nil {
		rv := reflect.ValueOf(req)
		if rv.Kind() != reflect.Ptr || !rv.IsNil() {
			if s := req.String(); s != "" {
				flags = append(flags, "--resource-requests", s)
			}
		}
	}
	if lim := resources.GetLimits(); lim != nil {
		rv := reflect.ValueOf(lim)
		if rv.Kind() != reflect.Ptr || !rv.IsNil() {
			if s := lim.String(); s != "" {
				flags = append(flags, "--resource-limits", s)
			}
		}
	}
	return flags
}

// ResourceSpec is an interface implemented by any API version's ResourceSpec
type ResourceSpec interface {
	GetRequests() ResourceValues
	GetLimits() ResourceValues
	GetQosClass() string
}

// ResourceValues is an interface implemented by any API version's ResourceValues
type ResourceValues interface {
	String() string
}

// SanitizeArgs redacts sensitive args to prevent leaking credentials
func SanitizeArgs(args []string) []string {
	return sanitizeSlice(args)
}

// SanitizeOutput redacts sensitive values from command output
func SanitizeOutput(output []byte) []byte {
	if len(output) == 0 {
		return output
	}

	// Convert bytes to string and split into words
	text := string(output)
	words := strings.Fields(text)

	// Sanitize the words
	sanitized := sanitizeSlice(words)

	// Join back and convert to bytes
	return []byte(strings.Join(sanitized, " "))
}

func sanitizeSlice(words []string) []string {
	if len(words) == 0 {
		return []string{}
	}

	// Start with a copy of all args
	out := make([]string, len(words))
	copy(out, words)

	// Iterate through and redact values following sensitive keys
	for i := 0; i < len(words); i++ {
		if isSensitiveKey(words[i]) {
			// Check if there's a next element to redact
			if i+1 < len(words) {
				out[i+1] = Redacted
				i++ // Skip the next element since we just redacted it
			}
		}
	}
	return out
}

func isSensitiveKey(key string) bool {
	norm := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(key)), "--")
	return slices.Contains(sensitiveKeys, norm)
}
