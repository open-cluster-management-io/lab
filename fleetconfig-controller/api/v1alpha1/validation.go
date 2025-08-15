package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// allowFleetConfigUpdate validates the FleetConfig update object to determine if the update action is valid.
// Only the following updates are allowed:
//   - spec.addOnConfig
//   - spec.registrationAuth.*
//   - spec.hub.clusterManager.source.*
//   - spec.spokes[*].addOns
//   - spec.spokes[*].klusterlet.annotations
//   - spec.spokes[*].klusterlet.source.*
//   - spec.spokes[*].klusterlet.values
//   - spec.spokes[*].kubeconfig
func allowFleetConfigUpdate(newObject *FleetConfig, oldObject *FleetConfig) error {

	// Hub check
	if !reflect.DeepEqual(newObject.Spec.Hub, oldObject.Spec.Hub) {
		oldHubCopy := oldObject.Spec.Hub
		newHubCopy := newObject.Spec.Hub

		if oldHubCopy.ClusterManager != nil {
			oldHubCopy.ClusterManager.Source = (OCMSource{})
		}
		if newHubCopy.ClusterManager != nil {
			newHubCopy.ClusterManager.Source = (OCMSource{})
		}

		if !reflect.DeepEqual(oldHubCopy, newHubCopy) {
			return errors.New("only changes to hub.spec.hub.clusterManager.source.* are allowed when updating the hub")
		}
	}

	// Spoke check
	if !reflect.DeepEqual(newObject.Spec.Spokes, oldObject.Spec.Spokes) {

		oldSpokes := make(map[string]Spoke)
		for _, spoke := range oldObject.Spec.Spokes {
			oldSpokes[spoke.Name] = spoke
		}

		// for spokes that exist in both old and new, check if the source changed
		for _, newSpoke := range newObject.Spec.Spokes {
			if oldSpoke, exists := oldSpokes[newSpoke.Name]; exists {
				oldSpokeCopy := oldSpoke
				newSpokeCopy := newSpoke
				newSpokeCopy.Klusterlet.Annotations = nil
				oldSpokeCopy.Klusterlet.Annotations = nil
				oldSpokeCopy.Klusterlet.Source = (OCMSource{})
				newSpokeCopy.Klusterlet.Source = (OCMSource{})
				oldSpokeCopy.Klusterlet.Values = nil
				newSpokeCopy.Klusterlet.Values = nil
				oldSpokeCopy.Kubeconfig = Kubeconfig{}
				newSpokeCopy.Kubeconfig = Kubeconfig{}
				newSpokeCopy.AddOns = []AddOn{}
				oldSpokeCopy.AddOns = []AddOn{}

				if !reflect.DeepEqual(oldSpokeCopy, newSpokeCopy) {
					return fmt.Errorf("spoke '%s' contains changes which are not allowed; only changes to spec.spokes[*].klusterlet.source.* are allowed when updating a spoke", newSpoke.Name)
				}
			}
		}
	}

	return nil
}

