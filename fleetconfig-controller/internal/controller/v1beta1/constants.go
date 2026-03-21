package v1beta1

import (
	"time"
)

// generic
const (
	clusteradm = "clusteradm"

	hubRequeuePreInit    = 30 * time.Second
	hubRequeuePostInit   = 2 * time.Minute
	requeueDeleting      = 5 * time.Second
	spokeRequeuePreJoin  = 15 * time.Second
	spokeRequeuePostJoin = 1 * time.Minute
	spokeWatchInterval   = 30 * time.Second
)

// addon
const (
	install   = "install"
	uninstall = "uninstall"
	hubAddon  = "hub-addon"

	managedClusterAddOn           = "ManagedClusterAddOn"
	AddOnDeploymentConfigResource = "addondeploymentconfigs"

	addonCleanupTimeout      = 1 * time.Minute
	addonCleanupPollInterval = 2 * time.Second

	manifestWorkAddOnLabelKey = "open-cluster-management.io/addon-name"
)
