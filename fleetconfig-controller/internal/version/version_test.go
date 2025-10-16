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

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
		wantErr bool
	}{
		{
			name:    "valid version with v prefix",
			version: "v1.2.3",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "valid version without v prefix",
			version: "1.2.3",
			want:    "1.2.3",
			wantErr: false,
		},
		{
			name:    "invalid version string",
			version: "invalid-version",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize(%v) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Normalize(%v) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestGetBundleSource(t *testing.T) {
	tests := []struct {
		name        string
		bundleSpecs []string
		want        string
		wantErr     bool
	}{
		{
			name: "matching sources",
			bundleSpecs: []string{
				"quay.io/open-cluster-management/registration:v1.0.0",
				"quay.io/open-cluster-management/work:v1.0.0",
				"quay.io/open-cluster-management/placement:v1.0.0",
			},
			want:    "quay.io/open-cluster-management",
			wantErr: false,
		},
		{
			name: "matching sources with ported registry",
			bundleSpecs: []string{
				"registry.io:5000/ocm/registration:v1.0.0",
				"registry.io:5000/ocm/work:v1.0.0",
				"registry.io:5000/ocm/placement:v1.0.0",
			},
			want:    "registry.io:5000/ocm",
			wantErr: false,
		},
		{
			name: "matching sources with sha256 digest",
			bundleSpecs: []string{
				"registry.io:5000/ocm/registration@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				"registry.io:5000/ocm/work@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				"registry.io:5000/ocm/placement@sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
			},
			want:    "registry.io:5000/ocm",
			wantErr: false,
		},
		{
			name: "mismatched sources",
			bundleSpecs: []string{
				"quay.io/foo/bar:v1.0.0",
				"quay.io/baz/qux:v1.0.0",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid spec format",
			bundleSpecs: []string{
				"invalid-spec-without-colon",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBundleSource(tt.bundleSpecs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBundleSource(%v) error = %v, wantErr %v", tt.bundleSpecs, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBundleSource(%v) = %v, want %v", tt.bundleSpecs, got, tt.want)
			}
		})
	}
}
