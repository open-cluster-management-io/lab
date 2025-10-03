package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	arg_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
)

// getManagedClusterAddOns returns the list of ManagedClusterAddOns currently installed on a spoke cluster
func getManagedClusterAddOns(ctx context.Context, addonC *addonapi.Clientset, spokeName string) ([]string, error) {
	managedClusterAddOns, err := addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).List(ctx, metav1.ListOptions{
		LabelSelector: v1beta1.ManagedBySelector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list ManagedClusterAddOns for spoke %s: %w", spokeName, err)
	}

	addons := make([]string, len(managedClusterAddOns.Items))
	for i, addon := range managedClusterAddOns.Items {
		addons[i] = addon.Name
	}
	return addons, nil
}

// getHubAddOns returns the list of hub addons (ClusterManagementAddOns without managed-by label)
func getHubAddOns(ctx context.Context, addonC *addonapi.Clientset) ([]string, error) {
	// Hub addons are ClusterManagementAddOns that don't have the managed-by label
	allClusterManagementAddOns, err := addonC.AddonV1alpha1().ClusterManagementAddOns().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list all ClusterManagementAddOns: %w", err)
	}

	var hubAddons []string
	for _, addon := range allClusterManagementAddOns.Items {
		if slices.Contains(v1beta1.SupportedHubAddons, addon.Name) && addon.Labels[v1beta1.LabelAddOnManagedBy] == "" {
			hubAddons = append(hubAddons, addon.Name)
		}
	}
	return hubAddons, nil
}

func handleAddonConfig(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, hub *v1beta1.Hub) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleAddOnConfig")

	requestedAddOns := hub.Spec.AddOnConfigs

	// get existing addon templates from cluster
	createdAddOns, err := addonC.AddonV1alpha1().AddOnTemplates().List(ctx, metav1.ListOptions{LabelSelector: v1beta1.ManagedBySelector.String()})
	if err != nil {
		logger.V(1).Info("failed to list AddOnTemplates, ensure CRDs are installed.", "error", err)
		return len(requestedAddOns) > 0, err
	}

	// nothing to do
	if len(requestedAddOns) == 0 && len(createdAddOns.Items) == 0 {
		logger.V(5).Info("no addons to reconcile")
		return false, nil
	}

	// compare existing to requested
	createdVersionedNames := make([]string, len(createdAddOns.Items))
	for i, ca := range createdAddOns.Items {
		createdVersionedNames[i] = ca.Name
	}

	requestedVersionedNames := make([]string, len(requestedAddOns))
	for i, ra := range requestedAddOns {
		requestedVersionedNames[i] = fmt.Sprintf("%s-%s", ra.Name, ra.Version)
	}

	// Find addons that need to be created (present in requested, missing from created)
	addonsToCreate := make([]v1beta1.AddOnConfig, 0)
	for i, requestedName := range requestedVersionedNames {
		if !slices.Contains(createdVersionedNames, requestedName) {
			addonsToCreate = append(addonsToCreate, requestedAddOns[i])
		}
	}

	// Find addons that need to be deleted (present in created, missing from requested)
	addonsToDelete := make([]string, 0)
	for _, createdName := range createdVersionedNames {
		if !slices.Contains(requestedVersionedNames, createdName) {
			addonsToDelete = append(addonsToDelete, createdName)
		}
	}

	// do deletes first, then creates.
	err = handleAddonDelete(ctx, addonC, addonsToDelete)
	if err != nil {
		return true, err
	}

	err = handleAddonCreate(ctx, kClient, hub, addonsToCreate)
	if err != nil {
		return true, err
	}

	return true, nil
}

