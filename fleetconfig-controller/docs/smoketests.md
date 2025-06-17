# Smoke Tests

## Single kind cluster

1. Deploy the kind cluster

```bash
kind create cluster --name ocm-demo --kubeconfig ~/Downloads/ocm-demo.kubeconfig
```

2. Build & load the latest fleetconfig-controller image

```bash
make docker-build && kind load docker-image quay.io/open-cluster-management/fleetconfig-controller:latest --name ocm-demo
```

3. Install the fleetconfig-controller

```bash
devspace run-pipeline deploy -n fleetconfig-system -b --kube-context kind-ocm-demo --silent
```

4. Apply the FleetConfig

```bash
kubectl apply -f - <<EOF
apiVersion: fleetconfig.open-cluster-management.io/v1alpha1
kind: FleetConfig
metadata:
  name: fleetconfig-kind-single
spec:
  hub:
    clusterManager:
      featureGates: "DefaultClusterSet=true,ManifestWorkReplicaSet=true,ResourceCleanup=true"
      source:
        bundleVersion: v0.16.0
        registry: quay.io/open-cluster-management
  spokes:
  - name: hub-as-spoke
    klusterlet:
      featureGates: "ClusterClaim=true,RawFeedbackJsonString=true"
      forceInternalEndpointLookup: true
      source:
        bundleVersion: v0.16.0
        registry: quay.io/open-cluster-management
    kubeconfig:
      inCluster: true
EOF
```
5. Verify that the FleetConfig is reconciled successfully
```bash
kubectl wait --for=jsonpath='{.status.phase}'=Running fleetconfigs.open-cluster-management.io/fleetconfig-kind-single
```

## Two kind clusters

1. Deploy the kind clusters

```bash
kind create cluster --name ocm-hub --kubeconfig ~/Downloads/ocm-hub.kubeconfig
kind create cluster --name ocm-spoke --kubeconfig ~/Downloads/ocm-spoke.kubeconfig
export KUBECONFIG=~/Downloads/ocm-hub.kubeconfig
```

2. Obtain an internal kubeconfig for the ocm-spoke cluster

```bash
kind get kubeconfig --name ocm-spoke --internal > ~/Downloads/ocm-spoke-internal.kubeconfig
kubectl create secret generic spoke-kubeconfig --from-file=kubeconfig=/Users/tylergillson/Downloads/spoke-internal.kubeconfig
```

3. Build & load the latest fleetconfig-controller image

```bash
make docker-build && kind load docker-image quay.io/open-cluster-management/fleetconfig-controller:latest --name ocm-hub
```

4. Install the fleetconfig-controller on the hub

```bash
devspace run-pipeline deploy -n fleet-config-system -b --kube-context kind-ocm-hub --silent
```

5. Apply the FleetConfig

```bash
kubectl apply -f - <<EOF
apiVersion: fleetconfig.open-cluster-management.io/v1alpha1
kind: FleetConfig
metadata:
  name: fleetconfig-kind-multiple
spec:
  hub:
    clusterManager:
      featureGates: "DefaultClusterSet=true,ManifestWorkReplicaSet=true,ResourceCleanup=true"
      source:
        bundleVersion: v0.16.0
        registry: quay.io/open-cluster-management
  spokes:
  - name: ocm-spoke
    klusterlet:
      featureGates: "ClusterClaim=true,RawFeedbackJsonString=true"
      forceInternalEndpointLookup: true
      source:
        bundleVersion: v0.16.0
        registry: quay.io/open-cluster-management
    kubeconfig:
      secretReference:
        name: spoke-kubeconfig
        namespace: default
EOF
```

5. Verify that the FleetConfig is reconciled successfully

```bash
kubectl wait --for=jsonpath='{.status.phase}'=Running fleetconfigs.open-cluster-management.io/fleetconfig-kind-multiple --timeout=5m
```
