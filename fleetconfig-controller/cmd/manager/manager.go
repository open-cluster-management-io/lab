// Package manager contains functions for configuring a controller manager.
package manager

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apiv1alpha1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	apiv1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	controllerv1alpha1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/controller/v1alpha1"
	controllerv1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/controller/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	webhookv1beta1 "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/webhook/v1beta1"
	// +kubebuilder:scaffold:imports
)

// Options are the options for configuring a controller manager
type Options struct {
	MetricsAddr               string
	EnableLeaderElection      bool
	ProbeAddr                 string
	SecureMetrics             bool
	EnableHTTP2               bool
	UseWebhook                bool
	CertDir                   string
	WebhookPort               int
	SpokeConcurrentReconciles int
	InstanceType              string
	EnableLegacyControllers   bool
	EnableTopologyResources   bool
	Scheme                    *runtime.Scheme
}

func setupServer(opts Options, setupLog logr.Logger) (webhook.Server, []func(*tls.Config)) {
	// if the EnableHTTP2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !opts.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		CertDir: opts.CertDir,
		Port:    opts.WebhookPort,
		TLSOpts: tlsOpts,
	})
	return webhookServer, tlsOpts
}

// ForHub configures a manager instance for a Hub cluster.
func ForHub(setupLog logr.Logger, opts Options) (ctrl.Manager, error) {
	webhookServer, tlsOpts := setupServer(opts, setupLog)
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: opts.Scheme,
		Metrics: metricsserver.Options{
			BindAddress:   opts.MetricsAddr,
			SecureServing: opts.SecureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         opts.EnableLeaderElection,
		LeaderElectionID:       "9aac6663.open-cluster-management.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		return nil, err
	}

	if opts.EnableTopologyResources {
		setupLog.Info("creating topology resources")
		// Wait for cache to sync before setting up controllers
		if err = mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
			setupLog.Info("waiting for cache to sync")
			if !mgr.GetCache().WaitForCacheSync(ctx) {
				err = fmt.Errorf("cache sync failed")
				setupLog.Error(err, "failed to sync cache")
				return ctx.Err()
			}

			setupLog.Info("cache synced successfully")
			if err := createTopologyResources(ctx, mgr.GetClient(), setupLog); err != nil {
				setupLog.Error(err, "failed to create topology resources at startup")
				return err
			}
			setupLog.Info("topology resources created successfully")
			return nil
		})); err != nil {
			return nil, err
		}
	}

	hubReconciler := &controllerv1beta1.HubReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Hub"),
		Scheme: mgr.GetScheme(),
	}

	spokeReconciler := &controllerv1beta1.SpokeReconciler{
		Client:               mgr.GetClient(),
		Log:                  ctrl.Log.WithName("controllers").WithName("Spoke"),
		ConcurrentReconciles: opts.SpokeConcurrentReconciles,
		Scheme:               mgr.GetScheme(),
		InstanceType:         opts.InstanceType,
	}
	if err := hubReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Hub")
		return nil, err
	}

	if err := spokeReconciler.SetupWithManagerForHub(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}

	// nolint:goconst
	if opts.UseWebhook || os.Getenv("ENABLE_WEBHOOKS") == "true" {
		if err := webhookv1beta1.SetupHubWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Hub")
			return nil, err
		}
		if err := webhookv1beta1.SetupSpokeWebhookWithManager(mgr, opts.InstanceType); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Spoke")
			return nil, err
		}
	}

	if opts.EnableLegacyControllers {
		fleetConfigReconciler := &controllerv1alpha1.FleetConfigReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("FleetConfig"),
			Scheme: mgr.GetScheme(),
		}
		if err = fleetConfigReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "FleetConfig")
			return nil, err
		}
		if err = apiv1alpha1.SetupFleetConfigWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "FleetConfig")
			return nil, err
		}
	}

	return mgr, nil
}

