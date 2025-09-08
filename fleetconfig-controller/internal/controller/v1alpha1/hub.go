// Package v1alpha1 contains the main reconciliation logic for fleetconfig-controller's v1alpha1 resources.
package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	clusterapi "open-cluster-management.io/api/client/cluster/clientset/versioned"
	operatorapi "open-cluster-management.io/api/client/operator/clientset/versioned"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/version"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// handleHub manages Hub cluster init and upgrade operations
func handleHub(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleHub", "fleetconfig", fc.Name)

	// check if the hub is already initialized
	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, kClient, fc.Spec.Hub.Kubeconfig)
	if err != nil {
		return err
	}
	operatorC, err := common.OperatorClient(hubKubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return err
	}
	cm, err := getClusterManager(ctx, operatorC)
	if err != nil {
		return err
	}

	// if a clustermanager already exists, we don't need to init the hub
	if cm != nil && cm.Status.Conditions != nil {
		msgs := make([]string, 0)
		for _, c := range cm.Status.Conditions {
			if c.Type == operatorv1.ConditionProgressing && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionClusterManagerApplied && c.Status == metav1.ConditionFalse {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionHubRegistrationDegraded && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
			if c.Type == operatorv1.ConditionHubPlacementDegraded && c.Status == metav1.ConditionTrue {
				msgs = append(msgs, fmt.Sprintf("%s: %s", c.Type, c.Message))
			}
		}
		if len(msgs) > 0 {
			msg := strings.TrimSuffix(strings.Join(msgs, "; "), "; ")
			msg = fmt.Sprintf("hub pending/degraded: %s", msg)
			fc.SetConditions(true, v1alpha1.NewCondition(
				msg, v1alpha1.FleetConfigHubInitialized, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return errors.New(msg)
		}
	} else {
		if err := initializeHub(ctx, kClient, fc); err != nil {
			return err
		}
	}

	fc.SetConditions(true, v1alpha1.NewCondition(
		v1alpha1.FleetConfigHubInitialized, v1alpha1.FleetConfigHubInitialized, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	addonConfigChanged, err := handleAddonConfig(ctx, kClient, addonC, fc)
	if err != nil && addonConfigChanged {
		fc.SetConditions(true, v1alpha1.NewCondition(
			err.Error(), v1alpha1.FleetConfigAddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	hubAddonChanged, err := handleHubAddons(ctx, addonC, fc)
	if err != nil && hubAddonChanged {
		fc.SetConditions(true, v1alpha1.NewCondition(
			err.Error(), v1alpha1.FleetConfigAddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	// only set success condition if we actually managed any addons
	if addonConfigChanged || hubAddonChanged {
		fc.SetConditions(true, v1alpha1.NewCondition(
			v1alpha1.FleetConfigAddonsConfigured, v1alpha1.FleetConfigAddonsConfigured, metav1.ConditionTrue, metav1.ConditionTrue,
		))
	}

	// attempt an upgrade whenever the clustermanager's bundleVersion changes
	upgrade, err := hubNeedsUpgrade(ctx, fc, operatorC)
	if err != nil {
		return fmt.Errorf("failed to check if hub needs upgrade: %w", err)
	}
	if upgrade {
		return upgradeHub(ctx, fc)
	}

	return nil
}

// initializeHub initializes the Hub cluster via 'clusteradm init'
func initializeHub(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("initHub", "fleetconfig", fc.Name)

	initArgs := append([]string{
		"init",
		fmt.Sprintf("--create-namespace=%t", fc.Spec.Hub.CreateNamespace),
		fmt.Sprintf("--force=%t", fc.Spec.Hub.Force),
		"--wait=true",
	}, fc.BaseArgs()...)

	if fc.Spec.RegistrationAuth.Driver == v1alpha1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-drivers=%s", fc.Spec.RegistrationAuth.Driver),
		}
		if fc.Spec.RegistrationAuth.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", fc.Spec.RegistrationAuth.HubClusterARN))
		}
		if len(fc.Spec.RegistrationAuth.AutoApprovedARNPatterns) > 0 {
			raArgs = append(raArgs, fmt.Sprintf("--auto-approved-arn-patterns=%s", strings.Join(fc.Spec.RegistrationAuth.AutoApprovedARNPatterns, ",")))
		}
		initArgs = append(initArgs, raArgs...)
	}

	if fc.Spec.Hub.SingletonControlPlane != nil {
		initArgs = append(initArgs, "--singleton=true")
		initArgs = append(initArgs, "--singleton-name", fc.Spec.Hub.SingletonControlPlane.Name)
		if fc.Spec.Hub.SingletonControlPlane.Helm != nil {
			if fc.Spec.Hub.SingletonControlPlane.Helm.Values != "" {
				values, cleanupValues, err := file.TmpFile([]byte(fc.Spec.Hub.SingletonControlPlane.Helm.Values), "values")
				if cleanupValues != nil {
					defer cleanupValues()
				}
				if err != nil {
					return err
				}
				initArgs = append(initArgs, "--values", values)
			}
			for _, s := range fc.Spec.Hub.SingletonControlPlane.Helm.Set {
				initArgs = append(initArgs, "--set", s)
			}
			for _, s := range fc.Spec.Hub.SingletonControlPlane.Helm.SetJSON {
				initArgs = append(initArgs, "--set-json", s)
			}
			for _, s := range fc.Spec.Hub.SingletonControlPlane.Helm.SetLiteral {
				initArgs = append(initArgs, "--set-literal", s)
			}
			for _, s := range fc.Spec.Hub.SingletonControlPlane.Helm.SetString {
				initArgs = append(initArgs, "--set-string", s)
			}
		}
	} else if fc.Spec.Hub.ClusterManager != nil {
		// clustermanager args
		initArgs = append(initArgs, "--feature-gates", fc.Spec.Hub.ClusterManager.FeatureGates)
		initArgs = append(initArgs, fmt.Sprintf("--use-bootstrap-token=%t", fc.Spec.Hub.ClusterManager.UseBootstrapToken))
		// source args
		initArgs = append(initArgs, "--bundle-version", fc.Spec.Hub.ClusterManager.Source.BundleVersion)
		initArgs = append(initArgs, "--image-registry", fc.Spec.Hub.ClusterManager.Source.Registry)
		// resources args
		initArgs = append(initArgs, common.PrepareResources(fc.Spec.Hub.ClusterManager.Resources)...)
	} else {
		// one of clusterManager or singletonControlPlane must be specified, per validating webhook, but handle the edge case anyway
		return fmt.Errorf("unknown hub type, must specify either hub.clusterManager or hub.singletonControlPlane")
	}

	initArgs, cleanupKcfg, err := common.PrepareKubeconfig(ctx, kClient, fc.Spec.Hub.Kubeconfig, initArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm init", "args", initArgs)

	cmd := exec.Command(clusteradm, initArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm init' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to init hub: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("hub initialized", "output", string(stdout))

	return nil
}

// hubNeedsUpgrade checks if the clustermanager on the Hub cluster has the desired bundle version
func hubNeedsUpgrade(ctx context.Context, fc *v1alpha1.FleetConfig, operatorC *operatorapi.Clientset) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("hubNeedsUpgrade", "fleetconfig", fc.Name)

	if fc.Spec.Hub.ClusterManager.Source.BundleVersion == "default" {
		logger.V(0).Info("clustermanager bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if fc.Spec.Hub.ClusterManager.Source.BundleVersion == "latest" {
		logger.V(0).Info("clustermanager bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	cm, err := getClusterManager(ctx, operatorC)
	if err != nil {
		return false, err
	}

	// identify lowest bundleVersion referenced in the clustermanager spec
	bundleSpecs := make([]string, 0)
	if cm.Spec.AddOnManagerImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.AddOnManagerImagePullSpec)
	}
	if cm.Spec.PlacementImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.PlacementImagePullSpec)
	}
	if cm.Spec.RegistrationImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.RegistrationImagePullSpec)
	}
	if cm.Spec.WorkImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, cm.Spec.WorkImagePullSpec)
	}
	activeBundleVersion, err := version.LowestBundleVersion(ctx, bundleSpecs)
	if err != nil {
		return false, fmt.Errorf("failed to detect bundleVersion from clustermanager spec: %w", err)
	}
	desiredBundleVersion, err := version.Normalize(fc.Spec.Hub.ClusterManager.Source.BundleVersion)
	if err != nil {
		return false, err
	}

	logger.V(0).Info("found clustermanager bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", desiredBundleVersion,
	)
	return activeBundleVersion != desiredBundleVersion, nil
}

// getClusterManager retrieves the ClusterManager resource from the Hub cluster
func getClusterManager(ctx context.Context, operatorC *operatorapi.Clientset) (*operatorv1.ClusterManager, error) {
	cm, err := operatorC.OperatorV1().ClusterManagers().Get(ctx, "cluster-manager", metav1.GetOptions{})
	if err != nil {
		if kerrs.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected error getting cluster-manager: %w", err)
	}
	return cm, nil
}

// upgradeHub upgrades the Hub cluster's clustermanager to the specified version
func upgradeHub(ctx context.Context, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeHub", "fleetconfig", fc.Name)

	upgradeArgs := append([]string{
		"upgrade", "clustermanager",
		"--bundle-version", fc.Spec.Hub.ClusterManager.Source.BundleVersion,
		"--image-registry", fc.Spec.Hub.ClusterManager.Source.Registry,
		"--wait=true",
	}, fc.BaseArgs()...)

	logger.V(1).Info("clusteradm upgrade clustermanager", "args", upgradeArgs)

	cmd := exec.Command(clusteradm, upgradeArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm upgrade clustermanager' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf(
			"failed to upgrade hub clustermanager to %s: %v, output: %s",
			fc.Spec.Hub.ClusterManager.Source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("clustermanager upgraded", "output", string(stdout))

	return nil
}

// cleanHub uninstalls OCM components from the Hub cluster via 'clusteradm clean'
// TODO: how to clean hub clusters using a singleton control plane?
func cleanHub(ctx context.Context, kClient client.Client, hubKubeconfig []byte, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanHub", "fleetconfig", fc.Name)

	clusterC, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}

	// delete all ManagedClusters before cleaning the hub
	if err := cleanManagedClusters(ctx, fc, clusterC); err != nil {
		return err
	}

	// manually clean all managed cluster namespaces
	if err := cleanNamespaces(ctx, kClient, fc); err != nil {
		return err
	}

	cleanArgs := append([]string{
		"clean",
		// name is omitted, as the default name, 'cluster-manager', is always used
		fmt.Sprintf("--purge-operator=%t", fc.Spec.Hub.ClusterManager.PurgeOperator),
	}, fc.BaseArgs()...)

	logger.V(1).Info("clusteradm clean", "args", cleanArgs)

	cmd := exec.Command(clusteradm, cleanArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm clean' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to clean hub cluster: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("hub cleaned", "output", string(stdout))

	return nil
}

var cleanupInterval = 5 * time.Second

// cleanManagedClusters deletes all ManagedClusters from the Hub cluster.
func cleanManagedClusters(ctx context.Context, fc *v1alpha1.FleetConfig, client *clusterapi.Clientset) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanManagedClusters", "fleetconfig", fc.Name)

	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	}
	if err := client.ClusterV1().ManagedClusters().DeleteCollection(ctx, deleteOpts, metav1.ListOptions{}); err != nil {
		if !kerrs.IsNotFound(err) {
			return fmt.Errorf("failed to delete managedClusters: %w", err)
		}
	}

	// Poll until all ManagedClusters are deleted
	logger.Info("waiting for all ManagedClusters to be deleted")

	err := wait.PollUntilContextCancel(ctx, cleanupInterval, true, func(ctx context.Context) (bool, error) {
		clusters, err := client.ClusterV1().ManagedClusters().List(ctx, metav1.ListOptions{})
		if err != nil {
			if kerrs.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if len(clusters.Items) == 0 {
			return true, nil
		}
		logger.V(1).Info("ManagedClusters still present", "count", len(clusters.Items))
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for ManagedClusters to be deleted: %w", err)
	}

	logger.Info("confirmed all ManagedClusters are deleted")
	return nil
}

func cleanNamespaces(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanNamespaces", "fleetconfig", fc.Name)

	deleteOpts := &client.DeleteOptions{
		PropagationPolicy: ptr.To(metav1.DeletePropagationForeground),
	}
	namespaces := make([]string, 0, len(fc.Spec.Spokes))

	for _, spoke := range fc.Spec.Spokes {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spoke.Name}}
		if err := kClient.Delete(ctx, ns, deleteOpts); err != nil && !kerrs.IsNotFound(err) {
			return err
		}
		logger.Info("deleted spoke namespace", "spokeNamespace", spoke.Name)
		namespaces = append(namespaces, spoke.Name)
	}
	if len(namespaces) == 0 {
		logger.Info("no spoke namespaces to delete")
		return nil
	}

	// Poll until all namespaces are deleted
	logger.Info("waiting for all spoke namespaces to be deleted")

	err := wait.PollUntilContextCancel(ctx, cleanupInterval, true, func(ctx context.Context) (bool, error) {
		for _, nsName := range namespaces {
			ns := &corev1.Namespace{}
			err := kClient.Get(ctx, client.ObjectKey{Name: nsName}, ns)
			if err == nil {
				logger.V(1).Info("namespace still present", "namespace", nsName)
				return false, nil
			} else if !kerrs.IsNotFound(err) {
				return false, err // unexpected error
			}
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for namespaces to be deleted: %w", err)
	}

	logger.Info("confirmed all spoke namespaces are deleted")
	return nil
}
