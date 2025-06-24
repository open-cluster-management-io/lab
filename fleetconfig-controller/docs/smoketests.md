# Smoke Tests

Before beginning either scenario, export your target directory. You will store kubeconfigs there in subsequent steps.

```bash
export TARGET_DIR=""
mkdir -p $TARGET_DIR
```

## Single kind cluster (hub-as-spoke)

1. Create a kind cluster

   ```bash
   kind create cluster --name ocm-hub-as-spoke --kubeconfig $TARGET_DIR/ocm-hub-as-spoke.kubeconfig
   export KUBECONFIG=$TARGET_DIR/ocm-hub-as-spoke.kubeconfig
   ```

1. Build & load the `fleetconfig-controller:latest` image

   ```bash
   IMAGE_FLAVOURS="fleetconfig-controller:./build/Dockerfile.base" make images && \
     kind load docker-image quay.io/open-cluster-management/fleetconfig-controller:latest \
       --name ocm-hub-as-spoke
   ```

1. Install the `fleetconfig-controller`

   ```bash
   devspace deploy -n fleetconfig-system
   ```

1. Verify that the `FleetConfig` is reconciled successfully

   ```bash
   kubectl wait --for=jsonpath='{.status.phase}'=Running fleetconfig/fleetconfig \
     -n fleetconfig-system \
     --timeout=10m
   ```

## Two kind clusters (hub and spoke)

1. Create two kind clusters

   ```bash
   kind create cluster --name ocm-hub --kubeconfig $TARGET_DIR/ocm-hub.kubeconfig
   kind create cluster --name ocm-spoke --kubeconfig $TARGET_DIR/ocm-spoke.kubeconfig
   export KUBECONFIG=$TARGET_DIR/ocm-hub.kubeconfig
   ```

1. Generate an internal kubeconfig for the `ocm-spoke` cluster and upload it to the `ocm-hub` cluster

   ```bash
   kind get kubeconfig --name ocm-spoke --internal > $TARGET_DIR/ocm-spoke-internal.kubeconfig
   kubectl create secret generic test-fleetconfig-kubeconfig \
     --from-file=value=$TARGET_DIR/ocm-spoke-internal.kubeconfig
   ```

1. Build & load the `fleetconfig-controller:local` image

   ```bash
   IMAGE_FLAVOURS="fleetconfig-controller:./build/Dockerfile.base" IMAGE_TAG=local make images && \
     kind load docker-image quay.io/open-cluster-management/fleetconfig-controller:local \
       --name ocm-hub
   ```

1. Install the `fleetconfig-controller` on the hub using the `deploy-local` pipeline

   ```bash
   devspace run-pipeline deploy-local -n fleet-config-system --skip-build
   ```

1. Verify that the `FleetConfig` is reconciled successfully

   ```bash
   kubectl wait --for=jsonpath='{.status.phase}'=Running fleetconfig/fleetconfig \
     -n fleetconfig-system \
     --timeout=10m
   ```