// checks that each addOnConfig specifies a valid source of manifests
func validateAddonConfigs(ctx context.Context, client client.Client, oldObject, newObject *FleetConfig) field.ErrorList {
	errs := field.ErrorList{}
	for i, a := range newObject.Spec.AddOnConfigs {
		cm := corev1.ConfigMap{}
		cmName := fmt.Sprintf("%s-%s-%s", AddonConfigMapNamePrefix, a.Name, a.Version)
		err := client.Get(ctx, types.NamespacedName{Name: cmName, Namespace: newObject.Namespace}, &cm)
		if err != nil {
			errs = append(errs, field.InternalError(field.NewPath("addOnConfigs").Index(i), err))
		}
		// Extract manifest configuration from ConfigMap
		_, hasRaw := cm.Data[AddonConfigMapManifestRawKey]
		manifestsURL, hasURL := cm.Data[AddonConfigMapManifestURLKey]

		// Validate manifest configuration
		if !hasRaw && !hasURL {
			// return fmt.Errorf("no inline manifests or URL found for addon %s version %s", a.Name, a.Version)
			errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(i), a.Name, fmt.Sprintf("no inline manifests or URL found for addon %s version %s", a.Name, a.Version)))
		}
		if hasRaw && hasURL {
			errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(i), a.Name, fmt.Sprintf("only 1 of inline manifests or URL can be set for addon %s version %s", a.Name, a.Version)))
		}

		if hasURL {
			url, err := url.Parse(manifestsURL)
			if err != nil {
				errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(i), a.Name, fmt.Sprintf("invalid URL '%s' for addon %s version %s. %v", manifestsURL, a.Name, a.Version, err.Error())))
				continue
			}
			if !slices.Contains(AllowedAddonURLSchemes, url.Scheme) {
				errs = append(errs, field.Invalid(field.NewPath("addOnConfigs").Index(i), a.Name, fmt.Sprintf("unsupported URL scheme %s for addon %s version %s. Must be one of %v", manifestsURL, a.Name, a.Version, AllowedAddonURLSchemes)))
			}
		}
	}

	if oldObject != nil {
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

		removedAddOnConfigs := make([]string, 0)
		for key := range oldAddOnConfigs {
			if _, found := newAddOnConfigs[key]; !found {
				removedAddOnConfigs = append(removedAddOnConfigs, key)
			}
		}

		// Check if any removed addon configs are still in use by managed cluster addons
		if len(removedAddOnConfigs) > 0 {
			mcAddOns, err := getManagedClusterAddOns(ctx)
			if err != nil {
				errs = append(errs, field.InternalError(field.NewPath("addOnConfigs"), err))
			} else {
				// Check if any removed addon configs are still in use
				for _, removedConfig := range removedAddOnConfigs {
					if isAddonConfigInUse(mcAddOns, removedConfig) {
						errs = append(errs, field.Invalid(field.NewPath("addOnConfigs"), removedConfig,
							fmt.Sprintf("cannot remove addon config %s as it is still in use by managedclusteraddons", removedConfig)))
					}
				}
			}
		}
	}

	return errs
}

// validates that any addon which is enabled on a spoke is configured
func validateAddons(newObject *FleetConfig) field.ErrorList {
	errs := field.ErrorList{}
	configuredAddons := make(map[string]bool)
	for _, ca := range newObject.Spec.AddOnConfigs {
		configuredAddons[ca.Name] = true
	}
	for i, s := range newObject.Spec.Spokes {
		for j, a := range s.AddOns {
			if !configuredAddons[a.ConfigName] {
				errs = append(errs, field.Invalid(field.NewPath("Spokes").Index(i).Child("AddOns").Index(j), a.ConfigName, fmt.Sprintf("cannot enable addon %s for spoke %s, no configuration found in spec.AddOnConfigs", a.ConfigName, s.Name)))
			}
		}
	}

	return errs
}

// getManagedClusterAddOns lists all ManagedClusterAddOns in all namespaces.
func getManagedClusterAddOns(ctx context.Context) ([]addonv1alpha1.ManagedClusterAddOn, error) {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	addonClientset, err := addonapi.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create addon clientset: %w", err)
	}
	addonList, err := addonClientset.AddonV1alpha1().ManagedClusterAddOns(metav1.NamespaceAll).List(ctx, metav1.ListOptions{LabelSelector: ManagedBySelector.String()})
	if err != nil {
		return nil, fmt.Errorf("failed to list ManagedClusterAddOns: %w", err)
	}
	return addonList.Items, nil
}

// isAddonConfigInUse checks if a removed addon config is still referenced by any ManagedClusterAddOn.
func isAddonConfigInUse(mcAddOns []addonv1alpha1.ManagedClusterAddOn, removedConfig string) bool {
	for _, mcao := range mcAddOns {
		for _, cr := range mcao.Status.ConfigReferences {
			if cr.DesiredConfig.Name == removedConfig {
				return true
			}
		}
	}
	return false
}
