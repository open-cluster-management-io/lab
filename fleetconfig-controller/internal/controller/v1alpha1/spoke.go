package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/hash"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/version"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

var csrSuffixPattern = regexp.MustCompile(`-[a-zA-Z0-9]{5}$`)

const (
	amwExistsError           = "you should manually clean them, uninstall kluster will cause those works out of control."
	managedClusterAddOn      = "ManagedClusterAddOn"
	addonCleanupTimeout      = 1 * time.Minute
	addonCleanupPollInterval = 2 * time.Second
)

// handleSpokes manages Spoke cluster join and upgrade operations
func handleSpokes(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleSpokes", "fleetconfig", fc.Name)

	hubKubeconfig, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, fc.Spec.Hub.Kubeconfig)
	if err != nil {
		return err
	}
	clusterClient, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}

	// clean up deregistered spokes
	joinedSpokes := make([]v1alpha1.JoinedSpoke, 0)
	for _, js := range fc.Status.JoinedSpokes {
		if !slices.ContainsFunc(fc.Spec.Spokes, func(spoke v1alpha1.Spoke) bool {
			return spoke.Name == js.Name
		}) {
			err = deregisterSpoke(ctx, kClient, hubKubeconfig, fc, &js)
			if err != nil {
				fc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), js.UnjoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
				))
				// if deregistration fails, retain the joined spoke in the status
				joinedSpokes = append(joinedSpokes, js)
				continue
			}
			fc.SetConditions(true, v1alpha1.NewCondition("unjoined", js.UnjoinType(), metav1.ConditionTrue, metav1.ConditionTrue))
		}
	}

	for _, spoke := range fc.Spec.Spokes {
		logger.V(0).Info("handleSpokes: reconciling spoke cluster", "name", spoke.Name)

		// check if the spoke has already been joined to the hub
		managedCluster, err := common.GetManagedCluster(ctx, clusterClient, spoke.Name)
		if err != nil {
			logger.Error(err, "failed to get managedCluster", "spoke", spoke.Name)
			continue
		}

		// attempt to join the spoke cluster if it hasn't already been joined
		if managedCluster == nil {
			tokenMeta, err := getToken(ctx, fc, hubKubeconfig)
			if err != nil {
				return fmt.Errorf("failed to get join token: %w", err)
			}
			if err := joinSpoke(ctx, kClient, fc, spoke, tokenMeta); err != nil {
				fc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
				))
				continue
			}
			// run `clusteradm accept` even if auto acceptance is enabled, as it's just a no-op if the spoke is already accepted
			if err := acceptCluster(ctx, fc, spoke.Name, false); err != nil {
				fc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
				))
				continue
			}
			logger.V(0).Info("handleSpokes: accepted spoke cluster", "name", spoke.Name)

			managedCluster, err = common.GetManagedCluster(ctx, clusterClient, spoke.Name)
			if err != nil {
				logger.Error(err, "failed to get managedCluster after join", "spoke", spoke.Name)
				continue
			}
		}

		// check managed clusters joined condition
		jc := getJoinedCondition(managedCluster)
		if jc == nil {
			logger.V(0).Info("handleSpokes: waiting for spoke cluster to join", "name", spoke.Name)
			msg := fmt.Sprintf("ManagedClusterJoined condition not found in ManagedCluster for spoke cluster %s", spoke.Name)
			fc.SetConditions(true, v1alpha1.NewCondition(
				msg, spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			// Re-accept all join requests for the spoke cluster. This is a workaround for the issue
			// that duplicate CSRs are sometimes created for the same spoke cluster when the klusterlet
			// controller bounces the klusterlet registration agent.
			if err := acceptCluster(ctx, fc, spoke.Name, true); err != nil {
				logger.Error(err, "failed to accept spoke cluster join request(s)", "spoke", spoke.Name)
			}
			continue
		}

		logger.V(0).Info("handleSpokes: found join condition", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
		if jc.Status != metav1.ConditionTrue {
			msg := fmt.Sprintf("failed to join spoke cluster %s: %s", spoke.Name, jc.Message)
			fc.SetConditions(true, v1alpha1.NewCondition(
				msg, spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			logger.V(0).Info("handleSpokes: join failed", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
			continue
		}

		// spoke cluster has joined successfully
		fc.SetConditions(true, v1alpha1.NewCondition(
			"Joined", spoke.JoinType(), metav1.ConditionTrue, metav1.ConditionTrue,
		))

		// Label the spoke ManagedCluster corresponding to the hub if in hub-as-spoke mode.
		// This allows the 'spoke' ManagedClusterSet to omit the hub-as-spoke cluster from its list
		// of spoke clusters.
		if managedCluster != nil && spoke.Kubeconfig.InCluster {
			if managedCluster.Labels == nil {
				managedCluster.Labels = make(map[string]string)
			}
			managedCluster.Labels[v1alpha1.LabelManagedClusterType] = v1alpha1.ManagedClusterTypeHubAsSpoke
			if err := common.UpdateManagedCluster(ctx, clusterClient, managedCluster); err != nil {
				return err
			}
			logger.V(0).Info("handleSpokes: labeled ManagedCluster as hub-as-spoke", "name", spoke.Name)
		}

		// attempt an upgrade whenever the klusterlet's bundleVersion or values change
		currKlusterletHash, err := hash.ComputeHash(spoke.Klusterlet.Values)
		if err != nil {
			return fmt.Errorf("failed to compute hash of spoke %s klusterlet values: %w", spoke.Name, err)
		}
		upgrade, err := spokeNeedsUpgrade(ctx, kClient, spoke, fc.Status.JoinedSpokes, currKlusterletHash)
		if err != nil {
			return fmt.Errorf("failed to check if spoke cluster needs upgrade: %w", err)
		}

		if upgrade {
			if err := upgradeSpoke(ctx, kClient, fc, spoke); err != nil {
				return fmt.Errorf("failed to upgrade spoke cluster %s: %w", spoke.Name, err)
			}
		}

		enabledAddons, err := handleSpokeAddons(ctx, spoke, fc)
		if err != nil {
			msg := fmt.Sprintf("failed to enable addons for spoke cluster %s: %s", spoke.Name, err.Error())
			fc.SetConditions(true, v1alpha1.NewCondition(
				msg, spoke.AddonEnableType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			continue
		}

		if len(enabledAddons) > 0 {
			fc.SetConditions(true, v1alpha1.NewCondition(
				"AddonsEnabled", spoke.AddonEnableType(), metav1.ConditionTrue, metav1.ConditionTrue,
			))
		}

		js := v1alpha1.JoinedSpoke{
			Name:                    spoke.Name,
			Kubeconfig:              spoke.Kubeconfig,
			PurgeKlusterletOperator: spoke.Klusterlet.PurgeOperator,
			EnabledAddons:           enabledAddons,
			KlusterletHash:          currKlusterletHash,
		}
		joinedSpokes = append(joinedSpokes, js)

	}

	fc.Status.JoinedSpokes = joinedSpokes

	return nil
}

func getJoinedSpoke(js []v1alpha1.JoinedSpoke, spokeName string) (v1alpha1.JoinedSpoke, bool) {
	i := slices.IndexFunc(js, func(s v1alpha1.JoinedSpoke) bool {
		return spokeName == s.Name
	})
	if i == -1 {
		return v1alpha1.JoinedSpoke{}, false
	}
	return js[i], true
}

func getJoinedCondition(managedCluster *clusterv1.ManagedCluster) *metav1.Condition {
	if managedCluster == nil || managedCluster.Status.Conditions == nil {
		return nil
	}

	for _, c := range managedCluster.Status.Conditions {
		if c.Type == "ManagedClusterJoined" {
			return &c
		}
	}

	return nil
}

// acceptCluster accepts a Spoke cluster's join request via 'clusteradm accept'
func acceptCluster(ctx context.Context, fc *v1alpha1.FleetConfig, name string, skipApproveCheck bool) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("acceptCluster")

	acceptArgs := append([]string{
		"accept", "--cluster", name,
	}, fc.BaseArgs()...)

	logger.V(1).Info("clusteradm accept", "args", acceptArgs)

	// TODO: handle other args:
	// --requesters=[]:
	//     Common Names of agents to be approved.

	if skipApproveCheck {
		acceptArgs = append(acceptArgs, "--skip-approve-check")
	}

	cmd := exec.Command(clusteradm, acceptArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm accept' to complete for spoke %s...", name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to accept spoke cluster join request: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("spoke cluster join request accepted", "output", string(stdout))

	return nil
}

type tokenMeta struct {
	Token        string `json:"hub-token"`
	HubAPIServer string `json:"hub-apiserver"`
}

// getToken gets a join token from the Hub cluster via 'clusteradm get token'
func getToken(ctx context.Context, fc *v1alpha1.FleetConfig, hubKubeconfig []byte) (*tokenMeta, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("getToken")

	tokenArgs := append([]string{
		"get", "token", "--output=json",
	}, fc.BaseArgs()...)

	if fc.Spec.Hub.ClusterManager != nil {
		tokenArgs = append(tokenArgs, fmt.Sprintf("--use-bootstrap-token=%t", fc.Spec.Hub.ClusterManager.UseBootstrapToken))
	}
	tokenArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, hubKubeconfig, fc.Spec.Hub.Kubeconfig.Context, tokenArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to prepare kubeconfig: %w", err)
	}

	logger.V(1).Info("clusteradm get token", "args", tokenArgs)

	cmd := exec.Command(clusteradm, tokenArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm get token' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return nil, fmt.Errorf("failed to get join token: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("got join token", "output", string(stdout))

	tokenMeta := &tokenMeta{}
	if err := json.Unmarshal(stdout, &tokenMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal join token: %w", err)
	}
	return tokenMeta, nil
}

// joinSpoke joins a Spoke cluster to the Hub cluster via 'clusteradm join'
func joinSpoke(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig, spoke v1alpha1.Spoke, tokenMeta *tokenMeta) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("joinSpoke", "spoke", spoke.Name)

	joinArgs := append([]string{
		"join",
		"--cluster-name", spoke.Name,
		fmt.Sprintf("--create-namespace=%t", spoke.CreateNamespace),
		fmt.Sprintf("--enable-sync-labels=%t", spoke.SyncLabels),
		"--hub-token", tokenMeta.Token,
		"--wait=true",
		// klusterlet args
		"--mode", spoke.Klusterlet.Mode,
		"--feature-gates", spoke.Klusterlet.FeatureGates,
		fmt.Sprintf("--force-internal-endpoint-lookup=%t", spoke.Klusterlet.ForceInternalEndpointLookup),
		fmt.Sprintf("--singleton=%t", spoke.Klusterlet.Singleton),
		// source args
		"--bundle-version", spoke.Klusterlet.Source.BundleVersion,
		"--image-registry", spoke.Klusterlet.Source.Registry,
	}, fc.BaseArgs()...)

	for k, v := range spoke.Klusterlet.Annotations {
		joinArgs = append(joinArgs, fmt.Sprintf("--klusterlet-annotation=%s=%s", k, v))
	}

	// resources args
	joinArgs = append(joinArgs, args.PrepareResources(spoke.Klusterlet.Resources)...)

	// Use hub API server from spec if provided and not forced to use internal endpoint,
	// otherwise fall back to the hub API server from the tokenMeta
	if fc.Spec.Hub.APIServer != "" && !spoke.Klusterlet.ForceInternalEndpointLookup {
		joinArgs = append(joinArgs, "--hub-apiserver", fc.Spec.Hub.APIServer)
	} else if tokenMeta.HubAPIServer != "" {
		joinArgs = append(joinArgs, "--hub-apiserver", tokenMeta.HubAPIServer)
	}

	if fc.Spec.Hub.Ca != "" {
		caFile, caCleanup, err := file.TmpFile([]byte(fc.Spec.Hub.Ca), "ca")
		if caCleanup != nil {
			defer caCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write hub CA to disk: %w", err)
		}
		joinArgs = append([]string{fmt.Sprintf("--ca-file=%s", caFile)}, joinArgs...)
	}

	if fc.Spec.RegistrationAuth.Driver == v1alpha1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-auth=%s", fc.Spec.RegistrationAuth.Driver),
		}
		if fc.Spec.RegistrationAuth.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", fc.Spec.RegistrationAuth.HubClusterARN))
		}
		if spoke.ClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--managed-cluster-arn=%s", spoke.ClusterARN))
		}

		joinArgs = append(joinArgs, raArgs...)
	}

	if spoke.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
		joinArgs = append(joinArgs,
			fmt.Sprintf("--force-internal-endpoint-lookup-managed=%t", spoke.Klusterlet.ForceInternalEndpointLookupManaged),
		)
		raw, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, spoke.Klusterlet.ManagedClusterKubeconfig)
		if err != nil {
			return err
		}
		mgdKcfg, mgdKcfgCleanup, err := file.TmpFile(raw, "kubeconfig")
		if mgdKcfgCleanup != nil {
			defer mgdKcfgCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write managedClusterKubeconfig to disk: %w", err)
		}
		joinArgs = append(joinArgs, "--managed-cluster-kubeconfig", mgdKcfg)
	}

	if spoke.ProxyCa != "" {
		proxyCaFile, proxyCaCleanup, err := file.TmpFile([]byte(spoke.ProxyCa), "proxy-ca")
		if proxyCaCleanup != nil {
			defer proxyCaCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write proxy CA to disk: %w", err)
		}
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-ca-file=%s", proxyCaFile))
	}
	if spoke.ProxyURL != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-url=%s", spoke.ProxyURL))
	}

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(spoke.Klusterlet.Values)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	joinArgs = append(joinArgs, valuesArgs...)

	kubeconfig, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, spoke.Kubeconfig)
	if err != nil {
		return err
	}
	joinArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, kubeconfig, spoke.Kubeconfig.Context, joinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm join", "args", joinArgs)

	cmd := exec.Command(clusteradm, joinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm join' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("clusteradm join command failed for spoke %s: %v, output: %s", spoke.Name, err, string(out))
	}
	logger.V(1).Info("successfully requested spoke cluster join", "output", string(stdout))

	return nil
}

