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

// Package main is the entrypoint for fleetconfig-controller
package main

import (
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1alpha1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	apiv1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/cmd/manager"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(apiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	mOpts := manager.Options{
		Scheme: scheme,
	}

	flag.StringVar(&mOpts.MetricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to. Use the port :8080. If not set, it will be 0 to disable the metrics server.")
	flag.StringVar(&mOpts.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&mOpts.EnableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&mOpts.SecureMetrics, "metrics-secure", false, "If set, the metrics endpoint is served securely.")
	flag.BoolVar(&mOpts.EnableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the metrics and webhook servers.")

	flag.BoolVar(&mOpts.UseWebhook, "use-webhook", mOpts.UseWebhook, "Enable admission webhooks")
	flag.StringVar(&mOpts.CertDir, "webhook-cert-dir", "/etc/k8s-webhook-certs", "Admission webhook cert/key dir")
	flag.IntVar(&mOpts.WebhookPort, "webhook-port", 9443, "Admission webhook port")

	flag.IntVar(&mOpts.SpokeConcurrentReconciles, "spoke-concurrent-reconciles", apiv1beta1.SpokeDefaultMaxConcurrentReconciles, fmt.Sprintf("Maximum number of Spoke resources that may be reconciled in parallel. Defaults to %d.", apiv1beta1.SpokeDefaultMaxConcurrentReconciles))
	flag.StringVar(&mOpts.InstanceType, "instance-type", apiv1beta1.InstanceTypeManager, fmt.Sprintf("The type of cluster that this controller instance is installed in. Defaults to %s", apiv1beta1.InstanceTypeManager))
	flag.BoolVar(&mOpts.EnableLegacyControllers, "enable-legacy-controllers", false, "Enable legacy FleetConfig resource and controllers")

	zOpts := zap.Options{
		Development: true,
	}
	zOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zOpts)))

	var (
		mgr ctrl.Manager
		err error
	)

	switch mOpts.InstanceType {
	case apiv1beta1.InstanceTypeManager, apiv1beta1.InstanceTypeUnified:
		mgr, err = manager.ForHub(setupLog, mOpts)
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
	case apiv1beta1.InstanceTypeAgent:
		mgr, err = manager.ForSpoke(setupLog, mOpts)
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			os.Exit(1)
		}
	default:
		setupLog.Info("unable to create controller for unknown instance type", "instanceType", mOpts.InstanceType, "allowed", apiv1beta1.SupportedInstanceTypes)
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
