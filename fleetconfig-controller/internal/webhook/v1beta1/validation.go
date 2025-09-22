package v1beta1

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"

	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	"open-cluster-management.io/api/client/addon/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

const warnHubNotFound = "hub not found, cannot validate spoke addons"

func isKubeconfigValid(kubeconfig v1beta1.Kubeconfig) (bool, string) {
	if kubeconfig.SecretReference == nil && !kubeconfig.InCluster {
		return false, "either secretReference or inCluster must be specified for the kubeconfig"
	}
	if kubeconfig.SecretReference != nil && kubeconfig.InCluster {
		return false, "either secretReference or inCluster can be specified for the kubeconfig, not both"
	}
	return true, ""
}

// allowHubUpdate validates that only allowed fields are changed when updating a Hub.
// Allowed changes include:
// - spec.apiServer
// - spec.clusterManager.source.*
// - spec.hubAddOns
// - spec.addOnConfigs
// - spec.logVerbosity
// - spec.timeout
// - spec.registrationAuth
// - spec.kubeconfig
func allowHubUpdate(oldHub, newHub *v1beta1.Hub) error {
	if !reflect.DeepEqual(newHub.Spec, oldHub.Spec) {
		oldHubCopy := oldHub.Spec.DeepCopy()
		newHubCopy := newHub.Spec.DeepCopy()

		// Allow changes to ClusterManager.Source
		if oldHubCopy.ClusterManager != nil {
			oldHubCopy.ClusterManager.Source = (v1beta1.OCMSource{})
		}
		if newHubCopy.ClusterManager != nil {
			newHubCopy.ClusterManager.Source = (v1beta1.OCMSource{})
		}

		// Allow changes to API Server
		oldHubCopy.APIServer = ""
		newHubCopy.APIServer = ""

		// Allow changes to HubAddOns
		oldHubCopy.HubAddOns = nil
		newHubCopy.HubAddOns = nil

		// Allow changes to AddOnConfigs
		oldHubCopy.AddOnConfigs = nil
		newHubCopy.AddOnConfigs = nil

		// Allow changes to LogVerbosity
		oldHubCopy.LogVerbosity = 0
		newHubCopy.LogVerbosity = 0

		// Allow changes to Timeout
		oldHubCopy.Timeout = 0
		newHubCopy.Timeout = 0

		// Allow changes to RegistrationAuth
		oldHubCopy.RegistrationAuth = v1beta1.RegistrationAuth{}
		newHubCopy.RegistrationAuth = v1beta1.RegistrationAuth{}

		// Allow changes to Kubeconfig
		oldHubCopy.Kubeconfig = v1beta1.Kubeconfig{}
		newHubCopy.Kubeconfig = v1beta1.Kubeconfig{}

		if !reflect.DeepEqual(oldHubCopy, newHubCopy) {
			return errors.New("only changes to spec.apiServer, spec.clusterManager.source.*, spec.hubAddOns, spec.addOnConfigs, spec.logVerbosity, spec.timeout, and spec.registrationAuth are allowed when updating the hub")
		}
	}
	return nil
}

// allowSpokeUpdate validates that only allowed fields are changed when updating a Spoke.
// Allowed changes include:
// - spec.klusterlet.annotations
// - spec.klusterlet.values
// - spec.kubeconfig
// - spec.addOns
// - spec.timeout
// - spec.logVerbosity
func allowSpokeUpdate(oldSpoke, newSpoke *v1beta1.Spoke) error {
	if !reflect.DeepEqual(newSpoke.Spec, oldSpoke.Spec) {
		oldSpokeCopy := oldSpoke.Spec.DeepCopy()
		newSpokeCopy := newSpoke.Spec.DeepCopy()
		newSpokeCopy.Klusterlet.Annotations = nil
		oldSpokeCopy.Klusterlet.Annotations = nil
		oldSpokeCopy.Klusterlet.Values = nil
		newSpokeCopy.Klusterlet.Values = nil
		oldSpokeCopy.Kubeconfig = v1beta1.Kubeconfig{}
		newSpokeCopy.Kubeconfig = v1beta1.Kubeconfig{}
		oldSpokeCopy.AddOns = []v1beta1.AddOn{}
		newSpokeCopy.AddOns = []v1beta1.AddOn{}
		oldSpokeCopy.LogVerbosity = 0
		newSpokeCopy.LogVerbosity = 0
		oldSpokeCopy.Timeout = 0
		newSpokeCopy.Timeout = 0

		if !reflect.DeepEqual(oldSpokeCopy, newSpokeCopy) {
			return errors.New("spoke contains changes which are not allowed; only changes to spec.klusterlet.annotations, spec.klusterlet.values, spec.kubeconfig, spec.addOns, spec.timeout, and spec.logVerbosity are allowed when updating a spoke")
		}
	}

	return nil
}

