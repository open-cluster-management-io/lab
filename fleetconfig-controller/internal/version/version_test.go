package version

import (
	"context"
	"testing"
)

func TestLowestBundleVersion(t *testing.T) {
	tests := []struct {
		name        string
		bundleSpecs []string
		want        string
		wantErr     bool
	}{
		{
			name:        "no bundle specs",
			bundleSpecs: []string{},
			want:        "",
			wantErr:     true,
		},
		{
			name:        "invalid bundle spec",
			bundleSpecs: []string{"fleetconfig-controller:invalid"},
			want:        "",
			wantErr:     true,
		},
		{
			name:        "single bundle spec",
			bundleSpecs: []string{"fleetconfig-controller:v0.1.0"},
			want:        "0.1.0",
			wantErr:     false,
		},
		{
			name:        "multiple bundle specs",
			bundleSpecs: []string{"fleetconfig-controller:v0.1.0", "fleetconfig-controller:v0.2.0"},
			want:        "0.1.0",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LowestBundleVersion(context.Background(), tt.bundleSpecs)
			if (err != nil) != tt.wantErr {
				t.Errorf("LowestBundleVersion(%v) error = %v, wantErr %v", tt.bundleSpecs, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LowestBundleVersion(%v) = %v, want %v", tt.bundleSpecs, got, tt.want)
			}
		})
	}
}