func handleAddonCreate(ctx context.Context, kClient client.Client, hub *v1beta1.Hub, addons []v1beta1.AddOnConfig) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("createAddOns")

	// set up array of clusteradm addon create commands
	for _, a := range addons {
		// look up manifests CM for the addon
		cm := corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-%s-%s", v1beta1.AddonConfigMapNamePrefix, a.Name, a.Version)
		err := kClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: hub.Namespace}, &cm)
		if err != nil {
			return errors.Wrapf(err, "could not load configuration for add-on %s version %s", a.Name, a.Version)
		}

		args := append([]string{
			addon,
			create,
			a.Name,
			fmt.Sprintf("--version=%s", a.Version),
			fmt.Sprintf("--labels=%v", v1beta1.ManagedBySelector.String()),
		}, hub.BaseArgs()...)

		// Extract manifest configuration from ConfigMap
		// validation was already done by the webhook, so simply check if raw manifests are provided and if not, use the URL.
		manifestsRaw, ok := cm.Data[v1beta1.AddonConfigMapManifestRawKey]
		if ok {
			// Write raw manifests to temporary file
			filename, cleanup, err := file.TmpFile([]byte(manifestsRaw), "yaml")
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				return err
			}
			args = append(args, fmt.Sprintf("--filename=%s", filename))
		} else {
			manifestsURL := cm.Data[v1beta1.AddonConfigMapManifestURLKey]
			url, err := url.Parse(manifestsURL)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to create addon %s version %s", a.Name, a.Version))
			}
			switch url.Scheme {
			case "http", "https":
				// pass URL directly
				args = append(args, fmt.Sprintf("--filename=%s", manifestsURL))
			default:
				return fmt.Errorf("unsupported URL scheme %s for addon %s version %s. Must be one of %v", url.Scheme, a.Name, a.Version, v1beta1.AllowedAddonURLSchemes)
			}
		}

		if a.HubRegistration {
			args = append(args, "--hub-registration")
		}
		if a.Overwrite {
			args = append(args, "--overwrite")
		}
		if a.ClusterRoleBinding != "" {
			args = append(args, fmt.Sprintf("--cluster-role-bind=%s", a.ClusterRoleBinding))
		}

		logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(args))
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon create' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			return fmt.Errorf("failed to create addon: %v, output: %s", err, string(out))
		}
		logger.V(0).Info("created addon", "AddOnTemplate", a.Name, "output", string(stdout))
	}
	return nil
}

func handleAddonDelete(ctx context.Context, addonC *addonapi.Clientset, addons []string) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("deleteAddOns")

	// a list of addons which may or may not need to be purged at the end (ClusterManagementAddOns needs to be deleted)
	purgeList := make([]string, 0)
	errs := make([]error, 0)
	for _, addonName := range addons {
		// get the addon template, so we can extract spec.addonName
		addon, err := addonC.AddonV1alpha1().AddOnTemplates().Get(ctx, addonName, metav1.GetOptions{})
		if err != nil && !kerrs.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to delete addon %s: %v", addonName, err))
			continue
		}

		// delete the addon template
		if addon == nil {
			logger.V(0).Info("addon not found, nothing to do", "AddOnTemplate", addonName)
			continue
		}

		err = addonC.AddonV1alpha1().AddOnTemplates().Delete(ctx, addonName, metav1.DeleteOptions{})
		if err != nil && !kerrs.IsNotFound(err) {
			errs = append(errs, fmt.Errorf("failed to delete addon %s: %v", addonName, err))
			continue
		}
		baseAddonName := addon.Spec.AddonName
		// get the addon name without a version suffix, add it to purge list
		purgeList = append(purgeList, baseAddonName)

		logger.V(0).Info("deleted addon", "AddOnTemplate", addonName)
	}

	// check if there are any remaining addon templates for the same addon names as what was just deleted (different versions of the same addon)
	// dont use a label selector here - in case an addon with the same name was created out of band, and it is the last remaining version, we dont want
	// to delete its ClusterManagementAddOn
	allAddons, err := addonC.AddonV1alpha1().AddOnTemplates().List(ctx, metav1.ListOptions{})
	if err != nil && !kerrs.IsNotFound(err) {
		return fmt.Errorf("failed to clean up addons %v: %v", purgeList, err)
	}
	for _, a := range allAddons.Items {
		// if other versions of the same addon remain, remove it from the purge list
		purgeList = slices.DeleteFunc(purgeList, func(name string) bool {
			return name == a.Spec.AddonName
		})
	}
	// if list is empty, nothing else to do
	if len(purgeList) == 0 {
		return nil
	}

	// delete the ClusterManagementAddOn for any addon which has no active templates left
	for _, name := range purgeList {
		err = addonC.AddonV1alpha1().ClusterManagementAddOns().Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil && !kerrs.IsNotFound(err) {
			return fmt.Errorf("failed to purge addon %s: %v", name, err)
		}
		logger.V(0).Info("purged addon", "ClusterManagementAddOn", name)
	}

	// only return aggregated errs after trying to delete ClusterManagementAddOns.
	// this way, we dont accidentally leave any orphaned resources for addons which were successfully deleted.
	if len(errs) > 0 {
		return fmt.Errorf("one or more addons were not deleted: %v", errs)
	}

	return nil
}

