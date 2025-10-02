# ✨ Unlock the power of GitOps for OCM! ✨

## 🌱 Project Overview

The `fleetconfig-controller` introduces 2 new custom resources to the OCM ecosystem: `Hub` and `Spoke` . It reconciles `Hub` and `Spoke` resources to declaratively manage the lifecycle of Open Cluster Management (OCM) multi-clusters. The `fleetconfig-controller` will initialize an OCM hub and one or more spoke clusters; add, remove, and upgrade clustermanagers and klusterlets when their bundle versions change, manage their feature gates, and uninstall all OCM components properly whenever a `Hub` or `Spoke`s are deleted.

The controller is a lightweight wrapper around [clusteradm](https://github.com/open-cluster-management-io/clusteradm). Anything you can accomplish imperatively via a series of `clusteradm` commands can now be accomplished declaratively using the `fleetconfig-controller`.

`fleetconfig-controller` supports 2 modes of operation:
- `addonMode: true` (recommended): After the initial join, a `fleetconfig-controller-agent` will be installed on the spoke cluster as an OCM addon. Once installed, the agent will manage all day 2 operations for the spoke cluster asynchronously. For more information about addon mode, see [2-phase-spoke-reconcile.md](./docs/2-phase-spoke-reconcile.md).
- `addonMode: false`: All management of all spokes is done from the hub cluster. No agent is installed on the spoke cluster. Currently, this is the only mode supported for EKS.

For the deprecated `v1alpha1` `FleetConfig` API, addon mode is not supported.

## 🔧 Installation

The controller is installed via Helm.

```bash
helm repo add ocm https://open-cluster-management.io/helm-charts
helm repo update ocm
helm install fleetconfig-controller ocm/fleetconfig-controller -n fleetconfig-system --create-namespace
```

By default the Helm chart will also produce a `Hub` and 1 `Spoke` (`hub-as-spoke`) to orchestrate, however that behaviour can be disabled. Refer to the chart [README](./charts/fleetconfig-controller/README.md) for full documentation.

## 🏗️ Support Matrix

Support for orchestration of OCM multi-clusters varies based on the Kubernetes distribution and/or cloud provider.

| Kubernetes Distribution | Support Level                         |
|-------------------------|---------------------------------------|
| Vanilla Kubernetes      | ✅ Fully Supported                    |
| Amazon EKS              | ✅ Fully Supported (addonMode: false) |
| Google GKE              | ✅ Fully Supported                    |
| Azure AKS               | 🚧 On Roadmap                         |

## 🏃🏼‍♂️ Quick Start

### Prerequisites

- `go` version v1.22.0+
- `docker` version 17.03+
- `kind` version v0.23.0+
- `kubectl` version v1.11.3+

### Onboarding

To familiarize yourself with the `Hub` and `Spoke` APIs and the `fleetconfig-controller`, we recommend doing one or more of the following onboarding steps.

1. Step through a [smoke test](./docs/smoketests.md)
1. Invoke the [end-to-end tests](./test/e2e/v1beta1_hub_spoke.go) and inspect the content of the kind clusters that the E2E suite automatically creates

   ```bash
   SKIP_CLEANUP=true make test-e2e
   ```

## 🔣 Development

The `fleetconfig-controller` repository is pre-wired for development using [DevSpace](https://www.devspace.sh/docs/getting-started/introduction).

### Single cluster (Hub and `hub-as-spoke` Spoke development)
```bash
# Create a dev kind cluster
kind create cluster \
  --name fleetconfig-dev \
  --kubeconfig ~/Downloads/fleetconfig-dev.kubeconfig

export KUBECONFIG=~/Downloads/fleetconfig-dev.kubeconfig

# Initialize a devspace development container
devspace run-pipeline dev -n fleetconfig-system
```
See [Debugging](#debugging) for instructions on how to start the fleetconfig controller manager in debug mode.

### Two clusters (Hub and Spoke development)
```bash
# Create two dev kind clusters
kind create cluster \
  --name fleetconfig-dev-hub \
  --kubeconfig ~/Downloads/fleetconfig-dev-hub.kubeconfig
export KUBECONFIG=~/Downloads/fleetconfig-dev-hub.kubeconfig

kind create cluster \
  --name fleetconfig-dev-spoke \
  --kubeconfig ~/Downloads/fleetconfig-dev-spoke.kubeconfig

# Get the spoke kind cluster's internal kubeconfig
kind get kubeconfig --name fleetconfig-dev-spoke --internal > ~/Downloads/fleetconfig-dev-spoke-internal.kubeconfig

# Initialize a devspace development container. This will bootstrap in hub-as-spoke mode.
devspace run-pipeline dev --namespace fleetconfig-system --force-build
```
See [Debugging](#debugging) for instructions on how to start the fleetconfig controller manager in debug mode.

In a new terminal session, execute the following commands to create a Spoke resource and start the fleetconfig controller agent on the spoke cluster.

```bash
# Create a secret containing the spoke cluster kubeconfig
export KUBECONFIG=~/Downloads/fleetconfig-dev-hub.kubeconfig
kubectl --namespace fleetconfig-system create secret generic spoke-kubeconfig \
  --from-file=value=<absolute/path/to/fleetconfig-dev-spoke-internal.kubeconfig>

# Create a minimal Spoke resource
kubectl apply -f hack/dev/spoke.yaml

# Once fleetconfig-controller-agent is created on the spoke cluster, start the debug session
export KUBECONFIG=~/Downloads/fleetconfig-dev-spoke.kubeconfig
devspace run-pipeline debug-spoke --namespace fleetconfig-system --force-build --profile v1alpha1
```
The `--profile v1alpha1` flag disables installing the default Hub and Spoke resources.

See [Debugging](#debugging) for instructions on how to start the fleetconfig controller agent in debug mode.

### Debugging

- Hit up arrow, then enter from within the dev container to start a headless delve session
- Use one of the following launch configs to connect VSCode with the delve session running in the dev container:

  ```json
  {
      "version": "0.2.0",
      "configurations": [
          {
              "name": "DevSpace - Hub",
              "type": "go",
              "request": "attach",
              "mode": "remote",
              "port": 2344,
              "host": "127.0.0.1",
              "substitutePath": [
                  {
                      "from": "${workspaceFolder}/fleetconfig-controller",
                      "to": "/workspace",
                  }
              ],
              "showLog": true,
              // "trace": "verbose", // useful for debugging delve (breakpoints not working, etc.)
          },
          {
              "name": "DevSpace - Spoke",
              "type": "go",
              "request": "attach",
              "mode": "remote",
              "port": 2345,
              "host": "127.0.0.1",
              "substitutePath": [
                  {
                      "from": "${workspaceFolder}/fleetconfig-controller",
                      "to": "/workspace",
                  }
              ],
              "showLog": true,
              // "trace": "verbose", // useful for debugging delve (breakpoints not working, etc.)
          }
      ]
  }
  ```
