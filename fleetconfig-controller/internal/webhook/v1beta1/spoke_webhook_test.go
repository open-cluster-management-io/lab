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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

var _ = Describe("Spoke Webhook", func() {
	var (
		obj       *v1beta1.Spoke
		oldObj    *v1beta1.Spoke
		validator SpokeCustomValidator
	)

	BeforeEach(func() {
		obj = &v1beta1.Spoke{}
		oldObj = &v1beta1.Spoke{}
		validator = SpokeCustomValidator{client: k8sClient}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	Context("When creating Spoke under Validating Webhook", func() {
		It("Should allow creation with valid configuration", func() {
			By("setting up a valid Spoke resource")
			obj.ObjectMeta.Name = "test-spoke"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.HubRef = v1beta1.HubRef{
				Name:      "hub",
				Namespace: "default",
			}
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			obj.Spec.Klusterlet.Mode = "Default"

			By("validating the creation")
			warnings, err := validator.ValidateCreate(ctx, obj)
			// Expect warnings about Hub not found, but no errors
			Expect(err).NotTo(HaveOccurred())
			if warnings != nil {
				Expect(warnings).To(ContainElement(ContainSubstring("hub not found")))
			}
		})

		It("Should deny creation when hosted mode lacks managedClusterKubeconfig", func() {
			By("setting up a Spoke in hosted mode without managedClusterKubeconfig")
			obj.ObjectMeta.Name = "test-spoke-hosted"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.HubRef = v1beta1.HubRef{
				Name:      "hub",
				Namespace: "default",
			}
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			obj.Spec.Klusterlet.Mode = "Hosted"
			// Missing ManagedClusterKubeconfig.SecretReference

			By("validating the creation should fail")
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("managedClusterKubeconfig.secretReference is required in hosted mode"))
		})
	})

	Context("When updating Spoke under Validating Webhook", func() {
		BeforeEach(func() {
			// Set up valid old and new objects
			oldObj.ObjectMeta.Name = "test-spoke"
			oldObj.ObjectMeta.Namespace = "default"
			oldObj.Spec.HubRef = v1beta1.HubRef{
				Name:      "hub",
				Namespace: "default",
			}
			oldObj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			oldObj.Spec.Klusterlet.Mode = "Default"

			obj.ObjectMeta.Name = "test-spoke"
			obj.ObjectMeta.Namespace = "default"
			obj.Spec.HubRef = v1beta1.HubRef{
				Name:      "hub",
				Namespace: "default",
			}
			obj.Spec.Kubeconfig = v1beta1.Kubeconfig{
				InCluster: true,
			}
			obj.Spec.Klusterlet.Mode = "Default"
		})

		It("Should allow valid updates", func() {
			By("updating Spoke with valid changes - only annotations are allowed")
			obj.Spec.Klusterlet.Annotations = map[string]string{
				"cluster.open-cluster-management.io/clusterset": "default",
			}

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