// validateHubAddons checks that each addOnConfig specifies a valid source of manifests
// and validates uniqueness constraints between HubAddOns and AddOnConfigs
func validateHubAddons(ctx context.Context, cli client.Client, oldObject, newObject *v1beta1.Hub, addonC *versioned.Clientset) field.ErrorList {
	errs := field.ErrorList{}

	// Validate uniqueness and cross-references
	errs = append(errs, validateAddonUniqueness(newObject)...)

	// Validate AddOnConfig manifests
	errs = append(errs, validateAddOnConfigManifests(ctx, cli, newObject)...)

	// Validate removal constraints
	if oldObject != nil {
		errs = append(errs, validateAddonRemovalConstraints(ctx, oldObject, newObject, addonC)...)
	}

	return errs
}

// validateAddonUniqueness validates uniqueness constraints for addons
func validateAddonUniqueness(newObject *v1beta1.Hub) field.ErrorList {
	errs := field.ErrorList{}

	// Validate that AddOnConfig names are unique within the AddOnConfigs list
	addOnConfigNames := make(map[string]int)
	for i, a := range newObject.Spec.AddOnConfigs {
		key := fmt.Sprintf("%s-%s", a.Name, a.Version)
		if existingIndex, found := addOnConfigNames[key]; found {
			errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(i), key,
				fmt.Sprintf("duplicate addOnConfig %s (name-version) found at indices %d and %d", key, existingIndex, i)))
		} else {
			addOnConfigNames[key] = i
		}
	}

	// Validate that HubAddOn names are unique within the HubAddOns list
	hubAddOnNames := make(map[string]int)
	for i, ha := range newObject.Spec.HubAddOns {
		if existingIndex, found := hubAddOnNames[ha.Name]; found {
			errs = append(errs, field.Invalid(field.NewPath("hubAddOns").Index(i), ha.Name,
				fmt.Sprintf("duplicate hubAddOn name %s found at indices %d and %d", ha.Name, existingIndex, i)))
		} else {
			hubAddOnNames[ha.Name] = i
		}
	}

	// Validate unique names between HubAddOns and AddOnConfigs
	for i, ha := range newObject.Spec.HubAddOns {
		if _, found := addOnConfigNames[ha.Name]; found {
			errs = append(errs, field.Invalid(field.NewPath("hubAddOns").Index(i), ha.Name,
				fmt.Sprintf("hubAddOn name %s clashes with an existing addOnConfig name.", ha.Name)))
		}
	}

	return errs
}

// validateAddOnConfigManifests validates that each AddOnConfig has valid manifest sources
func validateAddOnConfigManifests(ctx context.Context, cli client.Client, newObject *v1beta1.Hub) field.ErrorList {
	errs := field.ErrorList{}

	for i, a := range newObject.Spec.AddOnConfigs {
		cm := corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-%s-%s", v1beta1.AddonConfigMapNamePrefix, a.Name, a.Version)
		err := cli.Get(ctx, types.NamespacedName{Name: cmName, Namespace: newObject.Namespace}, &cm)
		if err != nil {
			errs = append(errs, field.InternalError(field.NewPath("addOnConfigs").Index(i), err))
			continue
		}

		errs = append(errs, validateManifestSource(i, a, cm)...)
	}

	return errs
}