func handleSpokeAddons(ctx context.Context, addonC *addonapi.Clientset, spoke *v1beta1.Spoke) ([]string, error) {
	logger := log.FromContext(ctx)
	addons := spoke.Spec.AddOns

	// Get actual enabled addons from cluster instead of status
	enabledAddons, err := getManagedClusterAddOns(ctx, addonC, spoke.Name)
	if err != nil {
		logger.V(1).Info("failed to get ManagedClusterAddOns, assuming none enabled", "error", err, "spokeName", spoke.Name)
		enabledAddons = []string{}
	}

	if len(addons) == 0 && len(enabledAddons) == 0 {
		// nothing to do
		return enabledAddons, nil
	}

	// compare existing to requested
	requestedAddonNames := make([]string, len(addons))
	for i, addon := range addons {
		requestedAddonNames[i] = addon.ConfigName
	}

	// Find addons that need to be enabled (present in requested, missing from enabledAddons)
	addonsToEnable := make([]v1beta1.AddOn, 0)
	for i, requestedName := range requestedAddonNames {
		if !slices.Contains(enabledAddons, requestedName) {
			addonsToEnable = append(addonsToEnable, addons[i])
		}
	}

	// Find addons that need to be disabled (present in enabledAddons, missing from requested)
	addonsToDisable := make([]string, 0)
	for _, enabledAddon := range enabledAddons {
		if !slices.Contains(requestedAddonNames, enabledAddon) {
			addonsToDisable = append(addonsToDisable, enabledAddon)
		}
	}

	// do disables first, then enables/updates
	err = handleAddonDisable(ctx, spoke, addonsToDisable)
	if err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return enabledAddons, err
	}

	// Remove disabled addons from enabledAddons
	for _, disabledAddon := range addonsToDisable {
		enabledAddons = slices.DeleteFunc(enabledAddons, func(ea string) bool {
			return ea == disabledAddon
		})
	}

	// Enable new addons and updated addons
	newEnabledAddons, err := handleAddonEnable(ctx, spoke, addonsToEnable, addonC)
	// even if an error is returned, any addon which was successfully enabled is tracked, so append before returning
	enabledAddons = append(enabledAddons, newEnabledAddons...)
	if err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return enabledAddons, err
	}
	spoke.SetConditions(true, v1beta1.NewCondition(
		v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	return enabledAddons, nil
}

func handleAddonEnable(ctx context.Context, spoke *v1beta1.Spoke, addons []v1beta1.AddOn, addonC *addonapi.Clientset) ([]string, error) {
	if len(addons) == 0 {
		return nil, nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("enableAddOns", "managedcluster", spoke.Name)

	baseArgs := append([]string{
		addon,
		enable,
		fmt.Sprintf("--cluster=%s", spoke.Name),
		fmt.Sprintf("--labels=%v", v1beta1.ManagedBySelector.String()),
	}, spoke.BaseArgs()...)

	var enableErrs []error
	enabledAddons := make([]string, 0)
	for _, a := range addons {
		args := []string{
			fmt.Sprintf("--names=%s", a.ConfigName),
		}
		if a.InstallNamespace != "" {
			args = append(args, fmt.Sprintf("--namespace=%s", a.InstallNamespace))
		}
		var annots []string
		for k, v := range a.Annotations {
			annots = append(annots, fmt.Sprintf("%s=%s", k, v))
		}
		annot := strings.Join(annots, ",")
		if annot != "" {
			args = append(args, fmt.Sprintf("--annotate=%s", annot))
		}

		args = append(baseArgs, args...)
		logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(args))
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon enable' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			enableErrs = append(enableErrs, fmt.Errorf("failed to enable addon: %v, output: %s", err, string(out)))
			continue
		}
		// TODO - do this natively with clusteradm once https://github.com/open-cluster-management-io/clusteradm/issues/501 is resolved.
		if a.ConfigName == v1beta1.FCCAddOnName {
			err = patchFCCMca(ctx, spoke.Name, addonC)
			if err != nil {
				enableErrs = append(enableErrs, err)
				continue
			}
		}

		enabledAddons = append(enabledAddons, a.ConfigName)
		logger.V(1).Info("enabled addon", "managedcluster", spoke.Name, "addon", a.ConfigName, "output", string(stdout))
	}

	if len(enableErrs) > 0 {
		return enabledAddons, fmt.Errorf("one or more addons were not enabled: %v", enableErrs)
	}
	return enabledAddons, nil
}

