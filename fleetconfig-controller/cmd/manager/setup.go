package manager

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// CreateTopologyResourcesAtStartup creates the default topology resources at controller startup
func createTopologyResources(ctx context.Context, cli client.Client, log logr.Logger) error {
	logger := log.WithName("topology-resources")
	logger.Info("creating default topology resources")

	// Create managed-cluster-set-global namespace
	globalNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: v1beta1.NamespaceManagedClusterSetGlobal}}
	result, err := controllerutil.CreateOrUpdate(ctx, cli, globalNs, func() error {
		return nil // namespace has no spec to mutate
	})
	if err != nil {
		return fmt.Errorf("failed to create or update %s namespace: %v", v1beta1.NamespaceManagedClusterSetGlobal, err)
	}
	logger.V(1).Info("global namespace", "result", result)

	// Create managed-cluster-set-global binding
	globalBinding := &clusterv1beta2.ManagedClusterSetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.ManagedClusterSetGlobal,
			Namespace: v1beta1.NamespaceManagedClusterSetGlobal,
		},
	}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, globalBinding, func() error {
		globalBinding.Spec.ClusterSet = v1beta1.ManagedClusterSetGlobal
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update global ManagedClusterSetBinding: %v", err)
	}
	logger.V(1).Info("global ManagedClusterSetBinding", "result", result)

	// Create managed-cluster-set-default namespace
	defaultNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: v1beta1.NamespaceManagedClusterSetDefault}}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, defaultNs, func() error {
		return nil // namespace has no spec to mutate
	})
	if err != nil {
		return fmt.Errorf("failed to create or update %s namespace: %v", v1beta1.NamespaceManagedClusterSetDefault, err)
	}
	logger.V(1).Info("default namespace", "result", result)

	// Create managed-cluster-set-default binding
	defaultBinding := &clusterv1beta2.ManagedClusterSetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.ManagedClusterSetDefault,
			Namespace: v1beta1.NamespaceManagedClusterSetDefault,
		},
	}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, defaultBinding, func() error {
		defaultBinding.Spec.ClusterSet = v1beta1.ManagedClusterSetDefault
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update default ManagedClusterSetBinding: %v", err)
	}
	logger.V(1).Info("default ManagedClusterSetBinding", "result", result)

	// Create managed-cluster-set-spokes namespace
	spokesNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: v1beta1.NamespaceManagedClusterSetSpokes}}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, spokesNs, func() error {
		return nil // namespace has no spec to mutate
	})
	if err != nil {
		return fmt.Errorf("failed to create or update %s namespace: %v", v1beta1.NamespaceManagedClusterSetSpokes, err)
	}
	logger.V(1).Info("spokes namespace", "result", result)

	// Create spokes ManagedClusterSet
	spokesClusterSet := &clusterv1beta2.ManagedClusterSet{ObjectMeta: metav1.ObjectMeta{Name: v1beta1.ManagedClusterSetSpokes}}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, spokesClusterSet, func() error {
		spokesClusterSet.Spec.ClusterSelector = clusterv1beta2.ManagedClusterSelector{
			SelectorType: clusterv1beta2.LabelSelector,
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      v1beta1.LabelManagedClusterType,
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{v1beta1.ManagedClusterTypeHub, v1beta1.ManagedClusterTypeHubAsSpoke},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update spokes ManagedClusterSet: %v", err)
	}
	logger.V(1).Info("spokes ManagedClusterSet", "result", result)

	// Create spokes ManagedClusterSetBinding
	spokesBinding := &clusterv1beta2.ManagedClusterSetBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.ManagedClusterSetSpokes,
			Namespace: v1beta1.NamespaceManagedClusterSetSpokes,
		},
	}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, spokesBinding, func() error {
		spokesBinding.Spec.ClusterSet = v1beta1.ManagedClusterSetSpokes
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update spokes ManagedClusterSetBinding: %v", err)
	}
	logger.V(1).Info("spokes ManagedClusterSetBinding", "result", result)

	// Create spokes Placement
	spokesPlacement := &clusterv1beta1.Placement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.PlacementSpokes,
			Namespace: v1beta1.NamespaceManagedClusterSetSpokes,
		},
	}
	result, err = controllerutil.CreateOrUpdate(ctx, cli, spokesPlacement, func() error {
		spokesPlacement.Spec = clusterv1beta1.PlacementSpec{}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update spokes Placement: %v", err)
	}
	logger.V(1).Info("spokes Placement", "result", result)

	logger.Info("topology resources created successfully at startup")
	return nil
}