// spokeNeedsUpgrade checks if the klusterlet on a Spoke cluster requires an upgrade. Upgrades are required when any of the following are true:
//   - The bundle version in the spec does not match the klusterlet's active bundle version
//   - The hash of the klusterlet chart values in the spec does not match the hash of the last applied klusterlet chart values
func spokeNeedsUpgrade(ctx context.Context, kClient client.Client, spoke v1alpha1.Spoke, joinedSpokes []v1alpha1.JoinedSpoke, currKlusterletHash string) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("spokeNeedsUpgrade", "spokeClusterName", spoke.Name)

	hashChanged := false
	prevJs, found := getJoinedSpoke(joinedSpokes, spoke.Name)
	if found {
		hashChanged = prevJs.KlusterletHash != currKlusterletHash
		logger.V(2).Info("comparing klusterlet values hash",
			"spoke", spoke.Name,
			"prevHash", prevJs.KlusterletHash,
			"currHash", currKlusterletHash,
		)
	}
	if hashChanged {
		return true, nil
	}

	if spoke.Klusterlet.Source.BundleVersion == "default" {
		logger.V(0).Info("klusterlet bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if spoke.Klusterlet.Source.BundleVersion == "latest" {
		logger.V(0).Info("klusterlet bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	kubeconfig, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, spoke.Kubeconfig)
	if err != nil {
		return false, err
	}
	operatorC, err := common.OperatorClient(kubeconfig)
	if err != nil {
		return false, err
	}

	k, err := operatorC.OperatorV1().Klusterlets().Get(ctx, "klusterlet", metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get klusterlet: %w", err)
	}

	// identify lowest bundleVersion referenced in the klusterlet spec
	bundleSpecs := make([]string, 0)
	if k.Spec.ImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.ImagePullSpec)
	}
	if k.Spec.RegistrationImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.RegistrationImagePullSpec)
	}
	if k.Spec.WorkImagePullSpec != "" {
		bundleSpecs = append(bundleSpecs, k.Spec.WorkImagePullSpec)
	}
	activeBundleVersion, err := version.LowestBundleVersion(ctx, bundleSpecs)
	if err != nil {
		return false, fmt.Errorf("failed to detect bundleVersion from klusterlet spec: %w", err)
	}
	desiredBundleVersion, err := version.Normalize(spoke.Klusterlet.Source.BundleVersion)
	if err != nil {
		return false, err
	}

	logger.V(0).Info("found klusterlet bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", desiredBundleVersion,
	)
	return activeBundleVersion != desiredBundleVersion, nil
}

