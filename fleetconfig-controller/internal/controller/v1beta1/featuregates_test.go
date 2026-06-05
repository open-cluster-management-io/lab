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
	"reflect"
	"strings"
	"testing"

	operatorv1 "open-cluster-management.io/api/operator/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// gateMode returns the mode set for feature, and whether it is present at all.
func gateMode(gates []operatorv1.FeatureGate, feature string) (operatorv1.FeatureGateModeType, bool) {
	for _, g := range gates {
		if g.Feature == feature {
			return g.Mode, true
		}
	}
	return "", false
}

func TestHubFeatureGates(t *testing.T) {
	tests := []struct {
		name         string
		featureGates string
		// wantPresent maps "component/feature" to its expected mode.
		wantPresent map[string]operatorv1.FeatureGateModeType
		// wantAbsent lists "component/feature" pairs that must NOT be written, so the
		// operator applies its own default for them.
		wantAbsent []string
	}{
		{
			name:         "empty string writes nothing, leaving all gates to operator defaults",
			featureGates: "",
			wantAbsent: []string{
				"registration/DefaultClusterSet",
				"registration/ResourceCleanup",
				"work/ManifestWorkReplicaSet",
				"addOnManager/AddonManagement",
			},
		},
		{
			name:         "only specified gates are written, routed to their component",
			featureGates: "ManifestWorkReplicaSet=true",
			wantPresent: map[string]operatorv1.FeatureGateModeType{
				"work/ManifestWorkReplicaSet": operatorv1.FeatureGateModeTypeEnable,
			},
			// Unspecified gates (incl. default-on ones) are left to the operator.
			wantAbsent: []string{
				"registration/DefaultClusterSet",
				"registration/ResourceCleanup",
				"addOnManager/AddonManagement",
			},
		},
		{
			name:         "disabling a default-off gate is written explicitly so it overrides",
			featureGates: "ManifestWorkReplicaSet=false",
			wantPresent: map[string]operatorv1.FeatureGateModeType{
				"work/ManifestWorkReplicaSet": operatorv1.FeatureGateModeTypeDisable,
			},
		},
		{
			name:         "mixed enable/disable across components",
			featureGates: "DefaultClusterSet=true,ManifestWorkReplicaSet=false,ResourceCleanup=false",
			wantPresent: map[string]operatorv1.FeatureGateModeType{
				"registration/DefaultClusterSet": operatorv1.FeatureGateModeTypeEnable,
				"registration/ResourceCleanup":   operatorv1.FeatureGateModeTypeDisable,
				"work/ManifestWorkReplicaSet":    operatorv1.FeatureGateModeTypeDisable,
			},
		},
		{
			name:         "unknown gate is skipped, not written",
			featureGates: "NotAFeature=true",
			wantAbsent: []string{
				"registration/NotAFeature",
				"work/NotAFeature",
				"addOnManager/NotAFeature",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registration, work, addOnManager := hubFeatureGates(tt.featureGates)
			byComponent := map[string][]operatorv1.FeatureGate{
				"registration": registration,
				"work":         work,
				"addOnManager": addOnManager,
			}
			for ref, wantMode := range tt.wantPresent {
				component, feature := splitRef(ref)
				gotMode, ok := gateMode(byComponent[component], feature)
				if !ok {
					t.Errorf("%s missing; want mode %q. list=%+v", ref, wantMode, byComponent[component])
					continue
				}
				if gotMode != wantMode {
					t.Errorf("%s mode = %q, want %q", ref, gotMode, wantMode)
				}
			}
			for _, ref := range tt.wantAbsent {
				component, feature := splitRef(ref)
				if _, ok := gateMode(byComponent[component], feature); ok {
					t.Errorf("%s should not be written (left to operator default), but was. list=%+v", ref, byComponent[component])
				}
			}
		})
	}
}

func splitRef(ref string) (component, feature string) {
	component, feature, _ = strings.Cut(ref, "/")
	return component, feature
}

func TestHubFeatureGatesDeterministic(t *testing.T) {
	const gates = "ManagedClusterAutoApproval=true,ManifestWorkReplicaSet=true,AddonManagement=false"
	reg, work, addon := hubFeatureGates(gates)
	for i := 0; i < 50; i++ {
		gotReg, gotWork, gotAddon := hubFeatureGates(gates)
		if !reflect.DeepEqual(gotReg, reg) ||
			!reflect.DeepEqual(gotWork, work) ||
			!reflect.DeepEqual(gotAddon, addon) {
			t.Fatalf("feature gate output is not deterministic across calls")
		}
	}
}

func TestMarshalClusterManagerValues(t *testing.T) {
	t.Run("prunes empty resourceRequirement so it cannot clobber --resource-qos-class", func(t *testing.T) {
		values := &v1beta1.ClusterManagerChartConfig{}
		applyHubFeatureGates(values, "AddonManagement=true")
		out, err := marshalClusterManagerValues(values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(string(out), "resourceRequirement") {
			t.Errorf("expected empty resourceRequirement to be pruned, got:\n%s", out)
		}
		if !strings.Contains(string(out), "AddonManagement") {
			t.Errorf("expected feature gates to be retained, got:\n%s", out)
		}
	})

	t.Run("keeps resourceRequirement when a QoS type is set", func(t *testing.T) {
		values := &v1beta1.ClusterManagerChartConfig{}
		values.ClusterManager.ResourceRequirement.Type = operatorv1.ResourceQosClassBestEffort
		out, err := marshalClusterManagerValues(values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(out), "BestEffort") {
			t.Errorf("expected resourceRequirement with a set type to be retained, got:\n%s", out)
		}
	})

	t.Run("disabled default-off work gate is written so it overrides the ClusterManager", func(t *testing.T) {
		values := &v1beta1.ClusterManagerChartConfig{}
		applyHubFeatureGates(values, "ManifestWorkReplicaSet=false")
		out, err := marshalClusterManagerValues(values)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The work feature gates must be present (not an empty workConfiguration: {})
		// so the merge replaces the CR value rather than leaving it stale.
		if !strings.Contains(string(out), "ManifestWorkReplicaSet") {
			t.Errorf("expected ManifestWorkReplicaSet to be written explicitly, got:\n%s", out)
		}
	})
}

func TestApplyHubFeatureGates(t *testing.T) {
	values := &v1beta1.ClusterManagerChartConfig{}
	applyHubFeatureGates(values, "ManifestWorkReplicaSet=true")

	mode, ok := gateMode(values.ClusterManager.WorkConfiguration.FeatureGates, "ManifestWorkReplicaSet")
	if !ok || mode != operatorv1.FeatureGateModeTypeEnable {
		t.Errorf("work ManifestWorkReplicaSet mode = %q present = %v, want Enable", mode, ok)
	}
	// Unspecified components are left empty so the operator applies its defaults.
	if len(values.ClusterManager.AddOnManagerConfiguration.FeatureGates) != 0 {
		t.Errorf("addon-manager gates should be empty when unspecified, got %+v",
			values.ClusterManager.AddOnManagerConfiguration.FeatureGates)
	}
}
