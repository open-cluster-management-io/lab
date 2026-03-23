package v1beta1

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	sigyaml "sigs.k8s.io/yaml"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	arg_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
)

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

	err = handleAddonCreate(ctx, kClient, addonC, hub, addonsToCreate)
	if err != nil {
		return true, err
	}

	return true, nil
}

func handleAddonCreate(ctx context.Context, kClient client.Client, addonC *addonapi.Clientset, hub *v1beta1.Hub, addons []v1beta1.AddOnConfig) error {
	if len(addons) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("createAddOns")

	for _, a := range addons {
		templateName := fmt.Sprintf("%s-%s", a.Name, a.Version)

		cm := corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-%s-%s", v1beta1.AddonConfigMapNamePrefix, a.Name, a.Version)
		if err := kClient.Get(ctx, types.NamespacedName{Name: cmName, Namespace: hub.Namespace}, &cm); err != nil {
			return errors.Wrapf(err, "could not load configuration for add-on %s version %s", a.Name, a.Version)
		}

		manifests, err := loadManifests(ctx, cm, a.Name, a.Version)
		if err != nil {
			return err
		}

		if err := applyCMA(ctx, addonC, a, templateName); err != nil {
			return err
		}
		if err := applyTemplate(ctx, addonC, a, templateName, manifests); err != nil {
			return err
		}
		logger.V(0).Info("created addon", "AddOnTemplate", templateName)
	}
	return nil
}

// applyCMA creates or updates the ClusterManagementAddOn for a custom addon template.
func applyCMA(ctx context.Context, addonC *addonapi.Clientset, a v1beta1.AddOnConfig, templateName string) error {
	logger := log.FromContext(ctx)

	cma := &addonv1alpha1.ClusterManagementAddOn{
		ObjectMeta: metav1.ObjectMeta{
			Name:   a.Name,
			Labels: v1beta1.ManagedByLabels,
			Annotations: map[string]string{
				"addon.open-cluster-management.io/lifecycle": "addon-manager",
			},
		},
		Spec: addonv1alpha1.ClusterManagementAddOnSpec{
			SupportedConfigs: []addonv1alpha1.ConfigMeta{
				{
					ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
						Group:    addonv1alpha1.GroupVersion.Group,
						Resource: "addontemplates",
					},
					DefaultConfig: &addonv1alpha1.ConfigReferent{
						Name: templateName,
					},
				},
			},
			InstallStrategy: addonv1alpha1.InstallStrategy{
				Type: addonv1alpha1.AddonInstallStrategyManual,
			},
		},
	}

	existing, err := addonC.AddonV1alpha1().ClusterManagementAddOns().Get(ctx, a.Name, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		_, err = addonC.AddonV1alpha1().ClusterManagementAddOns().Create(ctx, cma, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	if !a.Overwrite {
		logger.V(1).Info("overwrite is false - skipping updating ClusterManagementAddOn to use new AddOnTemplate")
		return nil
	}
	cma.ResourceVersion = existing.ResourceVersion
	_, err = addonC.AddonV1alpha1().ClusterManagementAddOns().Update(ctx, cma, metav1.UpdateOptions{})
	return err
}

// applyTemplate creates or updates the AddOnTemplate for a custom addon.
func applyTemplate(ctx context.Context, addonC *addonapi.Clientset, a v1beta1.AddOnConfig, templateName string, manifests []workv1.Manifest) error {
	logger := log.FromContext(ctx)

	tmpl := &addonv1alpha1.AddOnTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:   templateName,
			Labels: v1beta1.ManagedByLabels,
		},
		Spec: addonv1alpha1.AddOnTemplateSpec{
			AddonName: a.Name,
			AgentSpec: workv1.ManifestWorkSpec{
				Workload: workv1.ManifestsTemplate{
					Manifests: manifests,
				},
			},
			Registration: []addonv1alpha1.RegistrationSpec{},
		},
	}

	if a.HubRegistration {
		regSpec := addonv1alpha1.RegistrationSpec{
			Type: addonv1alpha1.RegistrationTypeKubeClient,
			KubeClient: &addonv1alpha1.KubeClientRegistrationConfig{
				HubPermissions: []addonv1alpha1.HubPermissionConfig{},
			},
		}
		if a.ClusterRoleBinding != "" {
			regSpec.KubeClient.HubPermissions = append(regSpec.KubeClient.HubPermissions,
				addonv1alpha1.HubPermissionConfig{
					Type: addonv1alpha1.HubPermissionsBindingCurrentCluster,
					CurrentCluster: &addonv1alpha1.CurrentClusterBindingConfig{
						ClusterRoleName: a.ClusterRoleBinding,
					},
				})
		}
		tmpl.Spec.Registration = []addonv1alpha1.RegistrationSpec{regSpec}
	}

	existing, err := addonC.AddonV1alpha1().AddOnTemplates().Get(ctx, templateName, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		_, err = addonC.AddonV1alpha1().AddOnTemplates().Create(ctx, tmpl, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	if !a.Overwrite {
		logger.V(1).Info("overwrite is false - skipping updating AddOnTemplate in-place")
		return nil
	}
	tmpl.ResourceVersion = existing.ResourceVersion
	_, err = addonC.AddonV1alpha1().AddOnTemplates().Update(ctx, tmpl, metav1.UpdateOptions{})
	return err
}

const maxManifestBytes = 10 << 20 // 10 MiB safety cap for URL-fetched manifests

var manifestClient = http.Client{
	Timeout: 30 * time.Second,
}

// loadManifests reads addon manifests from a ConfigMap (raw YAML or URL) and
// returns them as workv1.Manifest objects matching clusteradm's --filename behavior.
func loadManifests(ctx context.Context, cm corev1.ConfigMap, addonName, version string) ([]workv1.Manifest, error) {
	if raw, ok := cm.Data[v1beta1.AddonConfigMapManifestRawKey]; ok {
		return parseMultiDocYAML([]byte(raw))
	}

	manifestsURL := cm.Data[v1beta1.AddonConfigMapManifestURLKey]
	u, err := url.Parse(manifestsURL)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to create addon %s version %s", addonName, version))
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("unsupported URL scheme %s for addon %s version %s. Must be one of %v",
			u.Scheme, addonName, version, v1beta1.AllowedAddonURLSchemes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request for addon manifests: %w", err)
	}

	resp, err := manifestClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addon manifests from %s: %w", manifestsURL, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.FromContext(ctx).V(1).Info("close addon manifest response body", "url", manifestsURL, "error", closeErr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching addon manifests from %s", resp.StatusCode, manifestsURL)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read addon manifests from %s: %w", manifestsURL, err)
	}
	return parseMultiDocYAML(body)
}

