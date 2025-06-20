package client

import (
	"context"

	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	workv1 "open-cluster-management.io/api/work/v1"
)

type MockOCMClient struct {
	mock.Mock
	Interface dynamic.Interface
}

type MockClusterV1Interface struct {
	mock.Mock
}

type MockManagedClustersInterface struct {
	mock.Mock
}

type MockManagedClusterSetsInterface struct {
	mock.Mock
}

type MockPlacementsInterface struct {
	mock.Mock
}

type MockPlacementDecisionsInterface struct {
	mock.Mock
}

type MockAddonV1Alpha1Interface struct {
	mock.Mock
}

type MockManagedClusterAddOnsInterface struct {
	mock.Mock
}

type MockWorkV1Interface struct {
	mock.Mock
}

type MockManifestWorksInterface struct {
	mock.Mock
}

func (m *MockClusterV1Interface) ManagedClusters() *MockManagedClustersInterface {
	args := m.Called()
	return args.Get(0).(*MockManagedClustersInterface)
}

func (m *MockManagedClustersInterface) List(ctx context.Context, opts metav1.ListOptions) (*clusterv1.ManagedClusterList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*clusterv1.ManagedClusterList), args.Error(1)
}

func (m *MockManagedClustersInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*clusterv1.ManagedCluster, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*clusterv1.ManagedCluster), args.Error(1)
}

func (m *MockManagedClusterSetsInterface) List(ctx context.Context, opts metav1.ListOptions) (*clusterv1beta2.ManagedClusterSetList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*clusterv1beta2.ManagedClusterSetList), args.Error(1)
}

func (m *MockManagedClusterSetsInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*clusterv1beta2.ManagedClusterSet, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*clusterv1beta2.ManagedClusterSet), args.Error(1)
}

func (m *MockPlacementsInterface) List(ctx context.Context, opts metav1.ListOptions) (*clusterv1beta1.PlacementList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*clusterv1beta1.PlacementList), args.Error(1)
}

func (m *MockPlacementsInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*clusterv1beta1.Placement, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*clusterv1beta1.Placement), args.Error(1)
}

func (m *MockPlacementDecisionsInterface) List(ctx context.Context, opts metav1.ListOptions) (*clusterv1beta1.PlacementDecisionList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*clusterv1beta1.PlacementDecisionList), args.Error(1)
}

func (m *MockAddonV1Alpha1Interface) ManagedClusterAddOns(namespace string) *MockManagedClusterAddOnsInterface {
	args := m.Called(namespace)
	return args.Get(0).(*MockManagedClusterAddOnsInterface)
}

func (m *MockManagedClusterAddOnsInterface) List(ctx context.Context, opts metav1.ListOptions) (*addonv1alpha1.ManagedClusterAddOnList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*addonv1alpha1.ManagedClusterAddOnList), args.Error(1)
}

func (m *MockManagedClusterAddOnsInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*addonv1alpha1.ManagedClusterAddOn, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*addonv1alpha1.ManagedClusterAddOn), args.Error(1)
}

func (m *MockWorkV1Interface) ManifestWorks(namespace string) *MockManifestWorksInterface {
	args := m.Called(namespace)
	return args.Get(0).(*MockManifestWorksInterface)
}

func (m *MockManifestWorksInterface) List(ctx context.Context, opts metav1.ListOptions) (*workv1.ManifestWorkList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*workv1.ManifestWorkList), args.Error(1)
}

func (m *MockManifestWorksInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*workv1.ManifestWork, error) {
	args := m.Called(ctx, name, opts)
	return args.Get(0).(*workv1.ManifestWork), args.Error(1)
}
