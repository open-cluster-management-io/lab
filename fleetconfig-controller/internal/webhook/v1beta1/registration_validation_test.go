package v1beta1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

func TestValidateHubRegistrationAuth(t *testing.T) {
	tests := []struct {
		name    string
		hub     *v1beta1.Hub
		wantLen int
	}{
		{
			name: "grpc driver with full grpc",
			hub: &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{Name: "h"},
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.GRPCRegistrationDriver,
						GRPC: &v1beta1.RegistrationAuthGRPC{
							Init: &v1beta1.RegistrationAuthGRPCInit{
								EndpointType:           v1beta1.GRPCEndpointTypeHostname,
								HubServer:              "hub-grpc:8090",
								AutoApprovedIdentities: []string{"system:serviceaccount:open-cluster-management-hub:grpc"},
							},
							Join: &v1beta1.RegistrationAuthGRPCJoin{
								JoinServer:           "grpc.example:8090",
								CertificateAuthority: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
							},
						},
					},
				},
			},
			wantLen: 0,
		},
		{
			name: "grpc driver missing join server",
			hub: &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{Name: "h"},
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.GRPCRegistrationDriver,
						GRPC: &v1beta1.RegistrationAuthGRPC{
							Join: &v1beta1.RegistrationAuthGRPCJoin{
								JoinServer:           "",
								CertificateAuthority: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
							},
						},
					},
				},
			},
			wantLen: 1,
		},
		{
			name: "awsirsa with grpc block",
			hub: &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{Name: "h"},
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.AWSIRSARegistrationDriver,
						GRPC:   &v1beta1.RegistrationAuthGRPC{Init: &v1beta1.RegistrationAuthGRPCInit{HubServer: "x"}},
					},
				},
			},
			wantLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateHubRegistrationAuth(tt.hub)
			if len(got) != tt.wantLen {
				t.Fatalf("want %d errors, got %d: %v", tt.wantLen, len(got), got)
			}
		})
	}
}

func TestValidateSpokeRegistrationWithHub(t *testing.T) {
	hub := &v1beta1.Hub{
		Spec: v1beta1.HubSpec{RegistrationAuth: v1beta1.RegistrationAuth{Driver: v1beta1.GRPCRegistrationDriver}},
	}
	spoke := &v1beta1.Spoke{
		Spec: v1beta1.SpokeSpec{
			Klusterlet: v1beta1.Klusterlet{
				Values: &v1beta1.KlusterletChartConfig{},
			},
		},
	}
	spoke.Spec.Klusterlet.Values.GRPCConfig = "url: test"
	errs := validateSpokeRegistrationWithHub(spoke, hub)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
}
