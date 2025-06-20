package client

import (
	"log"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	addonv1alpha1informers "open-cluster-management.io/api/client/addon/informers/externalversions"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	workv1informers "open-cluster-management.io/api/client/work/informers/externalversions"
)

// OCMClient holds clients for OCM resources
type OCMClient struct {
	// Dynamic client for backward compatibility
	dynamic.Interface

	// Standard Kubernetes client for authentication operations
	KubernetesClient kubernetes.Interface

	// OCM typed clients
	ClusterClient clusterv1client.Interface
	AddonClient   addonv1alpha1client.Interface
	WorkClient    workv1client.Interface

	// OCM informers
	ClusterInformerFactory clusterv1informers.SharedInformerFactory
	AddonInformerFactory   addonv1alpha1informers.SharedInformerFactory
	WorkInformerFactory    workv1informers.SharedInformerFactory
}

// CreateOCMClient initializes OCM clients using the provided config
func CreateOCMClient(config *rest.Config) (*OCMClient, error) {
	// Create dynamic client (for backward compatibility)
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create standard Kubernetes client
	kubernetesClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create cluster client
	clusterClient, err := clusterv1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create addon client
	addonClient, err := addonv1alpha1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create work client
	workClient, err := workv1client.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Create informer factories
	clusterInformerFactory := clusterv1informers.NewSharedInformerFactory(clusterClient, 0)
	addonInformerFactory := addonv1alpha1informers.NewSharedInformerFactory(addonClient, 0)
	workInformerFactory := workv1informers.NewSharedInformerFactory(workClient, 0)

	log.Println("Successfully created OCM clients")

	return &OCMClient{
		Interface:              dynamicClient,
		KubernetesClient:       kubernetesClient,
		ClusterClient:          clusterClient,
		AddonClient:            addonClient,
		WorkClient:             workClient,
		ClusterInformerFactory: clusterInformerFactory,
		AddonInformerFactory:   addonInformerFactory,
		WorkInformerFactory:    workInformerFactory,
	}, nil
}
