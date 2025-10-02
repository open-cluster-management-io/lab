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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ktypes "k8s.io/apimachinery/pkg/types"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"open-cluster-management.io/ocm/pkg/operator/helpers/chart"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

		By("loading the fcc image into the spoke cluster")
		Expect(utils.DevspaceRunPipeline(tc.ctx, tc.spokeKubeconfig, "load-local", fcNamespace, "v1beta1")).To(Succeed())

		By("deploying fleetconfig-controller")
		Expect(utils.DevspaceRunPipeline(tc.ctx, tc.hubKubeconfig, "deploy-local", fcNamespace, "v1beta1")).To(Succeed())
	})

	AfterAll(func() {
		teardownTestEnvironment(tc)
	})

	// Tests Hub and Spoke operations with ResourceCleanup feature gate enabled, verifying:
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
			// NOTE: The Hub and Spoke CRs are created by devspace when the fleetconfig-controller chart is installed.
			//       Its configuration is defined via the fleetConfig values.
			ensureHubAndSpokesProvisioned(tc, hub, []*v1beta1.Spoke{spoke, hubAsSpoke}, nil)

			By("cloning the Hub and Spoke resources for further scenarios")
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

		It("should successfully upgrade spoke Klusterlet, with no kubeconfig secret", func() {
			By("confirming the kubeconfig secret is deleted")
			EventuallyWithOffset(1, func() error {
				secret := &corev1.Secret{}
				err := tc.kClient.Get(tc.ctx, types.NamespacedName{Namespace: fcNamespace, Name: spokeSecretName}, secret)
				if err != nil {
					return client.IgnoreNotFound(err)
				}
				utils.Info("kubeconfig secret still exists")
				return err
			}, 1*time.Minute, 1*time.Second).Should(Succeed())

			By("updating the klusterlet values and verifying that the upgrade is successful")
			EventuallyWithOffset(1, func() error {
				err := tc.kClient.Get(tc.ctx, v1beta1spokeNN, spokeClone)
				if err != nil {
					utils.WarnError(err, "failed to get spoke")
					return err
				}
				newDuration := 5 * time.Second
				spokeClone.Spec.Klusterlet.Values = &v1beta1.KlusterletChartConfig{
					KlusterletChartConfig: chart.KlusterletChartConfig{
						Klusterlet: chart.KlusterletConfig{
							WorkConfiguration: operatorv1.WorkAgentConfiguration{
								StatusSyncInterval: &metav1.Duration{
									Duration: newDuration,
								},
							},
						},
					},
				}
				err = tc.kClient.Update(tc.ctx, spokeClone)
				if err != nil {
					utils.WarnError(err, "failed to patch spoke")
					return err
				}
				klusterlet := &operatorv1.Klusterlet{}
				if err := tc.kClientSpoke.Get(tc.ctx, klusterletNN, klusterlet); err != nil {
					utils.WarnError(err, "failed to get klusterlet")
					return err
				}
				if klusterlet.Spec.WorkConfiguration == nil || klusterlet.Spec.WorkConfiguration.StatusSyncInterval == nil {
					err = errors.New("klusterlet status sync interval is nil")
					utils.WarnError(err, "klusterlet not upgraded")
					return err
				}
				if klusterlet.Spec.WorkConfiguration.StatusSyncInterval.Duration != newDuration {
					err = fmt.Errorf("wrong status sync interval found on Klusterlet. want: %s, got: %s", newDuration, klusterlet.Spec.WorkConfiguration.StatusSyncInterval.Duration)
					utils.WarnError(err, "failed to upgrade klusterlet")
					return err
				}
				return nil
			}, 3*time.Minute, 5*time.Second).Should(Succeed())
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

		It("should not allow changes to the Hub resource", func() {

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

				By("ensuring the spoke agent is uninstalled and ocm resources are cleaned up")
				deploy := &appsv1.Deployment{}
				err = tc.kClientSpoke.Get(tc.ctx, v1beta1fccAddOnAgentNN, deploy)
				if err != nil {
					if !kerrs.IsNotFound(err) {
						return err
					}
					utils.Info("fleetconfig-controller addon agent deleted successfully")
				} else {
					err := errors.New("fleetconfig-controller addon agent not deleted yet")
					utils.WarnError(err, "fleetconfig-controller addon agent still exists")
					return err
				}

				namespacesToDelete := []string{
					"open-cluster-management-agent",
					"open-cluster-management-agent-addon",
					"open-cluster-management",
					fcNamespace,
				}
				for _, n := range namespacesToDelete {
					spokeNs := &corev1.Namespace{}
					err = tc.kClientSpoke.Get(tc.ctx, ktypes.NamespacedName{Name: n}, spokeNs)
					if err != nil {
						if !kerrs.IsNotFound(err) {
							return err
						}
						utils.Info(fmt.Sprintf("namespace %s deleted successfully", n))
					} else {
						err := fmt.Errorf("namespace %s not deleted yet", n)
						utils.WarnError(err, "namespace still exists")
						return err
					}

				}

				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("deleting the Hub and ensuring that it isn't deleted until the ManifestWork is deleted")
			ExpectWithOffset(1, tc.kClient.Delete(tc.ctx, hubClone)).To(Succeed())
			EventuallyWithOffset(1, func() error {
				if err := tc.kClient.Get(tc.ctx, v1beta1hubNN, hubClone); err != nil {
					utils.WarnError(err, "failed to get Hub")
					return err
				}
				if hubClone.Status.Phase != v1beta1.Deleting {
					err := fmt.Errorf("expected %s, got %s", v1beta1.Deleting, hubClone.Status.Phase)
					utils.WarnError(err, "Hub deletion not started")
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
					v1beta1.HubUpgradeFailed: metav1.ConditionFalse,
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
					fmt.Println(hubClone.Status)
					return errors.New("hub still exists")
				},
			)
		})
	})
})