// upgradeSpoke upgrades the Spoke cluster's klusterlet to the specified version
func upgradeSpoke(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig, spoke v1alpha1.Spoke) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeSpoke", "spoke", spoke.Name)

	upgradeArgs := append([]string{
		"upgrade", "klusterlet",
		"--bundle-version", spoke.Klusterlet.Source.BundleVersion,
		"--image-registry", spoke.Klusterlet.Source.Registry,
		"--wait=true",
	}, fc.BaseArgs()...)

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(spoke.Klusterlet.Values)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	upgradeArgs = append(upgradeArgs, valuesArgs...)

	kubeconfig, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, spoke.Kubeconfig)
	if err != nil {
		return err
	}
	upgradeArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, kubeconfig, spoke.Kubeconfig.Context, upgradeArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm upgrade klusterlet", "args", upgradeArgs)

	cmd := exec.Command(clusteradm, upgradeArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm upgrade klusterlet' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf(
			"failed to upgrade klusterlet on spoke cluster %s to %s: %v, output: %s",
			spoke.Name, spoke.Klusterlet.Source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("klusterlet upgraded", "output", string(stdout))

	return nil
}

// cleanupSpokes deregisters Spoke cluster(s) from the Hub cluster via 'clusteradm unjoin'
func cleanupSpokes(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanupSpokes", "fleetconfig", fc.Name)

	for _, spoke := range fc.Spec.Spokes {
		joinedCondition := fc.GetCondition(spoke.JoinType())
		if joinedCondition == nil || joinedCondition.Status != metav1.ConditionTrue {
			logger.V(0).Info("skipping cleanup for unjoined spoke cluster",
				"spoke", spoke.Name, "message", joinedCondition.Message, "reason", joinedCondition.Reason,
			)
			continue
		}

		if err := unjoinSpoke(ctx, kClient, fc, &spoke); err != nil {
			return err
		}
	}

	return nil
}

