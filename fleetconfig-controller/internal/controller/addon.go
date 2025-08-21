package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os/exec"
	"slices"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
)

const (
	// commands
	addon   = "addon"
	create  = "create"
	enable  = "enable"
	disable = "disable"

	install   = "install"
	uninstall = "uninstall"
	hubAddon  = "hub-addon"

	// accepetd hub addon names
	hubAddOnArgoCD = "argocd"
	hubAddOnGPF    = "governance-policy-framework"

	argocdNamespace = "argocd"
)

func handleAddonConfig(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleAddOnConfig", "fleetconfig", fc.Name)

	// get existing addons
	createdAddOns, err := addonC.AddonV1alpha1().AddOnTemplates().List(ctx, metav1.ListOptions{LabelSelector: v1alpha1.ManagedBySelector.String()})
	if err != nil {
		return err
	}

	requestedAddOns := fc.Spec.AddOnConfigs

	// nothing to do
	if len(requestedAddOns) == 0 && len(createdAddOns.Items) == 0 {
		logger.V(5).Info("no addons to reconcile")
		return nil
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
	addonsToCreate := make([]v1alpha1.AddOnConfig, 0)
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
	err = handleAddonDelete(ctx, addonC, fc, addonsToDelete)
	if err != nil {
		return err
	}

	err = handleAddonCreate(ctx, kClient, addonC, fc, addonsToCreate)
	if err != nil {
		return err
	}

	return nil
}

func handleAddonCreate(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, fc *v1alpha1.FleetConfig, addons []v1alpha1.AddOnConfig) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("createAddOns", "fleetconfig", fc.Name)

	// set up array of clusteradm addon create commands
	for _, a := range addons {
		// look up manifests CM for the addon
		cm := corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-%s-%s", v1alpha1.AddonConfigMapNamePrefix, a.Name, a.Version)
		err := kClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: fc.Namespace}, &cm)
		if err != nil {
			return errors.Wrapf(err, "could not load configuration for add-on %s version %s", a.Name, a.Version)
		}

		args := append([]string{
			addon,
			create,
			a.Name,
			fmt.Sprintf("--version=%s", a.Version),
		}, fc.BaseArgs()...)

		// Extract manifest configuration from ConfigMap
		// validation was already done by the webhook, so simply check if raw manifests are provided and if not, use the URL.
		manifestsRaw, ok := cm.Data[v1alpha1.AddonConfigMapManifestRawKey]
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
			manifestsURL := cm.Data[v1alpha1.AddonConfigMapManifestURLKey]
			url, err := url.Parse(manifestsURL)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("failed to create addon %s version %s", a.Name, a.Version))
			}
			switch url.Scheme {
			case "http", "https":
				// pass URL directly
				args = append(args, fmt.Sprintf("--filename=%s", manifestsURL))
			default:
				return fmt.Errorf("unsupported URL scheme %s for addon %s version %s. Must be one of %v", url.Scheme, a.Name, a.Version, v1alpha1.AllowedAddonURLSchemes)
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

		logger.V(7).Info("running", "command", clusteradm, "args", args)
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon create' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			return fmt.Errorf("failed to create addon: %v, output: %s", err, string(out))
		}
		logger.V(0).Info("created addon", "AddOnTemplate", a.Name, "output", string(stdout))

		// label created resources
		err = labelConfigurationResources(ctx, addonC, a)
		if err != nil {
			logger.V(0).Error(err, "failed to label addon resources", "addon", a.Name, "version", a.Version)
		}
	}
	return nil
}

