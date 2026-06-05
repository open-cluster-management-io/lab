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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
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
	// 2. Addon configuration on hub and installation on spoke; ClusterManager.Values applied on the hub
	// 3. ManifestWork creation in hub-as-spoke namespace and namespace creation validation
	// 4. Propagation of feature gate modifications to the ClusterManager during active operation
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
			ensureAddonCreated(tc, 0, spoke.Status.ManagedClusterName)
		})

		It("should reconcile innocuous ClusterManager chart values onto the hub", func() {
			const ocmNamespace = "open-cluster-management"
			clusterManagerOperatorNN := ktypes.NamespacedName{Name: "cluster-manager", Namespace: ocmNamespace}
			wantCPU := resource.MustParse("3m")
			wantMemory := resource.MustParse("20Mi")

			By("setting Hub.spec.clusterManager.values (operator resources) and waiting for the cluster-manager deployment to match")
			EventuallyWithOffset(1, func() error {
				h := &v1beta1.Hub{}
				if err := tc.kClient.Get(tc.ctx, v1beta1hubNN, h); err != nil {
					return err
				}
				if h.Spec.ClusterManager == nil {
					return fmt.Errorf("hub.spec.clusterManager is nil")
				}
				h.Spec.ClusterManager.Values = &v1beta1.ClusterManagerChartConfig{
					ClusterManagerChartConfig: chart.ClusterManagerChartConfig{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    wantCPU,
								corev1.ResourceMemory: wantMemory,
							},
						},
						// Hub CRD validates embedded operator types: ResourceRequirement.type must be
						// Default | BestEffort | ResourceRequirement, never empty.
						ClusterManager: chart.ClusterManagerConfig{
							ResourceRequirement: operatorv1.ResourceRequirement{
								Type: operatorv1.ResourceQosClassDefault,
							},
						},
					},
				}
				if err := tc.kClient.Update(tc.ctx, h); err != nil {
					return err
				}
				return nil
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			EventuallyWithOffset(1, func() error {
				dep := &appsv1.Deployment{}
				if err := tc.kClient.Get(tc.ctx, clusterManagerOperatorNN, dep); err != nil {
					return err
				}
				for _, c := range dep.Spec.Template.Spec.Containers {
					if c.Name != "registration-operator" {
						continue
					}
					if c.Resources.Requests == nil {
						return fmt.Errorf("registration-operator container has no resource requests")
					}
					gotCPU, ok := c.Resources.Requests[corev1.ResourceCPU]
					if !ok || gotCPU.Cmp(wantCPU) != 0 {
						return fmt.Errorf("registration-operator cpu request: want %s, got %v", wantCPU.String(), c.Resources.Requests[corev1.ResourceCPU])
					}
					gotMem, ok := c.Resources.Requests[corev1.ResourceMemory]
					if !ok || gotMem.Cmp(wantMemory) != 0 {
						return fmt.Errorf("registration-operator memory request: want %s, got %v", wantMemory.String(), c.Resources.Requests[corev1.ResourceMemory])
					}
					return nil
				}
				return fmt.Errorf("registration-operator container not found on cluster-manager deployment")
			}, 5*time.Minute, 5*time.Second).Should(Succeed())

			By("refreshing hubClone so a later Delete uses the current resourceVersion")
			Expect(tc.kClient.Get(tc.ctx, v1beta1hubNN, hubClone)).To(Succeed())
		})

		It("should verify initial addon variables are correctly resolved", func() {
			By("verifying that the initial FOO and CLUSTER_NAME variables are resolved in deployed resources")
			ensureAddonVariablesResolved(tc, 0, spoke.Status.ManagedClusterName, "initial-foo-value")
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
				err := tc.kClient.Get(tc.ctx, ktypes.NamespacedName{Namespace: fcNamespace, Name: spokeSecretName}, secret)
				if err != nil {
					return client.IgnoreNotFound(err)
				}
				utils.Info("kubeconfig secret still exists")
				return err
			}, 1*time.Minute, 1*time.Second).Should(Succeed())

			By("updating the klusterlet values and annotations, and verifying that the upgrade is successful")
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
				// Update annotations - change existing ones and add a new one
				spokeClone.Spec.Klusterlet.Annotations = map[string]string{
					"foo": "updated-bar",
					"baz": "updated-quux",
					"new": "annotation",
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

			By("verifying that the annotation updates are propagated to the ManagedCluster")
			EventuallyWithOffset(1, func() error {
				kcfg, err := os.ReadFile(tc.hubKubeconfig)
				if err != nil {
					return err
				}
				clusterC, err := common.ClusterClient(kcfg)
				if err != nil {
					return err
				}
				managedCluster, err := clusterC.ClusterV1().ManagedClusters().Get(tc.ctx, spoke.Status.ManagedClusterName, metav1.GetOptions{})
				if err != nil {
					utils.WarnError(err, "failed to get ManagedCluster")
					return err
				}
				annotations := managedCluster.GetAnnotations()
				expectedAnnotations := map[string]string{
					fmt.Sprintf("%s/foo", klusterletAnnotationPrefix): "updated-bar",
					fmt.Sprintf("%s/baz", klusterletAnnotationPrefix): "updated-quux",
					fmt.Sprintf("%s/new", klusterletAnnotationPrefix): "annotation",
				}
				for key, expectedValue := range expectedAnnotations {
					actualValue, ok := annotations[key]
					if !ok {
						return fmt.Errorf("expected annotation %s not found", key)
					}
					if actualValue != expectedValue {
						return fmt.Errorf("annotation %s has wrong value. want: %s, got: %s", key, expectedValue, actualValue)
					}
				}
				return nil
			}, 1*time.Minute, 1*time.Second).Should(Succeed())
		})

		It("should successfully create a namespace in the hub-as-spoke cluster", func() {

			By("creating a ManifestWork in the hub-as-spoke cluster namespace")
			EventuallyWithOffset(1, func() error {
				return createManifestWork(tc.ctx, hubAsSpoke.Status.ManagedClusterName)
			}, 1*time.Minute, 1*time.Second).Should(Succeed())

			By("ensuring the test-namespace namespace is created on the hub")
			EventuallyWithOffset(1, func() error {
				return assertNamespace(tc.ctx, hubAsSpoke.Status.ManagedClusterName, tc.kClient)
			}, 2*time.Minute, 10*time.Second).Should(Succeed())
		})

		It("should propagate Hub feature gate changes to the ClusterManager", func() {
			clusterManagerNN := ktypes.NamespacedName{Name: "cluster-manager"}

			By("capturing the ClusterManager feature gates before the change")
			cmBefore := &operatorv1.ClusterManager{}
			Expect(tc.kClient.Get(tc.ctx, clusterManagerNN, cmBefore)).To(Succeed())
			var workBefore []operatorv1.FeatureGate
			if cmBefore.Spec.WorkConfiguration != nil {
				workBefore = cmBefore.Spec.WorkConfiguration.FeatureGates
			}
			Expect(featureGateHasMode(workBefore, "ManifestWorkReplicaSet", operatorv1.FeatureGateModeTypeEnable)).
				To(BeTrue(), "ManifestWorkReplicaSet should be enabled before the change")

			By("patching the Hub's feature gates")
			hub, err := utils.GetHub(tc.ctx, tc.kClient, v1beta1hubNN)
			Expect(err).NotTo(HaveOccurred())
			patchFeatureGates := "DefaultClusterSet=true,ManifestWorkReplicaSet=false,ResourceCleanup=false"
			Expect(utils.UpdateHubFeatureGates(tc.ctx, tc.kClient, hub, patchFeatureGates)).To(Succeed())

			By("verifying the feature gate change is propagated to the ClusterManager")
			EventuallyWithOffset(1, func() error {
				cm := &operatorv1.ClusterManager{}
				if err := tc.kClient.Get(tc.ctx, clusterManagerNN, cm); err != nil {
					return err
				}
				var work, registration []operatorv1.FeatureGate
				if cm.Spec.WorkConfiguration != nil {
					work = cm.Spec.WorkConfiguration.FeatureGates
				}
				if cm.Spec.RegistrationConfiguration != nil {
					registration = cm.Spec.RegistrationConfiguration.FeatureGates
				}
				if featureGateHasMode(work, "ManifestWorkReplicaSet", operatorv1.FeatureGateModeTypeEnable) {
					return fmt.Errorf("ManifestWorkReplicaSet still enabled on workConfiguration: %+v", work)
				}
				if !featureGateHasMode(registration, "ResourceCleanup", operatorv1.FeatureGateModeTypeDisable) {
					return fmt.Errorf("ResourceCleanup not disabled on registrationConfiguration: %+v", registration)
				}
				return nil
			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("re-enabling the ManifestWorkReplicaSet feature gate")
			hub, err = utils.GetHub(tc.ctx, tc.kClient, v1beta1hubNN)
			Expect(err).NotTo(HaveOccurred())
			patchFeatureGates = "DefaultClusterSet=true,ManifestWorkReplicaSet=true,ResourceCleanup=true"
			Expect(utils.UpdateHubFeatureGates(tc.ctx, tc.kClient, hub, patchFeatureGates)).To(Succeed())
		})

		It("should update addon variable values and verify resources are updated", func() {
			By("updating the spoke addon config to change the FOO variable value")
			EventuallyWithOffset(1, func() error {
				err := tc.kClient.Get(tc.ctx, v1beta1spokeNN, spokeClone)
				if err != nil {
					utils.WarnError(err, "failed to get spoke")
					return err
				}

				// Find the test-addon in the addons list and update the FOO variable value
				for i, addon := range spokeClone.Spec.AddOns {
					if addon.ConfigName == "test-addon" {
						if spokeClone.Spec.AddOns[i].DeploymentConfig == nil {
							spokeClone.Spec.AddOns[i].DeploymentConfig = &addonv1alpha1.AddOnDeploymentConfigSpec{}
						}
						// Update FOO variable value (keep CLUSTER_NAME the same)
						spokeClone.Spec.AddOns[i].DeploymentConfig.CustomizedVariables = []addonv1alpha1.CustomizedVariable{
							{
								Name:  "FOO",
								Value: "updated-foo-value",
							},
						}
						break
					}
				}

				err = tc.kClient.Update(tc.ctx, spokeClone)
				if err != nil {
					utils.WarnError(err, "failed to update spoke")
					return err
				}
				return nil
			}, 1*time.Minute, 5*time.Second).Should(Succeed())

			By("verifying that the updated FOO variable is correctly resolved in deployed resources")
			ensureAddonVariablesResolved(tc, 0, spoke.Status.ManagedClusterName, "updated-foo-value")
		})

		It("should update addon template version and verify new resources are deployed", func() {
			updateHubAddon(tc, hub)
			ensureAddonCreated(tc, 1, spoke.Status.ManagedClusterName)

			By("verifying that the new addon template (v2.0.0) has variables correctly resolved")
			ensureAddonVariablesResolved(tc, 1, spoke.Status.ManagedClusterName, "updated-foo-value")
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
				_, err = clusterC.ClusterV1().ManagedClusters().Get(tc.ctx, spoke.Status.ManagedClusterName, metav1.GetOptions{})
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
				err = tc.kClient.Get(tc.ctx, ktypes.NamespacedName{Name: spoke.Status.ManagedClusterName}, ns)
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

			By("deleting the Hub and ensuring that the Hub and Spoke aren't deleted until the ManifestWork is deleted")
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
				if err := tc.kClient.Get(tc.ctx, v1beta1hubAsSpokeNN, hubAsSpokeClone); err != nil {
					utils.WarnError(err, "failed to get Spoke")
					return err
				}
				if hubAsSpokeClone.Status.Phase != v1beta1.Deleting {
					err := fmt.Errorf("expected %s, got %s", v1beta1.Deleting, hubAsSpokeClone.Status.Phase)
					utils.WarnError(err, "Spoke deletion not started")
					return err
				}
				conditions := make([]metav1.Condition, len(hubAsSpokeClone.Status.Conditions))
				for i, c := range hubAsSpokeClone.Status.Conditions {
					conditions[i] = c.Condition
				}
				if err := utils.AssertConditions(conditions, map[string]metav1.ConditionStatus{
					v1beta1.SpokeJoined:      metav1.ConditionTrue,
					v1beta1.CleanupFailed:    metav1.ConditionTrue,
					v1beta1.AddonsConfigured: metav1.ConditionTrue,
					v1beta1.KlusterletSynced: metav1.ConditionTrue,
					v1beta1.PivotComplete:    metav1.ConditionTrue,
				}); err != nil {
					utils.WarnError(err, "Hub deletion not blocked")
					return err
				}
				return nil

			}, 5*time.Minute, 10*time.Second).Should(Succeed())

			By("deleting the manifest work from the hub")
			ExpectWithOffset(1, deleteManifestWork(tc.ctx, hubAsSpoke.Status.ManagedClusterName)).To(Succeed())

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
