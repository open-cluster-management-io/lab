// Package kube contains helpers for interacting with a kubernetes cluster
package kube

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1alpha1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
)

// RestConfigFromKubeconfig either creates a rest.Config from a v1alpha1.Kubeconfig or
// returns an in-cluster config if the kubeconfig is nil.
func RestConfigFromKubeconfig(kubeconfig []byte) (*rest.Config, error) {
	if kubeconfig == nil {
		return ctrl.GetConfig()
	}
	config, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	return clientConfig.ClientConfig()
}

// RawFromRestConfig creates a raw kubeconfig from a REST Config
func RawFromRestConfig(rc *rest.Config) ([]byte, error) {
	// cluster config
	clusterConfig := &clientcmdapi.Cluster{
		Server: rc.Host,
	}
	if rc.CAFile != "" {
		clusterConfig.CertificateAuthority = rc.CAFile
	} else if rc.CAData != nil {
		clusterConfig.CertificateAuthorityData = rc.CAData
	}
	// auth config
	authInfo := &clientcmdapi.AuthInfo{}
	if rc.BearerToken != "" {
		authInfo.Token = rc.BearerToken
	} else if rc.CertData != nil && rc.KeyData != nil {
		authInfo.ClientCertificateData = rc.CertData
		authInfo.ClientKeyData = rc.KeyData
	}
	// finalize
	clientConfig := clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters: map[string]*clientcmdapi.Cluster{
			"default-cluster": clusterConfig,
		},
		Contexts: map[string]*clientcmdapi.Context{
			"default-context": {
				Cluster:  "default-cluster",
				AuthInfo: "default-user",
			},
		},
		CurrentContext: "default-context",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"default-user": authInfo,
		},
	}
	return clientcmd.Write(clientConfig)
}

// RawFromInClusterRestConfig creates a kubeconfig from an incluster rest config
func RawFromInClusterRestConfig() ([]byte, error) {
	rc, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	return RawFromRestConfig(rc)
}

// KubeconfigFromNamespacedSecretOrCluster loads a kubeconfig from a cross-namespace secret or generates one from inCluster
func KubeconfigFromNamespacedSecretOrCluster(ctx context.Context, kClient client.Client, kubeconfig v1alpha1.Kubeconfig) (raw []byte, err error) {
	// exactly 1 of these 2 cases is always true
	if kubeconfig.InCluster {
		return RawFromInClusterRestConfig()
	}
	return KubeconfigFromNamespacedSecret(ctx, kClient, kubeconfig)
}

// KubeconfigFromNamespacedSecret loads a kubeconfig from a cross-namespace secret in the cluster
func KubeconfigFromNamespacedSecret(ctx context.Context, kClient client.Client, kubeconfig v1alpha1.Kubeconfig) ([]byte, error) {
	secretRef := kubeconfig.SecretReference
	secret := corev1.Secret{}
	nn := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}
	if err := kClient.Get(ctx, nn, &secret); err != nil {
		return nil, err
	}

	raw, ok := secret.Data[secretRef.KubeconfigKey]
	if !ok {
		return nil, fmt.Errorf("kubeconfig key '%s' not found in %v secret", secretRef.KubeconfigKey, nn)
	}

	return raw, nil
}

// KubeconfigFromSecretOrCluster loads a kubeconfig from a secret or generates one from inCluster
func KubeconfigFromSecretOrCluster(ctx context.Context, kClient client.Client, kubeconfig v1beta1.Kubeconfig, namespace string) (raw []byte, err error) {
	// exactly 1 of these 2 cases is always true
	if kubeconfig.InCluster {
		return RawFromInClusterRestConfig()
	}
	return KubeconfigFromSecret(ctx, kClient, kubeconfig, namespace)
}

// KubeconfigFromSecret loads a kubeconfig from a secret in the cluster
func KubeconfigFromSecret(ctx context.Context, kClient client.Client, kubeconfig v1beta1.Kubeconfig, namespace string) ([]byte, error) {
	secretRef := kubeconfig.SecretReference
	secret := corev1.Secret{}
	nn := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}
	if err := kClient.Get(ctx, nn, &secret); err != nil {
		return nil, err
	}

	raw, ok := secret.Data[secretRef.KubeconfigKey]
	if !ok {
		return nil, fmt.Errorf("kubeconfig key '%s' not found in %v secret", secretRef.KubeconfigKey, nn)
	}

	return raw, nil
}
