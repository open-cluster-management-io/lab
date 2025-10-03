package v1beta1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strconv"

	"dario.cat/mergo"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonapi "open-cluster-management.io/api/client/addon/clientset/versioned"
	workapi "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	operatorv1 "open-cluster-management.io/api/operator/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	arg_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/args"
	exec_utils "github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/exec"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/file"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/hash"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/version"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/pkg/common"
)

// cleanup cleans up a Spoke and its associated resources.
func (r *SpokeReconciler) cleanup(ctx context.Context, spoke *v1beta1.Spoke, hubKubeconfig []byte) error {
	switch r.InstanceType {
	case v1beta1.InstanceTypeManager:
		pivotComplete := spoke.PivotComplete()
		err := r.doHubCleanup(ctx, spoke, hubKubeconfig, pivotComplete)
		if err != nil {
			return err
		}
		if spoke.IsHubAsSpoke() || !pivotComplete {
			err = r.doSpokeCleanup(ctx, spoke, false)
			if err != nil {
				return err
			}
		}
		return nil
	case v1beta1.InstanceTypeUnified:
		err := r.doHubCleanup(ctx, spoke, hubKubeconfig, false)
		if err != nil {
			return err
		}
		return r.doSpokeCleanup(ctx, spoke, false)
	case v1beta1.InstanceTypeAgent:
		return r.doSpokeCleanup(ctx, spoke, true)
	default:
		// this is guarded against when the manager is initialized. should never reach this point
		panic(fmt.Sprintf("unknown instance type %s. Must be one of %v", r.InstanceType, v1beta1.SupportedInstanceTypes))
	}
}

// handleSpoke manages Spoke cluster join and upgrade operations
func (r *SpokeReconciler) handleSpoke(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta) error {
	klusterletValues, err := r.mergeKlusterletValues(ctx, spoke)
	if err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.KlusterletSynced, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}

	switch r.InstanceType {
	case v1beta1.InstanceTypeManager:
		err = r.doHubWork(ctx, spoke, hubMeta, klusterletValues)
		if err != nil {
			return err
		}
		if spoke.IsHubAsSpoke() { // hub-as-spoke
			err = r.doSpokeWork(ctx, spoke, hubMeta.hub, klusterletValues)
			if err != nil {
				spoke.SetConditions(true, v1beta1.NewCondition(
					err.Error(), v1beta1.KlusterletSynced, metav1.ConditionFalse, metav1.ConditionTrue,
				))
				return err
			}
		}
		return nil
	case v1beta1.InstanceTypeUnified:
		err = r.doHubWork(ctx, spoke, hubMeta, klusterletValues)
		if err != nil {
			return err
		}
		err = r.doSpokeWork(ctx, spoke, hubMeta.hub, klusterletValues)
		if err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.KlusterletSynced, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}
		return nil
	case v1beta1.InstanceTypeAgent:
		err = r.doSpokeWork(ctx, spoke, hubMeta.hub, klusterletValues)
		if err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.KlusterletSynced, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}
		return nil
	default:
		// this is guarded against when the manager is initialized. should never reach this point
		panic(fmt.Sprintf("unknown cluster type %s. Must be one of %v", r.InstanceType, v1beta1.SupportedInstanceTypes))
	}
}