// labelConfigurationResources labels the AddOnTemplate and ClusterManagementAddOn resources created for an addon
func labelConfigurationResources(ctx context.Context, addonC *addonapi.Clientset, addon v1alpha1.AddOnConfig) error {
	logger := log.FromContext(ctx)

	// Label AddOnTemplate with a.Name-a.Version
	addonTemplateName := fmt.Sprintf("%s-%s", addon.Name, addon.Version)
	addonTemplate, err := addonC.AddonV1alpha1().AddOnTemplates().Get(ctx, addonTemplateName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get AddOnTemplate %s: %v", addonTemplateName, err)
	}

	// Add managedBy label to AddOnTemplate
	if addonTemplate.Labels == nil {
		addonTemplate.Labels = make(map[string]string)
	}
	maps.Copy(addonTemplate.Labels, v1alpha1.ManagedByLabels)

	patchBytes, err := json.Marshal(labelPatchData(addonTemplate.Labels))
	if err != nil {
		return fmt.Errorf("failed to marshal patch data for AddOnTemplate %s: %v", addonTemplateName, err)
	}

	_, err = addonC.AddonV1alpha1().AddOnTemplates().Patch(ctx, addonTemplate.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to update AddOnTemplate %s with labels: %v", addonTemplateName, err)
	}
	logger.V(2).Info("labeled AddOnTemplate", "name", addonTemplateName, "label", v1alpha1.LabelAddOnManagedBy)

	// Label ClusterManagementAddOn with a.Name
	clusterMgmtAddOn, err := addonC.AddonV1alpha1().ClusterManagementAddOns().Get(ctx, addon.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ClusterManagementAddOn %s: %v", addon.Name, err)
	}

	// Add managedBy label to ClusterManagementAddOn
	if clusterMgmtAddOn.Labels == nil {
		clusterMgmtAddOn.Labels = make(map[string]string)
	}
	maps.Copy(clusterMgmtAddOn.Labels, v1alpha1.ManagedByLabels)

	patchBytes, err = json.Marshal(labelPatchData(clusterMgmtAddOn.Labels))
	if err != nil {
		return fmt.Errorf("failed to marshal patch data for ClusterManagementAddOn %s: %v", addon.Name, err)
	}

	_, err = addonC.AddonV1alpha1().ClusterManagementAddOns().Patch(ctx, clusterMgmtAddOn.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ClusterManagementAddOn %s with labels: %v", addon.Name, err)
	}
	logger.V(2).Info("labeled ClusterManagementAddOn", "name", addon.Name, "label", v1alpha1.LabelAddOnManagedBy)

	return nil
}

func handleAddonDelete(ctx context.Context, addonC *addonapi.Clientset, fc *v1alpha1.FleetConfig, addons []string) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("deleteAddOns", "fleetconfig", fc.Name)

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
		if addon != nil {
			err = addonC.AddonV1alpha1().AddOnTemplates().Delete(ctx, addonName, metav1.DeleteOptions{})
			if err != nil && !kerrs.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("failed to delete addon %s: %v", addonName, err))
				continue
			}
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

func handleSpokeAddons(ctx context.Context, addonC *addonapi.Clientset, spoke v1alpha1.Spoke, fc *v1alpha1.FleetConfig) ([]string, error) {
	var enabledAddons []string

	addons := spoke.AddOns
	// check if this spoke already has any addons enabled, or if it has been unjoined
	if len(fc.Status.JoinedSpokes) > 0 {
		js := v1alpha1.JoinedSpoke{}
		idx := slices.IndexFunc(fc.Status.JoinedSpokes, func(s v1alpha1.JoinedSpoke) bool {
			return s.Name == spoke.Name
		})
		if idx == -1 {
			return nil, nil
		}
		js = fc.Status.JoinedSpokes[idx]

		// if unjoined, return early since addons are already uninstalled
		if fc.IsUnjoined(spoke, js) {
			return nil, nil
		}
		enabledAddons = append(enabledAddons, js.EnabledAddons...)
	}

	if len(addons) == 0 && len(enabledAddons) == 0 {
		// nothing to do
		return nil, nil
	}

	// compare existing to requested
	requestedAddonNames := make([]string, len(addons))
	for i, addon := range addons {
		requestedAddonNames[i] = addon.ConfigName
	}

	// Find addons that need to be enabled (present in requested, missing from prevEnabledAddons)
	addonsToEnable := make([]v1alpha1.AddOn, 0)
	for i, requestedName := range requestedAddonNames {
		if !slices.Contains(enabledAddons, requestedName) {
			addonsToEnable = append(addonsToEnable, addons[i])
		}
	}

	// Find addons that need to be disabled (present in prevEnabledAddons, missing from requested)
	addonsToDisable := make([]string, 0)
	for _, prevEnabledAddon := range enabledAddons {
		if !slices.Contains(requestedAddonNames, prevEnabledAddon) {
			addonsToDisable = append(addonsToDisable, prevEnabledAddon)
		}
	}

	// do disables first, then enables/updates
	err := handleAddonDisable(ctx, spoke.Name, addonsToDisable)
	if err != nil {
		return enabledAddons, err
	}

	// Remove disabled addons from enabledAddons
	for _, disabledAddon := range addonsToDisable {
		enabledAddons = slices.DeleteFunc(enabledAddons, func(ea string) bool {
			return ea == disabledAddon
		})
	}

	// Enable new addons and updated addons
	newEnabledAddons, err := handleAddonEnable(ctx, addonC, spoke.Name, addonsToEnable)
	// even if an error is returned, any addon which was successfully enabled is tracked, so append before returning
	enabledAddons = append(enabledAddons, newEnabledAddons...)
	if err != nil {
		return enabledAddons, err
	}

	return enabledAddons, nil
}

