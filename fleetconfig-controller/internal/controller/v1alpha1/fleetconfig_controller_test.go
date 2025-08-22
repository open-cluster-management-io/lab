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

package v1alpha1

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

	v1alpha1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
)

var (
	fc           *v1alpha1.FleetConfig
	fcReconciler *FleetConfigReconciler
	nn           types.NamespacedName
)

var _ = Describe("FleetConfig Controller", Ordered, func() {
	Context("When reconciling a FleetConfig", func() {
		ctx := context.Background()

		BeforeAll(func() {
			nn = types.NamespacedName{
				Name:      "test-fleetconfig",
				Namespace: "default",
			}
			fcReconciler = &FleetConfigReconciler{
				Client: kClient,
				Log:    logr.Logger{},
				Scheme: kClient.Scheme(),
			}
			fc = &v1alpha1.FleetConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
				},
				Spec: v1alpha1.FleetConfigSpec{
					Hub: v1alpha1.Hub{
						Kubeconfig: v1alpha1.Kubeconfig{
							InCluster: true,
						},
					},
					Spokes: []v1alpha1.Spoke{},
				},
			}
		})

		It("Should create a FleetConfig", func() {
			Expect(kClient.Create(ctx, fc)).To(Succeed())
		})

		It("Should add a finalizer to the FleetConfig", func() {
			By("Reconciling the FleetConfig")
			Expect(reconcileFleetConfig(ctx)).To(Succeed())

			By("Verifying the FleetConfig's finalizer")
			Expect(kClient.Get(ctx, nn, fc)).To(Succeed())
			Expect(fc.Finalizers[0]).To(Equal(v1alpha1.FleetConfigFinalizer),
				"FleetConfig %s wasn't given a finalizer", nn.Name)
		})

		It("Should initialize the FleetConfig", func() {
			By("Reconciling the FleetConfig")
			Expect(reconcileFleetConfig(ctx)).To(Succeed())

			By("Verifying the FleetConfig's phase and conditions")
			Expect(kClient.Get(ctx, nn, fc)).To(Succeed())
			Expect(fc.Status.Phase).To(Equal(v1alpha1.FleetConfigStarting),
				"FleetConfig %s is not in the Initializing phase", nn.Name)
			Expect(assertConditions(fc.Status.Conditions, map[string]metav1.ConditionStatus{
				v1alpha1.FleetConfigHubInitialized: metav1.ConditionFalse,
				v1alpha1.FleetConfigCleanupFailed:  metav1.ConditionFalse,
			})).To(Succeed())
		})

		// cannot test provisioning without an e2e test

		It("Should delete the FleetConfig", func() {
			By("Deleting the FleetConfig")
			Expect(kClient.Delete(ctx, fc)).To(Succeed())
			Eventually(func() error {
				err := kClient.Get(ctx, nn, fc)
				if kerrs.IsNotFound(err) {
					return nil
				}
				return err
			}, 5*time.Minute).Should(Succeed())
		})
	})
})

func reconcileFleetConfig(ctx context.Context) error {
	_, err := fcReconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: nn,
	})
	return err
}

// assertConditions asserts that two sets of conditions match.
func assertConditions(conditions []v1alpha1.Condition, expected map[string]metav1.ConditionStatus) error {
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
