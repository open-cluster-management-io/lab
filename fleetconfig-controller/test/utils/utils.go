/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:golint,revive
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

const (
	kindImage = "kindest/node:v1.31.9"
)

var (
	HubClusterName   string
	SpokeClusterName string
)

func init() {
	r := rand.New(rand.NewSource(time.Now().UnixNano())) //#nosec G404
	intHash := r.Intn(900000) + 100000
	SpokeClusterName = fmt.Sprintf("kind-spoke-%d", intHash)
	HubClusterName = fmt.Sprintf("kind-hub-%d", intHash)
}

// NewClient creates a new client for the given kubeconfig and scheme
func NewClient(kubeconfig string, scheme *runtime.Scheme) (client.Client, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return client.New(restConfig, client.Options{Scheme: scheme})
}

// Info prints an info message to the GinkgoWriter
func Info(format string, a ...any) {
	_, _ = fmt.Fprintf(GinkgoWriter, "info: "+format, a...)
	_, _ = fmt.Fprint(GinkgoWriter, "\n")
}

// WarnError prints a warning message to the GinkgoWriter
func WarnError(err error, format string, a ...any) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v: "+format, append([]any{err}, a...))
	_, _ = fmt.Fprint(GinkgoWriter, "\n")
}

// DevspaceRunPipeline runs a devspace pipeline
func DevspaceRunPipeline(ctx context.Context, kubeconfig, pipeline, namespace, profile string) error {
	projDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %v", err)
	}

	cmd := exec.Command(
		"devspace", "run-pipeline", pipeline,
		"--kubeconfig", kubeconfig,
		"--namespace", namespace,
		"--profile", profile,
		"--no-warn", "--force-build",
		// "--debug",
	)
	cmd.Dir = projDir

	Info("Running devspace pipeline: %s", cmd.String())
	_, err = RunCommand(cmd, "", false)
	return err
}

// DevspacePurge purges all resources created by devspace
func DevspacePurge(ctx context.Context, kubeconfig, namespace string) error {
	projDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %v", err)
	}

	cmd := exec.Command(
		"devspace", "purge",
		"--kubeconfig", kubeconfig,
		"--namespace", namespace,
		"--no-warn",
		// "--debug",
	)
	cmd.Dir = projDir

	Info("Running devspace purge: %s", cmd.String())
	_, err = RunCommand(cmd, "", false)
	return err
}

// RunCommand executes the provided command with various options.
// - If returnOutput is true, stdout is returned; otherwise, it's piped to GinkgoWriter.
// - If subdir is not empty, the command is executed in that subdirectory of the project.
// Stderr is always piped to the GinkgoWriter.
func RunCommand(cmd *exec.Cmd, subdir string, returnOutput bool) ([]byte, error) {
	// Set up directory if needed
	var projDir string
	if subdir != "" {
		var err error
		projDir, err = GetProjectDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get project directory: %v", err)
		}

		dir := projDir
		if subdir != "" {
			dir = fmt.Sprintf("%s/%s", projDir, subdir)
		}
		cmd.Dir = dir

		if err := os.Chdir(cmd.Dir); err != nil {
			_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
		}
		defer func() {
			if err := os.Chdir(projDir); err != nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %s\n", err)
			}
		}()
	}

	// Set environment
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	// Log the command
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)

	if !returnOutput {
		// Simple case: pipe everything to GinkgoWriter
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		return nil, cmd.Run()
	}

	// Case where we need to return stdout
	// Create a pipe for stderr
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Create a pipe for stdout to capture output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	// Copy stderr to GinkgoWriter in a goroutine
	go func() {
		if _, err = io.Copy(GinkgoWriter, stderrPipe); err != nil {
			WarnError(err, "failed to copy stderr to GinkgoWriter")
		}
	}()

	// Read stdout to return as output
	output, err := io.ReadAll(stdoutPipe)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %v", err)
	}

	// Wait for the command to complete
	waitErr := cmd.Wait()
	if waitErr != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, waitErr, string(output))
	}

	return output, nil
}

// CreateKindCluster creates a kind cluster
func CreateKindCluster(name, kubeconfig string) error {
	kindOptions := []string{
		"create", "cluster",
		"--name", name,
		"--kubeconfig", kubeconfig,
		"--image", kindImage,
	}
	cmd := exec.Command("kind", kindOptions...)
	_, err := RunCommand(cmd, "", false)
	return err
}

// DeleteKindCluster deletes a kind cluster
func DeleteKindCluster(name string) error {
	kindOptions := []string{"delete", "cluster", "--name", name}
	cmd := exec.Command("kind", kindOptions...)
	_, err := RunCommand(cmd, "", false)
	return err
}

