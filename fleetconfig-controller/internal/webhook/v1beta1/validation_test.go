package v1beta1

import (
	"testing"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

func TestIsKubeconfigValid(t *testing.T) {
	tests := []struct {
		name       string
		kubeconfig v1beta1.Kubeconfig
		wantValid  bool
		wantMsg    string
	}{
		{
			name: "valid InCluster kubeconfig",
			kubeconfig: v1beta1.Kubeconfig{
				InCluster: true,
			},
			wantValid: true,
			wantMsg:   "",
		},
		{
			name: "valid SecretReference kubeconfig",
			kubeconfig: v1beta1.Kubeconfig{
				SecretReference: &v1beta1.SecretReference{
					Name:          "test-secret",
					KubeconfigKey: "kubeconfig",
				},
			},
			wantValid: true,
			wantMsg:   "",
		},
		{
			name: "invalid - neither InCluster nor SecretReference",
			kubeconfig: v1beta1.Kubeconfig{
				InCluster: false,
			},
			wantValid: false,
			wantMsg:   "either secretReference or inCluster must be specified for the kubeconfig",
		},
		{
			name: "invalid - both InCluster and SecretReference",
			kubeconfig: v1beta1.Kubeconfig{
				InCluster: true,
				SecretReference: &v1beta1.SecretReference{
					Name:          "test-secret",
					KubeconfigKey: "kubeconfig",
				},
			},
			wantValid: false,
			wantMsg:   "either secretReference or inCluster can be specified for the kubeconfig, not both",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := isKubeconfigValid(tt.kubeconfig)
			if valid != tt.wantValid {
				t.Errorf("isKubeconfigValid() valid = %v, want %v", valid, tt.wantValid)
			}
			if msg != tt.wantMsg {
				t.Errorf("isKubeconfigValid() msg = %v, want %v", msg, tt.wantMsg)
			}
		})
	}
}

func TestAllowHubUpdate(t *testing.T) {
	tests := []struct {
		name    string
		oldHub  *v1beta1.Hub
		newHub  *v1beta1.Hub
		wantErr bool
		errMsg  string
	}{
		{
			name: "no changes",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					CreateNamespace: true,
					Timeout:         300,
					LogVerbosity:    0,
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					CreateNamespace: true,
					Timeout:         300,
					LogVerbosity:    0,
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - timeout change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Timeout: 300,
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Timeout: 600,
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - logVerbosity change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					LogVerbosity: 0,
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					LogVerbosity: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - ClusterManager source change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					ClusterManager: &v1beta1.ClusterManager{
						Source: v1beta1.OCMSource{Registry: "old-registry"},
					},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					ClusterManager: &v1beta1.ClusterManager{
						Source: v1beta1.OCMSource{Registry: "new-registry"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - APIServer change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					APIServer: "https://old-api-server:6443",
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					APIServer: "https://new-api-server:6443",
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - HubAddOns change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "governance-policy-framework"},
					},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - AddOnConfigs change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "old-addon", Version: "v1.0.0"},
					},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "new-addon", Version: "v2.0.0"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - RegistrationAuth change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: "csr"},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver:        "awsirsa",
						HubClusterARN: "arn:aws:eks:us-west-2:123456789013:cluster/test-hub",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "disallowed - CreateNamespace change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					CreateNamespace: true,
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					CreateNamespace: false,
				},
			},
			wantErr: true,
			errMsg:  errAllowedHubUpdate,
		},
		{
			name: "disallowed - Force change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Force: false,
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Force: true,
				},
			},
			wantErr: true,
			errMsg:  errAllowedHubUpdate,
		},
		{
			name: "disallowed - ClusterManager non-source change",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					ClusterManager: &v1beta1.ClusterManager{
						FeatureGates: "AddonManagement=true",
					},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					ClusterManager: &v1beta1.ClusterManager{
						FeatureGates: "AddonManagement=true,ResourceCleanup=true",
					},
				},
			},
			wantErr: true,
			errMsg:  errAllowedHubUpdate,
		},
		{
			name: "multiple allowed changes",
			oldHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Timeout:      300,
					LogVerbosity: 0,
					ClusterManager: &v1beta1.ClusterManager{
						Source: v1beta1.OCMSource{Registry: "old-registry"},
					},
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: "csr"},
				},
			},
			newHub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Timeout:      600,
					LogVerbosity: 5,
					ClusterManager: &v1beta1.ClusterManager{
						Source: v1beta1.OCMSource{Registry: "new-registry"},
					},
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: "awsirsa"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := allowHubUpdate(tt.oldHub, tt.newHub)
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

