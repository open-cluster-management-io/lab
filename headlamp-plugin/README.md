# headlamp-plugin

A [Headlamp](https://headlamp.dev/) plugin that provides a dashboard for [Open Cluster Management (OCM)](https://open-cluster-management.io/) resources. View and manage your multicluster fleet directly from the Headlamp UI.

## Features

- Managed Cluster inventory — list with status, version, and capacity
- Cluster detail view — conditions, capacity/allocatable, cluster claims, taints

## Prerequisites

- [Headlamp](https://headlamp.dev/) (desktop app or in-cluster deployment)
- A Kubernetes cluster running as an OCM hub (with ManagedCluster CRDs)

## Quick Start

### Development

```bash
npm install
npm run start
```

Open Headlamp and connect to your OCM hub cluster. The "OCM" sidebar entry will appear automatically.

### Install in Headlamp Desktop

```bash
npm run build
cp -r dist ~/.config/Headlamp/plugins/headlamp-ocm
```

### Install in-cluster

Build and deploy the plugin as an init container alongside Headlamp:

```bash
make images
```

Then add the image as an init container in your Headlamp deployment:

```yaml
initContainers:
  - name: headlamp-ocm
    image: quay.io/open-cluster-management/headlamp-ocm:latest
    volumeMounts:
      - name: plugins
        mountPath: /headlamp-plugins
```

## OCM Resources

| Resource | API Group | Status |
|----------|-----------|--------|
| ManagedCluster | cluster.open-cluster-management.io/v1 | Done |
| ManagedClusterSet | cluster.open-cluster-management.io/v1beta2 | Planned |
| Placement | cluster.open-cluster-management.io/v1beta1 | Planned |
| ManagedClusterAddOn | addon.open-cluster-management.io/v1alpha1 | Planned |
| Policy | policy.open-cluster-management.io/v1 | Planned |
| ManifestWork | work.open-cluster-management.io/v1 | Planned |

## Development

```bash
npm run start       # Dev server with hot reload
npm run tsc         # Type check
npm run lint        # Lint
npm run build       # Production build
npm run test        # Run tests
```

## License

Apache 2.0