// doHubWork handles hub-side work such as joins and addons
func (r *SpokeReconciler) doHubWork(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta, klusterletValues *v1beta1.KlusterletChartConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleSpoke", "spoke", spoke.Name)

	hubKubeconfig := hubMeta.kubeconfig

	clusterClient, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create addon client: %w", err)
	}

	// check if the spoke has already been joined to the hub
	managedCluster, err := common.GetManagedCluster(ctx, clusterClient, spoke.Name)
	if err != nil {
		logger.Error(err, "failed to get managedCluster", "spoke", spoke.Name)
		return err
	}

	// attempt to join the spoke cluster if it hasn't already been joined
	if managedCluster == nil {
		spokeKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, spoke.Spec.Kubeconfig, spoke.Namespace)
		if err != nil {
			return fmt.Errorf("failed to load spoke kubeconfig: %v", err)
		}
		if err := r.joinSpoke(ctx, spoke, hubMeta, klusterletValues, spokeKubeconfig); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}

		// Accept the cluster join request
		if err := acceptCluster(ctx, spoke, false); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
			))
			return err
		}

		managedCluster, err = common.GetManagedCluster(ctx, clusterClient, spoke.Name)
		if err != nil {
			logger.Error(err, "failed to get managedCluster after join", "spoke", spoke.Name)
			return err
		}

		// precreate the namespace that the agent will be installed into
		// this prevents it from being automatically garbage collected when the spoke is deregistered
		if r.InstanceType != v1beta1.InstanceTypeUnified {
			err = r.createAgentNamespace(ctx, spoke.Name, spokeKubeconfig)
			if err != nil {
				logger.Error(err, "failed to create agent namespace", "spoke", spoke.Name)
				return err
			}
		}

	}

	// check managed clusters joined condition
	jc := r.getJoinedCondition(managedCluster)
	if jc == nil {
		logger.V(0).Info("waiting for spoke cluster to join", "name", spoke.Name)
		msg := fmt.Sprintf("ManagedClusterJoined condition not found in ManagedCluster for spoke cluster %s", spoke.Name)
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		// Re-accept all join requests for the spoke cluster
		if err := acceptCluster(ctx, spoke, true); err != nil {
			logger.Error(err, "failed to accept spoke cluster join request(s)", "spoke", spoke.Name)
		}
		return nil
	}

	logger.V(0).Info("found join condition", "reason", jc.Reason, "status", jc.Status, "message", jc.Message)
	if jc.Status != metav1.ConditionTrue {
		msg := fmt.Sprintf("failed to join spoke cluster %s: %s", spoke.Name, jc.Message)
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.SpokeJoined, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return errors.New(msg)
	}

	// spoke cluster has joined successfully
	spoke.SetConditions(true, v1beta1.NewCondition(
		"Joined", v1beta1.SpokeJoined, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	// Label the spoke ManagedCluster if in hub-as-spoke mode.
	// This allows the 'spoke' ManagedClusterSet to omit the hub-as-spoke cluster from its list
	// of spoke clusters.
	if managedCluster != nil && spoke.IsHubAsSpoke() {
		if managedCluster.Labels == nil {
			managedCluster.Labels = make(map[string]string)
		}
		managedCluster.Labels[v1beta1.LabelManagedClusterType] = v1beta1.ManagedClusterTypeHubAsSpoke
		if err := common.UpdateManagedCluster(ctx, clusterClient, managedCluster); err != nil {
			return err
		}
		logger.V(0).Info("labeled ManagedCluster as hub-as-spoke", "name", spoke.Name)
	}

	err = r.deleteKubeconfigSecret(ctx, spoke)
	if err != nil {
		logger.Error(err, "failed to remove spoke's kubeconfig secret", "spoke", spoke.Name)
		return err
	}

	if !spoke.IsHubAsSpoke() {
		adc := &addonv1alpha1.AddOnDeploymentConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v1beta1.FCCAddOnName,
				Namespace: spoke.Name,
			},
			Spec: addonv1alpha1.AddOnDeploymentConfigSpec{
				AgentInstallNamespace: os.Getenv(v1beta1.ControllerNamespaceEnvVar),
				CustomizedVariables: []addonv1alpha1.CustomizedVariable{
					{
						Name:  v1beta1.HubNamespaceEnvVar,
						Value: spoke.Spec.HubRef.Namespace,
					},
					{
						Name:  v1beta1.SpokeNamespaceEnvVar,
						Value: spoke.Namespace,
					},
					{
						Name:  v1beta1.PurgeAgentNamespaceEnvVar,
						Value: strconv.FormatBool(spoke.Spec.CleanupConfig.PurgeAgentNamespace),
					},
				},
			},
		}
		_, err = addonC.AddonV1alpha1().AddOnDeploymentConfigs(spoke.Name).Create(ctx, adc, metav1.CreateOptions{})
		if err != nil && !kerrs.IsAlreadyExists(err) {
			return err
		}

		err = r.bindAddonAgent(ctx, spoke)
		if err != nil {
			return err
		}
	}

	enabledAddons, err := handleSpokeAddons(ctx, addonC, spoke)
	if err != nil {
		msg := fmt.Sprintf("failed to enable addons for spoke cluster %s: %s", spoke.Name, err.Error())
		spoke.SetConditions(true, v1beta1.NewCondition(
			msg, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionTrue,
		))
		return err
	}
	spoke.Status.EnabledAddons = enabledAddons
	return nil
}

func (r *SpokeReconciler) createAgentNamespace(ctx context.Context, spokeName string, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	spokeRestCfg, err := kube.RestConfigFromKubeconfig(spokeKubeconfig)
	if err != nil {
		return err
	}
	spokeCli, err := client.New(spokeRestCfg, client.Options{})
	if err != nil {
		return err
	}
	agentNamespace := os.Getenv(v1beta1.ControllerNamespaceEnvVar) // manager.go enforces that this is not ""
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: agentNamespace,
		},
	}
	err = spokeCli.Create(ctx, ns)
	if err != nil && !kerrs.IsAlreadyExists(err) {
		return err
	}
	logger.V(1).Info("agent namespace configured", "spoke", spokeName, "namespace", agentNamespace)
	return nil
}