func handleAddonEnable(ctx context.Context, addonC *addonapi.Clientset, spokeName string, addons []v1alpha1.AddOn) ([]string, error) {
	if len(addons) == 0 {
		return nil, nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("enableAddOns", "managedcluster", spokeName)

	baseArgs := []string{
		addon,
		enable,
		fmt.Sprintf("--cluster=%s", spokeName),
	}

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
		args = append(args, fmt.Sprintf("--annotate=%s", annot))

		args = append(baseArgs, args...)
		logger.V(7).Info("running", "command", clusteradm, "args", args)
		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon enable' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			enableErrs = append(enableErrs, fmt.Errorf("failed to enable addon: %v, output: %s", err, string(out)))
			continue
		}
		err = labelManagedClusterAddOn(ctx, addonC, spokeName, a.ConfigName)
		if err != nil {
			enableErrs = append(enableErrs, err)
			continue
		}
		enabledAddons = append(enabledAddons, a.ConfigName)
		logger.V(1).Info("enabled addon", "managedcluster", spokeName, "addon", a.ConfigName, "output", string(stdout))
	}

	if len(enableErrs) > 0 {
		return enabledAddons, fmt.Errorf("one or more addons were not enabled: %v", enableErrs)
	}
	return enabledAddons, nil
}

func labelManagedClusterAddOn(ctx context.Context, addonC *addonapi.Clientset, spokeName, addonName string) error {
	logger := log.FromContext(ctx)

	mcao, err := addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).Get(ctx, addonName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ManagedClusterAddOn %s for spoke %s: %v", addonName, spokeName, err)
	}

	if mcao.Labels == nil {
		mcao.Labels = make(map[string]string)
	}
	maps.Copy(mcao.Labels, v1alpha1.ManagedByLabels)

	patchBytes, err := json.Marshal(labelPatchData(mcao.Labels))
	if err != nil {
		return fmt.Errorf("failed to marshal patch data for ManagedClusterAddOn %s: %v", addonName, err)
	}

	_, err = addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).Patch(ctx, mcao.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ManagedClusterAddOn %s for spoke %s with labels: %v", addonName, spokeName, err)
	}
	logger.V(2).Info("labeled ManagedClusterAddOn", "name", addonName, "spoke", spokeName, "label", v1alpha1.LabelAddOnManagedBy)

	return nil
}

func handleAddonDisable(ctx context.Context, spokeName string, addons []string) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("disableAddOns", "managedcluster", spokeName)

	args := []string{
		addon,
		disable,
		fmt.Sprintf("--names=%s", strings.Join(addons, ",")),
		fmt.Sprintf("--clusters=%s", spokeName),
	}

	logger.V(7).Info("running", "command", clusteradm, "args", args)
	cmd := exec.Command(clusteradm, args...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm addon disable' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		outStr := string(out)

		// Check if the error is due to addon not being found or cluster not found - these are success cases
		if strings.Contains(outStr, "add-on not found") {
			logger.V(5).Info("addon already disabled (not found)", "managedcluster", spokeName, "addons", addons, "output", outStr)
			return nil
		}
		if strings.Contains(outStr, "managedclusters.cluster.open-cluster-management.io") && strings.Contains(outStr, "not found") {
			logger.V(5).Info("addon disable skipped (cluster not found)", "managedcluster", spokeName, "addons", addons, "output", outStr)
			return nil
		}

		return fmt.Errorf("failed to disable addons: %v, output: %s", err, outStr)
	}
	logger.V(1).Info("disabled addons", "managedcluster", spokeName, "addons", addons, "output", string(stdout))
	return nil
}

// isHubAddOnMatching checks if an installed addon matches a desired addon spec
func isHubAddOnMatching(installed v1alpha1.InstalledHubAddOn, desired v1alpha1.HubAddOn, bundleVersion string) bool {
	return installed.Name == desired.Name &&
		installed.Namespace == desired.InstallNamespace &&
		installed.BundleVersion == bundleVersion
}

