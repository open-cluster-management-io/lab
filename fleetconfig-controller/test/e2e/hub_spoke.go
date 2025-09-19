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

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/test/utils"
)

var _ = Describe("hub and spoke", Label("v1beta1"), Serial, Ordered, func() {

	var (
		tc       *E2EContext
		hub      = &v1beta1.Hub{}
		hubClone = &v1beta1.Hub{}
		spoke    = &v1beta1.Spoke{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1spokeNN.Name,
				Namespace: v1beta1spokeNN.Namespace,
			},
		}
		hubAsSpoke = &v1beta1.Spoke{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1hubAsSpokeNN.Name,
				Namespace: v1beta1hubAsSpokeNN.Namespace,
			},
		}
		spokeClone      = &v1beta1.Spoke{}
		hubAsSpokeClone = &v1beta1.Spoke{}
	)

	BeforeAll(func() {
		tc = setupTestEnvironment()

		By("deploying fleetconfig")
		Expect(utils.DevspaceRunPipeline(tc.ctx, tc.hubKubeconfig, "deploy-local", fcNamespace, "v1beta1")).To(Succeed())
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
	// 8. Automatic ManifestWork cleanup when Hub and Spoke resource are deleted
	Context("deploy and teardown Hub and Spokes with ResourceCleanup feature gate enabled", func() {

		It("should join the spoke and hub-as-spoke clusters to the hub", func() {
			// NOTE: The FleetConfig CR is created by devspace when the fleetconfig-controller chart is installed.
			//       Its configuration is defined via the fleetConfig values.
			ensureHubAndSpokesProvisioned(tc, hub, []*v1beta1.Spoke{spoke, hubAsSpoke}, nil)

			By("cloning the FleetConfig resources for further scenarios")
			err := utils.CloneHub(hub, hubClone)
			Expect(err).NotTo(HaveOccurred())
			err = utils.CloneSpoke(spoke, spokeClone)
			Expect(err).NotTo(HaveOccurred())
			err = utils.CloneSpoke(hubAsSpoke, hubAsSpokeClone)
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

			By("failing to patch the Hub's feature gates")
			hub, err := utils.GetHub(tc.ctx, tc.kClient, v1beta1hubNN)
			Expect(err).NotTo(HaveOccurred())
			patchFeatureGates := "DefaultClusterSet=true,ManifestWorkReplicaSet=true,ResourceCleanup=false"
			Expect(utils.UpdateHubFeatureGates(tc.ctx, tc.kClient, hub, patchFeatureGates)).ToNot(Succeed())
		})

		It("should update an addon and make sure its propagated to the spoke", func() {
			updateHubAddon(tc, hub)
			ensureAddonCreated(tc, 1)
		})

		It("should delete a Spoke", func() {
			err := tc.kClient.Delete(tc.ctx, spoke)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should clean up the hub cluster", func() {

			By("ensuring the spoke is deregistered properly")
			EventuallyWithOffset(1, func() error {
				By("ensuring the Spoke resource is deleted")
				err := tc.kClient.Get(tc.ctx, v1beta1spokeNN, spoke)
				if err == nil {
					return errors.New("spoke still exists")
				}
				if err != nil && !kerrs.IsNotFound(err) {
					return err
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
				return nil
			})

			By("deleting the Hub and ensuring that it isn't deleted until the ManifestWork is deleted")
			ExpectWithOffset(1, tc.kClient.Delete(tc.ctx, hubClone)).To(Succeed())
			EventuallyWithOffset(1, func() error {
				if err := tc.kClient.Get(tc.ctx, v1beta1hubNN, hubClone); err != nil {
					utils.WarnError(err, "failed to get FleetConfig")
					return err
				}
				if hubClone.Status.Phase != v1beta1.Deleting {
					err := fmt.Errorf("expected %s, got %s", v1beta1.Deleting, hubClone.Status.Phase)
					utils.WarnError(err, "FleetConfig deletion not started")
					return err
				}
				conditions := make([]metav1.Condition, len(hubClone.Status.Conditions))
				for i, c := range hubClone.Status.Conditions {
					conditions[i] = c.Condition
				}
				if err := utils.AssertConditions(conditions, map[string]metav1.ConditionStatus{
					v1beta1.HubInitialized:   metav1.ConditionTrue,
					v1beta1.CleanupFailed:    metav1.ConditionTrue,
					v1beta1.AddonsConfigured: metav1.ConditionTrue,
				}); err != nil {
					utils.WarnError(err, "Hub deletion not blocked")
					return err
				}
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("deleting the manifest work from the hub")
			ExpectWithOffset(1, deleteManifestWork(tc.ctx, hubAsSpokeName)).To(Succeed())

			By("ensuring the Hub and hub-as-spoke Spoke are deleted once the ManifestWork is deleted")
			ensureResourceDeleted(
				func() error {
					err := tc.kClient.Get(tc.ctx, v1beta1hubAsSpokeNN, hubAsSpokeClone)
					if kerrs.IsNotFound(err) {
						utils.Info("Spoke deleted successfully")
						return nil
					} else if err != nil {
						utils.WarnError(err, "failed to check if Spoke was deleted")
					}
					return errors.New("spoke still exists")
				},
			)
			ensureResourceDeleted(
				func() error {
					err := tc.kClient.Get(tc.ctx, v1beta1hubNN, hubClone)
					if kerrs.IsNotFound(err) {
						utils.Info("Hub deleted successfully")
						return nil
					} else if err != nil {
						utils.WarnError(err, "failed to check if Hub was deleted")
					}
					return errors.New("hub still exists")
				},
			)
		})
	})
})
