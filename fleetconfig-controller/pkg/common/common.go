// Package common contains reusable helper functions for hub/spoke clients.
//
//nolint:revive // name matches import path pkg/common; "common" is established in this module
package common

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	clusterapi "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorapi "open-cluster-management.io/api/client/operator/clientset/versioned"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"

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

// AddOnClient creates an OCM addon v1 client.
func AddOnClient(kubeconfig []byte) (*addonapi.Clientset, error) {
	rc, err := kube.RestConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	addonC, err := addonapi.NewForConfig(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to create ocm addon client: %w", err)
	}
	return addonC, nil
}

// GetManagedCluster retrieves a Spoke's ManagedCluster. With ownerLabels, it prefers a label match
// and falls back to each name in fallbackNames in order, returning the first hit whose existing
// labels do not conflict with ownerLabels. Returns (nil, nil) if no ManagedCluster is found.
func GetManagedCluster(ctx context.Context, client clusterapi.Interface, fallbackNames []string, ownerLabels map[string]string) (*clusterv1.ManagedCluster, error) {
	if len(ownerLabels) > 0 {
		parts := make([]string, 0, len(ownerLabels))
		for k, v := range ownerLabels {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		list, err := client.ClusterV1().ManagedClusters().List(ctx, metav1.ListOptions{LabelSelector: strings.Join(parts, ",")})
		if err != nil {
			return nil, fmt.Errorf("list ManagedClusters by owner labels: %w", err)
		}
		if len(list.Items) > 1 {
			return nil, fmt.Errorf("expected at most one ManagedCluster matching owner labels %v, got %d", ownerLabels, len(list.Items))
		}
		if len(list.Items) == 1 {
			return &list.Items[0], nil
		}
		// No labeled match, fall through to name-based lookup for legacy / mid-join adoption.
	}

	for _, name := range fallbackNames {
		if name == "" {
			continue
		}
		mc, err := getManagedClusterLegacy(ctx, client, name)
		if err != nil {
			return nil, err
		}
		if mc == nil {
			continue
		}
		// Collision guard: skip a ManagedCluster owned by a different Spoke.
		if conflictsWithOwnerLabels(mc.Labels, ownerLabels) {
			continue
		}
		return mc, nil
	}
	return nil, nil
}

// conflictsWithOwnerLabels reports whether the existing labels carry any of the ownerLabels keys
// with a non-matching value, indicating the ManagedCluster is claimed by a different owner.
func conflictsWithOwnerLabels(existing, ownerLabels map[string]string) bool {
	for k, want := range ownerLabels {
		if got, has := existing[k]; has && got != want {
			return true
		}
	}
	return false
}

// getManagedClusterLegacy fetches a ManagedCluster by its cluster-scoped name. Returns (nil, nil) when the ManagedCluster does not exist.
func getManagedClusterLegacy(ctx context.Context, client clusterapi.Interface, name string) (*clusterv1.ManagedCluster, error) {
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
