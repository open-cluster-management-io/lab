// Package hash provides hashing utilities.
package hash

import (
	"strconv"

	"github.com/mitchellh/hashstructure/v2"
)

// ComputeHash computes the hash value of an arbitrary object
func ComputeHash(obj any) (string, error) {
	// compute a hash value of any object
	hash, err := hashstructure.Hash(obj, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	hashStr := strconv.FormatUint(hash, 16)
	return hashStr, nil
}
