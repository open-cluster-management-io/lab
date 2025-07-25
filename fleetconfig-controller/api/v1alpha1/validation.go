package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// allowFleetConfigUpdate validates the FleetConfig update object to determine if the update action is valid.
// Only the following updates are allowed:
//   - spec.addOnConfig
//   - spec.registrationAuth.*
//   - spec.hub.clusterManager.source.*
//   - spec.spokes[*].klusterlet.source.*
//   - spec.spokes[*].addOns
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
				oldSpokeCopy.Klusterlet.Source = (OCMSource{})
				newSpokeCopy.Klusterlet.Source = (OCMSource{})
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

func validateAddonConfigs(ctx context.Context, client client.Client, newObject *FleetConfig) field.ErrorList {
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

	return errs
}