// ForSpoke configures a manager instance for a Spoke cluster.
func ForSpoke(setupLog logr.Logger, opts Options) (ctrl.Manager, error) {
	_, tlsOpts := setupServer(opts, setupLog)

	var err error

	// Verify that all required environment variables have been set
	spokeNamespace := os.Getenv(apiv1beta1.SpokeNamespaceEnvVar)
	if spokeNamespace == "" {
		err = fmt.Errorf("%s environment variable must be set", apiv1beta1.SpokeNamespaceEnvVar)
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}
	hubNamespace := os.Getenv(apiv1beta1.HubNamespaceEnvVar)
	if hubNamespace == "" {
		err = fmt.Errorf("%s environment variable must be set", apiv1beta1.HubNamespaceEnvVar)
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}
	ctrlNamespace := os.Getenv(apiv1beta1.ControllerNamespaceEnvVar)
	if ctrlNamespace == "" {
		err = fmt.Errorf("%s environment variable must be set", apiv1beta1.ControllerNamespaceEnvVar)
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}
	// enables watching resources in the hub cluster
	hubRestCfg, err := getHubRestConfig()
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}

	// enables leader election in the spoke cluster
	localRestCfg, err := ctrl.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}

	mgr, err := ctrl.NewManager(hubRestCfg, ctrl.Options{
		Scheme: opts.Scheme,
		Metrics: metricsserver.Options{
			BindAddress:   opts.MetricsAddr,
			SecureServing: opts.SecureMetrics,
			TLSOpts:       tlsOpts,
		},
		// Configure Informer Cache to only watch the resources it needs to perform a reconcile, in specific namespaces.
		// There are 2 related reasons for this.
		// 1. Limit scope of access to the hub for the the spoke's controller. Since it only needs to watch a 3 resources across 2 namespaces, it should not watch other namespaces.
		// 2. OCM AddOn framework limits the scope of access that addon agents have to the hub.
		//    Only access to either the ManagedCluster namespace, or a user-specified namespace is allowed. No ClusterRoleBindings created by the addon manager controller.
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&apiv1beta1.Spoke{}: {
					Namespaces: map[string]cache.Config{
						spokeNamespace: {
							LabelSelector: labels.Everything(),                          // prevent default
							FieldSelector: fields.Everything(),                          // prevent default
							Transform:     func(in any) (any, error) { return in, nil }, // prevent default
						},
					},
				},
				// required to retrieve klusterlet values from configmap in the spoke namespace
				&corev1.ConfigMap{}: {
					Namespaces: map[string]cache.Config{
						spokeNamespace: {
							LabelSelector: labels.Everything(),                          // prevent default
							FieldSelector: fields.Everything(),                          // prevent default
							Transform:     func(in any) (any, error) { return in, nil }, // prevent default
						},
					},
				},
				// required to retrieve hub from the spoke.spec.hubRef namespace
				&apiv1beta1.Hub{}: {
					Namespaces: map[string]cache.Config{
						hubNamespace: {
							LabelSelector: labels.Everything(),                          // prevent default
							FieldSelector: fields.Everything(),                          // prevent default
							Transform:     func(in any) (any, error) { return in, nil }, // prevent default
						},
					},
				},
			},
		},

		WebhookServer:          nil,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElection:         opts.EnableLeaderElection,
		LeaderElectionConfig:   localRestCfg, // use local restConfig. alternatively, we can disable leader election if HA is not a concern.
		LeaderElectionID:       "9aac6663.open-cluster-management.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		return nil, err
	}

	if err := (&controllerv1beta1.SpokeReconciler{
		Client:               mgr.GetClient(),
		Log:                  ctrl.Log.WithName("controllers").WithName("Spoke"),
		ConcurrentReconciles: opts.SpokeConcurrentReconciles,
		Scheme:               mgr.GetScheme(),
		InstanceType:         opts.InstanceType,
	}).SetupWithManagerForSpoke(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Spoke")
		return nil, err
	}
	return mgr, nil
}

func getHubRestConfig() (*rest.Config, error) {
	hubKubeconfigPath := os.Getenv(apiv1beta1.HubKubeconfigEnvVar)
	if hubKubeconfigPath == "" {
		hubKubeconfigPath = apiv1beta1.DefaultHubKubeconfigPath
	}

	basePath := filepath.Dir(hubKubeconfigPath)
	certPath := filepath.Join(basePath, "tls.crt")
	keyPath := filepath.Join(basePath, "tls.key")

	hubKubeconfig, err := os.ReadFile(hubKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig from mount: %v", err)
	}

	// patch the kubeconfig with absolute paths to cert/key
	hubKubeconfig = bytes.ReplaceAll(hubKubeconfig, []byte("tls.crt"), []byte(certPath))
	hubKubeconfig = bytes.ReplaceAll(hubKubeconfig, []byte("tls.key"), []byte(keyPath))

	cfg, err := kube.RestConfigFromKubeconfig(hubKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config for hub: %v", err)
	}
	return cfg, nil
}
