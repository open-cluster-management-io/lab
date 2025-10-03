package v1beta1

import (
	"regexp"
	"time"
)

// generic
const (
	clusteradm     = "clusteradm"
	requeue        = 30 * time.Second
	amwExistsError = "you should manually clean them, uninstall kluster will cause those works out of control."
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
