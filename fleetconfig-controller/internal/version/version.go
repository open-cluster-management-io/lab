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
