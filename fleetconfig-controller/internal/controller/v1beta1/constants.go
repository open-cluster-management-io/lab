package v1beta1

import (
	"regexp"
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

var csrSuffixPattern = regexp.MustCompile(`-[a-zA-Z0-9]{5}$`)

// addon
const (
	// commands
	addon   = "addon"
	create  = "create"
	enable  = "enable"
	disable = "disable"

	install   = "install"
	uninstall = "uninstall"
	hubAddon  = "hub-addon"

	managedClusterAddOn           = "ManagedClusterAddOn"
	AddOnDeploymentConfigResource = "addondeploymentconfigs"

	addonCleanupTimeout      = 1 * time.Minute
	addonCleanupPollInterval = 2 * time.Second

	manifestWorkAddOnLabelKey = "open-cluster-management.io/addon-name"
)
