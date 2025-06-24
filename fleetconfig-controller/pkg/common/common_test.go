package common

import (
	"reflect"
	"testing"

	"k8s.io/component-base/featuregate"
	ocmfeature "open-cluster-management.io/api/feature"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
)

func TestExtractFeatureGates(t *testing.T) {
	tests := []struct {
		name string
		mc   *v1alpha1.FleetConfig
		want map[featuregate.Feature]bool
	}{
		{
			name: "nil fleetconfig",
			mc:   nil,
			want: map[featuregate.Feature]bool{},
		},
		{
			name: "nil cluster manager",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{},
			},
			want: map[featuregate.Feature]bool{},
		},
		{
			name: "empty feature gates",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{},
		},
		{
			name: "single feature gate enabled",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "AddonManagement=true",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{
				ocmfeature.AddonManagement: true,
			},
		},
		{
			name: "single feature gate disabled",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "ResourceCleanup=false",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{
				ocmfeature.ResourceCleanup: false,
			},
		},
		{
			name: "multiple feature gates",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "AddonManagement=true,ResourceCleanup=false",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{
				ocmfeature.AddonManagement: true,
				ocmfeature.ResourceCleanup: false,
			},
		},
		{
			name: "invalid boolean value",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "Feature=notabool",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{},
		},
		{
			name: "missing value",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: "Feature",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{},
		},
		{
			name: "with spaces",
			mc: &v1alpha1.FleetConfig{
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						ClusterManager: &v1alpha1.ClusterManager{
							FeatureGates: " AddonManagement = true , ResourceCleanup = false ",
						},
					},
				},
			},
			want: map[featuregate.Feature]bool{
				ocmfeature.AddonManagement: true,
				ocmfeature.ResourceCleanup: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFeatureGates(tt.mc)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractFeatureGates() = %v, want %v", got, tt.want)
			}
		})
	}
}
