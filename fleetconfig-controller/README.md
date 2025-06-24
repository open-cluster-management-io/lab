# ✨ Unlock the power of GitOps for OCM! ✨

## 🌱 Project Overview

The `fleetconfig-controller` introduces a new `FleetConfig` custom resource to the OCM ecosystem. It reconciles `FleetConfig` resources to declaratively manage the lifecycle of Open Cluster Management (OCM) multi-clusters. The `fleetconfig-controller` will initialize an OCM hub and one or more spoke clusters; add, remove, and upgrade clustermanagers and klusterlets when their bundle versions change, manage their feature gates, and uninstall all OCM components properly whenever a `FleetConfig` is deleted.

The controller is a lightweight wrapper around [clusteradm](https://github.com/open-cluster-management-io/clusteradm). Anything you can accomplish imperatively via a series of `clusteradm` commands can now be accomplished declaratively using the `fleetconfig-controller`.

## 🔧 Installation

The controller is installed via Helm.

```bash
helm repo add ocm https://open-cluster-management.io/helm-charts
helm repo update ocm
helm install fleetconfig-controller ocm/fleetconfig-controller -n fleetconfig-system --create-namespace
```

By default the Helm chart will also produce a `FleetConfig` to orchestrate, however that behaviour can be disabled. Refer to the chart [README](./charts/fleetconfig-controller/README.md) for full documentation.

## 🏗️ Support Matrix

Support for orchestration of OCM multi-clusters varies based on the Kubernetes distribution and/or cloud provider.

| Kubernetes Distribution | Support Level      |
|-------------------------|--------------------|
| Vanilla Kubernetes      | ✅ Fully Supported |
| Amazon EKS              | ✅ Fully Supported |
| Google GKE              | ✅ Fully Supported |
| Azure AKS               | 🚧 On Roadmap      |

## 🏃🏼‍♂️ Quick Start

### Prerequisites

- `go` version v1.22.0+
- `docker` version 17.03+
- `kind` version v0.23.0+
- `kubectl` version v1.11.3+

### Onboarding

To familiarize yourself with the `FleetConfig` API and the `fleetconfig-controller`, we recommend doing one or more of the following onboarding steps.

1. Step through a [smoke test](./docs/smoketests.md)
1. Invoke the [end-to-end tests](./test/e2e/fleetconfig.go) and inspect the content of the kind clusters that the E2E suite automatically creates

   ```bash
   SKIP_CLEANUP=true make test-e2e
   ```

## 🔣 Development

The `fleetconfig-controller` repository is pre-wired for development using [DevSpace](https://www.devspace.sh/docs/getting-started/introduction).

```bash
# Create a dev kind cluster
kind create cluster \
  --name fleetconfig-dev \
  --kubeconfig ~/Downloads/fleetconfig-dev.kubeconfig

export KUBECONFIG=~/Downloads/fleetconfig-dev.kubeconfig

# Initialize a devspace development container
devspace run-pipeline dev -n fleetconfig-system
```

### Debugging

- Hit up arrow, then enter from within the dev container to start a headless delve session
- Use the following launch config to connect VSCode with the delve session running in the dev container:

  ```json
  {
      "version": "0.2.0",
      "configurations": [
          {
              "name": "DevSpace",
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
          }
      ]
  }
  ```
