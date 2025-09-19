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

package e2e

import (
	"errors"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	operatorv1 "open-cluster-management.io/api/operator/v1"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/test/utils"
)

var _ = Describe("fleetconfig", Label("v1alpha1"), Serial, Ordered, func() {

	var (
		tc      *E2EContext
		fc      = &v1alpha1.FleetConfig{}
		fcClone = &v1alpha1.FleetConfig{}
	)

	BeforeAll(func() {
		tc = setupTestEnvironment()

		By("deploying fleetconfig")
		Expect(utils.DevspaceRunPipeline(tc.ctx, tc.hubKubeconfig, "deploy-local", fcNamespace, "v1alpha1")).To(Succeed())
		Expect(deployV1alpha1FleetConfig(tc)).To(Succeed())
	})

	AfterAll(func() {
		teardownTestEnvironment(tc)
	})

	// Tests FleetConfig operations with ResourceCleanup feature gate enabled, verifying:
	// 1. Cluster joining (spoke and hub-as-spoke) to the hub
	// 2. Addon configuration on hub and installation on spoke
	// 3. ManifestWork creation in hub-as-spoke namespace and namespace creation validation
	// 4. Prevention of feature gate modifications during active operation
	// 5. Addon update and propagation
	// 6. Spoke removal with proper deregistration from hub
	// 7. ManagedCluster and namespace deletion validation
	// 8. Automatic ManifestWork cleanup when FleetConfig resource is deleted
	Context("deploy and teardown FleetConfig with ResourceCleanup feature gate enabled", func() {

		It("should join the spoke and hub-as-spoke clusters to the hub", func() {
			// NOTE: The FleetConfig CR is created by devspace when the fleetconfig-controller chart is installed.
			//       Its configuration is defined via the fleetConfig values.
			ensureFleetConfigProvisioned(tc, fc, nil)

			By("cloning the FleetConfig resource for further scenarios")
			err := utils.CloneFleetConfig(fc, fcClone)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should verify addons configured on the hub and enabled on the spoke", func() {
			ensureAddonCreated(tc, 0)
		})

		It("should verify spoke cluster annotations", func() {
			EventuallyWithOffset(1, func() error {
				klusterlet := &operatorv1.Klusterlet{}
				if err := tc.kClientSpoke.Get(tc.ctx, klusterletNN, klusterlet); err != nil {
					return err
				}
				if err := assertKlusterletAnnotation(klusterlet, "foo", "bar"); err != nil {
					return err
				}
				if err := assertKlusterletAnnotation(klusterlet, "baz", "quux"); err != nil {
					return err
				}
				return nil
			}, 1*time.Minute, 1*time.Second).Should(Succeed())
		})

		It("should successfully create a namespace in the hub-as-spoke cluster", func() {

			By("creating a ManifestWork in the hub-as-spoke cluster namespace")
			EventuallyWithOffset(1, func() error {
				return createManifestWork(tc.ctx, hubAsSpokeName)
			}, 1*time.Minute, 1*time.Second).Should(Succeed())

			By("ensuring the test-namespace namespace is created on the hub")
			EventuallyWithOffset(1, func() error {
				return assertNamespace(tc.ctx, hubAsSpokeName, tc.kClient)
			}, 2*time.Minute, 10*time.Second).Should(Succeed())
		})

		It("should not allow changes to the FleetConfig resource", func() {

			By("failing to patch the FleetConfig's feature gates")
			fc, err := utils.GetFleetConfig(tc.ctx, tc.kClient, v1alpha1fleetConfigNN)
			Expect(err).NotTo(HaveOccurred())
			patchFeatureGates := "DefaultClusterSet=true,ManifestWorkReplicaSet=true,ResourceCleanup=false"
			Expect(utils.UpdateFleetConfigFeatureGates(tc.ctx, tc.kClient, fc, patchFeatureGates)).ToNot(Succeed())
		})

		It("should update an addon and make sure its propagated to the spoke", func() {
			updateFleetConfigAddon(tc, fc)
			ensureAddonCreated(tc, 1)
		})

		It("should remove a spoke from the hub", func() {
			removeSpokeFromHub(tc, fc)
		})

		It("should clean up the hub cluster", func() {

			By("ensuring the spoke is deregistered properly")
			EventuallyWithOffset(1, func() error {
				if err := tc.kClient.Get(tc.ctx, v1alpha1fleetConfigNN, fc); err != nil {
					return err
				}
				if len(fc.Status.JoinedSpokes) > 1 {
					return errors.New("spoke has not been unjoined")
				}

				kcfg, err := os.ReadFile(tc.hubKubeconfig)
				if err != nil {
					return err
				}
				clusterC, err := common.ClusterClient(kcfg)
				if err != nil {
					return err
				}

				By("ensuring the ManagedCluster is deleted")
				_, err = clusterC.ClusterV1().ManagedClusters().Get(tc.ctx, spokeName, metav1.GetOptions{})
				if err != nil {
					if !kerrs.IsNotFound(err) {
						return err
					}
					utils.Info("ManagedCluster successfully deleted")
				} else {
					err := errors.New("ManagedCluster not deleted yet")
					utils.WarnError(err, "ManagedCluster still exists")
					return err
				}

				By("ensuring the ManagedCluster namespace is deleted")
				ns := &corev1.Namespace{}
				err = tc.kClient.Get(tc.ctx, ktypes.NamespacedName{Name: spokeName}, ns)
				if err != nil {
					if !kerrs.IsNotFound(err) {
						return err
					}
					utils.Info("Managed Cluster namespace deleted successfully")
				} else {
					err := errors.New("ManagedCluster namespace not deleted yet")
					utils.WarnError(err, "ManagedCluster namespace still exists")
					return err
				}

				By("ensuring the FleetConfig is in the expected state")
				conditions := make([]metav1.Condition, len(fc.Status.Conditions))
				for i, c := range fc.Status.Conditions {
					conditions[i] = c.Condition
				}
				if err = utils.AssertConditions(conditions, map[string]metav1.ConditionStatus{
					v1alpha1.FleetConfigHubInitialized:                         metav1.ConditionTrue,
					v1alpha1.FleetConfigCleanupFailed:                          metav1.ConditionFalse,
					v1alpha1.FleetConfigAddonsConfigured:                       metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-joined", hubAsSpokeName):     metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-joined", spokeName):          metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-addons-enabled", spokeName):  metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-unjoined", spokeName):        metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-addons-disabled", spokeName): metav1.ConditionTrue,
				}); err != nil {
					utils.WarnError(err, "Spoke does not have expected condition")
					return err
				}
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("deleting the FleetConfig and ensuring that it isn't deleted until the ManifestWork is deleted")
			ExpectWithOffset(1, tc.kClient.Delete(tc.ctx, fcClone)).To(Succeed())
			EventuallyWithOffset(1, func() error {
				if err := tc.kClient.Get(tc.ctx, v1alpha1fleetConfigNN, fcClone); err != nil {
					utils.WarnError(err, "failed to get FleetConfig")
					return err
				}
				if fcClone.Status.Phase != v1alpha1.FleetConfigDeleting {
					err := fmt.Errorf("expected %s, got %s", v1alpha1.FleetConfigDeleting, fcClone.Status.Phase)
					utils.WarnError(err, "FleetConfig deletion not started")
					return err
				}
				conditions := make([]metav1.Condition, len(fcClone.Status.Conditions))
				for i, c := range fcClone.Status.Conditions {
					conditions[i] = c.Condition
				}
				if err := utils.AssertConditions(conditions, map[string]metav1.ConditionStatus{
					v1alpha1.FleetConfigHubInitialized:                         metav1.ConditionTrue,
					v1alpha1.FleetConfigCleanupFailed:                          metav1.ConditionTrue,
					v1alpha1.FleetConfigAddonsConfigured:                       metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-joined", hubAsSpokeName):     metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-joined", spokeName):          metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-addons-enabled", spokeName):  metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-unjoined", spokeName):        metav1.ConditionTrue,
					fmt.Sprintf("spoke-cluster-%s-addons-disabled", spokeName): metav1.ConditionTrue,
				}); err != nil {
					utils.WarnError(err, "FleetConfig deletion not blocked")
					return err
				}
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("deleting the manifest work from the hub")
			ExpectWithOffset(1, deleteManifestWork(tc.ctx, hubAsSpokeName)).To(Succeed())

			By("ensuring the FleetConfig is deleted once the ManifestWork is deleted")
			ensureResourceDeleted(
				func() error {
					err := tc.kClient.Get(tc.ctx, v1alpha1fleetConfigNN, fcClone)
					if kerrs.IsNotFound(err) {
						utils.Info("FleetConfig deleted successfully")
						return nil
					} else if err != nil {
						utils.WarnError(err, "failed to check if FleetConfig was deleted")
					}
					return errors.New("FleetConfig still exists")
				},
			)
		})
	})
})
