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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	v1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/test"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx               context.Context
	cancel            context.CancelFunc
	testEnv           *envtest.Environment
	cfg               *rest.Config
	k8sClient         client.Client
	testConfig        *test.Config
	err               error
	kubeconfigCleanup func()
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	testConfig, err = test.LoadConfig()
	if err != nil {
		panic(err)
	}

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	var err error
	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	root, err := test.GetProjectDir()
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(root, "charts", "fleetconfig-controller", "crds"),
			filepath.Join(root, "config", "crds"),
		},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	envTestBinaryDir := test.FindEnvTestBinaryDir(testConfig)
	if envTestBinaryDir != "" {
		testEnv.BinaryAssetsDirectory = envTestBinaryDir
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Generate, save, and configure kubeconfig so in-cluster client lookups succeed
	var kubeconfigPath string
	raw, err := kube.RawFromRestConfig(cfg)
	Expect(err).ShouldNot(HaveOccurred())
	kubeconfigPath, kubeconfigCleanup, err = file.TmpFile(raw, "kubeconfig")
	Expect(err).ShouldNot(HaveOccurred())

	Expect(os.Setenv("KUBECONFIG", kubeconfigPath)).To(Succeed())
	logf.Log.Info("Kubeconfig", "path", kubeconfigPath)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	kubeconfigCleanup()
})
