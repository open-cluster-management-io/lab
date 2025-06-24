package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
)

// allowFleetConfigUpdate validates the FleetConfig update object to determine if the update action is valid.
// Only the following updates are allowed:
//   - spec.registrationAuth.*
//   - spec.hub.clusterManager.source.*
//   - spec.spokes[*].klusterlet.source.*
func allowFleetConfigUpdate(newObject *FleetConfig, oldObject *FleetConfig) error {

	// Hub check
	if !reflect.DeepEqual(newObject.Spec.Hub, oldObject.Spec.Hub) {
		oldHubCopy := oldObject.Spec.Hub
		newHubCopy := newObject.Spec.Hub

		if oldHubCopy.ClusterManager != nil {
			oldHubCopy.ClusterManager.Source = nil
		}
		if newHubCopy.ClusterManager != nil {
			newHubCopy.ClusterManager.Source = nil
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
				oldSpokeCopy.Klusterlet.Source = nil
				newSpokeCopy.Klusterlet.Source = nil

				if !reflect.DeepEqual(oldSpokeCopy, newSpokeCopy) {
					return fmt.Errorf("spoke '%s' contains changes which are not allowed; only changes to spec.spokes[*].klusterlet.source.* are allowed when updating a spoke", newSpoke.Name)
				}
			}
		}
	}

	return nil
}
