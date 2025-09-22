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
			errMsg:  "only changes to spec.apiServer, spec.clusterManager.source.*, spec.hubAddOns, spec.addOnConfigs, spec.logVerbosity, spec.timeout, and spec.registrationAuth are allowed when updating the hub",
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
			errMsg:  "only changes to spec.apiServer, spec.clusterManager.source.*, spec.hubAddOns, spec.addOnConfigs, spec.logVerbosity, spec.timeout, and spec.registrationAuth are allowed when updating the hub",
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
			errMsg:  "only changes to spec.apiServer, spec.clusterManager.source.*, spec.hubAddOns, spec.addOnConfigs, spec.logVerbosity, spec.timeout, and spec.registrationAuth are allowed when updating the hub",
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
			errMsg:  "spoke contains changes which are not allowed; only changes to spec.klusterlet.annotations, spec.klusterlet.values, spec.kubeconfig, spec.addOns, spec.timeout, and spec.logVerbosity are allowed when updating a spoke",
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
			errMsg:  "spoke contains changes which are not allowed; only changes to spec.klusterlet.annotations, spec.klusterlet.values, spec.kubeconfig, spec.addOns, spec.timeout, and spec.logVerbosity are allowed when updating a spoke",
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
			errMsg:  "spoke contains changes which are not allowed; only changes to spec.klusterlet.annotations, spec.klusterlet.values, spec.kubeconfig, spec.addOns, spec.timeout, and spec.logVerbosity are allowed when updating a spoke",
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
			errMsg:  "spoke contains changes which are not allowed; only changes to spec.klusterlet.annotations, spec.klusterlet.values, spec.kubeconfig, spec.addOns, spec.timeout, and spec.logVerbosity are allowed when updating a spoke",
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
