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

func enable(feature string) operatorv1.FeatureGate {
	return operatorv1.FeatureGate{Feature: feature, Mode: operatorv1.FeatureGateModeTypeEnable}
}

func disable(feature string) operatorv1.FeatureGate {
	return operatorv1.FeatureGate{Feature: feature, Mode: operatorv1.FeatureGateModeTypeDisable}
}

func TestHubFeatureGates(t *testing.T) {
	tests := []struct {
		name             string
		featureGates     string
		wantRegistration []operatorv1.FeatureGate
		wantWork         []operatorv1.FeatureGate
		wantAddOnManager []operatorv1.FeatureGate
		wantErr          bool
	}{
		{
			name:         "empty string yields only default-enabled gates",
			featureGates: "",
			// DefaultClusterSet and ResourceCleanup default to true for registration.
			wantRegistration: []operatorv1.FeatureGate{enable("DefaultClusterSet"), enable("ResourceCleanup")},
			wantWork:         nil,
			// AddonManagement defaults to true.
			wantAddOnManager: []operatorv1.FeatureGate{enable("AddonManagement")},
		},
		{
			name:         "enable a default-off registration gate",
			featureGates: "ManagedClusterAutoApproval=true",
			wantRegistration: []operatorv1.FeatureGate{
				enable("DefaultClusterSet"), enable("ManagedClusterAutoApproval"), enable("ResourceCleanup"),
			},
			wantWork:         nil,
			wantAddOnManager: []operatorv1.FeatureGate{enable("AddonManagement")},
		},
		{
			name:         "disable a default-on gate is recorded as Disable",
			featureGates: "AddonManagement=false,ResourceCleanup=false",
			wantRegistration: []operatorv1.FeatureGate{
				enable("DefaultClusterSet"), disable("ResourceCleanup"),
			},
			wantWork:         nil,
			wantAddOnManager: []operatorv1.FeatureGate{disable("AddonManagement")},
		},
		{
			name:         "gates are routed to the owning component",
			featureGates: "ManifestWorkReplicaSet=true",
			wantRegistration: []operatorv1.FeatureGate{
				enable("DefaultClusterSet"), enable("ResourceCleanup"),
			},
			wantWork:         []operatorv1.FeatureGate{enable("ManifestWorkReplicaSet")},
			wantAddOnManager: []operatorv1.FeatureGate{enable("AddonManagement")},
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
			if !reflect.DeepEqual(registration, tt.wantRegistration) {
				t.Errorf("registration = %+v, want %+v", registration, tt.wantRegistration)
			}
			if !reflect.DeepEqual(work, tt.wantWork) {
				t.Errorf("work = %+v, want %+v", work, tt.wantWork)
			}
			if !reflect.DeepEqual(addOnManager, tt.wantAddOnManager) {
				t.Errorf("addOnManager = %+v, want %+v", addOnManager, tt.wantAddOnManager)
			}
		})
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
		// Feature gates must still be present.
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
}

func TestApplyHubFeatureGates(t *testing.T) {
	values := &v1beta1.ClusterManagerChartConfig{}
	if err := applyHubFeatureGates(values, "ManifestWorkReplicaSet=true"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := values.ClusterManager.WorkConfiguration.FeatureGates; !reflect.DeepEqual(
		got, []operatorv1.FeatureGate{enable("ManifestWorkReplicaSet")},
	) {
		t.Errorf("work feature gates = %+v, want ManifestWorkReplicaSet enabled", got)
	}
	if len(values.ClusterManager.AddOnManagerConfiguration.FeatureGates) == 0 {
		t.Error("expected addon-manager feature gates to be populated with defaults")
	}
	if err := applyHubFeatureGates(values, "BadGate"); err == nil {
		t.Error("expected error for invalid feature gate string")
	}
}
