package client

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resources to work with - ManagedCluster and ManagedClusterSet from OCM
var ManagedClusterResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1",
	Resource: "managedclusters",
}

var ManagedClusterSetResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta2",
	Resource: "managedclustersets",
}

// ManagedClusterAddon resource
var ManagedClusterAddonResource = schema.GroupVersionResource{
	Group:    "addon.open-cluster-management.io",
	Version:  "v1alpha1",
	Resource: "managedclusteraddons",
}

// Placement resource
var PlacementResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "placements",
}

// PlacementDecision resource
var PlacementDecisionResource = schema.GroupVersionResource{
	Group:    "cluster.open-cluster-management.io",
	Version:  "v1beta1",
	Resource: "placementdecisions",
}