// parseMultiDocYAML splits a multi-document YAML byte slice into workv1.Manifest objects.
func parseMultiDocYAML(data []byte) ([]workv1.Manifest, error) {
	var manifests []workv1.Manifest
	reader := yaml.NewYAMLReader(bufio.NewReader(strings.NewReader(string(data))))
	for {
		doc, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read YAML document: %w", err)
		}
		doc = []byte(strings.TrimSpace(string(doc)))
		if len(doc) == 0 {
			continue
		}
		jsonDoc, err := sigyaml.YAMLToJSON(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert YAML document to JSON: %w", err)
		}
		manifests = append(manifests, workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: jsonDoc},
		})
	}
	return manifests, nil
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

	mcas, err := addonC.AddonV1alpha1().ManagedClusterAddOns(spoke.Name).List(ctx, metav1.ListOptions{
		LabelSelector: v1beta1.ManagedBySelector.String(),
	})
	if err != nil {
		logger.V(1).Info("failed to list ManagedClusterAddOns, assuming none enabled", "error", err, "spokeName", spoke.Name)
		mcas = &addonv1alpha1.ManagedClusterAddOnList{}
	}

	currentAddons := make(map[string]struct{}, len(mcas.Items))
	for _, mca := range mcas.Items {
		currentAddons[mca.Name] = struct{}{}
	}

	if len(addons) == 0 && len(currentAddons) == 0 {
		return nil, nil
	}

	// Determine which addons to remove (present on cluster but no longer requested)
	requestedSet := make(map[string]struct{}, len(addons))
	for _, addon := range addons {
		requestedSet[addon.ConfigName] = struct{}{}
	}
	var addonsToDisable []string
	for name := range currentAddons {
		if _, requested := requestedSet[name]; !requested {
			addonsToDisable = append(addonsToDisable, name)
		}
	}

	if err := handleAddonDisable(ctx, addonC, spoke.Name, addonsToDisable); err != nil {
		return nil, err
	}

	// Enable/update all requested addons — applyMCA and applyAddOnDeploymentConfig
	// are idempotent and converge the spec directly, no hash comparison needed.
	enabledAddons, err := handleAddonEnable(ctx, addonC, spoke.Name, addons)
	if err != nil {
		return enabledAddons, err
	}

	return enabledAddons, nil
}

