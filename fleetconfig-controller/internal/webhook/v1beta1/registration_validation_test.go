package v1beta1

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
)

// klusterletValuesTestClient implements client.Reader for Merge tests (Get ConfigMap only).
type klusterletValuesTestClient struct {
	configMaps map[string]*corev1.ConfigMap
}

func (c *klusterletValuesTestClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
	cmOut, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return fmt.Errorf("unexpected object type %T", obj)
	}
	if c.configMaps == nil {
		return kerrs.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, key.Name)
	}
	k := key.Namespace + "/" + key.Name
	src, ok := c.configMaps[k]
	if !ok {
		return kerrs.NewNotFound(schema.GroupResource{Group: "", Resource: "configmaps"}, key.Name)
	}
	*cmOut = *src.DeepCopy()
	return nil
}

func (c *klusterletValuesTestClient) List(ctx context.Context, list client.ObjectList, _ ...client.ListOption) error {
	return nil
}

func TestValidateHubRegistrationAuth(t *testing.T) {
	tests := []struct {
		name    string
		hub     *v1beta1.Hub
		wantLen int
	}{
		{
			name: "grpc driver with hostname grpc",
			hub: &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{Name: "h"},
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.GRPCRegistrationDriver,
						GRPC: &v1beta1.RegistrationAuthGRPC{
							EndpointType:           v1beta1.GRPCEndpointTypeHostname,
							Server:                 "hub-grpc:8090",
							AutoApprovedIdentities: []string{"system:serviceaccount:open-cluster-management-hub:grpc"},
						},
					},
				},
			},
			wantLen: 0,
		},
		{
			name: "awsirsa with grpc block",
			hub: &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{Name: "h"},
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.AWSIRSARegistrationDriver,
						GRPC:   &v1beta1.RegistrationAuthGRPC{Server: "x"},
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
	ctx := context.Background()

	spokeMeta := metav1.ObjectMeta{Namespace: "ns"}

	tests := []struct {
		name       string
		configMaps map[string]*corev1.ConfigMap
		spoke      *v1beta1.Spoke
		wantErrs   int
	}{
		{
			name: "inline grpcConfig denied",
			spoke: &v1beta1.Spoke{
				ObjectMeta: spokeMeta,
				Spec: v1beta1.SpokeSpec{
					SpokeSpecBase: v1beta1.SpokeSpecBase{
						Klusterlet: v1beta1.Klusterlet{
							Values: &v1beta1.KlusterletChartConfig{
								KlusterletChartConfig: chart.KlusterletChartConfig{GRPCConfig: "url: test"},
							},
						},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "valuesFrom supplies grpcConfig denied",
			configMaps: map[string]*corev1.ConfigMap{
				"ns/v": {
					ObjectMeta: metav1.ObjectMeta{Name: "v", Namespace: "ns"},
					Data:       map[string]string{"k": "grpcConfig: sneaky\n"},
				},
			},
			spoke: &v1beta1.Spoke{
				ObjectMeta: spokeMeta,
				Spec: v1beta1.SpokeSpec{
					SpokeSpecBase: v1beta1.SpokeSpecBase{
						Klusterlet: v1beta1.Klusterlet{ValuesFrom: &v1beta1.ConfigMapRef{Name: "v", Key: "k"}},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "valuesFrom without grpc allowed",
			configMaps: map[string]*corev1.ConfigMap{
				"ns/v": {
					ObjectMeta: metav1.ObjectMeta{Name: "v", Namespace: "ns"},
					Data:       map[string]string{"k": "enableSyncLabels: true\n"},
				},
			},
			spoke: &v1beta1.Spoke{
				ObjectMeta: spokeMeta,
				Spec: v1beta1.SpokeSpec{
					SpokeSpecBase: v1beta1.SpokeSpecBase{
						Klusterlet: v1beta1.Klusterlet{ValuesFrom: &v1beta1.ConfigMapRef{Name: "v", Key: "k"}},
					},
				},
			},
			wantErrs: 0,
		},
		{
			name:       "valuesFrom ConfigMap missing denied",
			configMaps: map[string]*corev1.ConfigMap{},
			spoke: &v1beta1.Spoke{
				ObjectMeta: spokeMeta,
				Spec: v1beta1.SpokeSpec{
					SpokeSpecBase: v1beta1.SpokeSpecBase{
						Klusterlet: v1beta1.Klusterlet{ValuesFrom: &v1beta1.ConfigMapRef{Name: "missing", Key: "k"}},
					},
				},
			},
			wantErrs: 1,
		},
		{
			name: "valuesFrom key missing denied",
			configMaps: map[string]*corev1.ConfigMap{
				"ns/v": {
					ObjectMeta: metav1.ObjectMeta{Name: "v", Namespace: "ns"},
					Data:       map[string]string{"other": "x: 1\n"},
				},
			},
			spoke: &v1beta1.Spoke{
				ObjectMeta: spokeMeta,
				Spec: v1beta1.SpokeSpec{
					SpokeSpecBase: v1beta1.SpokeSpecBase{
						Klusterlet: v1beta1.Klusterlet{ValuesFrom: &v1beta1.ConfigMapRef{Name: "v", Key: "k"}},
					},
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &klusterletValuesTestClient{configMaps: tt.configMaps}
			errs := validateSpokeRegistrationWithHub(ctx, cli, tt.spoke, hub)
			if len(errs) != tt.wantErrs {
				t.Fatalf("want %d errors, got %d: %v", tt.wantErrs, len(errs), errs)
			}
		})
	}
}