func patchFCCMca(ctx context.Context, spokeName string, addonC *addonapi.Clientset) error {
	mca, err := addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).Get(ctx, v1beta1.FCCAddOnName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to configure %s: %v", v1beta1.FCCAddOnName, err)
	}
	desired := addonv1alpha1.AddOnConfig{
		ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
			Group:    addonv1alpha1.GroupName,
			Resource: AddOnDeploymentConfigResource,
		},
		ConfigReferent: addonv1alpha1.ConfigReferent{
			Name:      v1beta1.FCCAddOnName,
			Namespace: spokeName,
		},
	}
	if slices.ContainsFunc(mca.Spec.Configs, func(c addonv1alpha1.AddOnConfig) bool {
		return c.Group == desired.Group &&
			c.Resource == desired.Resource &&
			c.Name == desired.Name &&
			c.Namespace == desired.Namespace
	}) {
		return nil
	}
	mca.Spec.Configs = append(mca.Spec.Configs, desired)
	patchBytes, err := json.Marshal(map[string]any{
		"spec": map[string]any{"configs": mca.Spec.Configs},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal patch for %s: %v", v1beta1.FCCAddOnName, err)
	}
	if _, err = addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).Patch(
		ctx,
		v1beta1.FCCAddOnName,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	); err != nil {
		return fmt.Errorf("failed to patch %s: %v", v1beta1.FCCAddOnName, err)
	}
	return nil
}

func handleAddonDisable(ctx context.Context, spoke *v1beta1.Spoke, enabledAddons []string) error {
	if len(enabledAddons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("disableAddOns", "managedcluster", spoke.Name)

	args := append([]string{
		addon,
		disable,
		fmt.Sprintf("--names=%s", strings.Join(enabledAddons, ",")),
		fmt.Sprintf("--clusters=%s", spoke.Name),
	}, spoke.BaseArgs()...)

	logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(args))
	cmd := exec.Command(clusteradm, args...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon disable' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		outStr := string(out)

		// Check if the error is due to addon not being found or cluster not found - these are success cases
		if strings.Contains(outStr, "add-on not found") {
			logger.V(5).Info("addon already disabled (not found)", "managedcluster", spoke.Name, "addons", enabledAddons, "output", outStr)
			return nil
		}
		if strings.Contains(outStr, "managedclusters.cluster.open-cluster-management.io") && strings.Contains(outStr, "not found") {
			logger.V(5).Info("addon disable skipped (cluster not found)", "managedcluster", spoke.Name, "addons", enabledAddons, "output", outStr)
			return nil
		}

		return fmt.Errorf("failed to disable addons: %v, output: %s", err, outStr)
	}
	logger.V(1).Info("disabled addons", "managedcluster", spoke.Name, "addons", enabledAddons, "output", string(stdout))
	return nil
}

