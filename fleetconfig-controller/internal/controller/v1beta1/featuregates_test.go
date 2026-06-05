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
	type want struct {
		registration map[string]operatorv1.FeatureGateModeType
		work         map[string]operatorv1.FeatureGateModeType
		addOnManager map[string]operatorv1.FeatureGateModeType
	}
	tests := []struct {
		name         string
		featureGates string
		want         want
		wantErr      bool
	}{
		{
			name:         "empty string yields component defaults, fully explicit",
			featureGates: "",
			want: want{
				// DefaultClusterSet and ResourceCleanup default on; everything else off.
				registration: map[string]operatorv1.FeatureGateModeType{
					"DefaultClusterSet": operatorv1.FeatureGateModeTypeEnable,
					"ResourceCleanup":   operatorv1.FeatureGateModeTypeEnable,
					"ClusterProfile":    operatorv1.FeatureGateModeTypeDisable,
				},
				work: map[string]operatorv1.FeatureGateModeType{
					"ManifestWorkReplicaSet": operatorv1.FeatureGateModeTypeDisable,
				},
				addOnManager: map[string]operatorv1.FeatureGateModeType{
					"AddonManagement": operatorv1.FeatureGateModeTypeEnable,
				},
			},
		},
		{
			name:         "enabling a default-off work gate routes to workConfiguration",
			featureGates: "ManifestWorkReplicaSet=true",
			want: want{
				work: map[string]operatorv1.FeatureGateModeType{
					"ManifestWorkReplicaSet": operatorv1.FeatureGateModeTypeEnable,
				},
			},
		},
		{
			name:         "disabling a default-on gate is recorded as Disable",
			featureGates: "ResourceCleanup=false,AddonManagement=false",
			want: want{
				registration: map[string]operatorv1.FeatureGateModeType{
					"ResourceCleanup": operatorv1.FeatureGateModeTypeDisable,
				},
				addOnManager: map[string]operatorv1.FeatureGateModeType{
					"AddonManagement": operatorv1.FeatureGateModeTypeDisable,
				},
			},
		},
		{
			name:         "unparseable string returns an error",
			featureGates: "NotAFeature",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registration, work, addOnManager, err := hubFeatureGates(tt.featureGates)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertGateModes(t, "registration", registration, tt.want.registration)
			assertGateModes(t, "work", work, tt.want.work)
			assertGateModes(t, "addOnManager", addOnManager, tt.want.addOnManager)
		})
	}
}

func assertGateModes(t *testing.T, component string, gates []operatorv1.FeatureGate, want map[string]operatorv1.FeatureGateModeType) {
	t.Helper()
	for feature, wantMode := range want {
		gotMode, ok := gateMode(gates, feature)
		if !ok {
			t.Errorf("%s: feature %q missing from list %+v", component, feature, gates)
			continue
		}
		if gotMode != wantMode {
			t.Errorf("%s: feature %q mode = %q, want %q", component, feature, gotMode, wantMode)
		}
	}
}

// TestHubFeatureGatesExplicit asserts every known gate is listed explicitly, even
// when disabled. This is what lets a disabled default-off gate (e.g.
// ManifestWorkReplicaSet) override a previously-enabled ClusterManager value rather
// than being dropped by omitempty. See featureGatesForComponent.
func TestHubFeatureGatesExplicit(t *testing.T) {
	registration, work, addOnManager, err := hubFeatureGates("ManifestWorkReplicaSet=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All components list every known gate.
	if len(registration) != 6 {
		t.Errorf("registration: want 6 explicit gates, got %d: %+v", len(registration), registration)
	}
	if len(work) != 4 {
		t.Errorf("work: want 4 explicit gates, got %d: %+v", len(work), work)
	}
	if len(addOnManager) != 1 {
		t.Errorf("addOnManager: want 1 explicit gate, got %d: %+v", len(addOnManager), addOnManager)
	}
	// The disabled default-off gate must be present and explicitly Disable.
	mode, ok := gateMode(work, "ManifestWorkReplicaSet")
	if !ok || mode != operatorv1.FeatureGateModeTypeDisable {
		t.Errorf("ManifestWorkReplicaSet should be explicitly disabled, got mode=%q present=%v", mode, ok)
	}
}

func TestHubFeatureGatesDeterministic(t *testing.T) {
	const gates = "ManagedClusterAutoApproval=true,ManifestWorkReplicaSet=true,AddonManagement=false"
	reg, work, addon, err := hubFeatureGates(gates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 50; i++ {
		gotReg, gotWork, gotAddon, err := hubFeatureGates(gates)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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
		if err := applyHubFeatureGates(values, "AddonManagement=true"); err != nil {
			t.Fatal(err)
		}
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
		if err := applyHubFeatureGates(values, "ManifestWorkReplicaSet=false"); err != nil {
			t.Fatal(err)
		}
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
	if err := applyHubFeatureGates(values, "ManifestWorkReplicaSet=true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mode, ok := gateMode(values.ClusterManager.WorkConfiguration.FeatureGates, "ManifestWorkReplicaSet")
	if !ok || mode != operatorv1.FeatureGateModeTypeEnable {
		t.Errorf("work ManifestWorkReplicaSet mode = %q present = %v, want Enable", mode, ok)
	}
	if len(values.ClusterManager.AddOnManagerConfiguration.FeatureGates) == 0 {
		t.Error("expected addon-manager feature gates to be populated")
	}
	if err := applyHubFeatureGates(values, "BadGate"); err == nil {
		t.Error("expected error for invalid feature gate string")
	}
}
