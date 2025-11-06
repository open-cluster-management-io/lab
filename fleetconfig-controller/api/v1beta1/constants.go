package v1beta1

import "k8s.io/apimachinery/pkg/labels"

const (
	// HubCleanupPreflightFinalizer is the finalizer for cleanup preflight checks hub cluster's controller instance. Used to signal to the spoke's controller that unjoin can proceed.
	HubCleanupPreflightFinalizer = "fleetconfig.open-cluster-management.io/hub-cleanup-preflight"

	// HubCleanupFinalizer is the finalizer for cleanup by the hub cluster's controller instance.
	HubCleanupFinalizer = "fleetconfig.open-cluster-management.io/hub-cleanup"

	// SpokeCleanupFinalizer is the finalizer for cleanup by the spoke cluster's controller instance.
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

	// SpokeJoined means that the Spoke has successfully joined the Hub.
	SpokeJoined = "SpokeJoined"

	// PivotComplete means that the spoke cluster has successfully started managing itself.
	PivotComplete = "PivotComplete"

	// KlusterletSynced means that Klusterlet's OCM bundle version and values are up to date.
	KlusterletSynced = "KlusterletSynced"

	// HubUpgradeFailed means that the ClusterManager version upgrade failed.
	HubUpgradeFailed = "HubUpgradeFailed"
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

// Addon mode
const (
	// InstanceTypeManager indicates that the controller is running in a Hub cluster and only handles day 1 Spoke operations.
	InstanceTypeManager = "manager"

	// InstanceTypeAgent indicates that the controller is running in a Spoke cluster and only handles day 2 Spoke operations.
	InstanceTypeAgent = "agent"

	// InstanceTypeUnified indicates that the controller is running in a Hub cluster and handles the entire lifecycle of Spoke resources.
	InstanceTypeUnified = "unified"

	// HubKubeconfigEnvVar is the environment variable containing the path to the mounted Hub kubeconfig.
	HubKubeconfigEnvVar = "HUB_KUBECONFIG"

	// DefaultHubKubeconfigPath is the path of the mounted kubeconfig when the controller is running in a Spoke cluster. Used if the environment variable is not set.
	DefaultHubKubeconfigPath = "/managed/hub-kubeconfig/kubeconfig"

	// SpokeNameEnvVar is the environment variable containing the name of the Spoke resource.
	SpokeNameEnvVar = "CLUSTER_NAME"

	// SpokeNamespaceEnvVar is the environment variable containing the namespace of the Spoke resource.
	SpokeNamespaceEnvVar = "CLUSTER_NAMESPACE"

	// HubNamespaceEnvVar is the environment variable containing the namespace of the Hub resource.
	HubNamespaceEnvVar = "HUB_NAMESPACE"

	// ControllerNamespaceEnvVar is the environment variable containing the namespace that the controller is deployed to.
	ControllerNamespaceEnvVar = "CONTROLLER_NAMESPACE"

	// ClusterRoleNameEnvVar is the environment variable containing the name of the ClusterRole for fleetconfig-controller-manager.
	ClusterRoleNameEnvVar = "CLUSTER_ROLE_NAME"

	// PurgeAgentNamespaceEnvVar is the environment variable used to signal to the agent whether or not it should garbage collect it install namespace.
	PurgeAgentNamespaceEnvVar = "PURGE_AGENT_NAMESPACE"

	// FCCAddOnName is the name of the fleetconfig-controller addon.
	FCCAddOnName = "fleetconfig-controller-agent"

	// DefaultFCCManagerRole is the default name of the fleetconfig-controller-manager ClusterRole.
	DefaultFCCManagerRole = "fleetconfig-controller-manager-role"

	// NamespaceOCM is the open-cluster-management namespace.
	NamespaceOCM = "open-cluster-management"

	// NamespaceOCMAgent is the namespace for the open-cluster-management agent.
	NamespaceOCMAgent = "open-cluster-management-agent"

	// NamespaceOCMAgentAddOn is the namespace for open-cluster-management agent addons.
	NamespaceOCMAgentAddOn = "open-cluster-management-agent-addon"

	// AgentCleanupWatcherName is the name of the watcher for cleaning up the spoke agent.
	AgentCleanupWatcherName = "agent-cleanup-watcher"

	// ManagedClusterWorkloadCleanupTaint is applied to a ManagedCluster to remove non-addon workloads.
	// Addons can tolerate this taint to continue running during initial cleanup phase.
	ManagedClusterWorkloadCleanupTaint = "fleetconfig.open-cluster-management.io/workload-cleanup"

	// ManagedClusterTerminatingTaint is applied to remove all workloads including addons.
	// Nothing should tolerate this taint - it signals final cluster termination.
	ManagedClusterTerminatingTaint = "fleetconfig.open-cluster-management.io/terminating"
)

// SupportedInstanceTypes are the valid cluster types that the controller can be installed in.
var SupportedInstanceTypes = []string{
	InstanceTypeManager,
	InstanceTypeAgent,
	InstanceTypeUnified,
}

// OCMSpokeNamespaces are the namespaces created on an OCM managed cluster.
var OCMSpokeNamespaces = []string{
	NamespaceOCM,
	NamespaceOCMAgent,
	NamespaceOCMAgentAddOn,
}

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

const (
	// AddonArgoCD is the name of the built-in ArgoCD hub addon.
	AddonArgoCD = "argocd"

	// AddonGPF is the name of the built-in Governance Policy Framework hub addon.
	AddonGPF = "governance-policy-framework"
)

// SupportedHubAddons are the built-in hub addons which clusteradm and fleetconfig-controller support.
var SupportedHubAddons = []string{
	AddonArgoCD,
	AddonGPF,
}

const (
	// BundleVersionLatest is the latest OCM source version
	BundleVersionLatest = "latest"

	// BundleVersionDefault is the default OCM source version
	BundleVersionDefault = "default"
)

// Topology resource names
const (
	// NamespaceManagedClusterSetGlobal is the namespace for the global managed cluster set
	NamespaceManagedClusterSetGlobal = "managed-cluster-set-global"

	// NamespaceManagedClusterSetDefault is the namespace for the default managed cluster set
	NamespaceManagedClusterSetDefault = "managed-cluster-set-default"

	// NamespaceManagedClusterSetSpokes is the namespace for the spokes managed cluster set
	NamespaceManagedClusterSetSpokes = "managed-cluster-set-spokes"

	// ManagedClusterSetGlobal is the name of the global managed cluster set
	ManagedClusterSetGlobal = "global"

	// ManagedClusterSetDefault is the name of the default managed cluster set
	ManagedClusterSetDefault = "default"

	// ManagedClusterSetSpokes is the name of the spokes managed cluster set
	ManagedClusterSetSpokes = "spokes"

	// PlacementSpokes is the name of the spokes placement
	PlacementSpokes = "spokes"
)