// isHubAddOnMatching checks if an installed addon matches a desired addon spec
func isHubAddOnMatching(installed v1beta1.InstalledHubAddOn, desired v1beta1.HubAddOn, bundleVersion string) bool {
	return installed.Name == desired.Name &&
		installed.Namespace == desired.InstallNamespace &&
		installed.BundleVersion == bundleVersion
}

func handleHubAddons(ctx context.Context, addonC *addonapi.Clientset, hub *v1beta1.Hub) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleHubAddons", "fleetconfig", hub.Name)

	desiredAddOns := hub.Spec.HubAddOns
	bundleVersion := v1beta1.BundleVersionLatest
	if hub.Spec.ClusterManager != nil {
		bundleVersion = hub.Spec.ClusterManager.Source.BundleVersion
	}

	hubAddons, err := getHubAddOns(ctx, addonC)
	if err != nil {
		logger.V(1).Info("failed to get hub addons, assuming none installed", "error", err)
		hubAddons = []string{}
	}

	// use status as the source of truth for detailed addon information (namespace, version)
	// but cross-reference with actual cluster state to handle discrepancies
	installedAddOns := hub.Status.DeepCopy().InstalledHubAddOns

	// reconcile status with actual cluster state - remove from status any addons not found in cluster
	reconciledInstalledAddOns := make([]v1beta1.InstalledHubAddOn, 0)
	for _, installed := range installedAddOns {
		if slices.Contains(hubAddons, installed.Name) {
			reconciledInstalledAddOns = append(reconciledInstalledAddOns, installed)
		} else {
			logger.V(1).Info("addon in status but not found in cluster, removing from status", "addon", installed.Name)
		}
	}

	// nothing to do
	if len(desiredAddOns) == 0 && len(reconciledInstalledAddOns) == 0 {
		logger.V(5).Info("no hub addons to reconcile")
		return false, nil
	}

	// find addons that need to be uninstalled (present in installed, missing from desired or version mismatch)
	addonsToUninstall := make([]v1beta1.InstalledHubAddOn, 0)
	for _, installed := range reconciledInstalledAddOns {
		found := slices.ContainsFunc(desiredAddOns, func(desired v1beta1.HubAddOn) bool {
			return isHubAddOnMatching(installed, desired, bundleVersion)
		})
		if !found {
			addonsToUninstall = append(addonsToUninstall, installed)
		}
	}

	// find addons that need to be installed (present in desired, missing from installed or version upgrade)
	addonsToInstall := make([]v1beta1.HubAddOn, 0)
	for _, desired := range desiredAddOns {
		found := slices.ContainsFunc(reconciledInstalledAddOns, func(installed v1beta1.InstalledHubAddOn) bool {
			return isHubAddOnMatching(installed, desired, bundleVersion)
		})
		if !found {
			addonsToInstall = append(addonsToInstall, desired)
		}
	}

	// do uninstalls first, then installs
	err = handleHubAddonUninstall(ctx, addonsToUninstall, hub)
	if err != nil {
		return true, err
	}

	err = handleHubAddonInstall(ctx, addonC, addonsToInstall, bundleVersion, hub)
	if err != nil {
		return true, err
	}

	// Update status to reflect desired state - build the new installed addons list
	newInstalledAddOns := make([]v1beta1.InstalledHubAddOn, 0, len(desiredAddOns))
	for _, d := range desiredAddOns {
		newInstalledAddOns = append(newInstalledAddOns, v1beta1.InstalledHubAddOn{
			Name:          d.Name,
			Namespace:     d.InstallNamespace,
			BundleVersion: bundleVersion,
		})
	}
	hub.Status.InstalledHubAddOns = newInstalledAddOns
	return true, nil
}

