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
	"fmt"
	"sort"

	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"
	operatorv1 "open-cluster-management.io/api/operator/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// applyHubFeatureGates folds the hub's feature-gate string into the cluster-manager
// chart values so that feature-gate changes are both detected and propagated.
//
// clusteradm init applies feature gates via the --feature-gates flag, but
// 'clusteradm upgrade clustermanager' has no such flag: it reconstructs the
// registration/work/addon-manager configurations from the live ClusterManager CR
// and then merges --cluster-manager-values-file on top (the file overrides the CR).
// Writing the gates into the merged chart values therefore lets feature-gate changes
// (a) alter the values hash, which is what triggers an upgrade, and (b) override the
// gates carried forward from the CR during that upgrade. The translation mirrors
// clusteradm's hub init so init and upgrade converge on the same ClusterManager.
func applyHubFeatureGates(values *v1beta1.ClusterManagerChartConfig, featureGates string) error {
	registration, work, addOnManager, err := hubFeatureGates(featureGates)
	if err != nil {
		return err
	}
	values.ClusterManager.RegistrationConfiguration.FeatureGates = registration
	values.ClusterManager.WorkConfiguration.FeatureGates = work
	values.ClusterManager.AddOnManagerConfiguration.FeatureGates = addOnManager
	return nil
}

// hubFeatureGates parses the comma-separated feature-gate string (same format as
// 'clusteradm init --feature-gates', e.g. "AddonManagement=true,ResourceCleanup=false")
// and returns the registration, work, and addon-manager FeatureGate lists that
// clusteradm would apply for a hub.
func hubFeatureGates(featureGates string) (registration, work, addOnManager []operatorv1.FeatureGate, err error) {
	fg := featuregate.NewFeatureGate()
	for _, defaults := range []map[featuregate.Feature]featuregate.FeatureSpec{
		ocmfeature.DefaultHubRegistrationFeatureGates,
		ocmfeature.DefaultHubWorkFeatureGates,
		ocmfeature.DefaultHubAddonManagerFeatureGates,
	} {
		if err := fg.Add(defaults); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to register hub feature gates: %w", err)
		}
	}

	if featureGates != "" {
		if err := fg.Set(featureGates); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse feature gates %q: %w", featureGates, err)
		}
	}

	return convertToFeatureGateAPI(fg, ocmfeature.DefaultHubRegistrationFeatureGates),
		convertToFeatureGateAPI(fg, ocmfeature.DefaultHubWorkFeatureGates),
		convertToFeatureGateAPI(fg, ocmfeature.DefaultHubAddonManagerFeatureGates),
		nil
}

// convertToFeatureGateAPI converts the parsed feature gates into the operatorv1
// FeatureGate list for a single component, scoped to that component's known gates.
//
// Ported from open-cluster-management.io/clusteradm
// pkg/genericclioptions.ConvertToFeatureGateAPI so that 'clusteradm init' and the
// values-file path used by 'clusteradm upgrade clustermanager' produce identical
// gates. The result is sorted by feature name so the cluster-manager values hash is
// stable across reconciles (Go map iteration order is otherwise non-deterministic,
// which would cause spurious upgrades).
func convertToFeatureGateAPI(featureGates featuregate.MutableFeatureGate, defaultFeatureGate map[featuregate.Feature]featuregate.FeatureSpec) []operatorv1.FeatureGate {
	var features []operatorv1.FeatureGate
	featureGatesMap := featureGates.GetAll()

	// enable user-specified feature gates
	for feature := range featureGatesMap {
		if _, ok := defaultFeatureGate[feature]; !ok {
			continue
		}
		if featureGates.Enabled(feature) {
			features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: operatorv1.FeatureGateModeTypeEnable})
		} else if defaultFeatureGate[feature].Default {
			// Explicitly disable the feature gate that is enabled by default
			features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: operatorv1.FeatureGateModeTypeDisable})
		}
	}

	// enable default feature gates
	for feature, spec := range defaultFeatureGate {
		if _, ok := featureGatesMap[feature]; !ok && spec.Default {
			features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: operatorv1.FeatureGateModeTypeEnable})
		}
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].Feature < features[j].Feature
	})
	return features
}