func (r *SpokeReconciler) deleteKubeconfigSecret(ctx context.Context, spoke *v1beta1.Spoke) error {
	if r.InstanceType != v1beta1.InstanceTypeManager ||
		!spoke.PivotComplete() ||
		spoke.Spec.Kubeconfig.InCluster ||
		!spoke.Spec.CleanupConfig.PurgeKubeconfigSecret {
		return nil
	}

	logger := log.FromContext(ctx)
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spoke.Spec.Kubeconfig.SecretReference.Name,
			Namespace: spoke.Namespace,
		},
	}
	err := r.Delete(ctx, sec)
	if err != nil && !kerrs.IsNotFound(err) {
		return err
	}
	logger.V(1).Info("kubeconfig secret purged", "spoke", spoke.Name)
	return nil
}

// doSpokeWork handles spoke-side work such as upgrades
func (r *SpokeReconciler) doSpokeWork(ctx context.Context, spoke *v1beta1.Spoke, hub *v1beta1.Hub, klusterletValues *v1beta1.KlusterletChartConfig) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("handleSpoke", "spoke", spoke.Name)

	spoke.SetConditions(true, v1beta1.NewCondition(
		v1beta1.PivotComplete, v1beta1.PivotComplete, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	spokeKubeconfig, err := kube.RawFromInClusterRestConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig from inCluster: %v", err)
	}
	// attempt an upgrade whenever the klusterlet's bundleVersion or values change
	currKlusterletHash, err := hash.ComputeHash(klusterletValues)
	if err != nil {
		return fmt.Errorf("failed to compute hash of spoke %s klusterlet values: %w", spoke.Name, err)
	}
	if hub != nil && hub.Spec.ClusterManager != nil && hub.Spec.ClusterManager.Source.BundleVersion != "" {
		upgrade, err := r.spokeNeedsUpgrade(ctx, spoke, currKlusterletHash, hub.Spec.ClusterManager.Source, spokeKubeconfig)
		if err != nil {
			return fmt.Errorf("failed to check if spoke cluster needs upgrade: %w", err)
		}

		if upgrade {
			if err := r.upgradeSpoke(ctx, spoke, klusterletValues, hub.Spec.ClusterManager.Source, spokeKubeconfig); err != nil {
				return fmt.Errorf("failed to upgrade spoke cluster %s: %w", spoke.Name, err)
			}
		}
	}
	spoke.Status.KlusterletHash = currKlusterletHash

	spoke.SetConditions(true, v1beta1.NewCondition(
		v1beta1.KlusterletSynced, v1beta1.KlusterletSynced, metav1.ConditionTrue, metav1.ConditionTrue,
	))

	return nil
}

// doHubCleanup handles all the required cleanup of a hub cluster when deregistering a Spoke
func (r *SpokeReconciler) doHubCleanup(ctx context.Context, spoke *v1beta1.Spoke, hubKubeconfig []byte, pivotComplete bool) error {
	logger := log.FromContext(ctx)
	clusterC, err := common.ClusterClient(hubKubeconfig)
	if err != nil {
		return err
	}
	workC, err := common.WorkClient(hubKubeconfig)
	if err != nil {
		return err
	}
	addonC, err := common.AddOnClient(hubKubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create addon client for cleanup: %w", err)
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
	spokeCopy := spoke.DeepCopy()
	spokeCopy.Spec.AddOns = nil

	// for hub-as-spoke, or if the addon agent never came up, disable all addons
	// otherwise, leave fleetconfig-controller-agent addon running so that it can do deregistration
	shouldCleanAll := spoke.IsHubAsSpoke() || !pivotComplete || r.InstanceType == v1beta1.InstanceTypeUnified

	if !shouldCleanAll {
		spokeCopy.Spec.AddOns = append(spokeCopy.Spec.AddOns, v1beta1.AddOn{ConfigName: v1beta1.FCCAddOnName})
	}
	if _, err := handleSpokeAddons(ctx, addonC, spokeCopy); err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionFalse,
		))
		return err
	}

	if len(spoke.Status.EnabledAddons) > 0 {
		// Wait for addon manifestWorks to be fully cleaned up before proceeding with unjoin
		if err := waitForAddonManifestWorksCleanup(ctx, workC, spoke.Name, addonCleanupTimeout, shouldCleanAll); err != nil {
			spoke.SetConditions(true, v1beta1.NewCondition(
				err.Error(), v1beta1.AddonsConfigured, metav1.ConditionTrue, metav1.ConditionFalse,
			))
			return fmt.Errorf("addon manifestWorks cleanup failed: %w", err)
		}
		spoke.SetConditions(true, v1beta1.NewCondition(
			v1beta1.AddonsConfigured, v1beta1.AddonsConfigured, metav1.ConditionFalse, metav1.ConditionFalse,
		))
	}

	// remove preflight cleanup finalizer - this lets the spoke's controller know to proceed with unjoin.
	spoke.Finalizers = slices.DeleteFunc(spoke.Finalizers, func(s string) bool {
		return s == v1beta1.HubCleanupPreflightFinalizer
	})

	// requeue until unjoin is complete by the spoke's controller
	if slices.Contains(spoke.Finalizers, v1beta1.SpokeCleanupFinalizer) {
		logger.V(1).Info("Hub preflight complete, waiting for spoke agent to deregister")
		return nil
	}

	csrList := &certificatesv1.CertificateSigningRequestList{}
	if err := r.List(ctx, csrList, client.HasLabels{"open-cluster-management.io/cluster-name"}); err != nil {
		return err
	}
	for _, c := range csrList.Items {
		trimmedName := csrSuffixPattern.ReplaceAllString(c.Name, "")
		if trimmedName == spoke.Name {
			if err := r.Delete(ctx, &c); err != nil {
				return err
			}
		}
	}

	err = r.waitForAgentAddonDeleted(ctx, spoke, spokeCopy, addonC, workC)
	if err != nil {
		return err
	}

	// remove ManagedCluster
	err = clusterC.ClusterV1().ManagedClusters().Delete(ctx, spoke.Name, metav1.DeleteOptions{})
	if err != nil && !kerrs.IsNotFound(err) {
		return err
	}
	// remove Namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: spoke.Name}}
	err = r.Delete(ctx, ns)
	if err != nil && !kerrs.IsNotFound(err) {
		return err
	}

	spoke.Finalizers = slices.DeleteFunc(spoke.Finalizers, func(s string) bool {
		return s == v1beta1.HubCleanupFinalizer
	})

	return nil
}