func handleHubAddonUninstall(ctx context.Context, addons []v1beta1.InstalledHubAddOn, hub *v1beta1.Hub) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("uninstalling hub addons", "count", len(addons))

	var errs []error
	for _, addon := range addons {
		args := append([]string{
			uninstall,
			hubAddon,
			fmt.Sprintf("--names=%s", addon.Name),
		}, hub.BaseArgs()...)
		if addon.Namespace != "" {
			args = append(args, fmt.Sprintf("--namespace=%s", addon.Namespace))
		}

		logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(args))
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm uninstall hub-addon' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			outStr := string(out)
			errs = append(errs, fmt.Errorf("failed to uninstall hubAddon %s: %v, output: %s", addon.Name, err, outStr))
			continue
		}
		logger.V(1).Info("uninstalled hub addon", "name", addon.Name, "namespace", addon.Namespace, "output", string(stdout))
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more hub addons were not uninstalled: %v", errs)
	}
	return nil
}

func handleHubAddonInstall(ctx context.Context, addonC *addonapi.Clientset, addons []v1beta1.HubAddOn, bundleVersion string, hub *v1beta1.Hub) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("installing hub addons", "count", len(addons))

	var errs []error
	for _, addon := range addons {
		// Check if already installed (defensive check)
		installed, err := isAddonInstalled(ctx, addonC, addon.Name)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to check if hubAddon %s is installed: %v", addon.Name, err))
			continue
		}
		if installed {
			logger.V(3).Info("hubAddon already installed, skipping", "name", addon.Name)
			continue
		}

		args := append([]string{
			install,
			hubAddon,
			fmt.Sprintf("--names=%s", addon.Name),
			fmt.Sprintf("--bundle-version=%s", bundleVersion),
			fmt.Sprintf("--create-namespace=%t", addon.CreateNamespace),
		}, hub.BaseArgs()...)
		if addon.InstallNamespace != "" {
			args = append(args, fmt.Sprintf("--namespace=%s", addon.InstallNamespace))
		}

		logger.V(7).Info("running", "command", clusteradm, "args", arg_utils.SanitizeArgs(args))
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm install hub-addon' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			outStr := string(out)
			errs = append(errs, fmt.Errorf("failed to install hubAddon %s: %v, output: %s", addon.Name, err, outStr))
			continue
		}
		// the argocd pull integration addon logs the entire helm template output including CRDs to stdout.
		// to prevent flooding the logs, overwrite it.
		if addon.Name == v1beta1.AddonArgoCD {
			stdout = []byte("ArgoCD hub addon successfully installed")
		}
		logger.V(1).Info("installed hubAddon", "name", addon.Name, "output", string(stdout))
	}

	if len(errs) > 0 {
		return fmt.Errorf("one or more hub addons were not installed: %v", errs)
	}
	return nil
}

func isAddonInstalled(ctx context.Context, addonC *addonapi.Clientset, addonName string) (bool, error) {
	if _, err := addonC.AddonV1alpha1().ClusterManagementAddOns().Get(ctx, addonName, metav1.GetOptions{}); err != nil {
		return false, client.IgnoreNotFound(err)
	}

	// we enforce unique names between hubAddOns and addOnConfigs,
	// and handle deleting addOnConfigs first
	// so if the addon is found here, we can assume it was previously installed by `install hub-addon`
	return true, nil
}

// waitForAddonManifestWorksCleanup polls for addon-related manifestWorks to be removed
// after addon disable operation to avoid race conditions during spoke unjoin
func waitForAddonManifestWorksCleanup(ctx context.Context, workC *workapi.Clientset, spokeName string, timeout time.Duration, shouldCleanAll bool) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("waiting for addon manifestWorks cleanup", "spokeName", spokeName, "timeout", timeout)

	err := wait.PollUntilContextTimeout(ctx, addonCleanupPollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		manifestWorks, err := workC.WorkV1().ManifestWorks(spokeName).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.V(3).Info("failed to list manifestWorks during cleanup wait", "error", err)
			// Return false to continue polling on transient errors
			return false, nil
		}

		// for hub-as-spoke, or if the pivot failed, all addons must be removed.
		// otherwise, fleetconfig-controller-agent must not be removed.
		var expectedWorks = 0
		if !shouldCleanAll {
			expectedWorks = 1
		}

		if len(manifestWorks.Items) == expectedWorks {
			if shouldCleanAll {
				logger.V(1).Info("addon manifestWorks cleanup completed", "spokeName", spokeName, "remainingManifestWorks", len(manifestWorks.Items))
				return true, nil
			}
			mw := manifestWorks.Items[0]
			val, ok := mw.Labels[addonv1alpha1.AddonLabelKey]
			if !ok || val != v1beta1.FCCAddOnName {
				return false, fmt.Errorf("unexpected remaining ManifestWork: expected %s, got label=%q (ok=%t)", v1beta1.FCCAddOnName, val, ok)
			}
			logger.V(1).Info("addon manifestWorks cleanup completed", "spokeName", spokeName, "remainingManifestWork", mw.Name)
			return true, nil
		}

		logger.V(3).Info("waiting for addon manifestWorks cleanup",
			"spokeName", spokeName,
			"addonManifestWorks", len(manifestWorks.Items))

		// Continue polling
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for addon manifestWorks cleanup for spoke %s: %w", spokeName, err)
	}

	return nil
}

