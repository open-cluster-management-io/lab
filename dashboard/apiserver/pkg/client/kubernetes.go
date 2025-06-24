package client

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

// CreateKubernetesClient initializes a connection to the Kubernetes API
func CreateKubernetesClient() *OCMClient {
	// Get kubeconfig
	var kubeconfig string

	// Check if running in-cluster
	_, inClusterErr := rest.InClusterConfig()
	inCluster := inClusterErr == nil

	if !inCluster {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Create kubernetes client
	var config *rest.Config
	var err error
	if inCluster {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("Error creating in-cluster config: %v", err)
		}
		log.Println("Using in-cluster configuration")
	} else {
		// First try to use the KUBECONFIG environment variable
		kubeconfigEnv := os.Getenv("KUBECONFIG")
		if kubeconfigEnv != "" {
			log.Printf("Using KUBECONFIG from environment: %s", kubeconfigEnv)
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
			if err != nil {
				log.Printf("Error building kubeconfig from KUBECONFIG env: %v", err)
				// Fall back to command line flag or default
			}
		}

		// If KUBECONFIG env var didn't work, try the flag or default path
		if config == nil {
			log.Printf("Using kubeconfig from flag or default: %s", kubeconfig)
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				// Try the load rules (will check multiple locations)
				log.Printf("Trying default client config loading rules")
				loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
				configOverrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmdapi.Cluster{Server: ""}}
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				config, err = kubeConfig.ClientConfig()
				if err != nil {
					log.Fatalf("Error building kubeconfig using defaults: %v", err)
				}
			}
		}
	}

	// Create OCM client
	ocmClient, err := CreateOCMClient(config)
	if err != nil {
		log.Fatalf("Error creating OCM client: %v", err)
	}

	// Debug message to verify connection
	log.Println("Successfully created Kubernetes client")

	return ocmClient
}
