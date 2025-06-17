package v1alpha1

const (
	// FleetConfigFinalizer is the finalizer for FleetConfig cleanup.
	FleetConfigFinalizer = "fleetconfig.open-cluster-management.io/cleanup"
)

// FleetConfig condition types
const (
	// FleetConfigHubInitialized means that the Hub has been initialized.
	FleetConfigHubInitialized = "HubInitialized"

	// FleetConfigCleanupFailed means that a failure occurred during cleanup.
	FleetConfigCleanupFailed = "CleanupFailed"
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
)

// Registration driver types
const (
	// CSRRegistrationDriver is the default CSR-based registration driver.
	CSRRegistrationDriver = "csr"

	// AWSIRSARegistrationDriver is the AWS IAM Role for Service Accounts (IRSA) registration driver.
	AWSIRSARegistrationDriver = "awsirsa"
)
