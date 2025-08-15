package v1alpha1

import "k8s.io/apimachinery/pkg/labels"

const (
	// FleetConfigFinalizer is the finalizer for FleetConfig cleanup.
	FleetConfigFinalizer = "fleetconfig.open-cluster-management.io/cleanup"
)

// FleetConfig condition types
const (
	// FleetConfigHubInitialized means that the Hub has been initialized.
	FleetConfigHubInitialized = "HubInitialized"

	// FleetConfigAddonsConfigured means that all addons have been configured on the Hub.
	FleetConfigAddonsConfigured = "AddonsConfigured"

	// FleetConfigCleanupFailed means that a failure occurred during cleanup.
	FleetConfigCleanupFailed = "CleanupFailed"

	// FleetConfigAddonsEnabled means that all addons have been enabled for a particular Spoke.
	FleetConfigAddonsEnabled = "AddonsEnabled"
)

// FleetConfig condition reasons
const (
	ReconcileSuccess = "ReconcileSuccess"
)

// FleetConfig phases
const (
	// FleetConfigStarting means that the Hub and Spoke(s) are being initialized / joined.
	FleetConfigStarting = "Initializing"

	// FleetConfigRunning means that the Hub is initialized and all Spoke(s) have joined successfully.
	FleetConfigRunning = "Running"

	// FleetConfigUnhealthy means that a failure occurred during Hub initialization and/or Spoke join attempt.
	FleetConfigUnhealthy = "Unhealthy"

	// FleetConfigDeleting means that the FleetConfig is being deleted.
	FleetConfigDeleting = "Deleting"
)

// ManagedClusterType is the type of a managed cluster.
type ManagedClusterType string

const (
	// ManagedClusterTypeHub is the type of managed cluster that is a hub.
	ManagedClusterTypeHub = "hub"

	// ManagedClusterTypeSpoke is the type of managed cluster that is a spoke.
	ManagedClusterTypeSpoke = "spoke"

	// ManagedClusterTypeHubAsSpoke is the type of managed cluster that is both a hub and a spoke.
	ManagedClusterTypeHubAsSpoke = "hub-as-spoke"
)

// FleetConfig labels
const (
	// LabelManagedClusterType is the label key for the managed cluster type.
	LabelManagedClusterType = "fleetconfig.open-cluster-management.io/managedClusterType"

	// LabelAddOnManagedBy is the label key for the lifecycle manager of an add-on resource.
	LabelAddOnManagedBy = "addon.open-cluster-management.io/managedBy"
)

// Registration driver types
const (
	// CSRRegistrationDriver is the default CSR-based registration driver.
	CSRRegistrationDriver = "csr"

	// AWSIRSARegistrationDriver is the AWS IAM Role for Service Accounts (IRSA) registration driver.
	AWSIRSARegistrationDriver = "awsirsa"
)

// Addon ConfigMap constants
const (
	// AddonConfigMapNamePrefix is the common name prefix for all configmaps containing addon configurations.
	AddonConfigMapNamePrefix = "fleet-addon"

	// AddonConfigMapManifestRawKey is the data key containing raw manifests.
	AddonConfigMapManifestRawKey = "manifestsRaw"

	// AddonConfigMapManifestRawKey is the data key containing a URL to download manifests.
	AddonConfigMapManifestURLKey = "manifestsURL"
)

// AllowedAddonURLSchemes are the URL schemes which can be used to provide manifests for configuring addons.
var AllowedAddonURLSchemes = []string{"http", "https"}

var (
	// ManagedByLabels are labeles applies to resources to denote that fleetconfig-controller is managing the lifecycle.
	ManagedByLabels = map[string]string{
		LabelAddOnManagedBy: "fleetconfig-controller",
	}
	// ManagedBySelector is a label selector for filtering add-on resources managed fleetconfig-controller.
	ManagedBySelector = labels.SelectorFromSet(labels.Set(ManagedByLabels))
)