func (r *SpokeReconciler) waitForAgentAddonDeleted(ctx context.Context, spoke *v1beta1.Spoke, spokeCopy *v1beta1.Spoke, addonC *addonapi.Clientset, workC *workapi.Clientset) error {
	// delete fcc agent addon
	spokeCopy.Spec.AddOns = nil
	if _, err := handleSpokeAddons(ctx, addonC, spokeCopy); err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
		))
		return err
	}

	// at this point, klusterlet-work-agent is uninstalled, so nothing can remove this finalizer. all resources are cleaned up by the spoke's controller, so to prevent a dangling mw/namespace, we remove the finalizer manually
	mwList, err := workC.WorkV1().ManifestWorks(spoke.Name).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", manifestWorkAddOnLabelKey, v1beta1.FCCAddOnName)})
	if err != nil {
		return err
	}
	for _, mw := range mwList.Items {
		patchBytes, err := json.Marshal(map[string]any{
			"metadata": map[string]any{
				"finalizers": nil,
			},
		})
		if err != nil {
			return err
		}

		_, err = workC.WorkV1().ManifestWorks(spoke.Name).Patch(
			ctx,
			mw.Name,
			types.MergePatchType,
			patchBytes,
			metav1.PatchOptions{},
		)
		if err != nil && !kerrs.IsNotFound(err) {
			return err
		}
	}

	// Wait for all manifestWorks to be cleaned up
	if err := waitForAddonManifestWorksCleanup(ctx, workC, spoke.Name, addonCleanupTimeout, true); err != nil {
		spoke.SetConditions(true, v1beta1.NewCondition(
			err.Error(), v1beta1.CleanupFailed, metav1.ConditionTrue, metav1.ConditionFalse,
		))
		return fmt.Errorf("addon manifestWorks cleanup failed: %w", err)
	}
	return nil
}

