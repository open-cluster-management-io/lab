package v1beta1

import (
	"context"
	"fmt"
	"strings"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-cluster-management-io/lab/fleetconfig-controller/api/v1beta1"
	"github.com/open-cluster-management-io/lab/fleetconfig-controller/internal/kube"
)

const (
	ocmHubNamespace       = "open-cluster-management-hub"
	grpcServerServiceName = "cluster-manager-grpc-server"
	grpcCABundleConfigMap = "ca-bundle-configmap"
	grpcCABundleDataKey   = "ca-bundle.crt"
	defaultHubGRPCPort    = 8090
)

// grpcHubKubernetesClient returns a clientset for the hub API server using the Hub kubeconfig.
func grpcHubKubernetesClient(ctx context.Context, hub *v1beta1.Hub, hubKubeconfig []byte) (*kubernetes.Clientset, error) {
	logger := log.FromContext(ctx)
	restCfg, err := kube.RestConfigFromKubeconfigWithContext(hubKubeconfig, hub.Spec.Kubeconfig.Context)
	if err != nil {
		logger.V(1).Info("gRPC hub client: kubeconfig", "hub", hub.Name, "error", err)
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		logger.V(1).Info("gRPC hub client: new clientset", "hub", hub.Name, "error", err)
		return nil, err
	}
	return k8s, nil
}

// grpcHubObservationEnabled is true when the hub runs or exposes gRPC registration material worth observing.
func grpcHubObservationEnabled(hub *v1beta1.Hub) bool {
	return hub.Spec.RegistrationAuth.Driver == v1beta1.GRPCRegistrationDriver ||
		hub.Spec.RegistrationAuth.GRPCInitEnabled()
}

// grpcHubLoadBalancerJoinAddressEnabled is true when clusteradm join should use status.grpcServer from the hub Service.
func grpcHubLoadBalancerJoinAddressEnabled(hub *v1beta1.Hub) bool {
	g := hub.Spec.RegistrationAuth.GRPC
	return g != nil && strings.EqualFold(g.EndpointType, v1beta1.GRPCEndpointTypeLoadBalancer)
}

// observeHubGRPCCABundleStatus sets hub.status.grpcServerCA from open-cluster-management-hub/ca-bundle-configmap.
// Hostname and loadBalancer endpoint types use the same CA on the hub.
func (r *HubReconciler) observeHubGRPCCABundleStatus(ctx context.Context, hub *v1beta1.Hub, k8s kubernetes.Interface) {
	logger := log.FromContext(ctx)
	cm, err := k8s.CoreV1().ConfigMaps(ocmHubNamespace).Get(ctx, grpcCABundleConfigMap, metav1.GetOptions{})
	if err != nil {
		if !kerrs.IsNotFound(err) {
			logger.V(1).Info("could not read gRPC CA configmap from hub", "hub", hub.Name, "configMap", grpcCABundleConfigMap, "error", err)
		}
		return
	}
	if v, ok := cm.Data[grpcCABundleDataKey]; ok && v != "" {
		hub.Status.GRPCServerCA = v
	}
}

// observeHubGRPCServerLoadBalancerStatus sets hub.status.grpcServer from cluster-manager-grpc-server LoadBalancer ingress.
func (r *HubReconciler) observeHubGRPCServerLoadBalancerStatus(ctx context.Context, hub *v1beta1.Hub, k8s kubernetes.Interface) {
	logger := log.FromContext(ctx)
	svc, err := k8s.CoreV1().Services(ocmHubNamespace).Get(ctx, grpcServerServiceName, metav1.GetOptions{})
	if err != nil {
		if !kerrs.IsNotFound(err) {
			logger.V(1).Info("could not read gRPC service from hub", "hub", hub.Name, "service", grpcServerServiceName, "error", err)
		}
		hub.Status.GRPCServer = ""
		return
	}
	host := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		in0 := svc.Status.LoadBalancer.Ingress[0]
		switch {
		case in0.Hostname != "":
			host = in0.Hostname
		case in0.IP != "":
			host = in0.IP
		}
	}
	port := defaultHubGRPCPort
	for _, p := range svc.Spec.Ports {
		if strings.EqualFold(p.Name, "grpc") && p.Port > 0 {
			port = int(p.Port)
			break
		}
	}
	if port == defaultHubGRPCPort {
		for _, p := range svc.Spec.Ports {
			if p.Port > 0 {
				port = int(p.Port)
				break
			}
		}
	}
	if host != "" {
		addr := host
		if !strings.Contains(addr, ":") {
			addr = fmt.Sprintf("%s:%d", addr, port)
		}
		hub.Status.GRPCServer = addr
	} else {
		hub.Status.GRPCServer = ""
	}
}