func allOwnersAddOns(mws []workv1.ManifestWork) bool {
	for _, m := range mws {
		if !slices.ContainsFunc(m.OwnerReferences, func(or metav1.OwnerReference) bool {
			return or.Kind == managedClusterAddOn
		}) {
			return false
		}
	}
	return true
}

// bindAddonAgent creates the necessary bindings for fcc agent to access hub resources
func (r *SpokeReconciler) bindAddonAgent(ctx context.Context, spoke *v1beta1.Spoke) error {
	roleName := os.Getenv(v1beta1.ClusterRoleNameEnvVar)
	if roleName == "" {
		roleName = v1beta1.DefaultFCCManagerRole
	}

	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
		Name:     roleName,
	}

	err := r.createBinding(ctx, roleRef, spoke.Namespace, spoke.Name)
	if err != nil {
		return err
	}
	if spoke.Spec.HubRef.Namespace != spoke.Namespace {
		err = r.createBinding(ctx, roleRef, spoke.Spec.HubRef.Namespace, spoke.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

// createBinding creates a role binding for a given role.
// The role binding follows a different naming format than OCM uses for addon agents.
// We need to append the spoke name to avoid possible conflicts in cases where multiple spokes exist in 1 namespace
func (r *SpokeReconciler) createBinding(ctx context.Context, roleRef rbacv1.RoleRef, namespace, spokeName string) error {
	logger := log.FromContext(ctx)

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("open-cluster-management:%s:%s:agent-%s",
				v1beta1.FCCAddOnName, strings.ToLower(roleRef.Kind), spokeName),
			Namespace: namespace,
			Labels: map[string]string{
				addonv1alpha1.AddonLabelKey: v1beta1.FCCAddOnName,
			},
		},
		RoleRef: roleRef,
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.GroupKind,
				APIGroup: rbacv1.GroupName,
				Name:     clusterAddonGroup(spokeName, v1beta1.FCCAddOnName),
			},
		},
	}

	err := r.Create(ctx, binding, &client.CreateOptions{})
	if err != nil {
		if !kerrs.IsAlreadyExists(err) {
			logger.Error(err, "failed to create role binding for addon")
			return err
		}
		curr := &rbacv1.RoleBinding{}
		err = r.Get(ctx, types.NamespacedName{Namespace: binding.Namespace, Name: binding.Name}, curr)
		if err != nil {
			logger.Error(err, "failed to get role binding for addon")
			return err
		}
		binding.SetResourceVersion(curr.ResourceVersion)
		err = r.Update(ctx, binding)
		if err != nil {
			logger.Error(err, "failed to update role binding for addon")
			return err
		}
	}
	return nil
}

// clusterAddonGroup returns the group that represents the addon for the cluster
// ref: https://github.com/open-cluster-management-io/ocm/blob/main/pkg/addon/templateagent/registration.go#L484
func clusterAddonGroup(clusterName, addonName string) string {
	return fmt.Sprintf("system:open-cluster-management:cluster:%s:addon:%s", clusterName, addonName)
}