// doSpokeCleanup handles all the required cleanup of a spoke cluster when deregistering a Spoke
func (r *SpokeReconciler) doSpokeCleanup(ctx context.Context, spoke *v1beta1.Spoke, pivotComplete bool) error {
	logger := log.FromContext(ctx)
	// requeue until preflight is complete by the hub's controller
	if slices.Contains(spoke.Finalizers, v1beta1.HubCleanupPreflightFinalizer) {
		logger.V(1).Info("Cleanup initiated, waiting for hub to complete preflight")
		return nil
	}

	var (
		spokeKubeconfig []byte
		err             error
	)

	// if the addon agent did not come up successfully, try to unjoin the spoke from the hub
	if pivotComplete {
		spokeKubeconfig, err = kube.RawFromInClusterRestConfig()
	} else {
		spokeKubeconfig, err = kube.KubeconfigFromSecretOrCluster(ctx, r.Client, spoke.Spec.Kubeconfig, spoke.Namespace)
	}
	if err != nil {
		return err
	}

	err = r.unjoinSpoke(ctx, spoke, spokeKubeconfig)
	if err != nil {
		return err
	}

	// unified manager/hub-as-spoke/failed pivot case, no further cleanup needed - clusteradm unjoin will have handled it all
	if r.InstanceType != v1beta1.InstanceTypeAgent {
		spoke.Finalizers = slices.DeleteFunc(spoke.Finalizers, func(s string) bool {
			return s == v1beta1.SpokeCleanupFinalizer
		})
		return nil
	}

	// remove all remaining klusterlet resources that unjoin did not remove (because of the remaining AMW)
	if spoke.Spec.CleanupConfig.PurgeKlusterletOperator {
		restCfg, err := kube.RestConfigFromKubeconfig(spokeKubeconfig)
		if err != nil {
			return err
		}
		spokeClient, err := client.New(restCfg, client.Options{})
		if err != nil {
			return err
		}
		operatorClient, err := common.OperatorClient(spokeKubeconfig)
		if err != nil {
			return err
		}

		if err := operatorClient.OperatorV1().Klusterlets().Delete(ctx, "klusterlet", metav1.DeleteOptions{}); err != nil && !kerrs.IsNotFound(err) {
			return err
		}

		for _, nsName := range v1beta1.OCMSpokeNamespaces {
			if nsName == "" {
				continue
			}
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
			if err := spokeClient.Delete(ctx, ns); err != nil && !kerrs.IsNotFound(err) {
				return err
			}
		}
	}

	spoke.Finalizers = slices.DeleteFunc(spoke.Finalizers, func(s string) bool {
		return s == v1beta1.SpokeCleanupFinalizer
	})

	logger.V(1).Info("Klusterlet cleanup complete")
	return nil
}

type tokenMeta struct {
	Token        string `json:"hub-token"`
	HubAPIServer string `json:"hub-apiserver"`
}

type hubMeta struct {
	hub        *v1beta1.Hub
	kubeconfig []byte
}

// joinSpoke joins a Spoke cluster to the Hub cluster
func (r *SpokeReconciler) joinSpoke(ctx context.Context, spoke *v1beta1.Spoke, hubMeta hubMeta, klusterletValues *v1beta1.KlusterletChartConfig, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("joinSpoke", "spoke", spoke.Name)

	hub := hubMeta.hub

	if hub == nil {
		return errors.New("hub not found")
	}
	// dont start join until the hub is ready
	hubInitCond := hubMeta.hub.GetCondition(v1beta1.HubInitialized)
	if hubInitCond == nil || hubInitCond.Status != metav1.ConditionTrue {
		return errors.New("hub does not have initialized condition")
	}

	tokenMeta, err := getToken(ctx, hubMeta)
	if err != nil {
		return fmt.Errorf("failed to get join token: %w", err)
	}

	joinArgs := append([]string{
		"join",
		"--cluster-name", spoke.Name,
		fmt.Sprintf("--create-namespace=%t", spoke.Spec.CreateNamespace),
		fmt.Sprintf("--enable-sync-labels=%t", spoke.Spec.SyncLabels),
		"--hub-token", tokenMeta.Token,
		"--wait=true",
		// klusterlet args
		"--mode", spoke.Spec.Klusterlet.Mode,
		"--feature-gates", spoke.Spec.Klusterlet.FeatureGates,
		fmt.Sprintf("--force-internal-endpoint-lookup=%t", spoke.Spec.Klusterlet.ForceInternalEndpointLookup),
		fmt.Sprintf("--singleton=%t", spoke.Spec.Klusterlet.Singleton),
	}, spoke.BaseArgs()...)

	if hub.Spec.ClusterManager != nil {
		// source args
		joinArgs = append(joinArgs,
			"--bundle-version", hub.Spec.ClusterManager.Source.BundleVersion,
			"--image-registry", hub.Spec.ClusterManager.Source.Registry)
	}
	for k, v := range spoke.Spec.Klusterlet.Annotations {
		joinArgs = append(joinArgs, fmt.Sprintf("--klusterlet-annotation=%s=%s", k, v))
	}

	// resources args
	joinArgs = append(joinArgs, arg_utils.PrepareResources(spoke.Spec.Klusterlet.Resources)...)

	// Use hub API server from spec if provided and not forced to use internal endpoint,
	// otherwise fall back to the hub API server from the tokenMeta
	if hub.Spec.APIServer != "" && !spoke.Spec.Klusterlet.ForceInternalEndpointLookup {
		joinArgs = append(joinArgs, "--hub-apiserver", hub.Spec.APIServer)
	} else if tokenMeta.HubAPIServer != "" {
		joinArgs = append(joinArgs, "--hub-apiserver", tokenMeta.HubAPIServer)
	}

	if hub.Spec.Ca != "" {
		caFile, caCleanup, err := file.TmpFile([]byte(hub.Spec.Ca), "ca")
		if caCleanup != nil {
			defer caCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write hub CA to disk: %w", err)
		}
		joinArgs = append([]string{fmt.Sprintf("--ca-file=%s", caFile)}, joinArgs...)
	}

	ra := hub.Spec.RegistrationAuth
	if ra.Driver == v1alpha1.AWSIRSARegistrationDriver {
		raArgs := []string{
			fmt.Sprintf("--registration-auth=%s", ra.Driver),
		}
		if ra.HubClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--hub-cluster-arn=%s", ra.HubClusterARN))
		}
		if spoke.Spec.ClusterARN != "" {
			raArgs = append(raArgs, fmt.Sprintf("--managed-cluster-arn=%s", spoke.Spec.ClusterARN))
		}

		joinArgs = append(joinArgs, raArgs...)
	}

	if spoke.Spec.Klusterlet.Mode == string(operatorv1.InstallModeHosted) {
		joinArgs = append(joinArgs,
			fmt.Sprintf("--force-internal-endpoint-lookup-managed=%t", spoke.Spec.Klusterlet.ForceInternalEndpointLookupManaged),
		)
		raw, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, spoke.Spec.Klusterlet.ManagedClusterKubeconfig, spoke.Namespace)
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

	if spoke.Spec.ProxyCa != "" {
		proxyCaFile, proxyCaCleanup, err := file.TmpFile([]byte(spoke.Spec.ProxyCa), "proxy-ca")
		if proxyCaCleanup != nil {
			defer proxyCaCleanup()
		}
		if err != nil {
			return fmt.Errorf("failed to write proxy CA to disk: %w", err)
		}
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-ca-file=%s", proxyCaFile))
	}
	if spoke.Spec.ProxyURL != "" {
		joinArgs = append(joinArgs, fmt.Sprintf("--proxy-url=%s", spoke.Spec.ProxyURL))
	}

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(klusterletValues)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	joinArgs = append(joinArgs, valuesArgs...)

	joinArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, joinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm join", "args", arg_utils.SanitizeArgs(joinArgs))

	cmd := exec.Command(clusteradm, joinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm join' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("clusteradm join command failed for spoke %s: %v, output: %s", spoke.Name, err, string(out))
	}
	logger.V(1).Info("successfully requested spoke cluster join", "output", string(stdout))

	return nil
}

