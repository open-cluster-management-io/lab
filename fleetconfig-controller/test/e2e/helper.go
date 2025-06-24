package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	workv1 "open-cluster-management.io/api/work/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/test/utils"
)

const (
	fcNamespace         = "fleetconfig-system"
	spokeSecretName     = "test-fleetconfig-kubeconfig"
	devspaceLocal       = "local"
	devspaceCI          = "ci"
	kubeconfigSecretKey = "value"
	hubAsSpokeName      = v1alpha1.ManagedClusterTypeHubAsSpoke
	spokeName           = v1alpha1.ManagedClusterTypeSpoke
)

var (
	// global test context variables
	devspaceProfile    string
	useExistingCluster bool

	// global test variables
	multiClusterNN = ktypes.NamespacedName{Name: "fleetconfig", Namespace: fcNamespace}
)

// E2EContext holds all the test-specific state.
// It allows multiple test suites to run in parallel.
type E2EContext struct {
	ctx                     context.Context
	hubKubeconfig           string
	spokeKubeconfig         string
	spokeKubeconfigInternal string
	kClient                 client.Client
	kClientSpoke            client.Client
}

// setupTestEnvironment sets up the test environment for a context
func setupTestEnvironment() *E2EContext {
	tc := &E2EContext{
		ctx:           context.Background(),
		hubKubeconfig: os.Getenv("KUBECONFIG"),
	}

	var (
		err error
		f   *os.File
	)

	if tc.hubKubeconfig == "" {
		utils.Info("No KUBECONFIG detected - provisioning hub kind cluster for E2E tests.")
		By("creating Hub Kind cluster")
		f, err := os.CreateTemp("", "kubeconfig")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		tc.hubKubeconfig = f.Name()
		Expect(os.Setenv("KUBECONFIG", tc.hubKubeconfig)).To(Succeed())
		Expect(utils.CreateKindCluster(utils.HubClusterName, tc.hubKubeconfig)).To(Succeed())
		if hkDest := os.Getenv("HUB_KUBECONFIG_DEST"); hkDest != "" {
			bs, err := os.ReadFile(tc.hubKubeconfig) // #nosec G304
			Expect(err).NotTo(HaveOccurred())
			Expect(os.WriteFile(hkDest, bs, 0600)).To(Succeed())
		}
	} else {
		utils.Info("KUBECONFIG detected - using existing cluster as hub for E2E tests.")
		useExistingCluster = true
		if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
			utils.HubClusterName = v
		} else {
			Fail("KIND_CLUSTER environment variable must be set when using an existing cluster")
		}
	}

	By("creating Spoke Kind cluster")
	f, err = os.CreateTemp("", "kubeconfig")
	Expect(err).NotTo(HaveOccurred())
	Expect(f.Close()).To(Succeed())
	tc.spokeKubeconfig = f.Name()
	Expect(utils.CreateKindCluster(utils.SpokeClusterName, tc.spokeKubeconfig)).To(Succeed())
	if skDest := os.Getenv("SPOKE_KUBECONFIG_DEST"); skDest != "" {
		bs, err := os.ReadFile(tc.spokeKubeconfig) // #nosec G304
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(skDest, bs, 0600)).To(Succeed())
	}

	By("getting spoke internal kubeconfig")
	f, err = os.CreateTemp("", "kubeconfig")
	Expect(err).NotTo(HaveOccurred())
	tc.spokeKubeconfigInternal = f.Name()
	cmd := exec.Command("kind", "get", "kubeconfig", "--name", utils.SpokeClusterName, "--internal")
	res, err := utils.RunCommand(cmd, "", true)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	_, err = f.Write(res)
	Expect(err).NotTo(HaveOccurred())
	Expect(f.Close()).To(Succeed())

	By("adding external APIs to the client-go scheme")
	Expect(v1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(clusterv1beta1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(clusterv1beta2.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(workv1.AddToScheme(scheme.Scheme)).To(Succeed())

	By("creating a kubernetes client for the hub cluster")
	tc.kClient, err = utils.NewClient(tc.hubKubeconfig, scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("creating a kubeconfig secret for the spoke's internal kubeconfig")
	kcfg, err := os.ReadFile(tc.spokeKubeconfigInternal) // #nosec G304
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spokeSecretName,
			Namespace: "default",
		},
		Data: map[string][]byte{
			kubeconfigSecretKey: kcfg,
		},
	}
	err = tc.kClient.Create(tc.ctx, secret)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating a kubernetes client for the spoke cluster")
	tc.kClientSpoke, err = utils.NewClient(tc.spokeKubeconfig, scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	By("creating fleetconfig namespace")
	cmd = exec.Command("kubectl", "create", "ns", fcNamespace)
	_, err = utils.RunCommand(cmd, "", false)
	Expect(err).NotTo(HaveOccurred())

	return tc
}

// teardownTestEnvironment cleans up the test environment after a context.
//
//   - If tests failed, support bundles are collected before teardown.
//   - If SKIP_CLEANUP is set, teardown is skipped, regardless of test outcome.
//   - If SKIP_CLEANUP_ON_FAILURE is set and a top-level Context fails, the entire suite is aborted.
//
// Note: you must set SKIP_CLEANUP_ON_FAILURE=true and SKIP_CLEANUP=false when running
// suites with multiple invocations of teardownTestEnvironment (e.g., hue). If
// SKIP_CLEANUP=true, the KUBECONFIG env var will not be unset after the 1st context,
// causing the 2nd context to fail because KIND_CLUSTER is not set.
func teardownTestEnvironment(tc *E2EContext) {
	failed := CurrentSpecReport().Failed()

	if failed {
		By("collecting support bundle from spoke cluster")
		if err := utils.GetSupportBundle(tc.ctx, tc.spokeKubeconfig, "spoke"); err != nil {
			utils.WarnError(err, "failed to collect support bundle from spoke cluster")
		}
		By("collecting support bundle from hub cluster")
		if err := utils.GetSupportBundle(tc.ctx, tc.hubKubeconfig, "hub"); err != nil {
			utils.WarnError(err, "failed to collect support bundle from hub cluster")
		}
	}

	if os.Getenv("SKIP_CLEANUP") != "" {
		return
	}
	if os.Getenv("SKIP_CLEANUP_ON_FAILURE") != "" && failed {
		AbortSuite("Aborting suite because SKIP_CLEANUP_ON_FAILURE is set and a top-level Context failed!")
	}

	By("deleting Spoke cluster")
	if err := utils.DeleteKindCluster(utils.SpokeClusterName); err != nil {
		utils.WarnError(err, "failed to delete spoke cluster")
	}
	if err := os.Remove(tc.spokeKubeconfig); err != nil {
		utils.WarnError(err, "failed to remove spoke kubeconfig")
	}
	if err := os.Remove(tc.spokeKubeconfigInternal); err != nil {
		utils.WarnError(err, "failed to remove spoke kubeconfig internal")
	}

	if !useExistingCluster {
		By("deleting Hub cluster")
		if err := utils.DeleteKindCluster(utils.HubClusterName); err != nil {
			utils.WarnError(err, "failed to delete hub cluster")
		}
		if err := os.Remove(tc.hubKubeconfig); err != nil {
			utils.WarnError(err, "failed to remove hub kubeconfig")
		}
		if err := os.Unsetenv("KUBECONFIG"); err != nil {
			utils.WarnError(err, "failed to unset KUBECONFIG")
		}
	} else {
		By("purging fleetconfig")
		if err := utils.DevspacePurge(tc.ctx, tc.hubKubeconfig, fcNamespace); err != nil {
			utils.WarnError(err, "failed to purge from hub cluster")
		}
	}
}

// ensureFleetConfigProvisioned checks that the FleetConfig is properly provisioned with expected conditions
func ensureFleetConfigProvisioned(tc *E2EContext, fc *v1alpha1.FleetConfig, extraExpectedConditions map[string]metav1.ConditionStatus) {
	expectedConditions := map[string]metav1.ConditionStatus{
		v1alpha1.FleetConfigHubInitialized:                     metav1.ConditionTrue,
		v1alpha1.FleetConfigCleanupFailed:                      metav1.ConditionFalse,
		fmt.Sprintf("spoke-cluster-%s-joined", hubAsSpokeName): metav1.ConditionTrue,
		fmt.Sprintf("spoke-cluster-%s-joined", spokeName):      metav1.ConditionTrue,
	}
	for k, v := range extraExpectedConditions {
		expectedConditions[k] = v
	}

	By("ensuring the FleetConfig is provisioned")
	EventuallyWithOffset(1, func() error {
		if err := tc.kClient.Get(tc.ctx, multiClusterNN, fc); err != nil {
			utils.WarnError(err, "FleetConfig not provisioned")
			return err
		}
		conditions := make([]metav1.Condition, len(fc.Status.Conditions))
		for i, c := range fc.Status.Conditions {
			conditions[i] = c.Condition
		}
		if err := utils.AssertConditions(conditions, expectedConditions); err != nil {
			utils.WarnError(err, "FleetConfig not provisioned")
			return err
		}
		if fc.Status.Phase != v1alpha1.FleetConfigRunning {
			err := fmt.Errorf("expected %s, got %s", v1alpha1.FleetConfigRunning, fc.Status.Phase)
			utils.WarnError(err, "FleetConfig not provisioned")
			return err
		}
		return nil
	}, 20*time.Minute, 10*time.Second).Should(Succeed())
}

// removeSpokeFromHub removes the spoke from the FleetConfig
func removeSpokeFromHub(tc *E2EContext, fc *v1alpha1.FleetConfig) {
	By("removing the spoke")
	if err := tc.kClient.Get(tc.ctx, multiClusterNN, fc); err != nil {
		utils.WarnError(err, "failed to get FleetConfig")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}
	fc.Spec.Spokes = slices.DeleteFunc(fc.Spec.Spokes, func(s v1alpha1.Spoke) bool {
		return s.Name == spokeName
	})
	ExpectWithOffset(1, tc.kClient.Update(tc.ctx, fc)).NotTo(HaveOccurred())
}

// ensureResourceDeleted is a generic helper to check if a resource is deleted
func ensureResourceDeleted(checkFn func() error) {
	EventuallyWithOffset(1, func() error {
		if err := checkFn(); err != nil {
			utils.WarnError(err, "waiting for deletion")
			return err
		}
		return nil
	}, 5*time.Minute, 10*time.Second).Should(Succeed())
}

// createManifestWork creates a ManifestWork in the given namespace
func createManifestWork(ctx context.Context, namespace string) error {
	workC, err := common.WorkClient(nil)
	if err != nil {
		return err
	}
	nnManifestWork := ktypes.NamespacedName{
		Name:      "test-manifest-work",
		Namespace: namespace,
	}
	manifestWork := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nnManifestWork.Name,
			Namespace: nnManifestWork.Namespace,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					{
						RawExtension: runtime.RawExtension{
							Raw: []byte(`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"test-namespace"}}`),
						},
					},
				},
			},
		},
	}
	_, err = workC.WorkV1().ManifestWorks(namespace).Create(ctx, manifestWork, metav1.CreateOptions{})
	return err
}

// deleteManifestWork deletes a ManifestWork in the given namespace
func deleteManifestWork(ctx context.Context, namespace string) error {
	workC, err := common.WorkClient(nil)
	if err != nil {
		return err
	}
	return workC.WorkV1().ManifestWorks(namespace).Delete(ctx, "test-manifest-work", metav1.DeleteOptions{})
}

// assertNamespace asserts that a namespace exists in the given cluster
func assertNamespace(ctx context.Context, cluster string, kClient client.Client) error {
	namespace := &corev1.Namespace{}
	namespaceName := "test-namespace"
	if err := kClient.Get(ctx, ktypes.NamespacedName{Name: namespaceName}, namespace); err != nil {
		if kerrs.IsNotFound(err) {
			utils.WarnError(err, "namespace %s not created yet in cluster '%s'", namespaceName, cluster)
			return err
		}
		utils.WarnError(err, "failed to fetch namespace %s from cluster '%s'", err, namespaceName, cluster)
		return errors.New("namespace not found")
	}
	utils.Info("Namespace %s is now created in cluster '%s'", namespaceName, cluster)
	return nil
}
