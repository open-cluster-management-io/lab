// Package version contains helpers for working with versions
package version

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/semver"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// LowestBundleVersion finds the lowest semantic version from a list of bundle specifications.
func LowestBundleVersion(ctx context.Context, bundleSpecs []string) (string, error) {
	logger := log.FromContext(ctx)

	semvers := make([]*semver.Version, 0)
	for _, bundleSpec := range bundleSpecs {
		specParts := strings.Split(bundleSpec, ":")
		if len(specParts) != 2 {
			logger.V(0).Info("invalid bundleSpec", "bundleSpec", bundleSpec)
			continue
		}
		version, err := semver.NewVersion(specParts[1])
		if err != nil {
			logger.V(0).Info("invalid bundleSpec version", "version", specParts[1])
			continue
		}
		semvers = append(semvers, version)
	}
	if len(semvers) == 0 {
		return "", fmt.Errorf("no valid bundle versions detected")
	}

	slices.SortFunc(semvers, func(a, b *semver.Version) int {
		if a.LessThan(b) {
			return -1
		} else if a.GreaterThan(b) {
			return 1
		}
		return 0
	})

	return semvers[0].String(), nil
}

// Normalize returns a semver string with the leading `v` prefix stripped off
func Normalize(v string) (string, error) {
	sv, err := semver.NewVersion(v)
	if err != nil {
		return "", err
	}
	return sv.String(), nil
}

// GetBundleSource returns the bundle source, or an error if they do not match.
func GetBundleSource(bundleSpecs []string) (string, error) {
	seen := make(map[string]struct{})
	for _, spec := range bundleSpecs {
		var imageSpec string

		// Check if using digest
		if atIdx := strings.Index(spec, "@"); atIdx != -1 {
			imageSpec = spec[:atIdx]
		} else {
			// Split on the last colon to handle registry URLs with ports (e.g., registry.io:5000/image:tag)
			lastColonIdx := strings.LastIndex(spec, ":")
			if lastColonIdx == -1 {
				return "", fmt.Errorf("invalid image spec %s: missing version tag or digest", spec)
			}
			imageSpec = spec[:lastColonIdx]
		}

		parts := strings.Split(imageSpec, "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid image spec %s: missing repository path", spec)
		}
		// Extract repository source (everything except the last part which is the image name)
		source := strings.Join(parts[:len(parts)-1], "/")
		seen[source] = struct{}{}
	}

	if len(seen) == 0 {
		return "", fmt.Errorf("no bundle specs provided")
	}

	if len(seen) > 1 {
		sources := make([]string, 0, len(seen))
		for s := range seen {
			sources = append(sources, s)
		}
		return "", fmt.Errorf("bundle specs have mismatched sources: %v", sources)
	}

	// Return the single source
	for source := range seen {
		return source, nil
	}

	return "", fmt.Errorf("unexpected error: no source found")
}