// acceptCluster accepts a Spoke cluster's join request
func acceptCluster(ctx context.Context, spoke *v1beta1.Spoke, skipApproveCheck bool) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("acceptCluster", "spoke", spoke.Name)

	acceptArgs := append([]string{
		"accept", "--cluster", spoke.Name,
	}, spoke.BaseArgs()...)

	logger.V(1).Info("clusteradm accept", "args", arg_utils.SanitizeArgs(acceptArgs))

	// TODO: handle other args:
	// --requesters=[]:
	//     Common Names of agents to be approved.

	if skipApproveCheck {
		acceptArgs = append(acceptArgs, "--skip-approve-check")
	}

	cmd := exec.Command(clusteradm, acceptArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm accept' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf("failed to accept spoke cluster join request: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("spoke cluster join request accepted", "output", string(stdout))

	return nil
}

// getJoinedCondition gets the joined condition from a managed cluster
func (r *SpokeReconciler) getJoinedCondition(managedCluster *clusterv1.ManagedCluster) *metav1.Condition {
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

// spokeNeedsUpgrade checks if the klusterlet on a Spoke cluster requires an upgrade
func (r *SpokeReconciler) spokeNeedsUpgrade(ctx context.Context, spoke *v1beta1.Spoke, currKlusterletHash string, source v1beta1.OCMSource, spokeKubeconfig []byte) (bool, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("spokeNeedsUpgrade", "spokeClusterName", spoke.Name)

	prevHash := spoke.Status.KlusterletHash
	hashChanged := prevHash != currKlusterletHash && prevHash != ""
	logger.V(2).Info("comparing klusterlet values hash",
		"spoke", spoke.Name,
		"prevHash", spoke.Status.KlusterletHash,
		"currHash", currKlusterletHash,
	)
	if hashChanged {
		logger.V(0).Info("hash changed", "old", spoke.Status.KlusterletHash, "new", currKlusterletHash)
		return true, nil
	}

	if source.BundleVersion == v1beta1.BundleVersionDefault {
		logger.V(0).Info("klusterlet bundleVersion is default, skipping upgrade")
		return false, nil
	}
	if source.BundleVersion == v1beta1.BundleVersionLatest {
		logger.V(0).Info("klusterlet bundleVersion is latest, attempting upgrade")
		return true, nil
	}

	operatorC, err := common.OperatorClient(spokeKubeconfig)
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
	desiredBundleVersion, err := version.Normalize(source.BundleVersion)
	if err != nil {
		return false, err
	}

	logger.V(0).Info("found klusterlet bundleVersions",
		"activeBundleVersion", activeBundleVersion,
		"desiredBundleVersion", desiredBundleVersion,
	)
	return activeBundleVersion != desiredBundleVersion, nil
}

// upgradeSpoke upgrades the Spoke cluster's klusterlet
func (r *SpokeReconciler) upgradeSpoke(ctx context.Context, spoke *v1beta1.Spoke, klusterletValues *v1beta1.KlusterletChartConfig, source v1beta1.OCMSource, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("upgradeSpoke", "spoke", spoke.Name)

	upgradeArgs := append([]string{
		"upgrade", "klusterlet",
		"--bundle-version", source.BundleVersion,
		"--image-registry", source.Registry,
		"--wait=true",
	}, spoke.BaseArgs()...)

	valuesArgs, valuesCleanup, err := prepareKlusterletValuesFile(klusterletValues)
	if valuesCleanup != nil {
		defer valuesCleanup()
	}
	if err != nil {
		return err
	}
	upgradeArgs = append(upgradeArgs, valuesArgs...)

	upgradeArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, upgradeArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return err
	}

	logger.V(1).Info("clusteradm upgrade klusterlet", "args", arg_utils.SanitizeArgs(upgradeArgs))

	cmd := exec.Command(clusteradm, upgradeArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm upgrade klusterlet' to complete for spoke %s...", spoke.Name))
	if err != nil {
		out := append(stdout, stderr...)
		return fmt.Errorf(
			"failed to upgrade klusterlet on spoke cluster %s to %s: %v, output: %s",
			spoke.Name, source.BundleVersion, err, string(out),
		)
	}
	logger.V(1).Info("klusterlet upgraded", "output", string(stdout))

	return nil
}

// unjoinSpoke unjoins a spoke from the hub
func (r *SpokeReconciler) unjoinSpoke(ctx context.Context, spoke *v1beta1.Spoke, spokeKubeconfig []byte) error {
	logger := log.FromContext(ctx)
	logger.V(0).Info("unjoinSpoke", "spoke", spoke.Name)

	unjoinArgs := append([]string{
		"unjoin",
		"--cluster-name", spoke.GetName(),
		fmt.Sprintf("--purge-operator=%t", spoke.Spec.CleanupConfig.PurgeKlusterletOperator),
	}, spoke.BaseArgs()...)

	unjoinArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, spokeKubeconfig, spoke.Spec.Kubeconfig.Context, unjoinArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %w", spoke.GetName(), err)
	}

	logger.V(1).Info("clusteradm unjoin", "args", arg_utils.SanitizeArgs(unjoinArgs))

	cmd := exec.Command(clusteradm, unjoinArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, fmt.Sprintf("waiting for 'clusteradm unjoin' to complete for spoke %s...", spoke.GetName()))
	out := append(stdout, stderr...)
	if err != nil { //|| strings.Contains(string(out), amwExistsError) {
		return fmt.Errorf("failed to unjoin spoke cluster %s: %v, output: %s", spoke.GetName(), err, string(out))
	}
	logger.V(1).Info("spoke cluster unjoined", "output", string(stdout))

	return nil
}