func handleHubAddons(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleHubAddons", "fleetconfig", fc.Name)

	installedAddOns := fc.Status.DeepCopy().InstalledHubAddOns
	desiredAddOns := fc.Spec.HubAddOns
	bundleVersion := fc.Spec.Hub.ClusterManager.Source.BundleVersion

	// nothing to do
	if len(desiredAddOns) == 0 && len(installedAddOns) == 0 {
		logger.V(5).Info("no hub addons to reconcile")
		return nil
	}

	// Find addons that need to be uninstalled (present in installed, missing from desired or version mismatch)
	addonsToUninstall := make([]v1alpha1.InstalledHubAddOn, 0)
	for _, installed := range installedAddOns {
		found := slices.ContainsFunc(desiredAddOns, func(desired v1alpha1.HubAddOn) bool {
			return isHubAddOnMatching(installed, desired, bundleVersion)
		})
		if !found {
			addonsToUninstall = append(addonsToUninstall, installed)
		}
	}

	// Find addons that need to be installed (present in desired, missing from installed or version upgrade)
	addonsToInstall := make([]v1alpha1.HubAddOn, 0)
	for _, desired := range desiredAddOns {
		found := slices.ContainsFunc(installedAddOns, func(installed v1alpha1.InstalledHubAddOn) bool {
			return isHubAddOnMatching(installed, desired, bundleVersion)
		})
		if !found {
			addonsToInstall = append(addonsToInstall, desired)
		}
	}

	// do uninstalls first, then installs
	err := handleHubAddonUninstall(ctx, addonsToUninstall)
	if err != nil {
		return err
	}

	err = handleHubAddonInstall(ctx, kClient, addonC, addonsToInstall, bundleVersion)
	if err != nil {
		return err
	}

	// build the new installed addons list
	newInstalledAddOns := make([]v1alpha1.InstalledHubAddOn, 0, len(desiredAddOns))
	for _, d := range desiredAddOns {
		newInstalledAddOns = append(newInstalledAddOns, v1alpha1.InstalledHubAddOn{
			Name:          d.Name,
			Namespace:     d.InstallNamespace,
			BundleVersion: bundleVersion,
		})
	}
	fc.Status.InstalledHubAddOns = newInstalledAddOns
	return nil
}

func handleHubAddonUninstall(ctx context.Context, addons []v1alpha1.InstalledHubAddOn) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("uninstalling hub addons", "count", len(addons))

	var errs []error
	for _, addon := range addons {
		args := []string{
			uninstall,
			hubAddon,
			fmt.Sprintf("--names=%s", addon.Name),
		}
		if addon.Namespace != "" {
			args = append(args, fmt.Sprintf("--namespace=%s", addon.Namespace))
		}

		logger.V(7).Info("running", "command", clusteradm, "args", args)
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

func handleHubAddonInstall(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, addons []v1alpha1.HubAddOn, bundleVersion string) error {
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

		// workaround until https://github.com/open-cluster-management-io/clusteradm/pull/510 is merged/released
		if addon.Name == hubAddOnArgoCD && addon.CreateNamespace {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: argocdNamespace,
				},
			}
			err := kClient.Create(ctx, ns)
			if err != nil && !kerrs.IsAlreadyExists(err) {
				errs = append(errs, fmt.Errorf("failed to create namespace for hubAddon %s: %v", addon.Name, err))
				continue
			}
		}

		args := []string{
			install,
			hubAddon,
			fmt.Sprintf("--names=%s", addon.Name),
			fmt.Sprintf("--bundle-version=%s", bundleVersion),
			fmt.Sprintf("--create-namespace=%t", addon.CreateNamespace),
		}
		if addon.InstallNamespace != "" {
			args = append(args, fmt.Sprintf("--namespace=%s", addon.InstallNamespace))
		}

		cmd := exec.Command(clusteradm, args...)
		stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm install hub-addon' to complete...")
		if err != nil {
			out := append(stdout, stderr...)
			outStr := string(out)
			errs = append(errs, fmt.Errorf("failed to install hubAddon %s: %v, output: %s", addon.Name, err, outStr))
			continue
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

func labelPatchData(labels map[string]string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"labels": labels,
		},
	}
}
