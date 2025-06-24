// Package file contains file helpers
package file

import (
	"fmt"
	"os"
)

// TmpFile writes content to a temp fileand returns the tmp filepath and cleanup function
func TmpFile(content []byte, pattern string) (string, func(), error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = f.Close() }()
	cleanup := func() {
		_ = os.Remove(f.Name())
	}
	if _, err := f.Write(content); err != nil {
		return "", cleanup, fmt.Errorf("failed to write temp file: %w", err)
	}
	return f.Name(), cleanup, nil
}