// unjoinSpoke unjoins a single spoke cluster from the Hub cluster via `clusteradm unjoin`
func unjoinSpoke(ctx context.Context, kClient client.Client, fc *v1alpha1.FleetConfig, spoke v1alpha1.ISpoke) error {
	logger := log.FromContext(ctx)

	unjoinArgs := append([]string{
		"unjoin",
		"--cluster-name", spoke.GetName(),
		fmt.Sprintf("--purge-operator=%t", spoke.GetPurgeKlusterletOperator()),
	}, fc.BaseArgs()...)

	kubeconfig, err := kube.KubeconfigFromNamespacedSecretOrCluster(ctx, kClient, spoke.GetKubeconfig())
	if err != nil {
		return err
	}
	unjoinArgs, cleanupKcfg, err := args.PrepareKubeconfig(ctx, kubeconfig, spoke.GetKubeconfig().Context, unjoinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %w", spoke.GetName(), err)
	}

	logger.V(1).Info("clusteradm unjoin", "args", unjoinArgs)

	cmd := exec.Command(clusteradm, unjoinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm unjoin' to complete for spoke %s...", spoke.GetName()))
	out := append(stdout, stderr...)
	if err != nil || strings.Contains(string(out), amwExistsError) {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %v, output: %s", spoke.GetName(), err, string(out))
	}
	logger.V(1).Info("spoke cluster unjoined", "output", string(stdout))

	return nil
}

// deregisterSpoke fully deregisters a spoke cluster, including cleaning up all relevant resources on the hub
func deregisterSpoke(ctx context.Context, kClient client.Client, hubKubeconfig []byte, fc *v1alpha1.FleetConfig, spoke *v1alpha1.JoinedSpoke) error {
	logger := log.FromContext(ctx)
	clusterC, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}
	workC, err := common.WorkClient(hubKubeconfig)
	if err != nil {
		return err
	}

	// skip clean up if the ManagedCluster resource is not found or if any manifestWorks exist
	managedCluster, err := clusterC.ClusterV1().ManagedClusters().Get(ctx, spoke.Name, metav1.GetOptions{})
	if kerrs.IsNotFound(err) {
		logger.Info("ManagedCluster resource not found; nothing to do")
		return nil
	} else if err != nil {
		return fmt.Errorf("unexpected error listing managedClusters: %w", err)
	}
	manifestWorks, err := workC.WorkV1().ManifestWorks(managedCluster.Name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list manifestWorks for managedCluster %s: %w", managedCluster.Name, err)
	}

	// check that the number of manifestWorks is the same as the number of addons enabled for that spoke
	if len(manifestWorks.Items) > 0 && !allOwnersAddOns(manifestWorks.Items) {
		msg := fmt.Sprintf("Found manifestWorks for ManagedCluster %s; cannot unjoin spoke cluster while it has active ManifestWorks", managedCluster.Name)
		logger.Info(msg)
		return errors.New(msg)

	}

	// remove addons only after confirming that the cluster can be unjoined - this avoids leaving dangling resources that may rely on the addon
	if err := handleAddonDisable(ctx, spoke.Name, spoke.EnabledAddons, fc); err != nil {
		fc.SetConditions(true, v1alpha1.NewCondition(
			err.Error(), spoke.AddonDisableType(), metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	if len(spoke.EnabledAddons) > 0 {
		// Wait for addon manifestWorks to be fully cleaned up before proceeding with unjoin
		if err := waitForAddonManifestWorksCleanup(ctx, workC, spoke.Name, addonCleanupTimeout); err != nil {
			fc.SetConditions(true, v1alpha1.NewCondition(
				err.Error(), spoke.AddonDisableType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return fmt.Errorf("addon manifestWorks cleanup failed: %w", err)
		}
		fc.SetConditions(true, v1alpha1.NewCondition(
			"AddonsDisabled", spoke.AddonDisableType(), metav1.ConditionTrue, metav1.ConditionTrue,
		))
	}

	// unjoin spoke - safe to proceed now that addon cleanup is confirmed
	if err := unjoinSpoke(ctx, kClient, fc, spoke); err != nil {
		return err
	}

	// remove CSR
	csrList := &certificatesv1.CertificateSigningRequestList{}
	if err := kClient.List(ctx, csrList, client.HasLabels{"open-cluster-management.io/cluster-name"}); err != nil {
		return err
	}
	for _, c := range csrList.Items {
		trimmedName := csrSuffixPattern.ReplaceAllString(c.Name, "")
		if trimmedName == spoke.Name {
			if err := kClient.Delete(ctx, &c); err != nil {
				return err
			}
		}
	}

	// remove ManagedCluster
	if err = clusterC.ClusterV1().ManagedClusters().Delete(ctx, spoke.Name, metav1.DeleteOptions{}); err != nil {
		return client.IgnoreNotFound(err)
	}

	// remove Namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spoke.Name}}
	if err := kClient.Delete(ctx, ns); err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

func allOwnersAddOns(mws []workv1.ManifestWork) bool {
	for _, m := range mws {
		if !slices.ContainsFunc(m.OwnerReferences, func(or metav1.OwnerReference) bool {
			return or.Kind == managedClusterAddOn
		}) {
			return false
		}
	}
	return true
}

// waitForAddonManifestWorksCleanup polls for addon-related manifestWorks to be removed
// after addon disable operation to avoid race conditions during spoke unjoin
func waitForAddonManifestWorksCleanup(ctx context.Context, workC *workapi.Clientset, spokeName string, timeout time.Duration) error {
	logger := log.FromContext(ctx)
	logger.V(1).Info("waiting for addon manifestWorks cleanup", "spokeName", spokeName, "timeout", timeout)

	err := wait.PollUntilContextTimeout(ctx, addonCleanupPollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		manifestWorks, err := workC.WorkV1().ManifestWorks(spokeName).List(ctx, metav1.ListOptions{})
		if err != nil {
			logger.V(3).Info("failed to list manifestWorks during cleanup wait", "error", err)
			// Return false to continue polling on transient errors
			return false, nil
		}

		// Success condition: no manifestWorks remaining
		if len(manifestWorks.Items) == 0 {
			logger.V(1).Info("addon manifestWorks cleanup completed", "spokeName", spokeName, "remainingManifestWorks", len(manifestWorks.Items))
			return true, nil
		}

		logger.V(3).Info("waiting for addon manifestWorks cleanup",
			"spokeName", spokeName,
			"addonManifestWorks", len(manifestWorks.Items))

		// Continue polling
		return false, nil
	})

	if err != nil {
		return fmt.Errorf("timeout waiting for addon manifestWorks cleanup for spoke %s: %w", spokeName, err)
	}

	return nil
}

// prepareKlusterletValuesFile creates a temporary file with klusterlet values and returns
// args to append and a cleanup function. Returns empty slice if values are empty.
func prepareKlusterletValuesFile(values *v1alpha1.KlusterletChartConfig) ([]string, func(), error) {
	if values == nil {
		return nil, nil, nil
	}
	if values.IsEmpty() {
		return nil, nil, nil
	}
	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal klusterlet values to YAML: %w", err)
	}
	valuesFile, valuesCleanup, err := file.TmpFile(valuesYAML, "klusterlet-values")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write klusterlet values to disk: %w", err)
	}
	return []string{"--klusterlet-values-file", valuesFile}, valuesCleanup, nil
}
