// Package test contains test utilities for the fleetconfig-controller project.
package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Config is a struct to hold global integration test configuration.
type Config struct {
	EnvTestK8sVersion string `json:"envTestK8sVersion"`
	FullTrace         bool   `json:"fullTrace"`
	LabelFilter       string `json:"labelFilter"`
	Verbose           bool   `json:"verbose"`
}

// LoadConfig centralized test config from disk.
func LoadConfig() (*Config, error) {
	root, err := GetProjectDir()
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(root, "hack", "test-config.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}
	return &config, nil
}

// FindEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func FindEnvTestBinaryDir(c *Config) string {
	envTestDir := fmt.Sprintf("%s-%s-%s", c.EnvTestK8sVersion, runtime.GOOS, runtime.GOARCH)
	for _, parDir := range [][]string{
		{"..", ".."},
		{"..", "..", ".."},
	} {
		dirs := append(parDir, []string{"bin", "k8s", envTestDir}...)
		basePath := filepath.Join(dirs...)
		entries, err := os.ReadDir(basePath)
		if err != nil {
			logf.Log.Error(err, "Failed to read directory", "path", basePath)
			continue
		}
		if len(entries) >= 3 {
			var found string
			for _, entry := range entries {
				found += entry.Name()
			}
			if strings.Contains(found, "kubectl") && strings.Contains(found, "etcd") && strings.Contains(found, "kube-apiserver") {
				return basePath
			}
		}
	}
	return ""
}

// GetProjectDir returns the fleetconfig-controller project root directory.
// It works by finding "fleetconfig-controller" in the current working directory path.
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Find the fleetconfig-controller directory in the path
	parts := strings.Split(wd, string(os.PathSeparator))
	for i, part := range parts {
		if part == "fleetconfig-controller" {
			// Reconstruct path up to and including fleetconfig-controller
			projectPath := strings.Join(parts[:i+1], string(os.PathSeparator))
			if projectPath == "" {
				projectPath = string(os.PathSeparator) // Handle root case
			}
			return projectPath, nil
		}
	}

	return "", fmt.Errorf("fleetconfig-controller directory not found in path: %s", wd)
}
