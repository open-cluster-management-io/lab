# fleetconfig-controller

Controller for OCM multi-clusters; orchestrates registration & lifecycle management declaratively via `clusteradm`.

All configuration options for `clusteradm` are supported by the `FleetConfig` custom resource.

## Description

The FleetConfig custom resource models the state of an Open Cluster Management (OCM) multi-cluster in a declarative manner. The fleetconfig-controller will initialize the designated hub and spoke clusters, add, remove, and upgrade clustermanagers and klusterlets when their bundle versions change, manage their feature gates, and uninstall all OCM components properly whenever a FleetConfig is deleted.

## Getting Started

### Prerequisites
- go version v1.22.0+
- docker version 17.03+
- kind version v0.23.0
- kubectl version v1.11.3+

### Run a smoke test locally

Refer to [scenarios](docs/smoketests.md).

### Run E2E tests using an existing cluster

For debugging E2E tests, it can be helpful to use an existing cluster. Do so as follows:

```bash
# (assuming you have a preexisting kind cluster named 'fleetconfig-dev')
SKIP_CLEANUP=true KIND_CLUSTER=fleetconfig-dev make test-e2e
```

## Development

```bash
# Create a dev kind cluster
kind create cluster --name ocm-dev --kubeconfig ~/Downloads/ocm-dev.kubeconfig

# Initialize a devspace development container
devspace run-pipeline dev -n fleetconfig-system
```

### Debugging

- Hit up arrow, then enter from within the dev container to start a headless delve session
- Execute the DevSpace launch config to connect VSCode with the delve session running in the dev container
