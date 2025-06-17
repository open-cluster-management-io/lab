package v1alpha1

import (
	"testing"
)

func TestAllowFleetConfigUpdate(t *testing.T) {
	tests := []struct {
		name      string
		oldObject *FleetConfig
		newObject *FleetConfig
		wantErr   bool
		errMsg    string
	}{
		{
			name: "RegistrationAuth change are allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					RegistrationAuth: &RegistrationAuth{Driver: "csr"},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					RegistrationAuth: &RegistrationAuth{Driver: "awsirsa", HubClusterARN: "11111111:11111111:11111111:11111111"},
				},
			},
			wantErr: false,
		},
		{
			name: "no changes",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{CreateNamespace: true},
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{CreateNamespace: true},
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "hub change not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{CreateNamespace: true},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{CreateNamespace: false},
				},
			},
			wantErr: true,
			errMsg:  "only changes to hub.spec.hub.clusterManager.source.* are allowed when updating the hub",
		},
		{
			name: "hub ClusterManager source change allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "old-registry"},
						},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "new-registry"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "hub ClusterManager source bundle version change allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{BundleVersion: "v0.6.0"},
						},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{BundleVersion: "v0.7.0"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "hub ClusterManager source and non-source change not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						CreateNamespace: true,
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "old-registry"},
						},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						CreateNamespace: false,
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "new-registry"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "only changes to hub.spec.hub.clusterManager.source.* are allowed when updating the hub",
		},
		{
			name: "spoke source change allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spoke addition allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry2"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spoke removal allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry2"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spoke source change and addition allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry2"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spoke non-source change not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", CreateNamespace: true, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", CreateNamespace: false, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "registry"}}},
					},
				},
			},
			wantErr: true,
			errMsg:  "spoke 'spoke1' contains changes which are not allowed; only changes to spec.spokes[*].klusterlet.source.* are allowed when updating a spoke",
		},
		{
			name: "spoke source and non-source change not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", CreateNamespace: true, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", CreateNamespace: false, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry"}}},
					},
				},
			},
			wantErr: true,
			errMsg:  "spoke 'spoke1' contains changes which are not allowed; only changes to spec.spokes[*].klusterlet.source.* are allowed when updating a spoke",
		},
		{
			name: "spoke deletion and source change allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple spoke source changes allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry1"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry2"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry1"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry2"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "spoke source bundle version change allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{BundleVersion: "v0.6.0"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{BundleVersion: "v0.7.0"}}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "hub change with registration auth not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub:              Hub{CreateNamespace: true},
					RegistrationAuth: &RegistrationAuth{Driver: "csr"},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub:              Hub{CreateNamespace: false},
					RegistrationAuth: &RegistrationAuth{Driver: "awsirsa"},
				},
			},
			wantErr: true,
			errMsg:  "only changes to hub.spec.hub.clusterManager.source.* are allowed when updating the hub",
		},
		{
			name: "multiple spokes with one having non-source change not allowed",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry1"}}},
						{Name: "spoke2", CreateNamespace: true, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-registry2"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry1"}}},
						{Name: "spoke2", CreateNamespace: false, Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-registry2"}}},
					},
				},
			},
			wantErr: true,
			errMsg:  "spoke 'spoke2' contains changes which are not allowed; only changes to spec.spokes[*].klusterlet.source.* are allowed when updating a spoke",
		},
		{
			name: "hub ClusterManager source change with spoke changes allowed and adding a spoke while removing another",
			oldObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "old-hub-registry", BundleVersion: "v0.6.0"},
						},
					},
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-spoke1-registry"}}},
						{Name: "spoke2", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "old-spoke2-registry"}}},
					},
				},
			},
			newObject: &FleetConfig{
				Spec: FleetConfigSpec{
					Hub: Hub{
						ClusterManager: &ClusterManager{
							Source: &OCMSource{Registry: "new-hub-registry", BundleVersion: "v0.7.0"},
						},
					},
					Spokes: []Spoke{
						{Name: "spoke1", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-spoke1-registry"}}},
						{Name: "spoke3", Klusterlet: Klusterlet{Source: &OCMSource{Registry: "new-spoke3-registry"}}},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := allowFleetConfigUpdate(tt.newObject, tt.oldObject)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error message %q but got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
