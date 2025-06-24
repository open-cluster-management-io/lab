package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestWorkModel(t *testing.T) {
	manifestWork := ManifestWork{
		ID:        "test-id",
		Name:      "test-manifestwork",
		Namespace: "test-cluster",
		Labels: map[string]string{
			"app": "test-app",
		},
		Manifests: []Manifest{
			{
				RawExtension: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name": "test-config",
					},
				},
			},
		},
		Conditions: []Condition{
			{
				Type:   "Applied",
				Status: "True",
			},
		},
		ResourceStatus: ManifestResourceStatus{
			Manifests: []ManifestCondition{
				{
					ResourceMeta: ManifestResourceMeta{
						Ordinal:   0,
						Group:     "",
						Version:   "v1",
						Kind:      "ConfigMap",
						Resource:  "configmaps",
						Name:      "test-config",
						Namespace: "default",
					},
					Conditions: []Condition{
						{
							Type:   "Available",
							Status: "True",
						},
					},
				},
			},
		},
		CreationTimestamp: "2023-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-id", manifestWork.ID)
	assert.Equal(t, "test-manifestwork", manifestWork.Name)
	assert.Equal(t, "test-cluster", manifestWork.Namespace)
	assert.Equal(t, "test-app", manifestWork.Labels["app"])
	assert.Len(t, manifestWork.Manifests, 1)
	assert.Equal(t, "ConfigMap", manifestWork.Manifests[0].RawExtension["kind"])
	assert.Len(t, manifestWork.Conditions, 1)
	assert.Equal(t, "Applied", manifestWork.Conditions[0].Type)
	assert.Len(t, manifestWork.ResourceStatus.Manifests, 1)
	assert.Equal(t, "test-config", manifestWork.ResourceStatus.Manifests[0].ResourceMeta.Name)
	assert.Equal(t, "2023-01-01T00:00:00Z", manifestWork.CreationTimestamp)
}

func TestManifestModel(t *testing.T) {
	manifest := Manifest{
		RawExtension: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "test-deployment",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"replicas": 3,
			},
		},
	}

	assert.Equal(t, "apps/v1", manifest.RawExtension["apiVersion"])
	assert.Equal(t, "Deployment", manifest.RawExtension["kind"])
	metadata := manifest.RawExtension["metadata"].(map[string]interface{})
	assert.Equal(t, "test-deployment", metadata["name"])
}

func TestManifestResourceStatusModel(t *testing.T) {
	status := ManifestResourceStatus{
		Manifests: []ManifestCondition{
			{
				ResourceMeta: ManifestResourceMeta{
					Ordinal:  0,
					Kind:     "Service",
					Name:     "test-service",
					Resource: "services",
				},
				Conditions: []Condition{
					{
						Type:   "Available",
						Status: "True",
					},
				},
			},
		},
	}

	assert.Len(t, status.Manifests, 1)
	assert.Equal(t, "Service", status.Manifests[0].ResourceMeta.Kind)
	assert.Equal(t, "test-service", status.Manifests[0].ResourceMeta.Name)
	assert.Len(t, status.Manifests[0].Conditions, 1)
}

func TestManifestConditionModel(t *testing.T) {
	condition := ManifestCondition{
		ResourceMeta: ManifestResourceMeta{
			Ordinal:   1,
			Group:     "apps",
			Version:   "v1",
			Kind:      "Deployment",
			Resource:  "deployments",
			Name:      "test-deployment",
			Namespace: "default",
		},
		Conditions: []Condition{
			{
				Type:    "Progressing",
				Status:  "True",
				Reason:  "NewReplicaSetAvailable",
				Message: "ReplicaSet has successfully progressed",
			},
		},
	}

	assert.Equal(t, int32(1), condition.ResourceMeta.Ordinal)
	assert.Equal(t, "apps", condition.ResourceMeta.Group)
	assert.Equal(t, "v1", condition.ResourceMeta.Version)
	assert.Equal(t, "Deployment", condition.ResourceMeta.Kind)
	assert.Equal(t, "deployments", condition.ResourceMeta.Resource)
	assert.Equal(t, "test-deployment", condition.ResourceMeta.Name)
	assert.Equal(t, "default", condition.ResourceMeta.Namespace)
	assert.Len(t, condition.Conditions, 1)
	assert.Equal(t, "Progressing", condition.Conditions[0].Type)
}

func TestManifestResourceMetaModel(t *testing.T) {
	meta := ManifestResourceMeta{
		Ordinal:   0,
		Group:     "networking.k8s.io",
		Version:   "v1",
		Kind:      "Ingress",
		Resource:  "ingresses",
		Name:      "test-ingress",
		Namespace: "default",
	}

	assert.Equal(t, int32(0), meta.Ordinal)
	assert.Equal(t, "networking.k8s.io", meta.Group)
	assert.Equal(t, "v1", meta.Version)
	assert.Equal(t, "Ingress", meta.Kind)
	assert.Equal(t, "ingresses", meta.Resource)
	assert.Equal(t, "test-ingress", meta.Name)
	assert.Equal(t, "default", meta.Namespace)
}

