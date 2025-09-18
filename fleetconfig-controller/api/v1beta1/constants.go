package v1beta1

import "k8s.io/apimachinery/pkg/labels"

const (
	// HubCleanupFinalizer is the finalizer for Hub cleanup.
	HubCleanupFinalizer = "fleetconfig.open-cluster-management.io/hub-cleanup"

	// SpokeCleanupFinalizer is the finalizer for Spoke cleanup.
	SpokeCleanupFinalizer = "fleetconfig.open-cluster-management.io/spoke-cleanup"
)

// Hub and Spoke condition types
const (
	// HubInitialized means that the Hub has been initialized.
	HubInitialized = "HubInitialized"

	// AddonsConfigured means that all addons have been configured on the Hub, or enabled/disabled on a Spoke.
	AddonsConfigured = "AddonsConfigured"

	// CleanupFailed means that a failure occurred during cleanup.
	CleanupFailed = "CleanupFailed"

	// SpokeJoined means that the spoke has successfully joined the Hub.
	SpokeJoined = "SpokeJoined"
)

// Hub and Spoke condition reasons
const (
	ReconcileSuccess = "ReconcileSuccess"
)

// Hub and Spoke phases
const (
	// HubStarting means that the Hub is being initialized.
	HubStarting = "Initializing"

	// HubRunning means that the Hub is initialized successfully.
	HubRunning = "Running"

	// SpokeJoining means that the Spoke is being joined to the Hub.
	SpokeJoining = "Joining"

	// SpokeRunning means that the Spoke has successfully joined the Hub.
	SpokeRunning = "Running"

	// Unhealthy means that a failure occurred during Hub initialization and/or Spoke join attempt.
	Unhealthy = "Unhealthy"

	// Deleting means that the Hub or Spoke is being deleted.
	Deleting = "Deleting"
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

// Reconcile parameters
const (
	// SpokeDefaultMaxConcurrentReconciles is the default maximum number of Spoke resources that may be reconciled in parallel.
	SpokeDefaultMaxConcurrentReconciles = 5
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
