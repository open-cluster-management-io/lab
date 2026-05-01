package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"open-cluster-management-io/lab/apiserver/pkg/models"
)

func IsMockMode() bool {
	return os.Getenv("DASHBOARD_USE_MOCK") == "true"
}

func mockTime(daysAgo int) string {
	return time.Now().AddDate(0, 0, -daysAgo).Format(time.RFC3339)
}

func MockGetClusters(c *gin.Context) {
	clusters := []models.Cluster{
		{
			ID: "uid-cluster-1", Name: "spoke-1", Status: "Online", Version: "v1.28.4",
			HubAccepted: true, CreationTimestamp: mockTime(30),
			Labels: map[string]string{
				"cloud": "AWS", "region": "us-east-1", "env": "production",
				"cluster.open-cluster-management.io/clusterset": "default",
			},
			ClusterClaims: []models.ClusterClaim{
				{Name: "id.k8s.io", Value: "spoke-1"},
				{Name: "platform.open-cluster-management.io", Value: "AWS"},
			},
			Capacity:    map[string]string{"cpu": "16", "memory": "64Gi", "pods": "250"},
			Allocatable: map[string]string{"cpu": "14", "memory": "60Gi", "pods": "250"},
			Conditions: []models.Condition{
				{Type: "ManagedClusterConditionAvailable", Status: "True", Reason: "ManagedClusterAvailable", Message: "Cluster is available", LastTransitionTime: mockTime(1)},
				{Type: "HubAcceptedManagedCluster", Status: "True", Reason: "HubClusterAdminAccepted", Message: "Accepted by hub", LastTransitionTime: mockTime(30)},
				{Type: "ManagedClusterJoined", Status: "True", Reason: "ManagedClusterJoined", Message: "Cluster joined", LastTransitionTime: mockTime(29)},
			},
		},
		{
			ID: "uid-cluster-2", Name: "spoke-2", Status: "Online", Version: "v1.29.1",
			HubAccepted: true, CreationTimestamp: mockTime(25),
			Labels: map[string]string{
				"cloud": "GCP", "region": "europe-west1", "env": "staging",
				"cluster.open-cluster-management.io/clusterset": "default",
			},
			ClusterClaims: []models.ClusterClaim{
				{Name: "id.k8s.io", Value: "spoke-2"},
				{Name: "platform.open-cluster-management.io", Value: "GCP"},
			},
			Capacity:    map[string]string{"cpu": "8", "memory": "32Gi", "pods": "150"},
			Allocatable: map[string]string{"cpu": "7", "memory": "30Gi", "pods": "150"},
			Conditions: []models.Condition{
				{Type: "ManagedClusterConditionAvailable", Status: "True", Reason: "ManagedClusterAvailable", Message: "Cluster is available", LastTransitionTime: mockTime(0)},
				{Type: "HubAcceptedManagedCluster", Status: "True", Reason: "HubClusterAdminAccepted", Message: "Accepted by hub", LastTransitionTime: mockTime(25)},
				{Type: "ManagedClusterJoined", Status: "True", Reason: "ManagedClusterJoined", Message: "Cluster joined", LastTransitionTime: mockTime(24)},
			},
		},
		{
			ID: "uid-cluster-3", Name: "spoke-3", Status: "Offline", Version: "v1.27.8",
			HubAccepted: true, CreationTimestamp: mockTime(60),
			Labels: map[string]string{
				"cloud": "Azure", "region": "eastus", "env": "development",
				"cluster.open-cluster-management.io/clusterset": "global",
			},
			ClusterClaims: []models.ClusterClaim{
				{Name: "id.k8s.io", Value: "spoke-3"},
				{Name: "platform.open-cluster-management.io", Value: "Azure"},
			},
			Capacity:    map[string]string{"cpu": "4", "memory": "16Gi", "pods": "100"},
			Allocatable: map[string]string{"cpu": "3", "memory": "14Gi", "pods": "100"},
			Conditions: []models.Condition{
				{Type: "ManagedClusterConditionAvailable", Status: "False", Reason: "ManagedClusterNotReachable", Message: "Cluster is not reachable", LastTransitionTime: mockTime(2)},
				{Type: "HubAcceptedManagedCluster", Status: "True", Reason: "HubClusterAdminAccepted", Message: "Accepted by hub", LastTransitionTime: mockTime(60)},
				{Type: "ManagedClusterJoined", Status: "True", Reason: "ManagedClusterJoined", Message: "Cluster joined", LastTransitionTime: mockTime(59)},
			},
		},
		{
			ID: "uid-cluster-4", Name: "spoke-4", Status: "Online", Version: "v1.29.0",
			HubAccepted: true, CreationTimestamp: mockTime(10),
			Labels: map[string]string{
				"cloud": "AWS", "region": "ap-southeast-1", "env": "production",
				"cluster.open-cluster-management.io/clusterset": "global",
			},
			ClusterClaims: []models.ClusterClaim{
				{Name: "id.k8s.io", Value: "spoke-4"},
				{Name: "platform.open-cluster-management.io", Value: "AWS"},
			},
			Capacity:    map[string]string{"cpu": "32", "memory": "128Gi", "pods": "500"},
			Allocatable: map[string]string{"cpu": "30", "memory": "120Gi", "pods": "500"},
			Conditions: []models.Condition{
				{Type: "ManagedClusterConditionAvailable", Status: "True", Reason: "ManagedClusterAvailable", Message: "Cluster is available", LastTransitionTime: mockTime(0)},
				{Type: "HubAcceptedManagedCluster", Status: "True", Reason: "HubClusterAdminAccepted", Message: "Accepted by hub", LastTransitionTime: mockTime(10)},
				{Type: "ManagedClusterJoined", Status: "True", Reason: "ManagedClusterJoined", Message: "Cluster joined", LastTransitionTime: mockTime(9)},
			},
		},
		{
			ID: "uid-cluster-5", Name: "spoke-5", Status: "Online", Version: "v1.28.6",
			HubAccepted: true, CreationTimestamp: mockTime(15),
			Labels: map[string]string{
				"cloud": "GCP", "region": "us-central1", "env": "staging",
				"cluster.open-cluster-management.io/clusterset": "default",
			},
			ClusterClaims: []models.ClusterClaim{
				{Name: "id.k8s.io", Value: "spoke-5"},
				{Name: "platform.open-cluster-management.io", Value: "GCP"},
			},
			Capacity:    map[string]string{"cpu": "12", "memory": "48Gi", "pods": "200"},
			Allocatable: map[string]string{"cpu": "11", "memory": "44Gi", "pods": "200"},
			Conditions: []models.Condition{
				{Type: "ManagedClusterConditionAvailable", Status: "True", Reason: "ManagedClusterAvailable", Message: "Cluster is available", LastTransitionTime: mockTime(0)},
				{Type: "HubAcceptedManagedCluster", Status: "True", Reason: "HubClusterAdminAccepted", Message: "Accepted by hub", LastTransitionTime: mockTime(15)},
				{Type: "ManagedClusterJoined", Status: "True", Reason: "ManagedClusterJoined", Message: "Cluster joined", LastTransitionTime: mockTime(14)},
			},
		},
	}
	c.JSON(http.StatusOK, clusters)
}

