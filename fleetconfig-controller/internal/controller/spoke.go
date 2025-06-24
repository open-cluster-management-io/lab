package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
	"slices"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
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

var csrSuffixPattern = regexp.MustCompile(`-[a-zA-Z0-9]{5}$`)

// handleSpokes manages Spoke cluster join and upgrade operations
func handleSpokes(ctx context.Context, kClient client.Client, mc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleSpokes", "fleetconfig", mc.Name)

	hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, kClient, mc.Spec.Hub.Kubeconfig)
	if err != nil {
		return err
	}
	clusterClient, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}

	// clean up deregistered spokes
	joinedSpokes := make([]v1alpha1.JoinedSpoke, 0)
	for _, js := range mc.Status.JoinedSpokes {
		if !slices.ContainsFunc(mc.Spec.Spokes, func(spoke v1alpha1.Spoke) bool {
			return spoke.Name == js.Name && reflect.DeepEqual(spoke.Kubeconfig, js.Kubeconfig)
		}) {
			err = deregisterSpoke(ctx, kClient, hubKubeconfig, &js)
			if err != nil {
				mc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), js.UnjoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
				))
				joinedSpokes = append(joinedSpokes, js)
				continue
			}
			mc.SetConditions(true, v1alpha1.NewCondition("unjoined", js.UnjoinType(), metav1.ConditionTrue, metav1.ConditionTrue))
		}
	}

	for _, spoke := range mc.Spec.Spokes {
		logger.V(0).Info("handleSpokes: reconciling spoke cluster", "name", spoke.Name)

		// check if the spoke has already been joined to the hub
		managedCluster, err := common.GetManagedCluster(ctx, clusterClient, spoke.Name)
		if err != nil {
			logger.Error(err, "failed to get managedCluster", "spoke", spoke.Name)
			continue
		}

		// attempt to join the spoke cluster if it hasn't already been joined
		if managedCluster == nil {
			tokenMeta, err := getToken(ctx, kClient, mc)
			if err != nil {
				return fmt.Errorf("failed to get join token: %w", err)
			}
			if err := joinSpoke(ctx, kClient, mc.Spec, spoke, tokenMeta); err != nil {
				mc.SetConditions(true, v1alpha1.NewCondition(
					err.Error(), spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
				))
				continue
			}
			// run `clusteradm accept` even if auto acceptance is enabled, as it's just a no-op if the spoke is already accepted
			if err := acceptCluster(ctx, spoke.Name); err != nil {
				mc.SetConditions(true, v1alpha1.NewCondition(
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
			mc.SetConditions(true, v1alpha1.NewCondition(
				msg, spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			continue
		}

		logger.V(0).Info("handleSpokes: found join condition", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
		if jc.Status != metav1.ConditionTrue {
			msg := fmt.Sprintf("failed to join spoke cluster %s: %s", spoke.Name, jc.Message)
			mc.SetConditions(true, v1alpha1.NewCondition(
				msg, spoke.JoinType(), metav1.ConditionFalse, metav1.ConditionTrue,
			))
			logger.V(0).Info("handleSpokes: join failed", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
			continue
		}

		// spoke cluster has joined successfully
		mc.SetConditions(true, v1alpha1.NewCondition(
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

		// attempt an upgrade whenever the klusterlet's bundleVersion changes
		upgrade, err := spokeNeedsUpgrade(ctx, kClient, spoke)
		if err != nil {
			return fmt.Errorf("failed to check if spoke cluster needs upgrade: %w", err)
		}
		if upgrade {
			if err := upgradeSpoke(ctx, kClient, spoke); err != nil {
				return fmt.Errorf("failed to upgrade spoke cluster %s: %w", spoke.Name, err)
			}
		}
	}

	// Only spokes which are joined, are eligible to be unjoined
	for _, spoke := range mc.Spec.Spokes {
		joinedCondition := mc.GetCondition(spoke.JoinType())
		if joinedCondition == nil || joinedCondition.Status != metav1.ConditionTrue {
			continue
		}
		js := v1alpha1.JoinedSpoke{
			Name:                    spoke.Name,
			Kubeconfig:              spoke.Kubeconfig,
			PurgeKlusterletOperator: spoke.Klusterlet.PurgeOperator,
		}
		joinedSpokes = append(joinedSpokes, js)
	}
	mc.Status.JoinedSpokes = joinedSpokes

	return nil
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
func acceptCluster(ctx context.Context, name string) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("acceptCluster")

	acceptArgs := []string{"accept", "--cluster", name}
	logger.V(1).Info("clusteradm accept", "args", acceptArgs)

	// TODO: handle other args:
	// --requesters=[]:
	//     Common Names of agents to be approved.

	// --skip-approve-check=false:
	//     If set, then skip check and approve csr directly.

	cmd := exec.Command(clusteradm, acceptArgs...)
	out, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm accept' to complete for spoke %s...", name))
	if err != nil {
		return fmt.Errorf("failed to accept spoke cluster join request: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("spoke cluster join request accepted", "output", string(out))

	return nil
}

type tokenMeta struct {
	Token        string `json:"hub-token"`
	HubAPIServer string `json:"hub-apiserver"`
}

// getToken gets a join token from the Hub cluster via 'clusteradm get token'
func getToken(ctx context.Context, kClient client.Client, mc *v1alpha1.FleetConfig) (*tokenMeta, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("getToken")

	tokenArgs := []string{"get", "token", "--output=json"}
	if mc.Spec.Hub.ClusterManager != nil {
		tokenArgs = append(tokenArgs, fmt.Sprintf("--use-bootstrap-token=%t", mc.Spec.Hub.ClusterManager.UseBootstrapToken))
	}
	tokenArgs, cleanupKcfg, err := common.PrepareKubeconfig(ctx, kClient, mc.Spec.Hub.Kubeconfig, tokenArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to prepare kubeconfig: %w", err)
	}
	logger.V(1).Info("clusteradm get token", "args", tokenArgs)

	cmd := exec.Command(clusteradm, tokenArgs...)
	out, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm get token' to complete...")
	if err != nil {
		return nil, fmt.Errorf("failed to get join token: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("got join token", "output", string(out))

	tokenMeta := &tokenMeta{}
	if err := json.Unmarshal(out, &tokenMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal join token: %w", err)
	}
	return tokenMeta, nil
}

// joinSpoke joins a Spoke cluster to the Hub cluster via 'clusteradm join'
func joinSpoke(ctx context.Context, kClient client.Client, spec v1alpha1.FleetConfigSpec, spoke v1alpha1.Spoke, tokenMeta *tokenMeta) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("joinSpoke", "spoke", spoke.Name)

	joinArgs := []string{"join",
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
	}

	// Use hub API server from spec if provided, otherwise fall back to tokenMeta
	if spec.Hub.APIServer != nil {
		joinArgs = append(joinArgs, "--hub-apiserver", *spec.Hub.APIServer)
	} else if tokenMeta.HubAPIServer != "" {
		joinArgs = append(joinArgs, "--hub-apiserver", tokenMeta.HubAPIServer)
	}

	registrationDriver := spec.RegistrationAuth.GetDriver()
	if registrationDriver == v1alpha1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-auth=%s", registrationDriver),
		}
		if spec.RegistrationAuth.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", spec.RegistrationAuth.HubClusterARN))
		}
		if spoke.ClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--managed-cluster-arn=%s", spoke.ClusterARN))
		}

		joinArgs = append(joinArgs, raArgs...)
	}

	if spoke.Klusterlet.Resources != nil {
		joinArgs = append(joinArgs, common.PrepareResources(*spoke.Klusterlet.Resources)...)
	}

	if spoke.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
		joinArgs = append(joinArgs,
			fmt.Sprintf("--force-internal-endpoint-lookup-managed=%t", spoke.Klusterlet.ForceInternalEndpointLookupManaged),
		)
		raw, err := kube.KubeconfigFromSecretOrCluster(ctx, kClient, spoke.Klusterlet.ManagedClusterKubeconfig)
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

	if spoke.Ca != "" {
		caFile, caCleanup, err := file.TmpFile([]byte(spoke.Ca), "ca")
		if caCleanup != nil {
			defer caCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write CA to disk: %w", err)
		}
		joinArgs = append([]string{fmt.Sprintf("--ca-file=%s", caFile)}, joinArgs...)
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

	joinArgs, cleanupKcfg, err := common.PrepareKubeconfig(ctx, kClient, spoke.Kubeconfig, joinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm join", "args", joinArgs)

	cmd := exec.Command(clusteradm, joinArgs...)
	out, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm join' to complete for spoke %s...", spoke.Name))
	if err != nil {
		return fmt.Errorf("clusteradm join command failed for spoke %s: %v, output: %s", spoke.Name, err, string(out))
	}
	logger.V(1).Info("successfully requested spoke cluster join", "output", string(out))

	return nil
}

// spokeNeedsUpgrade checks if the klusterlet on a Spoke cluster has the desired bundle version
func spokeNeedsUpgrade(ctx context.Context, kClient client.Client, spoke v1alpha1.Spoke) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("spokeNeedsUpgrade", "spokeClusterName", spoke.Name)

	if spoke.Klusterlet.Source.BundleVersion == "default" {
		logger.V(0).Info("klusterlet bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if spoke.Klusterlet.Source.BundleVersion == "latest" {
		logger.V(0).Info("klusterlet bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	kubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, kClient, spoke.Kubeconfig)
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

	logger.V(0).Info("found klusterlet bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", spoke.Klusterlet.Source.BundleVersion,
	)
	return activeBundleVersion == spoke.Klusterlet.Source.BundleVersion, nil
}

// upgradeSpoke upgrades the Spoke cluster's klusterlet to the specified version
func upgradeSpoke(ctx context.Context, kClient client.Client, spoke v1alpha1.Spoke) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeSpoke", "spoke", spoke.Name)

	upgradeArgs := []string{"upgrade", "klusterlet",
		"--bundle-version", spoke.Klusterlet.Source.BundleVersion,
		"--image-registry", spoke.Klusterlet.Source.Registry,
		"--wait=true",
	}

	upgradeArgs, cleanupKcfg, err := common.PrepareKubeconfig(ctx, kClient, spoke.Kubeconfig, upgradeArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}
	logger.V(1).Info("clusteradm upgrade klusterlet", "args", upgradeArgs)

	cmd := exec.Command(clusteradm, upgradeArgs...)
	out, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm upgrade klusterlet' to complete for spoke %s...", spoke.Name))
	if err != nil {
		return fmt.Errorf(
			"failed to upgrade klusterlet on spoke cluster %s to %s: %v, output: %s",
			spoke.Name, spoke.Klusterlet.Source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("klusterlet upgraded", "output", string(out))

	return nil
}

// cleanupSpokes deregisters Spoke cluster(s) from the Hub cluster via 'clusteradm unjoin'
func cleanupSpokes(ctx context.Context, kClient client.Client, mc *v1alpha1.FleetConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("cleanupSpokes", "fleetconfig", mc.Name)

	for _, spoke := range mc.Spec.Spokes {
		joinedCondition := mc.GetCondition(spoke.JoinType())
		if joinedCondition == nil || joinedCondition.Status != metav1.ConditionTrue {
			logger.V(0).Info("skipping cleanup for unjoined spoke cluster",
				"spoke", spoke.Name, "message", joinedCondition.Message, "reason", joinedCondition.Reason,
			)
			continue
		}

		if err := unjoinSpoke(ctx, kClient, spoke.Kubeconfig, spoke.Name, spoke.Klusterlet.PurgeOperator); err != nil {
			return err
		}
	}

	return nil
}

// unjoinSpoke unjoins a single spoke cluster from the Hub cluster via `clusteradm unjoin`
func unjoinSpoke(ctx context.Context, kClient client.Client, kubeconfig *v1alpha1.Kubeconfig, spokeName string, purgeOperator bool) error {
	logger := log.FromContext(ctx)

	unjoinArgs := []string{
		"unjoin",
		"--cluster-name", spokeName,
		fmt.Sprintf("--purge-operator=%t", purgeOperator),
	}
	unjoinArgs, cleanupKcfg, err := common.PrepareKubeconfig(ctx, kClient, kubeconfig, unjoinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %w", spokeName, err)
	}
	logger.V(1).Info("clusteradm unjoin", "args", unjoinArgs)

	cmd := exec.Command(clusteradm, unjoinArgs...)
	out, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm unjoin' to complete for spoke %s...", spokeName))
	if err != nil {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %v, output: %s", spokeName, err, string(out))
	}
	logger.V(1).Info("spoke cluster unjoined", "output", string(out))

	return nil
}

// deregisterSpoke fully deregisters a spoke cluster, including cleaning up all relevant resources on the hub
func deregisterSpoke(ctx context.Context, kClient client.Client, hubKubeconfig []byte, spoke *v1alpha1.JoinedSpoke) error {
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
	if len(manifestWorks.Items) > 0 {
		msg := fmt.Sprintf("Found manifestWorks for ManagedCluster %s; cannot unjoin spoke cluster while it has active ManifestWorks", managedCluster.Name)
		logger.Info(msg)
		return errors.New(msg)
	}

	// unjoin spoke
	if err := unjoinSpoke(ctx, kClient, spoke.Kubeconfig, spoke.Name, spoke.PurgeKlusterletOperator); err != nil {
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
	if err = clusterC.ClusterV1().ManagedClusters().Delete(ctx, spoke.Name, metav1.DeleteOptions{}); err != nil && !kerrs.IsNotFound(err) {
		return err
	}

	// remove Namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spoke.Name}}
	if err := kClient.Delete(ctx, ns); err != nil && !kerrs.IsNotFound(err) {
		return err
	}

	return nil
}
