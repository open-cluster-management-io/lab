/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"sort"
	"strconv"
	"strings"

	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// applyHubFeatureGates writes the hub's explicitly-specified feature gates into the
// cluster-manager chart values, routed to the registration/work/addon-manager
// configuration each gate belongs to.
//
// Only gates named in featureGates are written. Gates that are not specified are
// deliberately left out so the ClusterManager operator applies its own default for
// them: the operator's default is tied to the running OCM bundle and may differ from
// (or be newer than) this controller's compiled-in defaults, and may be enabled. We
// must not pin an unspecified gate to a default we guessed.
//
// clusteradm upgrade clustermanager merges the values file onto the live
// ClusterManager via yaml.Unmarshal: a featureGates slice replaces that component's
// gates wholesale, while a missing key is left untouched. So for a component the hub
// specifies at least one gate in, the written slice fully defines that component's
// gates (unspecified ones fall back to the operator default); for a component the hub
// specifies nothing in, the slice is empty, omitempty drops the key, and the
// ClusterManager keeps whatever clusteradm init established (again the operator
// default). Writing each specified gate explicitly — including those set to false —
// is what lets a change reach the ClusterManager; dropping disabled gates produced an
// empty list that omitempty removed, so a disabled gate never overrode a
// previously-enabled value.
func applyHubFeatureGates(values *v1beta1.ClusterManagerChartConfig, featureGates string) {
	registration, work, addOnManager := hubFeatureGates(featureGates)
	values.ClusterManager.RegistrationConfiguration.FeatureGates = registration
	values.ClusterManager.WorkConfiguration.FeatureGates = work
	values.ClusterManager.AddOnManagerConfiguration.FeatureGates = addOnManager
}

// hubFeatureGates parses the feature-gate string and returns the explicitly-specified
// gates routed to the registration, work, and addon-manager components.
func hubFeatureGates(featureGates string) (registration, work, addOnManager []operatorv1.FeatureGate) {
	specified := parseFeatureGates(featureGates)
	return componentFeatureGates(specified, ocmfeature.DefaultHubRegistrationFeatureGates),
		componentFeatureGates(specified, ocmfeature.DefaultHubWorkFeatureGates),
		componentFeatureGates(specified, ocmfeature.DefaultHubAddonManagerFeatureGates)
}

// parseFeatureGates parses the comma-separated "key=bool" string (the same format as
// clusteradm init --feature-gates, e.g. "AddonManagement=true,ResourceCleanup=false")
// into a map of explicitly-set gates. Malformed or valueless pairs are skipped;
// clusteradm validates the raw string against the running bundle when the hub is
// initialized (the controller passes it verbatim via --feature-gates).
func parseFeatureGates(featureGates string) map[string]bool {
	specified := map[string]bool{}
	for _, pair := range strings.Split(featureGates, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(pair), "=")
		if !found {
			continue
		}
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		specified[strings.TrimSpace(key)] = enabled
	}
	return specified
}

// componentFeatureGates returns operatorv1 FeatureGate entries for the specified gates
// that belong to the given component, sorted by name so the cluster-manager values
// hash is stable across reconciles (Go map iteration order is otherwise
// non-deterministic). Gates not known to the component are skipped.
func componentFeatureGates(specified map[string]bool, known map[featuregate.Feature]featuregate.FeatureSpec) []operatorv1.FeatureGate {
	var features []operatorv1.FeatureGate
	for name, enabled := range specified {
		if _, ok := known[featuregate.Feature(name)]; !ok {
			continue
		}
		mode := operatorv1.FeatureGateModeTypeDisable
		if enabled {
			mode = operatorv1.FeatureGateModeTypeEnable
		}
		features = append(features, operatorv1.FeatureGate{Feature: name, Mode: mode})
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].Feature < features[j].Feature
	})
	return features
}
