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

	v1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

var (
	hub           *v1beta1.Hub
	hubReconciler *HubReconciler
	hubNN         types.NamespacedName
)

var _ = Describe("Hub Controller", Ordered, func() {
	Context("When reconciling a Hub", func() {
		ctx := context.Background()

		BeforeAll(func() {
			hubNN = types.NamespacedName{
				Name:      "test-hub",
				Namespace: "default",
			}
			hubReconciler = &HubReconciler{
				Client: k8sClient,
				Log:    logr.Logger{},
				Scheme: k8sClient.Scheme(),
			}
			hub = &v1beta1.Hub{
				ObjectMeta: metav1.ObjectMeta{
					Name:      hubNN.Name,
					Namespace: hubNN.Namespace,
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
		})

		It("Should create a Hub", func() {
			Expect(k8sClient.Create(ctx, hub)).To(Succeed())
		})

		It("Should add a finalizer to the Hub", func() {
			By("Reconciling the Hub")
			Expect(reconcileHub(ctx)).To(Succeed())

			By("Verifying the Hub's finalizer")
			Expect(k8sClient.Get(ctx, hubNN, hub)).To(Succeed())
			Expect(hub.Finalizers).To(ContainElement(v1beta1.HubCleanupFinalizer),
				"Hub %s wasn't given a finalizer", hubNN.Name)
		})

		It("Should initialize the Hub", func() {
			By("Reconciling the Hub")
			Expect(reconcileHub(ctx)).To(Succeed())

			By("Verifying the Hub's phase and conditions")
			Expect(k8sClient.Get(ctx, hubNN, hub)).To(Succeed())
			Expect(hub.Status.Phase).To(Equal(v1beta1.HubStarting),
				"Hub %s is not in the Initializing phase", hubNN.Name)
			Expect(assertHubConditions(hub.Status.Conditions, map[string]metav1.ConditionStatus{
				v1beta1.HubInitialized:   metav1.ConditionFalse,
				v1beta1.CleanupFailed:    metav1.ConditionFalse,
				v1beta1.AddonsConfigured: metav1.ConditionFalse,
			})).To(Succeed())
		})

		// cannot test full provisioning without an e2e test

		It("Should delete the Hub", func() {
			By("Deleting the Hub")
			Expect(k8sClient.Delete(ctx, hub)).To(Succeed())
			Eventually(func() error {
				err := k8sClient.Get(ctx, hubNN, hub)
				if kerrs.IsNotFound(err) {
					return nil
				}
				return err
			}, 5*time.Minute).Should(Succeed())
		})
	})
})

func reconcileHub(ctx context.Context) error {
	_, err := hubReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: hubNN,
	})
	return err
}

// assertHubConditions asserts that two sets of conditions match.
func assertHubConditions(conditions []v1beta1.Condition, expected map[string]metav1.ConditionStatus) error {
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