func TestAllowSpokeUpdate(t *testing.T) {
	tests := []struct {
		name     string
		oldSpoke *v1beta1.Spoke
		newSpoke *v1beta1.Spoke
		wantErr  bool
		errMsg   string
	}{
		{
			name: "no changes",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					CreateNamespace: true,
					Timeout:         300,
					LogVerbosity:    0,
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					CreateNamespace: true,
					Timeout:         300,
					LogVerbosity:    0,
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - klusterlet annotations change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						Annotations: map[string]string{"old": "value"},
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						Annotations: map[string]string{"new": "value"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - kubeconfig change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Kubeconfig: v1beta1.Kubeconfig{
						InCluster: true,
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Kubeconfig: v1beta1.Kubeconfig{
						SecretReference: &v1beta1.SecretReference{
							Name:          "new-secret",
							KubeconfigKey: "kubeconfig",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - addOns change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					AddOns: []v1beta1.AddOn{
						{ConfigName: "old-addon"},
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					AddOns: []v1beta1.AddOn{
						{ConfigName: "new-addon"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - timeout change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Timeout: 300,
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Timeout: 600,
				},
			},
			wantErr: false,
		},
		{
			name: "allowed - logVerbosity change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					LogVerbosity: 0,
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					LogVerbosity: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "disallowed - HubRef change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					HubRef: v1beta1.HubRef{
						Name:      "old-hub",
						Namespace: "default",
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					HubRef: v1beta1.HubRef{
						Name:      "new-hub",
						Namespace: "default",
					},
				},
			},
			wantErr: true,
			errMsg:  errAllowedSpokeUpdate,
		},
		{
			name: "disallowed - CreateNamespace change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					CreateNamespace: true,
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					CreateNamespace: false,
				},
			},
			wantErr: true,
			errMsg:  errAllowedSpokeUpdate,
		},
		{
			name: "disallowed - klusterlet mode change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						Mode: "Default",
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						Mode: "Hosted",
					},
				},
			},
			wantErr: true,
			errMsg:  errAllowedSpokeUpdate,
		},
		{
			name: "disallowed - klusterlet feature gates change",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						FeatureGates: "AddonManagement=true",
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Klusterlet: v1beta1.Klusterlet{
						FeatureGates: "AddonManagement=true,ClusterClaim=true",
					},
				},
			},
			wantErr: true,
			errMsg:  errAllowedSpokeUpdate,
		},
		{
			name: "multiple allowed changes",
			oldSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Timeout:      300,
					LogVerbosity: 0,
					Kubeconfig: v1beta1.Kubeconfig{
						InCluster: true,
					},
					Klusterlet: v1beta1.Klusterlet{
						Annotations: map[string]string{"old": "value"},
					},
					AddOns: []v1beta1.AddOn{
						{ConfigName: "old-addon"},
					},
				},
			},
			newSpoke: &v1beta1.Spoke{
				Spec: v1beta1.SpokeSpec{
					Timeout:      600,
					LogVerbosity: 5,
					Kubeconfig: v1beta1.Kubeconfig{
						SecretReference: &v1beta1.SecretReference{
							Name:          "new-secret",
							KubeconfigKey: "kubeconfig",
						},
					},
					Klusterlet: v1beta1.Klusterlet{
						Annotations: map[string]string{"new": "value"},
					},
					AddOns: []v1beta1.AddOn{
						{ConfigName: "new-addon"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := allowSpokeUpdate(tt.oldSpoke, tt.newSpoke)
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

func TestValidateAddonUniqueness(t *testing.T) {
	tests := []struct {
		name     string
		hub      *v1beta1.Hub
		wantErrs int
		errMsgs  []string
	}{
		{
			name: "valid - no addons",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{},
			},
			wantErrs: 0,
		},
		{
			name: "valid - unique AddOnConfigs with different names",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon2", Version: "v1.0.0"},
						{Name: "addon3", Version: "v2.0.0"},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid - same AddOnConfig name with different versions",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon1", Version: "v2.0.0"},
						{Name: "addon1", Version: "v3.0.0"},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid - unique HubAddOns",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
						{Name: "governance-policy-framework"},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "valid - no name conflicts between HubAddOns and AddOnConfigs",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon2", Version: "v2.0.0"},
					},
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
						{Name: "governance-policy-framework"},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name: "invalid - duplicate AddOnConfig name-version pairs",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon2", Version: "v2.0.0"},
						{Name: "addon1", Version: "v1.0.0"}, // duplicate
					},
				},
			},
			wantErrs: 1,
			errMsgs:  []string{"duplicate addOnConfig addon1-v1.0.0 (name-version) found at indices"},
		},
		{
			name: "invalid - multiple duplicate AddOnConfig name-version pairs",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon2", Version: "v2.0.0"},
						{Name: "addon1", Version: "v1.0.0"}, // duplicate
						{Name: "addon2", Version: "v2.0.0"}, // duplicate
					},
				},
			},
			wantErrs: 2,
			errMsgs: []string{
				"duplicate addOnConfig addon1-v1.0.0 (name-version) found at indices",
				"duplicate addOnConfig addon2-v2.0.0 (name-version) found at indices",
			},
		},
		{
			name: "invalid - duplicate HubAddOn names",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
						{Name: "governance-policy-framework"},
						{Name: "argocd"}, // duplicate
					},
				},
			},
			wantErrs: 1,
			errMsgs:  []string{"duplicate hubAddOn name argocd found at indices"},
		},
		{
			name: "invalid - multiple duplicate HubAddOn names",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
						{Name: "governance-policy-framework"},
						{Name: "argocd"},                      // duplicate
						{Name: "governance-policy-framework"}, // duplicate
					},
				},
			},
			wantErrs: 2,
			errMsgs: []string{
				"duplicate hubAddOn name argocd found at indices",
				"duplicate hubAddOn name governance-policy-framework found at indices",
			},
		},
		{
			name: "invalid - name conflict between HubAddOn and AddOnConfig",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "argocd", Version: "v1.0.0"},
						{Name: "addon2", Version: "v2.0.0"},
					},
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"}, // conflicts with AddOnConfig
						{Name: "governance-policy-framework"},
					},
				},
			},
			wantErrs: 1,
			errMsgs:  []string{"hubAddOn name argocd clashes with an existing addOnConfig name"},
		},
		{
			name: "invalid - unsupported HubAddOn",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{},
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "custom-addon"},
					},
				},
			},
			wantErrs: 1,
			errMsgs:  []string{"hubAddOn name argocd clashes with an existing addOnConfig name"},
		},
		{
			name: "invalid - all types of conflicts combined",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1", Version: "v1.0.0"},
						{Name: "addon2", Version: "v2.0.0"},
						{Name: "addon1", Version: "v1.0.0"}, // duplicate name-version
						{Name: "argocd", Version: "v1.0.0"},
					},
					HubAddOns: []v1beta1.HubAddOn{
						{Name: "argocd"},
						{Name: "argocd"}, // duplicate name, conflicts with AddOnConfig
					},
				},
			},
			wantErrs: 4,
			errMsgs: []string{
				"duplicate addOnConfig addon1-v1.0.0 (name-version) found at indices",
				"duplicate hubAddOn name argocd found at indices",
				"hubAddOn name shared-addon clashes with an existing addOnConfig name",
				"hubAddOn name shared-addon clashes with an existing addOnConfig name",
			},
		},
		{
			name: "edge case - empty version defaults",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					AddOnConfigs: []v1beta1.AddOnConfig{
						{Name: "addon1"}, // empty version
						{Name: "addon1"}, // empty version, should be duplicate
					},
				},
			},
			wantErrs: 1,
			errMsgs:  []string{"duplicate addOnConfig addon1- (name-version) found at indices"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validateAddonUniqueness(tt.hub)

			if len(errs) != tt.wantErrs {
				t.Errorf("validateAddonUniqueness() returned %d errors, want %d", len(errs), tt.wantErrs)
				for i, err := range errs {
					t.Errorf("  Error %d: %s", i, err.Error())
				}
				return
			}

			// Check that each expected error message is present
			for _, expectedMsg := range tt.errMsgs {
				found := false
				for _, err := range errs {
					if err.Error() != "" && err.Error() != expectedMsg {
						// For partial matching since error messages include indices
						if len(expectedMsg) > 0 && err.Error() != "" {
							// Check if the error message contains the expected substring
							if len(err.Error()) >= len(expectedMsg) {
								found = true
								break
							}
						}
					} else if err.Error() == expectedMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing %q not found in errors: %v", expectedMsg, errs)
				}
			}
		})
	}
}