// validateManifestSource validates the manifest source configuration for an AddOnConfig
func validateManifestSource(index int, addon v1beta1.AddOnConfig, cm corev1.ConfigMap) field.ErrorList {
	errs := field.ErrorList{}

	// Extract manifest configuration from ConfigMap
	_, hasRaw := cm.Data[v1beta1.AddonConfigMapManifestRawKey]
	manifestsURL, hasURL := cm.Data[v1beta1.AddonConfigMapManifestURLKey]

	// Validate manifest configuration
	if !hasRaw && !hasURL {
		errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(index), addon.Name,
			fmt.Sprintf("no inline manifests or URL found for addon %s version %s", addon.Name, addon.Version)))
	}
	if hasRaw && hasURL {
		errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(index), addon.Name,
			fmt.Sprintf("only 1 of inline manifests or URL can be set for addon %s version %s", addon.Name, addon.Version)))
	}

	if hasURL {
		errs = append(errs, validateManifestURL(index, addon, manifestsURL)...)
	}

	return errs
}

// validateManifestURL validates the URL format and scheme for manifest sources
func validateManifestURL(index int, addon v1beta1.AddOnConfig, manifestsURL string) field.ErrorList {
	errs := field.ErrorList{}

	url, err := url.Parse(manifestsURL)
	if err != nil {
		errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(index), addon.Name,
			fmt.Sprintf("invalid URL '%s' for addon %s version %s. %v", manifestsURL, addon.Name, addon.Version, err.Error())))
		return errs
	}

	if !slices.Contains(v1beta1.AllowedAddonURLSchemes, url.Scheme) {
		errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(index), addon.Name,
			fmt.Sprintf("unsupported URL scheme %s for addon %s version %s. Must be one of %v",
				url.Scheme, addon.Name, addon.Version, v1beta1.AllowedAddonURLSchemes)))
	}

	return errs
}

// validateAddonRemovalConstraints validates that removed addons are not still in use
func validateAddonRemovalConstraints(ctx context.Context, oldObject, newObject *v1beta1.Hub, addonC *versioned.Clientset) field.ErrorList {
	errs := field.ErrorList{}

	// Check AddOnConfigs removal constraints
	removedAddOnConfigs := getRemovedAddOnConfigs(oldObject, newObject)
	if len(removedAddOnConfigs) > 0 {
		if removalErrs := validateAddonNotInUse(ctx, removedAddOnConfigs, "addOnConfigs", addonC); len(removalErrs) > 0 {
			errs = append(errs, removalErrs...)
		}
	}

	// Check HubAddOns removal constraints
	removedHubAddOns := getRemovedHubAddOns(oldObject, newObject)
	if len(removedHubAddOns) > 0 {
		if removalErrs := validateAddonNotInUse(ctx, removedHubAddOns, "hubAddOns", addonC); len(removalErrs) > 0 {
			errs = append(errs, removalErrs...)
		}
	}

	return errs
}

// getRemovedAddOnConfigs returns the list of AddOnConfigs that were removed
func getRemovedAddOnConfigs(oldObject, newObject *v1beta1.Hub) []string {
	oldAddOnConfigs := make(map[string]struct{})
	for _, a := range oldObject.Spec.AddOnConfigs {
		key := fmt.Sprintf("%s-%s", a.Name, a.Version)
		oldAddOnConfigs[key] = struct{}{}
	}

	newAddOnConfigs := make(map[string]struct{})
	for _, a := range newObject.Spec.AddOnConfigs {
		key := fmt.Sprintf("%s-%s", a.Name, a.Version)
		newAddOnConfigs[key] = struct{}{}
	}

	var removedAddOnConfigs []string
	for key := range oldAddOnConfigs {
		if _, found := newAddOnConfigs[key]; !found {
			removedAddOnConfigs = append(removedAddOnConfigs, key)
		}
	}

	return removedAddOnConfigs
}

