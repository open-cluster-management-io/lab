// Package hash provides hashing utilities.
//
//nolint:revive // package name matches directory internal/hash; avoids churn vs stdlib "crypto/hash"
package hash

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
)

// ComputeHash computes the hash value of an arbitrary object
func ComputeHash(obj any) (string, error) {
	opts := &hashstructure.HashOptions{
		ZeroNil: true,
	}
	// compute a hash value of any object
	hash, err := hashstructure.Hash(obj, hashstructure.FormatV2, opts)
	if err != nil {
		return "", err
	}
	hashStr := strconv.FormatUint(hash, 16)
	return hashStr, nil
}
