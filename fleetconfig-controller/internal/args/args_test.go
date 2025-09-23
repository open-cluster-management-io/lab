package args

import (
	"context"
	"os"
	"reflect"
	"slices"
	"testing"
)

// Mock implementations for testing
type mockResourceValues struct {
	cpu    string
	memory string
}

func (m *mockResourceValues) String() string {
	if m.cpu != "" && m.memory != "" {
		return "cpu=" + m.cpu + ",memory=" + m.memory
	} else if m.cpu != "" {
		return "cpu=" + m.cpu
	} else if m.memory != "" {
		return "memory=" + m.memory
	}
	return ""
}

type mockResourceSpec struct {
	requests *mockResourceValues
	limits   *mockResourceValues
	qosClass string
}

func (m *mockResourceSpec) GetRequests() ResourceValues {
	if m.requests == nil {
		return &mockResourceValues{}
	}
	return m.requests
}

func (m *mockResourceSpec) GetLimits() ResourceValues {
	if m.limits == nil {
		return &mockResourceValues{}
	}
	return m.limits
}

func (m *mockResourceSpec) GetQosClass() string {
	return m.qosClass
}

func TestPrepareKubeconfig(t *testing.T) {
	ctx := context.Background()
	kubeconfig := []byte(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-server:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token`)

	args := []string{"init", "--hub-name", "test-hub"}

	t.Run("with context", func(t *testing.T) {
		resultArgs, cleanup, err := PrepareKubeconfig(ctx, kubeconfig, "test-context", args)
		defer cleanup()

		if err != nil {
			t.Errorf("PrepareKubeconfig() error = %v", err)
		}

		// Check that kubeconfig flag is added
		if !slices.Contains(resultArgs, "--kubeconfig") {
			t.Error("PrepareKubeconfig() should add --kubeconfig flag")
		}

		// Check that context flag is added
		if !slices.Contains(resultArgs, "--context") {
			t.Error("PrepareKubeconfig() should add --context flag")
		}

		// Check that the kubeconfig file exists
		kubeconfigIndex := slices.Index(resultArgs, "--kubeconfig")
		if kubeconfigIndex == -1 || kubeconfigIndex+1 >= len(resultArgs) {
			t.Fatal("PrepareKubeconfig() should add kubeconfig path")
		}

		kubeconfigPath := resultArgs[kubeconfigIndex+1]
		if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
			t.Errorf("PrepareKubeconfig() kubeconfig file should exist at %s", kubeconfigPath)
		}
	})

	t.Run("without context", func(t *testing.T) {
		resultArgs, cleanup, err := PrepareKubeconfig(ctx, kubeconfig, "", args)
		defer cleanup()

		if err != nil {
			t.Errorf("PrepareKubeconfig() error = %v", err)
		}

		// Check that kubeconfig flag is added but context is not
		if !slices.Contains(resultArgs, "--kubeconfig") {
			t.Error("PrepareKubeconfig() should add --kubeconfig flag")
		}

		if slices.Contains(resultArgs, "--context") {
			t.Error("PrepareKubeconfig() should not add --context flag when context is empty")
		}
	})
}

func TestPrepareResources(t *testing.T) {
	tests := []struct {
		name     string
		spec     ResourceSpec
		expected []string
	}{
		{
			name: "with requests and limits",
			spec: &mockResourceSpec{
				requests: &mockResourceValues{cpu: "100m", memory: "128Mi"},
				limits:   &mockResourceValues{cpu: "500m", memory: "512Mi"},
				qosClass: "BestEffort",
			},
			expected: []string{
				"--resource-qos-class", "BestEffort",
				"--resource-requests", "cpu=100m,memory=128Mi",
				"--resource-limits", "cpu=500m,memory=512Mi",
			},
		},
		{
			name: "with only requests",
			spec: &mockResourceSpec{
				requests: &mockResourceValues{cpu: "200m"},
				qosClass: "Default",
			},
			expected: []string{
				"--resource-qos-class", "Default",
				"--resource-requests", "cpu=200m",
			},
		},
		{
			name: "with only limits",
			spec: &mockResourceSpec{
				limits:   &mockResourceValues{memory: "1Gi"},
				qosClass: "ResourceRequirement",
			},
			expected: []string{
				"--resource-qos-class", "ResourceRequirement",
				"--resource-limits", "memory=1Gi",
			},
		},
		{
			name: "with empty resources",
			spec: &mockResourceSpec{
				qosClass: "Default",
			},
			expected: []string{
				"--resource-qos-class", "Default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrepareResources(tt.spec)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("PrepareResources() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMockResourceValues_String(t *testing.T) {
	tests := []struct {
		name     string
		values   *mockResourceValues
		expected string
	}{
		{
			name:     "both cpu and memory",
			values:   &mockResourceValues{cpu: "100m", memory: "128Mi"},
			expected: "cpu=100m,memory=128Mi",
		},
		{
			name:     "only cpu",
			values:   &mockResourceValues{cpu: "200m"},
			expected: "cpu=200m",
		},
		{
			name:     "only memory",
			values:   &mockResourceValues{memory: "256Mi"},
			expected: "memory=256Mi",
		},
		{
			name:     "empty values",
			values:   &mockResourceValues{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.values.String()
			if result != tt.expected {
				t.Errorf("mockResourceValues.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}