func TestDeleteOptionModel(t *testing.T) {
	deleteOption := DeleteOption{
		PropagationPolicy: "Foreground",
		SelectivelyOrphan: &SelectivelyOrphan{
			OrphaningRules: []OrphaningRule{
				{
					Group:     "apps",
					Resource:  "deployments",
					Name:      "keep-deployment",
					Namespace: "default",
				},
			},
		},
	}

	assert.Equal(t, "Foreground", deleteOption.PropagationPolicy)
	assert.NotNil(t, deleteOption.SelectivelyOrphan)
	assert.Len(t, deleteOption.SelectivelyOrphan.OrphaningRules, 1)
	assert.Equal(t, "apps", deleteOption.SelectivelyOrphan.OrphaningRules[0].Group)
	assert.Equal(t, "keep-deployment", deleteOption.SelectivelyOrphan.OrphaningRules[0].Name)
}

func TestOrphaningRuleModel(t *testing.T) {
	rule := OrphaningRule{
		Group:     "v1",
		Resource:  "secrets",
		Name:      "important-secret",
		Namespace: "kube-system",
	}

	assert.Equal(t, "v1", rule.Group)
	assert.Equal(t, "secrets", rule.Resource)
	assert.Equal(t, "important-secret", rule.Name)
	assert.Equal(t, "kube-system", rule.Namespace)
}

func TestManifestConfigOptionModel(t *testing.T) {
	config := ManifestConfigOption{
		ResourceIdentifier: ResourceIdentifier{
			Group:     "apps",
			Resource:  "deployments",
			Name:      "test-deployment",
			Namespace: "default",
		},
		FeedbackRules: []FeedbackRule{
			{
				Type: "JSONPaths",
				JsonPaths: []JsonPath{
					{
						Name:    "status",
						Version: "v1",
						Path:    ".status.readyReplicas",
					},
				},
			},
		},
		UpdateStrategy: &UpdateStrategy{
			Type: "ServerSideApply",
			ServerSideApply: &ServerSideApplyConfig{
				Force:        true,
				FieldManager: "manifestwork-agent",
			},
		},
	}

	assert.Equal(t, "apps", config.ResourceIdentifier.Group)
	assert.Equal(t, "test-deployment", config.ResourceIdentifier.Name)
	assert.Len(t, config.FeedbackRules, 1)
	assert.Equal(t, "JSONPaths", config.FeedbackRules[0].Type)
	assert.NotNil(t, config.UpdateStrategy)
	assert.Equal(t, "ServerSideApply", config.UpdateStrategy.Type)
	assert.True(t, config.UpdateStrategy.ServerSideApply.Force)
}

func TestResourceIdentifierModel(t *testing.T) {
	identifier := ResourceIdentifier{
		Group:     "batch",
		Resource:  "jobs",
		Name:      "test-job",
		Namespace: "default",
	}

	assert.Equal(t, "batch", identifier.Group)
	assert.Equal(t, "jobs", identifier.Resource)
	assert.Equal(t, "test-job", identifier.Name)
	assert.Equal(t, "default", identifier.Namespace)
}

func TestFeedbackRuleModel(t *testing.T) {
	rule := FeedbackRule{
		Type: "JSONPaths",
		JsonPaths: []JsonPath{
			{
				Name:    "replicas",
				Version: "v1",
				Path:    ".status.replicas",
			},
			{
				Name: "ready",
				Path: ".status.readyReplicas",
			},
		},
	}

	assert.Equal(t, "JSONPaths", rule.Type)
	assert.Len(t, rule.JsonPaths, 2)
	assert.Equal(t, "replicas", rule.JsonPaths[0].Name)
	assert.Equal(t, ".status.replicas", rule.JsonPaths[0].Path)
	assert.Equal(t, "ready", rule.JsonPaths[1].Name)
}

func TestJsonPathModel(t *testing.T) {
	jsonPath := JsonPath{
		Name:    "conditions",
		Version: "v1",
		Path:    ".status.conditions",
	}

	assert.Equal(t, "conditions", jsonPath.Name)
	assert.Equal(t, "v1", jsonPath.Version)
	assert.Equal(t, ".status.conditions", jsonPath.Path)
}

func TestManifestWorkListModel(t *testing.T) {
	list := ManifestWorkList{
		Items: []ManifestWork{
			{
				ID:   "work1",
				Name: "manifestwork-1",
			},
			{
				ID:   "work2",
				Name: "manifestwork-2",
			},
		},
	}

	assert.Len(t, list.Items, 2)
	assert.Equal(t, "work1", list.Items[0].ID)
	assert.Equal(t, "manifestwork-1", list.Items[0].Name)
	assert.Equal(t, "work2", list.Items[1].ID)
}