func handleAddonEnable(ctx context.Context, addonC *addonapi.Clientset, spokeName string, addons []v1beta1.AddOn) ([]string, error) {
	if len(addons) == 0 {
		return nil, nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("enableAddOns", "managedcluster", spokeName)

	var enableErrs []error
	enabledAddons := make([]string, 0, len(addons))
	for _, a := range addons {
		annotations := make(map[string]string)
		maps.Copy(annotations, a.Annotations)

		var mcaConfigs []addonv1alpha1.AddOnConfig
		if a.DeploymentConfig != nil {
			if err := applyAddOnDeploymentConfig(ctx, addonC, a.ConfigName, spokeName, a.DeploymentConfig); err != nil {
				enableErrs = append(enableErrs, fmt.Errorf("failed to apply addon deployment config for %s: %v", a.ConfigName, err))
				continue
			}
			mcaConfigs = []addonv1alpha1.AddOnConfig{
				{
					ConfigGroupResource: addonv1alpha1.ConfigGroupResource{
						Group:    addonv1alpha1.GroupVersion.Group,
						Resource: AddOnDeploymentConfigResource,
					},
					ConfigReferent: addonv1alpha1.ConfigReferent{
						Namespace: spokeName,
						Name:      a.ConfigName,
					},
				},
			}
		}

		mca := &addonv1alpha1.ManagedClusterAddOn{
			ObjectMeta: metav1.ObjectMeta{
				Name:        a.ConfigName,
				Namespace:   spokeName,
				Labels:      v1beta1.ManagedByLabels,
				Annotations: annotations,
			},
			Spec: addonv1alpha1.ManagedClusterAddOnSpec{
				Configs: mcaConfigs,
			},
		}

		if err := applyMCA(ctx, addonC, mca); err != nil {
			enableErrs = append(enableErrs, fmt.Errorf("failed to enable addon %s: %v", a.ConfigName, err))
			continue
		}

		enabledAddons = append(enabledAddons, a.ConfigName)
		logger.V(1).Info("enabled addon", "managedcluster", spokeName, "addon", a.ConfigName)
	}

	if len(enableErrs) > 0 {
		return enabledAddons, fmt.Errorf("one or more addons were not enabled: %v", enableErrs)
	}
	return enabledAddons, nil
}

// applyMCA creates or updates a ManagedClusterAddOn, matching upstream ApplyAddon semantics.
func applyMCA(ctx context.Context, addonC *addonapi.Clientset, mca *addonv1alpha1.ManagedClusterAddOn) error {
	existing, err := addonC.AddonV1alpha1().ManagedClusterAddOns(mca.Namespace).Get(ctx, mca.Name, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		_, err = addonC.AddonV1alpha1().ManagedClusterAddOns(mca.Namespace).Create(ctx, mca, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	// Overlay desired labels/annotations: keys from the Spoke win; keys only on the live object stay.
	// Keys removed from the Spoke are not deleted here. maps.Copy requires a non-nil dst, so clone when nil.
	// We do not set ManagedClusterAddOn.spec.installNamespace: OCM/addon-manager uses AddOnDeploymentConfig
	// (namespaced in the managed cluster namespace) and related defaults for install placement.
	if len(mca.Labels) > 0 {
		if existing.Labels == nil {
			existing.Labels = maps.Clone(mca.Labels)
		} else {
			maps.Copy(existing.Labels, mca.Labels)
		}
	}
	if len(mca.Annotations) > 0 {
		if existing.Annotations == nil {
			existing.Annotations = maps.Clone(mca.Annotations)
		} else {
			maps.Copy(existing.Annotations, mca.Annotations)
		}
	}
	existing.Spec.Configs = mca.Spec.Configs
	_, err = addonC.AddonV1alpha1().ManagedClusterAddOns(mca.Namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

// applyAddOnDeploymentConfig creates or updates an AddOnDeploymentConfig resource.
func applyAddOnDeploymentConfig(ctx context.Context, addonC *addonapi.Clientset, addonName, namespace string, config *addonv1alpha1.AddOnDeploymentConfigSpec) error {
	adc := &addonv1alpha1.AddOnDeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addonName,
			Namespace: namespace,
			Labels:    maps.Clone(v1beta1.ManagedByLabels),
		},
		Spec: *config,
	}

	existing, err := addonC.AddonV1alpha1().AddOnDeploymentConfigs(namespace).Get(ctx, addonName, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		_, err = addonC.AddonV1alpha1().AddOnDeploymentConfigs(namespace).Create(ctx, adc, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	existing.Spec = adc.Spec
	if len(adc.Labels) > 0 {
		if existing.Labels == nil {
			existing.Labels = maps.Clone(adc.Labels)
		} else {
			maps.Copy(existing.Labels, adc.Labels)
		}
	}
	_, err = addonC.AddonV1alpha1().AddOnDeploymentConfigs(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	return err
}

func handleAddonDisable(ctx context.Context, addonC *addonapi.Clientset, spokeName string, addonNames []string) error {
	if len(addonNames) == 0 {
		return nil
	}

	logger := log.FromContext(ctx)
	logger.V(0).Info("disableAddOns", "managedcluster", spokeName)

	var errs []error
	for _, name := range addonNames {
		err := addonC.AddonV1alpha1().ManagedClusterAddOns(spokeName).Delete(ctx, name, metav1.DeleteOptions{})
		if kerrs.IsNotFound(err) {
			logger.V(5).Info("addon already disabled (not found)", "managedcluster", spokeName, "addon", name)
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to disable addon %s: %w", name, err))
			continue
		}
		logger.V(1).Info("disabled addon", "managedcluster", spokeName, "addon", name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to disable addons: %v", errs)
	}
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

	bindingLabels := maps.Clone(v1beta1.ManagedByLabels)
	bindingLabels[addonv1alpha1.AddonLabelKey] = v1beta1.FCCAddOnName
	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("open-cluster-management:%s:%s:agent-%s",
				v1beta1.FCCAddOnName, strings.ToLower(roleRef.Kind), spokeName),
			Namespace: namespace,
			Labels:    bindingLabels,
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