// getToken gets a join token from the Hub cluster via 'clusteradm get token'
func getToken(ctx context.Context, hubMeta hubMeta) (*tokenMeta, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("getToken")

	tokenArgs := append([]string{
		"get", "token", "--output=json",
	}, hubMeta.hub.BaseArgs()...)

	if hubMeta.hub.Spec.ClusterManager != nil {
		tokenArgs = append(tokenArgs, fmt.Sprintf("--use-bootstrap-token=%t", hubMeta.hub.Spec.ClusterManager.UseBootstrapToken))
	}
	tokenArgs, cleanupKcfg, err := arg_utils.PrepareKubeconfig(ctx, hubMeta.kubeconfig, hubMeta.hub.Spec.Kubeconfig.Context, tokenArgs)
	if cleanupKcfg != nil {
		defer cleanupKcfg()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to prepare kubeconfig: %w", err)
	}

	logger.V(1).Info("clusteradm get token", "args", arg_utils.SanitizeArgs(tokenArgs))

	cmd := exec.Command(clusteradm, tokenArgs...)
	stdout, stderr, err := exec_utils.CmdWithLogs(ctx, cmd, "waiting for 'clusteradm get token' to complete...")
	if err != nil {
		out := append(stdout, stderr...)
		return nil, fmt.Errorf("failed to get join token: %v, output: %s", err, string(out))
	}
	logger.V(1).Info("got join token", "output", arg_utils.Redacted)

	tokenMeta := &tokenMeta{}
	if err := json.Unmarshal(stdout, &tokenMeta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal join token: %w", err)
	}
	return tokenMeta, nil
}

