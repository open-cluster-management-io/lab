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

package v1beta1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

var (
	spoke           *v1beta1.Spoke
	spokeReconciler *SpokeReconciler
	spokeNN         types.NamespacedName
	testHub         *v1beta1.Hub
	testHubNN       types.NamespacedName
)

var _ = Describe("Spoke Controller", Ordered, func() {
	Context("When reconciling a Spoke", func() {
		ctx := context.Background()

		BeforeAll(func() {
			// Create a test Hub first since Spoke references it
			testHubNN = types.NamespacedName{
				Name:      "test-hub-2",
				Namespace: "default",
			}
			testHub = &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testHubNN.Name,
					Namespace: testHubNN.Namespace,
				},
				Spec: v1beta1.HubSpec{
					Kubeconfig: v1beta1.Kubeconfig{
						InCluster: true,
					},
					CreateNamespace: true,
					Timeout:         300,
					LogVerbosity:    0,
				},
			}
			Expect(k8sClient.Create(ctx, testHub)).To(Succeed())

			spokeNN = types.NamespacedName{
				Name:      "hub-as-spoke",
				Namespace: "default",
			}
			spokeReconciler = &SpokeReconciler{
				Client:       k8sClient,
				Log:          logr.Logger{},
				Scheme:       k8sClient.Scheme(),
				InstanceType: v1beta1.InstanceTypeManager,
			}
			spoke = &v1beta1.Spoke{
				ObjectMeta: metav1.ObjectMeta{
					Name:      spokeNN.Name,
					Namespace: spokeNN.Namespace,
				},
				Spec: v1beta1.SpokeSpec{
					HubRef: v1beta1.HubRef{
						Name:      testHubNN.Name,
						Namespace: testHubNN.Namespace,
					},
					Kubeconfig: v1beta1.Kubeconfig{
						InCluster: true,
					},
					CreateNamespace: true,
					SyncLabels:      false,
					Timeout:         300,
					LogVerbosity:    0,
				},
			}
		})

		AfterAll(func() {
			// Clean up test Hub
			By("Cleaning up test Hub")
			err := k8sClient.Delete(ctx, testHub)
			if err != nil && !kerrs.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("Should create a Spoke", func() {
			Expect(k8sClient.Create(ctx, spoke)).To(Succeed())
		})

		It("Should add a finalizer to the Spoke", func() {
			By("Reconciling the Spoke")
			Expect(reconcileSpoke(ctx)).To(Succeed())

			By("Verifying the Spoke's finalizer")
			Expect(k8sClient.Get(ctx, spokeNN, spoke)).To(Succeed())
			Expect(spoke.Finalizers).To(ContainElement(v1beta1.SpokeCleanupFinalizer),
				"Spoke %s wasn't given a finalizer", spokeNN.Name)
		})

		It("Should initialize the Spoke", func() {
			By("Reconciling the Spoke")
			Expect(reconcileSpoke(ctx)).To(Succeed())

			By("Verifying the Spoke's phase and conditions")
			Expect(k8sClient.Get(ctx, spokeNN, spoke)).To(Succeed())
			Expect(spoke.Status.Phase).To(Equal(v1beta1.SpokeJoining),
				"Spoke %s is not in the Joining phase", spokeNN.Name)
			Expect(assertSpokeConditions(spoke.Status.Conditions, map[string]metav1.ConditionStatus{
				v1beta1.SpokeJoined:      metav1.ConditionFalse,
				v1beta1.CleanupFailed:    metav1.ConditionFalse,
				v1beta1.AddonsConfigured: metav1.ConditionFalse,
				v1beta1.PivotComplete:    metav1.ConditionFalse,
				v1beta1.KlusterletSynced: metav1.ConditionFalse,
			})).To(Succeed())
		})

		// cannot test full provisioning without an e2e test

		It("Should delete the Spoke", func() {
			By("Deleting the Spoke")
			Expect(k8sClient.Delete(ctx, spoke)).To(Succeed())
			Eventually(func() error {
				err := k8sClient.Get(ctx, spokeNN, spoke)
				if kerrs.IsNotFound(err) {
					return nil
				}
				return err
			}, 5*time.Minute).Should(Succeed())
		})
	})
})

func reconcileSpoke(ctx context.Context) error {
	_, err := spokeReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: spokeNN,
	})
	return err
}

// assertSpokeConditions asserts that two sets of conditions match.
func assertSpokeConditions(conditions []v1beta1.Condition, expected map[string]metav1.ConditionStatus) error {
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
