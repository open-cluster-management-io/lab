package v1beta1

import (
	"testing"

	operatorv1 "open-cluster-management.io/api/operator/v1"
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