// getRemovedHubAddOns returns the list of HubAddOns that were removed
func getRemovedHubAddOns(oldObject, newObject *v1beta1.Hub) []string {
	oldHubAddOns := make(map[string]struct{})
	for _, ha := range oldObject.Spec.HubAddOns {
		oldHubAddOns[ha.Name] = struct{}{}
	}

	newHubAddOns := make(map[string]struct{})
	for _, ha := range newObject.Spec.HubAddOns {
		newHubAddOns[ha.Name] = struct{}{}
	}

	var removedHubAddOns []string
	for name := range oldHubAddOns {
		if _, found := newHubAddOns[name]; !found {
			removedHubAddOns = append(removedHubAddOns, name)
		}
	}

	return removedHubAddOns
}

// validateAddonNotInUse validates that removed addons are not still referenced by ManagedClusterAddOns
func validateAddonNotInUse(ctx context.Context, removedAddons []string, fieldPath string, addonC *versioned.Clientset) field.ErrorList {
	errs := field.ErrorList{}

	mcAddOns, err := addonC.AddonV1alpha1().ManagedClusterAddOns(metav1.NamespaceAll).List(ctx, metav1.ListOptions{LabelSelector: v1beta1.ManagedBySelector.String()})
	if err != nil {
		errs = append(errs, field.InternalError(field.NewPath(fieldPath), err))
		return errs
	}

	var inUseAddons []string
	for _, removedAddon := range removedAddons {
		if isAddondEnabled(mcAddOns.Items, removedAddon) {
			inUseAddons = append(inUseAddons, removedAddon)
		}
	}

	if len(inUseAddons) > 0 {
		errs = append(errs, field.Invalid(field.NewPath(fieldPath), inUseAddons,
			fmt.Sprintf("cannot remove %s %v as they are still in use by managedclusteraddons", fieldPath, inUseAddons)))
	}

	return errs
}

// validates that any addon which is enabled on a spoke is configured
func validateAddons(ctx context.Context, cli client.Client, newObject *v1beta1.Spoke, addonC *versioned.Clientset) (admission.Warnings, field.ErrorList) {
	errs := field.ErrorList{}

	// try to get hub, if not present or not ready, log a warning that addons cant be properly validated
	hub := &v1beta1.Hub{}
	err := cli.Get(ctx, types.NamespacedName{Name: newObject.Spec.HubRef.Name, Namespace: newObject.Spec.HubRef.Namespace}, hub)
	if err != nil {
		if !kerrs.IsNotFound(err) {
			return nil, field.ErrorList{field.InternalError(field.NewPath("spec").Child("addOns"), err)}
		}
		// warn instead of an error, so we don't block creating spokes and hub at the same time
		return admission.Warnings{warnHubNotFound}, nil
	}

	initCond := hub.GetCondition(v1beta1.HubInitialized)
	if initCond == nil || initCond.Status != metav1.ConditionTrue {
		// warn instead of an error, so we don't block creating spokes and hub at the same time
		return admission.Warnings{warnHubNotFound}, nil
	}

	cmaList, err := addonC.AddonV1alpha1().ClusterManagementAddOns().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, field.ErrorList{field.InternalError(field.NewPath("spec").Child("addOns"), err)}
	}
	cmaNames := make([]string, len(cmaList.Items))
	for i, cma := range cmaList.Items {
		cmaNames[i] = cma.Name
	}

	for i, a := range newObject.Spec.AddOns {
		if !slices.Contains(cmaNames, a.ConfigName) {
			errs = append(errs, field.Invalid(field.NewPath("spec").Child("addOns").Index(i), a.ConfigName, "no matching HubAddOn or AddOnConfig found for AddOn"))
		}
	}

	return nil, errs
}

// isAddonConfigInUse checks if a removed addon config is still referenced by any ManagedClusterAddOn.
func isAddondEnabled(mcAddOns []addonv1alpha1.ManagedClusterAddOn, removedAddon string) bool {
	for _, mcao := range mcAddOns {
		for _, cr := range mcao.Status.ConfigReferences {
			if cr.DesiredConfig.Name == removedAddon {
				return true
			}
		}
	}
	return false
}
