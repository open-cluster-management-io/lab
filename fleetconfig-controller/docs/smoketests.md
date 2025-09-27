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

2. Build & load the `fleetconfig-controller:latest` image

   ```bash
   IMAGE_FLAVOURS="base:./build/Dockerfile.base" make images && \
     kind load docker-image quay.io/open-cluster-management/fleetconfig-controller:latest \
       --name ocm-hub-as-spoke
   ```

3. Install the `fleetconfig-controller`

   ```bash
   devspace deploy -n fleetconfig-system
   ```

4. Verify that the `Hub` and `Spoke` are reconciled successfully

   ```bash
   kubectl wait --for=jsonpath='{.status.phase}'=Running hub/hub \
     -n fleetconfig-system \
     --timeout=10m
   
   kubectl wait --for=jsonpath='{.status.phase}'=Running spoke/hub-as-spoke \
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

2. Generate an internal kubeconfig for the `ocm-spoke` cluster and upload it to the `ocm-hub` cluster

   ```bash
   kind get kubeconfig --name ocm-spoke --internal > $TARGET_DIR/ocm-spoke-internal.kubeconfig
   kubectl create namespace fleetconfig-system
   kubectl -n fleetconfig-system create secret generic test-spoke-kubeconfig \
     --from-file=value=$TARGET_DIR/ocm-spoke-internal.kubeconfig
   ```

3. Build & load the `fleetconfig-controller:local` image

   ```bash
   IMAGE_FLAVOURS="base:./build/Dockerfile.base" IMAGE_REPO=fleetconfig-controller-local IMAGE_TAG=local make images && \
     kind load docker-image quay.io/open-cluster-management/fleetconfig-controller-local:local \
       --name ocm-hub && \
     kind load docker-image quay.io/open-cluster-management/fleetconfig-controller-local:local \
       --name ocm-spoke
   ```

4. Install the `fleetconfig-controller` on the hub using the `deploy-local` pipeline

   ```bash
   devspace run-pipeline deploy-local -n fleetconfig-system --skip-build
   ```

5. Verify that the `Hub` and `Spoke` are reconciled successfully

   ```bash
   kubectl wait --for=jsonpath='{.status.phase}'=Running hub/hub \
     -n fleetconfig-system \
     --timeout=10m
   
   kubectl wait --for=jsonpath='{.status.phase}'=Running spoke/hub-as-spoke \
     -n fleetconfig-system \
     --timeout=10m
   
   kubectl wait --for=jsonpath='{.status.phase}'=Running spoke/spoke \
     -n fleetconfig-system \
     --timeout=10m
   ```
