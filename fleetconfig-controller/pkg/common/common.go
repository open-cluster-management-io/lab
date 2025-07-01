// Package common contains reusable helped functions
package common

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterapi "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorapi "open-cluster-management.io/api/client/operator/clientset/versioned"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
)

// ClusterClient creates an OCM cluster v1 client.
func ClusterClient(kubeconfig []byte) (*clusterapi.Clientset, error) {
	rc, err := kube.RestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	clusterC, err := clusterapi.NewForConfig(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to create ocm cluster client: %w", err)
	}
	return clusterC, nil
}

// OperatorClient creates an OCM operator v1 client.
func OperatorClient(kubeconfig []byte) (*operatorapi.Clientset, error) {
	rc, err := kube.RestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	operatorC, err := operatorapi.NewForConfig(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to create ocm operator client: %w", err)
	}
	return operatorC, nil
}

// WorkClient creates an OCM work v1 client.
func WorkClient(kubeconfig []byte) (*workapi.Clientset, error) {
	rc, err := kube.RestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	workC, err := workapi.NewForConfig(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to create ocm work client: %w", err)
	}
	return workC, nil
}

// GetManagedCluster retrieves a ManagedCluster resource from the Hub cluster for a particular Spoke cluster.
func GetManagedCluster(ctx context.Context, client *clusterapi.Clientset, name string) (*clusterv1.ManagedCluster, error) {
	managedCluster, err := client.ClusterV1().ManagedClusters().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected error getting ManagedCluster %s: %w", name, err)
	}
	return managedCluster, nil
}

// UpdateManagedCluster updates the ManagedCluster resource for a particular Spoke cluster.
func UpdateManagedCluster(ctx context.Context, client *clusterapi.Clientset, managedCluster *clusterv1.ManagedCluster) error {
	if _, err := client.ClusterV1().ManagedClusters().Update(ctx, managedCluster, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update ManagedCluster %s: %w", managedCluster.Name, err)
	}
	return nil
}

// PrepareKubeconfig parses a kubeconfig spec and returns updated clusteradm args.
// The '--kubeconfig' flag is added and a cleanup function is returned to remove the temp kubeconfig file.
func PrepareKubeconfig(ctx context.Context, kClient client.Client, kubeconfig *v1alpha1.Kubeconfig, args []string) ([]string, func(), error) {
	logger := log.FromContext(ctx)

	raw, err := kube.KubeconfigFromSecretOrCluster(ctx, kClient, kubeconfig)
	if err != nil {
		return args, nil, err
	}
	kubeconfigPath, cleanup, err := file.TmpFile(raw, "kubeconfig")
	if err != nil {
		return args, cleanup, err
	}
	if kubeconfig.Context != "" {
		args = append(args, "--context", kubeconfig.Context)
	}

	logger.V(1).Info("Using kubeconfig", "path", kubeconfigPath)
	args = append(args, "--kubeconfig", kubeconfigPath)
	return args, cleanup, nil
}

// PrepareResources returns resource-related flags
func PrepareResources(resources v1alpha1.ResourceSpec) []string {
	flags := []string{
		"--resource-qos-class", resources.QosClass,
	}
	requests := resources.Requests.String()
	if requests != "" {
		flags = append(flags, "--resource-requests", requests)
	}
	limits := resources.Limits.String()
	if limits != "" {
		flags = append(flags, "--resource-limits", limits)
	}
	return flags
}