// getHubMeta retrieves the Hub resource and it's associated kubeconfig
func (r *SpokeReconciler) getHubMeta(ctx context.Context, hubRef v1beta1.HubRef) (hubMeta, error) {
	hub := &v1beta1.Hub{}
	hubMeta := hubMeta{}
	nn := types.NamespacedName{Name: hubRef.Name, Namespace: hubRef.Namespace}

	// get Hub using local client
	err := r.Get(ctx, nn, hub)
	if err != nil {
		return hubMeta, client.IgnoreNotFound(err)
	}
	hubMeta.hub = hub
	// load the hub's kubeconfig. only needed on the hub's reconciler instance - the spoke's instance can access the hub using its default client
	if r.InstanceType != v1beta1.InstanceTypeAgent {
		hubKubeconfig, err := kube.KubeconfigFromSecretOrCluster(ctx, r.Client, hub.Spec.Kubeconfig, hub.Namespace)
		if err != nil {
			return hubMeta, err
		}
		hubMeta.kubeconfig = hubKubeconfig
	}
	return hubMeta, nil
}

// mergeKlusterletValues merges klusterlet values from a configmap in the Spoke namespace, and from the Spoke's spec. Spec takes precedence.
func (r *SpokeReconciler) mergeKlusterletValues(ctx context.Context, spoke *v1beta1.Spoke) (*v1beta1.KlusterletChartConfig, error) {
	logger := log.FromContext(ctx)

	if spoke.Spec.Klusterlet.ValuesFrom == nil && spoke.Spec.Klusterlet.Values == nil {
		logger.V(3).Info("no values or valuesFrom provided. Using default klusterlet chart values", "spoke", spoke.Name)
		return nil, nil
	}

	var fromInterface = map[string]any{}
	var specInterface = map[string]any{}

	if spoke.Spec.Klusterlet.ValuesFrom != nil {
		cm := &corev1.ConfigMap{}
		nn := types.NamespacedName{Name: spoke.Spec.Klusterlet.ValuesFrom.Name, Namespace: spoke.Namespace}
		err := r.Get(ctx, nn, cm)
		if err != nil {
			if kerrs.IsNotFound(err) {
				// cm not found, return spec's values
				logger.V(1).Info("warning: Klusterlet values ConfigMap not found", "spoke", spoke.Name, "configMap", nn)
				return spoke.Spec.Klusterlet.Values, nil
			}
			return nil, fmt.Errorf("failed to retrieve Klusterlet values ConfigMap %s: %w", nn, err)
		}
		fromValues, ok := cm.Data[spoke.Spec.Klusterlet.ValuesFrom.Key]
		if !ok {
			logger.V(1).Info("warning: Klusterlet values key not found in ConfigMap", "spoke", spoke.Name, "configMap", nn, "key", spoke.Spec.Klusterlet.ValuesFrom.Key)
			return spoke.Spec.Klusterlet.Values, nil
		}
		fromBytes := []byte(fromValues)
		err = yaml.Unmarshal(fromBytes, &fromInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML values from ConfigMap %s key %s: %w", nn, spoke.Spec.Klusterlet.ValuesFrom.Key, err)
		}
	}

	if spoke.Spec.Klusterlet.Values != nil {
		specBytes, err := yaml.Marshal(spoke.Spec.Klusterlet.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Klusterlet values from spoke spec for spoke %s: %w", spoke.Name, err)
		}
		err = yaml.Unmarshal(specBytes, &specInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal Klusterlet values from spoke spec for spoke %s: %w", spoke.Name, err)
		}
	}

	mergedMap := map[string]any{}
	maps.Copy(mergedMap, fromInterface)

	// Merge spec on top but ignore zero-values from spec
	if err := mergo.Map(&mergedMap, specInterface, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("merge failed for spoke %s: %w", spoke.Name, err)
	}

	mergedBytes, err := yaml.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged Klusterlet values for spoke %s: %w", spoke.Name, err)
	}

	merged := &v1beta1.KlusterletChartConfig{}
	err = yaml.Unmarshal(mergedBytes, merged)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal merged values into KlusterletChartConfig for spoke %s: %w", spoke.Name, err)
	}

	return merged, nil

}

// prepareKlusterletValuesFile creates a temporary file with klusterlet values and returns
// args to append and a cleanup function. Returns empty slice if values are empty.
func prepareKlusterletValuesFile(values *v1beta1.KlusterletChartConfig) ([]string, func(), error) {
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
