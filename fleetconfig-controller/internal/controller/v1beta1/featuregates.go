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
// and returns the registration, work, and addon-manager FeatureGate lists for a hub.
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

	return featureGatesForComponent(fg, ocmfeature.DefaultHubRegistrationFeatureGates),
		featureGatesForComponent(fg, ocmfeature.DefaultHubWorkFeatureGates),
		featureGatesForComponent(fg, ocmfeature.DefaultHubAddonManagerFeatureGates),
		nil
}

// featureGatesForComponent returns the operatorv1 FeatureGate list for a single
// component, with an explicit Enable/Disable entry for every gate known to that
// component.
//
// Unlike clusteradm's init-time conversion (which omits disabled default-off gates
// and lets the operator default apply), every gate is listed explicitly. The lists
// are written to --cluster-manager-values-file, which 'clusteradm upgrade
// clustermanager' merges onto the existing ClusterManager via yaml.Unmarshal: a
// slice replaces wholesale, but a *missing* key is left untouched. An empty list is
// dropped by omitempty, so a disabled default-off gate (e.g. ManifestWorkReplicaSet,
// whose component has no default-on gates) would otherwise produce
// `workConfiguration: {}` and never override a previously-enabled value. Emitting all
// gates explicitly makes the merge set the exact desired state on every reconcile.
//
// Sorted by feature name so the cluster-manager values hash is stable across
// reconciles (Go map iteration order is otherwise non-deterministic).
func featureGatesForComponent(featureGates featuregate.MutableFeatureGate, defaultFeatureGate map[featuregate.Feature]featuregate.FeatureSpec) []operatorv1.FeatureGate {
	features := make([]operatorv1.FeatureGate, 0, len(defaultFeatureGate))
	for feature := range defaultFeatureGate {
		mode := operatorv1.FeatureGateModeTypeDisable
		if featureGates.Enabled(feature) {
			mode = operatorv1.FeatureGateModeTypeEnable
		}
		features = append(features, operatorv1.FeatureGate{Feature: string(feature), Mode: mode})
	}

	sort.Slice(features, func(i, j int) bool {
		return features[i].Feature < features[j].Feature
	})
	return features
}
