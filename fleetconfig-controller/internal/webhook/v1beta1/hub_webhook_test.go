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

// Package v1beta1 contains webhooks for v1beta1 resources
package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

var _ = Describe("Hub Webhook", func() {
	var (
		obj       *v1beta1.Hub
		oldObj    *v1beta1.Hub
		validator HubCustomValidator
	)

	BeforeEach(func() {
		obj = &v1beta1.Hub{}
		oldObj = &v1beta1.Hub{}
		validator = HubCustomValidator{client: k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	Context("When creating Hub under Validating Webhook", func() {
		It("Should allow creation with valid configuration", func() {
			By("setting up a valid Hub resource")
			obj.ObjectMeta.Name = "hub"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			obj.Spec.ClusterManager = &v1beta1.ClusterManager{
				FeatureGates: "AddonManagement=true",
			}

			By("validating the creation")
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny creation when neither ClusterManager nor SingletonControlPlane is specified", func() {
			By("setting up a Hub without ClusterManager or SingletonControlPlane")
			obj.ObjectMeta.Name = "hub"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			// Both ClusterManager and SingletonControlPlane are nil

			By("validating the creation should fail")
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either hub.clusterManager or hub.singletonControlPlane must be specified"))
		})
	})

	Context("When updating Hub under Validating Webhook", func() {
		BeforeEach(func() {
			oldObj.ObjectMeta.Name = "hub"
			oldObj.ObjectMeta.Namespace = "default"
			oldObj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			oldObj.Spec.ClusterManager = &v1beta1.ClusterManager{
				FeatureGates: "AddonManagement=true",
			}

			obj.ObjectMeta.Name = "hub"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			obj.Spec.ClusterManager = &v1beta1.ClusterManager{
				FeatureGates: "AddonManagement=true",
			}
		})

		It("Should allow valid updates", func() {
			By("updating Hub with valid changes - only allowed fields")
			// Update timeout (which is allowed)
			obj.Spec.Timeout = 600

			By("validating the update")
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should deny updates with invalid kubeconfig", func() {
			By("setting up invalid kubeconfig in the update")
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: false,
				// Missing SecretReference when InCluster is false
			}

			By("validating the update should fail")
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("either secretReference or inCluster must be specified"))
		})
	})

})
