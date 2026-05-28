package common

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterfake "open-cluster-management.io/api/client/cluster/clientset/versioned/fake"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const (
	labelName = "fleetconfig.open-cluster-management.io/spoke-name"
	labelNs   = "fleetconfig.open-cluster-management.io/spoke-namespace"
)

func TestGetManagedCluster(t *testing.T) {
	tests := []struct {
		name          string
		fixtures      []*clusterv1.ManagedCluster
		fallbackNames []string
		ownerLabels   map[string]string
		wantName      string // "" means nil result
		wantErr       bool
	}{
		{
			name: "label hit returns labeled MC",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "foo-1234abcd",
					Labels: map[string]string{labelName: "foo", labelNs: "ns1"},
				}},
			},
			fallbackNames: []string{"foo"},
			ownerLabels:   map[string]string{labelName: "foo", labelNs: "ns1"},
			wantName:      "foo-1234abcd",
		},
		{
			name: "label miss falls back to name (legacy adoption)",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{Name: "legacy"}},
			},
			fallbackNames: []string{"legacy"},
			ownerLabels:   map[string]string{labelName: "legacy", labelNs: "ns"},
			wantName:      "legacy",
		},
		{
			name: "collision guard rejects MC owned by different Spoke",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{labelName: "foo", labelNs: "other-ns"},
				}},
			},
			fallbackNames: []string{"foo"},
			ownerLabels:   map[string]string{labelName: "foo", labelNs: "my-ns"},
			wantName:      "",
		},
		{
			name:          "not found returns nil",
			fixtures:      nil,
			fallbackNames: []string{"foo"},
			ownerLabels:   nil,
			wantName:      "",
		},
		{
			name: "fallback order honored: derived chosen before legacy",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{Name: "foo-abcd1234"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			},
			fallbackNames: []string{"foo-abcd1234", "foo"},
			ownerLabels:   nil,
			wantName:      "foo-abcd1234",
		},
		{
			name: "second fallback used when first missing",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			},
			fallbackNames: []string{"foo-abcd1234", "foo"},
			ownerLabels:   nil,
			wantName:      "foo",
		},
		{
			name: "nil ownerLabels skips label lookup, plain Get",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{Name: "plain"}},
			},
			fallbackNames: []string{"plain"},
			ownerLabels:   nil,
			wantName:      "plain",
		},
		{
			name: "multiple label matches returns error",
			fixtures: []*clusterv1.ManagedCluster{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "foo-1111",
					Labels: map[string]string{labelName: "foo", labelNs: "ns"},
				}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "foo-2222",
					Labels: map[string]string{labelName: "foo", labelNs: "ns"},
				}},
			},
			fallbackNames: []string{"foo"},
			ownerLabels:   map[string]string{labelName: "foo", labelNs: "ns"},
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			objs := make([]runtime.Object, 0, len(tc.fixtures))
			for _, mc := range tc.fixtures {
				objs = append(objs, mc)
			}
			client := clusterfake.NewSimpleClientset(objs...)

			got, err := GetManagedCluster(context.Background(), client, tc.fallbackNames, tc.ownerLabels)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			switch {
			case tc.wantName == "" && got != nil:
				t.Fatalf("want nil, got %q", got.Name)
			case tc.wantName != "" && got == nil:
				t.Fatalf("want %q, got nil", tc.wantName)
			case tc.wantName != "" && got.Name != tc.wantName:
				t.Fatalf("want %q, got %q", tc.wantName, got.Name)
			}
		})
	}
}
