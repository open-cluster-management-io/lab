// Package args provides helpers for formatting clusteradm args.
package args

import (
	"context"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
)

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
