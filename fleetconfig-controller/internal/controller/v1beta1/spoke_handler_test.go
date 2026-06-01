package v1beta1

import (
	"context"
	"os"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

func TestSyncManagedClusterAnnotations(t *testing.T) {
	const prefix = operatorv1.ClusterAnnotationsKeyPrefix + "/"

	tests := []struct {
		name      string
		current   map[string]string
		requested map[string]string
		want      map[string]string
	}{
		{
			name: "preserve non-klusterlet annotations",
			current: map[string]string{
				"other.io/annotation":     "keep-me",
				"another.io/annotation":   "also-keep",
				prefix + "klusterlet-key": "old-value",
			},
			requested: map[string]string{
				prefix + "klusterlet-key": "new-value",
			},
			want: map[string]string{
				"other.io/annotation":     "keep-me",
				"another.io/annotation":   "also-keep",
				prefix + "klusterlet-key": "new-value",
			},
		},
		{
			name: "add new klusterlet annotations",
			current: map[string]string{
				"other.io/annotation":   "keep-me",
				prefix + "existing-key": "existing-value",
			},
			requested: map[string]string{
				prefix + "existing-key": "existing-value",
				prefix + "new-key":      "new-value",
			},
			want: map[string]string{
				"other.io/annotation":   "keep-me",
				prefix + "existing-key": "existing-value",
				prefix + "new-key":      "new-value",
			},
		},
		{
			name: "update existing klusterlet annotations",
			current: map[string]string{
				"other.io/annotation": "keep-me",
				prefix + "key1":       "old-value-1",
				prefix + "key2":       "old-value-2",
			},
			requested: map[string]string{
				prefix + "key1": "new-value-1",
				prefix + "key2": "new-value-2",
			},
			want: map[string]string{
				"other.io/annotation": "keep-me",
				prefix + "key1":       "new-value-1",
				prefix + "key2":       "new-value-2",
			},
		},
		{
			name: "remove klusterlet annotations that are no longer requested",
			current: map[string]string{
				"other.io/annotation":  "keep-me",
				prefix + "keep-key":    "keep-value",
				prefix + "remove-key1": "will-be-removed",
				prefix + "remove-key2": "also-removed",
			},
			requested: map[string]string{
				prefix + "keep-key": "keep-value",
			},
			want: map[string]string{
				"other.io/annotation": "keep-me",
				prefix + "keep-key":   "keep-value",
			},
		},
		{
			name:    "handle nil current annotations",
			current: nil,
			requested: map[string]string{
				prefix + "new-key": "new-value",
			},
			want: map[string]string{
				prefix + "new-key": "new-value",
			},
		},
		{
			name:    "handle empty current annotations",
			current: map[string]string{},
			requested: map[string]string{
				prefix + "new-key": "new-value",
			},
			want: map[string]string{
				prefix + "new-key": "new-value",
			},
		},
		{
			name: "handle empty requested annotations",
			current: map[string]string{
				"other.io/annotation":  "keep-me",
				prefix + "remove-key1": "will-be-removed",
				prefix + "remove-key2": "also-removed",
			},
			requested: map[string]string{},
			want: map[string]string{
				"other.io/annotation": "keep-me",
			},
		},
		{
			name: "complex scenario with add, update, remove, and preserve",
			current: map[string]string{
				"other.io/annotation":       "keep-me",
				"third-party.io/annotation": "also-keep",
				prefix + "update-me":        "old-value",
				prefix + "keep-me":          "keep-value",
				prefix + "remove-me":        "will-be-removed",
			},
			requested: map[string]string{
				prefix + "update-me": "new-value",
				prefix + "keep-me":   "keep-value",
				prefix + "add-me":    "new-annotation",
			},
			want: map[string]string{
				"other.io/annotation":       "keep-me",
				"third-party.io/annotation": "also-keep",
				prefix + "update-me":        "new-value",
				prefix + "keep-me":          "keep-value",
				prefix + "add-me":           "new-annotation",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syncManagedClusterAnnotations(tt.current, tt.requested)

			if len(got) != len(tt.want) {
				t.Errorf("syncManagedClusterAnnotations() returned %d annotations, want %d", len(got), len(tt.want))
			}

			for key, wantValue := range tt.want {
				gotValue, ok := got[key]
				if !ok {
					t.Errorf("syncManagedClusterAnnotations() missing key %q", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("syncManagedClusterAnnotations() key %q = %q, want %q", key, gotValue, wantValue)
				}
			}

			for key := range got {
				if _, ok := tt.want[key]; !ok {
					t.Errorf("syncManagedClusterAnnotations() has unexpected key %q with value %q", key, got[key])
				}
			}
		})
	}
}

// TestAppendJoinSpokeTransportAndAuthArgs_CleanupLifecycle verifies that temp files
// referenced by --ca-file, --grpc-ca-file, and --proxy-ca-file exist when the helper
// returns, and are deleted once the returned cleanup func runs.
func TestAppendJoinSpokeTransportAndAuthArgs_CleanupLifecycle(t *testing.T) {
	tests := []struct {
		name      string
		hub       *v1beta1.Hub
		spoke     *v1beta1.Spoke
		wantFlags []string
	}{
		{
			name: "hub CA only",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Ca:               "fake-hub-ca",
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: v1alpha1.CSRRegistrationDriver},
				},
			},
			spoke:     &v1beta1.Spoke{ObjectMeta: metav1.ObjectMeta{Name: "spoke", Namespace: "ns"}},
			wantFlags: []string{"--ca-file="},
		},
		{
			name: "proxy CA only",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: v1alpha1.CSRRegistrationDriver},
				},
			},
			spoke: &v1beta1.Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: "spoke", Namespace: "ns"},
				Spec:       v1beta1.SpokeSpec{SpokeSpecBase: v1beta1.SpokeSpecBase{ProxyCa: "fake-proxy-ca"}},
			},
			wantFlags: []string{"--proxy-ca-file="},
		},
		{
			name: "hub CA + proxy CA",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					Ca:               "fake-hub-ca",
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: v1alpha1.CSRRegistrationDriver},
				},
			},
			spoke: &v1beta1.Spoke{
				ObjectMeta: metav1.ObjectMeta{Name: "spoke", Namespace: "ns"},
				Spec:       v1beta1.SpokeSpec{SpokeSpecBase: v1beta1.SpokeSpecBase{ProxyCa: "fake-proxy-ca"}},
			},
			wantFlags: []string{"--ca-file=", "--proxy-ca-file="},
		},
		{
			name: "gRPC registration driver",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{
						Driver: v1beta1.GRPCRegistrationDriver,
						GRPC: &v1beta1.RegistrationAuthGRPC{
							EndpointType: v1beta1.GRPCEndpointTypeHostname,
							Server:       "grpc.example.com:8090",
						},
					},
				},
				Status: v1beta1.HubStatus{GRPCServerCA: "fake-grpc-ca"},
			},
			spoke:     &v1beta1.Spoke{ObjectMeta: metav1.ObjectMeta{Name: "spoke", Namespace: "ns"}},
			wantFlags: []string{"--grpc-ca-file="},
		},
		{
			name: "no temp files when nothing to write",
			hub: &v1beta1.Hub{
				Spec: v1beta1.HubSpec{
					RegistrationAuth: v1beta1.RegistrationAuth{Driver: v1alpha1.CSRRegistrationDriver},
				},
			},
			spoke:     &v1beta1.Spoke{ObjectMeta: metav1.ObjectMeta{Name: "spoke", Namespace: "ns"}},
			wantFlags: nil,
		},
	}

	r := &SpokeReconciler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			joinArgs, cleanup, err := r.appendJoinSpokeTransportAndAuthArgs(context.Background(), nil, tt.spoke, tt.hub, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cleanup == nil {
				t.Fatal("expected non-nil cleanup func")
			}

			paths := map[string]string{}
			for _, arg := range joinArgs {
				for _, flag := range tt.wantFlags {
					if path, ok := strings.CutPrefix(arg, flag); ok {
						paths[flag] = path
					}
				}
			}
			if len(paths) != len(tt.wantFlags) {
				t.Fatalf("want %d temp file flags %v, got %d: %v", len(tt.wantFlags), tt.wantFlags, len(paths), paths)
			}

			for flag, path := range paths {
				if _, err := os.Stat(path); err != nil {
					t.Errorf("temp file for %s (%s) should exist before cleanup: %v", flag, path, err)
				}
			}

			cleanup()

			for flag, path := range paths {
				_, err := os.Stat(path)
				if err == nil {
					t.Errorf("temp file for %s (%s) should be deleted after cleanup but still exists", flag, path)
				} else if !os.IsNotExist(err) {
					t.Errorf("unexpected stat error for %s (%s) after cleanup: %v", flag, path, err)
				}
			}
		})
	}
}
