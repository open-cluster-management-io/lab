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
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/test"
)

var (
	testConfig *test.Config
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteConfig, reporterConfig := GinkgoConfiguration()

	_, _ = fmt.Fprintf(GinkgoWriter, "loading the test config\n")
	var err error
	testConfig, err = test.LoadConfig()
	if err != nil {
		panic(err)
	}
	reporterConfig.FullTrace = testConfig.FullTrace
	reporterConfig.VeryVerbose = testConfig.Verbose
	if testConfig.LabelFilter != "" {
		suiteConfig.LabelFilter = testConfig.LabelFilter
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "Starting E2E suite\n")
	RunSpecs(t, "e2e suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
})