// GetProjectDir returns the project root directory
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// AssertConditions asserts that a resource's conditions match a set of expected conditions.
func AssertConditions(conditions []metav1.Condition, expected map[string]metav1.ConditionStatus) error {
	Info("Asserting conditions")
	Info("Got: %v", conditions)
	Info("Expected: %v", expected)
	if len(conditions) != len(expected) {
		return fmt.Errorf("expected %d conditions, got %d", len(expected), len(conditions))
	}
	for _, c := range conditions {
		expectedCondition, ok := expected[c.Type]
		if !ok {
			return fmt.Errorf("unhandled condition %s", c.Type)
		}
		if c.Status != expectedCondition {
			return fmt.Errorf("condition %s has status %s, expected %s", c.Type, c.Status, expectedCondition)
		}
	}
	return nil
}

// GetSupportBundle gets a support bundle from a kind cluster.
func GetSupportBundle(ctx context.Context, kubeconfig, bundleName string) error {
	projDir, err := GetProjectDir()
	if err != nil {
		return fmt.Errorf("failed to get project directory: %v", err)
	}
	supportBundlePath := fmt.Sprintf("%s/bin/support-bundle", projDir)
	cmd := exec.Command(supportBundlePath,
		"--debug",
		"--kubeconfig", kubeconfig,
		"--output", fmt.Sprintf("%s-bundle.tar.gz", bundleName),
		fmt.Sprintf("%s/test/utils/resources/support-bundle.yaml", projDir),
	)
	_, err = RunCommand(cmd, "", false)
	return err
}

// GetFleetConfig gets a FleetConfig
func GetFleetConfig(ctx context.Context, kClient client.Client, nn ktypes.NamespacedName) (*v1alpha1.FleetConfig, error) {
	fc := &v1alpha1.FleetConfig{}
	return fc, kClient.Get(ctx, nn, fc)
}

// PatchFleetConfig patches a FleetConfig
func PatchFleetConfig(ctx context.Context, kClient client.Client, original *v1alpha1.FleetConfig, patch *v1alpha1.FleetConfig) error {
	patchObject := client.MergeFrom(original)
	return kClient.Patch(ctx, patch, patchObject)
}

// UpdateFleetConfigFeatureGates updates a FleetConfig's feature gates
func UpdateFleetConfigFeatureGates(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig, featureGates string) error {
	if fc.Spec.Hub.ClusterManager == nil {
		return fmt.Errorf("ClusterManager is nil")
	}

	original := fc.DeepCopy()
	fc.Spec.Hub.ClusterManager.FeatureGates = featureGates

	return PatchFleetConfig(ctx, kClient, original, fc)
}

// GetHub gets a Hub
func GetHub(ctx context.Context, kClient client.Client, nn ktypes.NamespacedName) (*v1beta1.Hub, error) {
	hub := &v1beta1.Hub{}
	return hub, kClient.Get(ctx, nn, hub)
}

// PatchHub patches a Hub
func PatchHub(ctx context.Context, kClient client.Client, original *v1beta1.Hub, patch *v1beta1.Hub) error {
	patchObject := client.MergeFrom(original)
	return kClient.Patch(ctx, patch, patchObject)
}

// UpdateHubFeatureGates updates a Hub's feature gates
func UpdateHubFeatureGates(ctx context.Context, kClient client.Client, hub *v1beta1.Hub, featureGates string) error {
	if hub.Spec.ClusterManager == nil {
		return fmt.Errorf("ClusterManager is nil")
	}

	original := hub.DeepCopy()
	hub.Spec.ClusterManager.FeatureGates = featureGates

	return PatchHub(ctx, kClient, original, hub)
}

// GetSpoke gets a Spoke
func GetSpoke(ctx context.Context, kClient client.Client, nn ktypes.NamespacedName) (*v1beta1.Spoke, error) {
	spoke := &v1beta1.Spoke{}
	return spoke, kClient.Get(ctx, nn, spoke)
}

// PatchSpoke patches a Spoke
func PatchSpoke(ctx context.Context, kClient client.Client, original *v1beta1.Spoke, patch *v1beta1.Spoke) error {
	patchObject := client.MergeFrom(original)
	return kClient.Patch(ctx, patch, patchObject)
}

// CloneFleetConfig clones a FleetConfig
func CloneFleetConfig(fc *v1alpha1.FleetConfig, dest *v1alpha1.FleetConfig) error {
	*dest = v1alpha1.FleetConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fc.Name,
			Namespace: fc.Namespace,
		},
		Spec: *fc.Spec.DeepCopy(),
	}
	return nil
}

// CloneHub clones a Hub
func CloneHub(hub *v1beta1.Hub, dest *v1beta1.Hub) error {
	*dest = v1beta1.Hub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hub.Name,
			Namespace: hub.Namespace,
		},
		Spec: *hub.Spec.DeepCopy(),
	}
	return nil
}

// CloneSpoke clones a Spoke
func CloneSpoke(spoke *v1beta1.Spoke, dest *v1beta1.Spoke) error {
	*dest = v1beta1.Spoke{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spoke.Name,
			Namespace: spoke.Namespace,
		},
		Spec: *spoke.Spec.DeepCopy(),
	}
	return nil
}
