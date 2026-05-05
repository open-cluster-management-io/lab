package klusterletvalues

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"dario.cat/mergo"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// ErrValuesFromKeyMissing is returned when StrictValuesFrom is set and the ConfigMap does not contain valuesFrom.key.
var ErrValuesFromKeyMissing = errors.New("klusterlet valuesFrom key missing from ConfigMap")

// MergeOptions controls how ValuesFrom references are resolved.
type MergeOptions struct {
	// StrictValuesFrom requires the ValuesFrom ConfigMap and data key to exist.
	// When false (default), a missing ConfigMap or key falls back to spec-only values (reconciler behavior).
	StrictValuesFrom bool
}

// Merge merges klusterlet chart values from an optional ConfigMap (ValuesFrom) and inline spec (Values).
// Spec values override ConfigMap values (non-zero fields from spec win).
func Merge(ctx context.Context, c client.Reader, spokeNamespace string, kl v1beta1.Klusterlet, opts MergeOptions) (*v1beta1.KlusterletChartConfig, error) {
	if kl.ValuesFrom == nil && kl.Values == nil {
		return nil, nil
	}

	var fromInterface = map[string]any{}
	var specInterface = map[string]any{}

	if kl.ValuesFrom != nil {
		cm := &corev1.ConfigMap{}
		nn := types.NamespacedName{Name: kl.ValuesFrom.Name, Namespace: spokeNamespace}
		err := c.Get(ctx, nn, cm)
		if err != nil {
			if kerrs.IsNotFound(err) {
				if opts.StrictValuesFrom {
					return nil, fmt.Errorf("klusterlet valuesFrom ConfigMap %s: %w", nn, err)
				}
				return kl.Values, nil
			}
			return nil, fmt.Errorf("failed to retrieve Klusterlet values ConfigMap %s: %w", nn, err)
		}
		fromValues, ok := cm.Data[kl.ValuesFrom.Key]
		if !ok {
			if opts.StrictValuesFrom {
				return nil, fmt.Errorf("%w: key %q missing in ConfigMap %s", ErrValuesFromKeyMissing, kl.ValuesFrom.Key, nn)
			}
			return kl.Values, nil
		}
		fromBytes := []byte(fromValues)
		err = yaml.Unmarshal(fromBytes, &fromInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML values from ConfigMap %s key %s: %w", nn, kl.ValuesFrom.Key, err)
		}
	}

	if kl.Values != nil {
		specBytes, err := yaml.Marshal(kl.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Klusterlet values from spoke spec: %w", err)
		}
		err = yaml.Unmarshal(specBytes, &specInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Klusterlet values from spoke spec: %w", err)
		}
	}

	mergedMap := map[string]any{}
	maps.Copy(mergedMap, fromInterface)

	if err := mergo.Map(&mergedMap, specInterface, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merge failed for klusterlet values: %w", err)
	}

	mergedBytes, err := yaml.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged Klusterlet values: %w", err)
	}

	merged := &v1beta1.KlusterletChartConfig{}
	err = yaml.Unmarshal(mergedBytes, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged values into KlusterletChartConfig: %w", err)
	}

	return merged, nil
}