func MockGetCluster(c *gin.Context) {
	name := c.Param("name")
	clusters := getMockClusters()
	for _, cluster := range clusters {
		if cluster.Name == name {
			c.JSON(http.StatusOK, cluster)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Cluster not found"})
}

func getMockClusters() []models.Cluster {
	return []models.Cluster{
		{ID: "uid-cluster-1", Name: "spoke-1", Status: "Online", Version: "v1.28.4", HubAccepted: true, CreationTimestamp: mockTime(30), Labels: map[string]string{"cloud": "AWS", "region": "us-east-1", "env": "production", "cluster.open-cluster-management.io/clusterset": "default"}},
		{ID: "uid-cluster-2", Name: "spoke-2", Status: "Online", Version: "v1.29.1", HubAccepted: true, CreationTimestamp: mockTime(25), Labels: map[string]string{"cloud": "GCP", "region": "europe-west1", "env": "staging", "cluster.open-cluster-management.io/clusterset": "default"}},
		{ID: "uid-cluster-3", Name: "spoke-3", Status: "Offline", Version: "v1.27.8", HubAccepted: true, CreationTimestamp: mockTime(60), Labels: map[string]string{"cloud": "Azure", "region": "eastus", "env": "development", "cluster.open-cluster-management.io/clusterset": "global"}},
		{ID: "uid-cluster-4", Name: "spoke-4", Status: "Online", Version: "v1.29.0", HubAccepted: true, CreationTimestamp: mockTime(10), Labels: map[string]string{"cloud": "AWS", "region": "ap-southeast-1", "env": "production", "cluster.open-cluster-management.io/clusterset": "global"}},
		{ID: "uid-cluster-5", Name: "spoke-5", Status: "Online", Version: "v1.28.6", HubAccepted: true, CreationTimestamp: mockTime(15), Labels: map[string]string{"cloud": "GCP", "region": "us-central1", "env": "staging", "cluster.open-cluster-management.io/clusterset": "default"}},
	}
}

func MockGetClusterSets(c *gin.Context) {
	clusterSets := []models.ClusterSet{
		{
			ID: "uid-set-1", Name: "default", CreationTimestamp: mockTime(90),
			Spec: models.ClusterSetSpec{
				ClusterSelector: models.ClusterSelector{SelectorType: "ExclusiveClusterSetLabel"},
			},
			Status: models.ClusterSetStatus{
				Conditions: []models.Condition{
					{Type: "ClusterSetEmpty", Status: "False", Reason: "ClustersSelected", Message: "3 ManagedClusters selected", LastTransitionTime: mockTime(1)},
				},
			},
		},
		{
			ID: "uid-set-2", Name: "global", CreationTimestamp: mockTime(90),
			Spec: models.ClusterSetSpec{
				ClusterSelector: models.ClusterSelector{SelectorType: "ExclusiveClusterSetLabel"},
			},
			Status: models.ClusterSetStatus{
				Conditions: []models.Condition{
					{Type: "ClusterSetEmpty", Status: "False", Reason: "ClustersSelected", Message: "2 ManagedClusters selected", LastTransitionTime: mockTime(1)},
				},
			},
		},
	}
	c.JSON(http.StatusOK, clusterSets)
}

func MockGetClusterSet(c *gin.Context) {
	name := c.Param("name")
	sets := []models.ClusterSet{
		{ID: "uid-set-1", Name: "default", CreationTimestamp: mockTime(90), Spec: models.ClusterSetSpec{ClusterSelector: models.ClusterSelector{SelectorType: "ExclusiveClusterSetLabel"}}},
		{ID: "uid-set-2", Name: "global", CreationTimestamp: mockTime(90), Spec: models.ClusterSetSpec{ClusterSelector: models.ClusterSelector{SelectorType: "ExclusiveClusterSetLabel"}}},
	}
	for _, s := range sets {
		if s.Name == name {
			c.JSON(http.StatusOK, s)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "ClusterSet not found"})
}

func MockGetPlacements(c *gin.Context) {
	three := int32(3)
	placements := []models.Placement{
		{
			ID: "uid-placement-1", Name: "placement-prod", Namespace: "default",
			CreationTimestamp: mockTime(20), ClusterSets: []string{"default"},
			NumberOfClusters: &three, NumberOfSelectedClusters: 3,
			Satisfied: true,
			Conditions: []models.Condition{
				{Type: "PlacementSatisfied", Status: "True", Reason: "AllDecisionsScheduled", Message: "All clusters scheduled", LastTransitionTime: mockTime(0)},
			},
			DecisionGroups: []models.DecisionGroupStatus{
				{DecisionGroupIndex: 0, DecisionGroupName: "", Decisions: []string{"placement-prod-decision-1"}, ClusterCount: 3},
			},
		},
		{
			ID: "uid-placement-2", Name: "placement-staging", Namespace: "default",
			CreationTimestamp: mockTime(15), ClusterSets: []string{"default", "global"},
			NumberOfSelectedClusters: 2,
			Satisfied:               true,
			Conditions: []models.Condition{
				{Type: "PlacementSatisfied", Status: "True", Reason: "AllDecisionsScheduled", Message: "All clusters scheduled", LastTransitionTime: mockTime(0)},
			},
			DecisionGroups: []models.DecisionGroupStatus{
				{DecisionGroupIndex: 0, DecisionGroupName: "", Decisions: []string{"placement-staging-decision-1"}, ClusterCount: 2},
			},
		},
		{
			ID: "uid-placement-3", Name: "placement-dev", Namespace: "open-cluster-management",
			CreationTimestamp: mockTime(5), ClusterSets: []string{"global"},
			NumberOfSelectedClusters: 0,
			Satisfied:               false, ReasonMessage: "No clusters available matching criteria",
			Conditions: []models.Condition{
				{Type: "PlacementSatisfied", Status: "False", Reason: "NotAllDecisionsScheduled", Message: "No clusters available matching criteria", LastTransitionTime: mockTime(0)},
			},
		},
	}
	c.JSON(http.StatusOK, placements)
}

func MockGetPlacement(c *gin.Context) {
	MockGetPlacements(c)
}

func MockGetPlacementsByNamespace(c *gin.Context) {
	MockGetPlacements(c)
}

func MockGetPlacementDecisions(c *gin.Context) {
	decisions := []models.PlacementDecision{
		{
			ID: "uid-pd-1", Name: "placement-prod-decision-1", Namespace: "default",
			Decisions: []models.ClusterDecision{
				{ClusterName: "spoke-1", Reason: "Score: 100"},
				{ClusterName: "spoke-2", Reason: "Score: 95"},
				{ClusterName: "spoke-5", Reason: "Score: 90"},
			},
		},
	}
	c.JSON(http.StatusOK, decisions)
}

func MockGetAllPlacementDecisions(c *gin.Context) {
	MockGetPlacementDecisions(c)
}

func MockGetPlacementDecisionsByNamespace(c *gin.Context) {
	MockGetPlacementDecisions(c)
}

func MockGetPlacementDecision(c *gin.Context) {
	decisions := []models.PlacementDecision{
		{
			ID: "uid-pd-1", Name: "placement-prod-decision-1", Namespace: "default",
			Decisions: []models.ClusterDecision{
				{ClusterName: "spoke-1", Reason: "Score: 100"},
				{ClusterName: "spoke-2", Reason: "Score: 95"},
			},
		},
	}
	c.JSON(http.StatusOK, decisions[0])
}

func MockGetPlacementDecisionsByPlacement(c *gin.Context) {
	MockGetPlacementDecisions(c)
}

func MockGetClusterAddons(c *gin.Context) {
	addons := []models.ManagedClusterAddon{
		{
			ID: "uid-addon-1", Name: "governance-policy-framework", Namespace: c.Param("name"),
			InstallNamespace: "open-cluster-management-agent-addon", CreationTimestamp: mockTime(28),
			Conditions: []models.Condition{
				{Type: "Available", Status: "True", Reason: "AddonAvailable", Message: "Addon is available", LastTransitionTime: mockTime(0)},
			},
		},
		{
			ID: "uid-addon-2", Name: "config-policy-controller", Namespace: c.Param("name"),
			InstallNamespace: "open-cluster-management-agent-addon", CreationTimestamp: mockTime(28),
			Conditions: []models.Condition{
				{Type: "Available", Status: "True", Reason: "AddonAvailable", Message: "Addon is available", LastTransitionTime: mockTime(0)},
			},
		},
		{
			ID: "uid-addon-3", Name: "work-manager", Namespace: c.Param("name"),
			InstallNamespace: "open-cluster-management-agent-addon", CreationTimestamp: mockTime(28),
			Conditions: []models.Condition{
				{Type: "Available", Status: "True", Reason: "AddonAvailable", Message: "Addon is available", LastTransitionTime: mockTime(0)},
			},
		},
	}
	c.JSON(http.StatusOK, addons)
}

func MockGetClusterAddon(c *gin.Context) {
	addon := models.ManagedClusterAddon{
		ID: "uid-addon-1", Name: c.Param("addonName"), Namespace: c.Param("name"),
		InstallNamespace: "open-cluster-management-agent-addon", CreationTimestamp: mockTime(28),
		Conditions: []models.Condition{
			{Type: "Available", Status: "True", Reason: "AddonAvailable", Message: "Addon is available", LastTransitionTime: mockTime(0)},
		},
	}
	c.JSON(http.StatusOK, addon)
}

func MockGetAllClusterSetBindings(c *gin.Context) {
	bindings := []models.ManagedClusterSetBinding{
		{
			ID: "uid-binding-1", Name: "default", Namespace: "default",
			CreationTimestamp: mockTime(85),
			Spec:             models.ManagedClusterSetBindingSpec{ClusterSet: "default"},
			Status: models.ManagedClusterSetBindingStatus{
				Conditions: []models.Condition{
					{Type: "Bound", Status: "True", Reason: "ClusterSetBound", Message: "Bound to ManagedClusterSet default", LastTransitionTime: mockTime(1)},
				},
			},
		},
		{
			ID: "uid-binding-2", Name: "global", Namespace: "open-cluster-management",
			CreationTimestamp: mockTime(85),
			Spec:             models.ManagedClusterSetBindingSpec{ClusterSet: "global"},
			Status: models.ManagedClusterSetBindingStatus{
				Conditions: []models.Condition{
					{Type: "Bound", Status: "True", Reason: "ClusterSetBound", Message: "Bound to ManagedClusterSet global", LastTransitionTime: mockTime(1)},
				},
			},
		},
	}
	c.JSON(http.StatusOK, bindings)
}

func MockGetClusterSetBindings(c *gin.Context) {
	MockGetAllClusterSetBindings(c)
}

func MockGetClusterSetBinding(c *gin.Context) {
	binding := models.ManagedClusterSetBinding{
		ID: "uid-binding-1", Name: c.Param("name"), Namespace: c.Param("namespace"),
		CreationTimestamp: mockTime(85),
		Spec:             models.ManagedClusterSetBindingSpec{ClusterSet: c.Param("name")},
		Status: models.ManagedClusterSetBindingStatus{
			Conditions: []models.Condition{
				{Type: "Bound", Status: "True", Reason: "ClusterSetBound", Message: "Bound to ManagedClusterSet", LastTransitionTime: mockTime(1)},
			},
		},
	}
	c.JSON(http.StatusOK, binding)
}

func MockGetManifestWorks(c *gin.Context) {
	works := []models.ManifestWork{
		{
			ID: "uid-mw-1", Name: "deploy-nginx", Namespace: c.Param("namespace"),
			CreationTimestamp: mockTime(5),
			Labels:           map[string]string{"app": "nginx"},
			Conditions: []models.Condition{
				{Type: "Applied", Status: "True", Reason: "AppliedManifestWorkComplete", Message: "Apply manifest work complete", LastTransitionTime: mockTime(0)},
				{Type: "Available", Status: "True", Reason: "ResourcesAvailable", Message: "All resources are available", LastTransitionTime: mockTime(0)},
			},
		},
	}
	c.JSON(http.StatusOK, works)
}

func MockGetManifestWork(c *gin.Context) {
	work := models.ManifestWork{
		ID: "uid-mw-1", Name: c.Param("name"), Namespace: c.Param("namespace"),
		CreationTimestamp: mockTime(5),
		Conditions: []models.Condition{
			{Type: "Applied", Status: "True", Reason: "AppliedManifestWorkComplete", Message: "Apply manifest work complete", LastTransitionTime: mockTime(0)},
			{Type: "Available", Status: "True", Reason: "ResourcesAvailable", Message: "All resources are available", LastTransitionTime: mockTime(0)},
		},
	}
	c.JSON(http.StatusOK, work)
}

func MockStreamClusters(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	c.Writer.Write([]byte("event: clusters\ndata: []\n\n"))
	c.Writer.Flush()
}
