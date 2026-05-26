package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
	workv1alpha1 "open-cluster-management.io/api/work/v1alpha1"
)

func TestConvertMWRSBasicFields(t *testing.T) {
	ts := metav1.NewTime(time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC))
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deploy-nginx",
			Namespace:         "default",
			UID:               types.UID("uid-mwrs-1"),
			CreationTimestamp: ts,
			Labels:            map[string]string{"app": "nginx"},
		},
		Status: workv1alpha1.ManifestWorkReplicaSetStatus{
			Summary: workv1alpha1.ManifestWorkReplicaSetSummary{
				Total:     3,
				Available: 2,
				Applied:   3,
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Equal(t, "deploy-nginx", result.Name)
	assert.Equal(t, "default", result.Namespace)
	assert.Equal(t, "uid-mwrs-1", result.ID)
	assert.Equal(t, "2025-06-15T10:30:00Z", result.CreationTimestamp)
	assert.Equal(t, "nginx", result.Labels["app"])
}

func TestConvertMWRSPlacementRefs(t *testing.T) {
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Spec: workv1alpha1.ManifestWorkReplicaSetSpec{
			PlacementRefs: []workv1alpha1.LocalPlacementReference{
				{
					Name: "prod-clusters",
					RolloutStrategy: clusterv1alpha1.RolloutStrategy{
						Type: clusterv1alpha1.Progressive,
					},
				},
				{
					Name: "dev-clusters",
					RolloutStrategy: clusterv1alpha1.RolloutStrategy{
						Type: clusterv1alpha1.All,
					},
				},
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Len(t, result.PlacementRefs, 2)
	assert.Equal(t, "prod-clusters", result.PlacementRefs[0].Name)
	assert.Equal(t, "Progressive", result.PlacementRefs[0].RolloutStrategyType)
	assert.Equal(t, "dev-clusters", result.PlacementRefs[1].Name)
	assert.Equal(t, "All", result.PlacementRefs[1].RolloutStrategyType)
}

func TestConvertMWRSConditions(t *testing.T) {
	now := metav1.Now()
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: workv1alpha1.ManifestWorkReplicaSetStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "PlacementVerified",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "AsExpected",
					Message:            "Placement verified",
				},
				{
					Type:               "ManifestworkApplied",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: now,
					Reason:             "AsExpected",
					Message:            "All ManifestWorks applied",
				},
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Len(t, result.Conditions, 2)
	assert.Equal(t, "PlacementVerified", result.Conditions[0].Type)
	assert.Equal(t, "True", result.Conditions[0].Status)
	assert.Equal(t, "AsExpected", result.Conditions[0].Reason)
	assert.Equal(t, "Placement verified", result.Conditions[0].Message)
	assert.Equal(t, "ManifestworkApplied", result.Conditions[1].Type)
}

func TestConvertMWRSSummary(t *testing.T) {
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: workv1alpha1.ManifestWorkReplicaSetStatus{
			Summary: workv1alpha1.ManifestWorkReplicaSetSummary{
				Total:       5,
				Available:   3,
				Progressing: 1,
				Degraded:    1,
				Applied:     4,
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Equal(t, 5, result.Summary.Total)
	assert.Equal(t, 3, result.Summary.Available)
	assert.Equal(t, 1, result.Summary.Progressing)
	assert.Equal(t, 1, result.Summary.Degraded)
	assert.Equal(t, 4, result.Summary.Applied)
}

func TestConvertMWRSPlacementsSummary(t *testing.T) {
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Status: workv1alpha1.ManifestWorkReplicaSetStatus{
			PlacementsSummary: []workv1alpha1.PlacementSummary{
				{
					Name:                    "all-clusters",
					AvailableDecisionGroups: "2/3",
					Summary: workv1alpha1.ManifestWorkReplicaSetSummary{
						Total:     3,
						Available: 2,
						Applied:   3,
					},
				},
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Len(t, result.PlacementsSummary, 1)
	assert.Equal(t, "all-clusters", result.PlacementsSummary[0].Name)
	assert.Equal(t, "2/3", result.PlacementsSummary[0].AvailableDecisionGroups)
	assert.Equal(t, 3, result.PlacementsSummary[0].Summary.Total)
	assert.Equal(t, 2, result.PlacementsSummary[0].Summary.Available)
}

func TestConvertMWRSManifestCount(t *testing.T) {
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Spec: workv1alpha1.ManifestWorkReplicaSetSpec{
			ManifestWorkTemplate: workv1.ManifestWorkSpec{
				Workload: workv1.ManifestsTemplate{
					Manifests: []workv1.Manifest{{}, {}, {}},
				},
			},
		},
	}

	result := convertMWRS(mwrs)
	assert.Equal(t, 3, result.ManifestCount)
}

func TestConvertMWRSEmpty(t *testing.T) {
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "ns"},
	}

	result := convertMWRS(mwrs)
	assert.Equal(t, "empty", result.Name)
	assert.Equal(t, 0, result.ManifestCount)
	assert.Nil(t, result.PlacementRefs)
	assert.Nil(t, result.Conditions)
	assert.Nil(t, result.PlacementsSummary)
	assert.Equal(t, 0, result.Summary.Total)
}

func TestConvertMWRSTimestamp(t *testing.T) {
	ts := metav1.NewTime(time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC))
	mwrs := &workv1alpha1.ManifestWorkReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ts-test",
			Namespace:         "ns",
			CreationTimestamp: ts,
		},
	}

	result := convertMWRS(mwrs)
	assert.Equal(t, "2025-01-15T08:00:00Z", result.CreationTimestamp)
}

// --- Handler nil-client tests ---

func TestGetAllManifestWorkReplicaSetsNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx := context.Background()

	GetAllManifestWorkReplicaSets(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetManifestWorkReplicaSetsNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "namespace", Value: "test-ns"}}
	ctx := context.Background()

	GetManifestWorkReplicaSets(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetManifestWorkReplicaSetNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "namespace", Value: "test-ns"},
		{Key: "name", Value: "test-mwrs"},
	}
	ctx := context.Background()

	GetManifestWorkReplicaSet(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetManifestWorksByReplicaSetNilClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "namespace", Value: "test-ns"},
		{Key: "name", Value: "test-mwrs"},
	}
	ctx := context.Background()

	GetManifestWorksByReplicaSet(c, nil, ctx)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
