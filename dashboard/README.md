# OCM Dashboard

![Node.js](https://img.shields.io/badge/node-%3E%3D22.0.0-green)
![Go](https://img.shields.io/badge/go-%3E%3D1.23-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-blue)

---

## 📑 Table of Contents

- [Project Overview](#project-overview)
- [Architecture](#architecture)
  - [Frontend Components](#frontend-components)
  - [Backend Components](#backend-components)
- [Current Features](#current-features)
- [Setup & Development](#setup--development)
  - [Prerequisites](#prerequisites)
  - [Frontend Development](#frontend-development)
  - [Backend Development](#backend-development)
  - [Connecting Frontend to Backend](#connecting-frontend-to-backend)
- [Testing](#testing)
- [Building for Production](#building-for-production)
  - [Build Application](#build-application)
  - [Docker Images](#docker-images)
  - [Available Make Targets](#available-make-targets)
- [Deployment](#deployment)
  - [Helm Chart Deployment](#helm-chart-deployment)
- [Configuration](#configuration)
- [RBAC Requirements (For Backend)](#rbac-requirements-for-backend)
- [Next Steps](#next-steps)
- [License](#license)

---

## Project Overview

A dashboard for displaying and monitoring Open Cluster Management (OCM) clusters, placements, cluster sets, manifest works, and addons.

![OCM Dashboard](./public/images/demo.gif)

---

## Architecture

The OCM Dashboard follows a modern architecture pattern for Kubernetes dashboards:

- **Frontend**: React + TypeScript SPA with Material UI and real-time updates
- **Backend**: Go API service that connects to the Kubernetes API for OCM resources

### Frontend Components

- **Authentication**: Bearer token authentication (JWT) stored in localStorage
- **Overview Page**: High-level KPIs for clusters, cluster sets, and placements
- **Cluster List & Detail**: Table view and detail drawer for clusters, including status, version, claims, and addons
- **Placement List & Detail**: Table view and detail drawer for placements, including status, predicates, and decisions
- **ClusterSet List & Detail**: Table view and detail drawer for ManagedClusterSets, including cluster and binding counts
- **ManifestWorks List**: View manifest works for clusters, including manifest and condition details
- **Addons List**: View managed cluster addons, including status, registrations, and supported configs
- **Login Page**: Token-based login with development mode support
- **Layout**: Responsive layout with navigation drawer and app bar
- **API Service Layer**: Abstraction for backend communication using `fetch`

### Backend Components

- **API Server**: Go service built with Gin, providing endpoints for OCM resources:
  - `GET /api/clusters` - List all ManagedClusters
  - `GET /api/clusters/:name` - Get details for a specific ManagedCluster
  - `GET /api/clustersets` - List all ManagedClusterSets
  - `GET /api/clustersets/:name` - Get details for a specific ManagedClusterSet
  - `GET /api/clustersetbindings` - List all ManagedClusterSetBindings
  - `GET /api/clustersetbindings/:namespace` - List bindings in a namespace
  - `GET /api/clustersetbindings/:namespace/:name` - Get a specific binding
  - `GET /api/placements` - List all Placements
  - `GET /api/placements/:namespace` - List Placements in a namespace
  - `GET /api/placements/:namespace/:name` - Get a specific Placement
  - `GET /api/placements/:namespace/:name/decisions` - Get PlacementDecisions for a Placement
  - `GET /api/manifestworks/:namespace` - List ManifestWorks in a namespace (cluster)
  - `GET /api/manifestworks/:namespace/:name` - Get a specific ManifestWork
  - `GET /api/addons/:name` - List all Addons for a cluster
  - `GET /api/addons/:name/:addonName` - Get a specific Addon for a cluster
  - `GET /api/stream/clusters` - SSE endpoint for real-time ManagedCluster updates
- **Authentication**: Basic authorization header check. TokenReview validation is a TODO. Can be bypassed with `DASHBOARD_BYPASS_AUTH=true`.
- **Kubernetes Client**: Uses `client-go` to interact with the Kubernetes API for OCM resources (ManagedCluster, ManagedClusterSet, Placement, ManifestWork, Addon, etc.)
- **Mock Data Mode**: Supports running with mock data for development via `DASHBOARD_USE_MOCK=true`.

---

## Current Features

**Frontend:**

- Read-only view of clusters, placements, cluster sets, manifest works, and addons
- Table and detail views for all major OCM resources
- Real-time cluster status updates via SSE
- Authentication flow with token support (bearer token in localStorage)
- Responsive UI with Material UI components
- Overview dashboard with KPIs
- Error handling and loading states

**Backend:**

- API endpoints for all major OCM resources (Clusters, ClusterSets, ClusterSetBindings, Placements, ManifestWorks, Addons)
- Placement and PlacementDecision support
- ManifestWork and Addon support
- SSE endpoint for streaming cluster updates
- Kubernetes client integration using `client-go`
- Support for in-cluster and out-of-cluster kubeconfig
- CORS configured for broad access (e.g. `*`)
- Debug mode (`DASHBOARD_DEBUG=true`) and mock data mode (`DASHBOARD_USE_MOCK=true`)

---

## Setup & Development

### Prerequisites

- Node.js 18+ and npm/pnpm
- Go 1.22+ (for backend development)
- Docker with buildx support (for building images)
- Access to a Kubernetes cluster with OCM installed (for backend integration)
- Make (for using the Makefile commands)

### Frontend Development

```bash
# Install dependencies and run development server
npm install
npm run dev

# Or use make:
make dev-ui
```

Open your browser at the URL shown in the terminal (usually http://localhost:5173)

### Backend Development

```bash
# Run API server with mock data (recommended for development)
make dev-apiserver

# Run API server with real Kubernetes connection
make dev-apiserver-real

# Run API server with debugger
make debug-apiserver

# Run UI server (serves built frontend)
make dev-uiserver
```

The `dev-apiserver` target runs the debug script which sets appropriate environment variables for development.
You can modify the `debug.sh` script or set environment variables directly to change behavior (e.g., `KUBECONFIG` path, `PORT`).

### Connecting Frontend to Backend

The frontend (`src/api/utils.ts`) is configured to connect to the backend API, typically running on `http://localhost:8080`. Ensure the backend server is running when developing the frontend.
The `VITE_API_BASE_URL` in `.env.development` (for frontend) should match the backend server address.

---

## Testing

Run tests for all components:

```bash
# Run all tests (frontend, API server, and UI server)
make test

# Run frontend tests only
make test-frontend

# Run API server tests only
make test-apiserver

# Run UI server tests only
make test-uiserver

# Run linting for all components
make lint

# Test UI server functionality
make test-uiserver-functionality
```

---

## Building for Production

### Build Application

```bash
# Build all components (UI, UI server, and API server)
make build

# Build UI only
make build-ui

# Build UI server only
make build-uiserver

# Build API server only
make build-apiserver
```

### Docker Images

The project builds two separate Docker images:

- **API Image**: `dashboard-api` (Go backend)
- **UI Image**: `dashboard-ui` (React frontend with nginx)

#### Configuration Variables

You can customize the build process using these variables:

- `API_IMAGE_NAME`: API image name (default: `dashboard-api`)
- `UI_IMAGE_NAME`: UI image name (default: `dashboard-ui`)
- `IMAGE_TAG`: Docker image tag (default: `latest`)
- `REGISTRY`: Docker registry (default: `quay.io/open-cluster-management`)
- `PLATFORMS`: Target platforms (default: `linux/amd64,linux/arm64`)

#### Using Make (Recommended)

```bash
# Setup Docker buildx for multi-arch builds (first time only)
make setup-buildx

# Build both images for local testing (single architecture)
make docker-build-local

# Build both images for production (multi-architecture)
make docker-build

# Build and push both images
make docker-build-push

# Build specific components
make docker-build-api
make docker-build-ui

# Build with custom configuration
make docker-build-push IMAGE_TAG=v1.0.0
make docker-build-push REGISTRY=myregistry.io/myorg IMAGE_TAG=dev
make docker-build PLATFORMS=linux/amd64
```

#### Individual Image Operations

```bash
# API Image
make docker-build-api          # Build API image
make docker-push-api           # Push API image
make docker-build-push-api     # Build and push API image

# UI Image
make docker-build-ui           # Build UI image
make docker-push-ui            # Push UI image
make docker-build-push-ui      # Build and push UI image
```

#### Using Docker Directly

```bash
# Build API image
docker buildx build -f Dockerfile.api -t dashboard-api:latest --load .

# Build UI image
docker buildx build -f Dockerfile.ui -t dashboard-ui:latest --load .

# Build and push multi-architecture images
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.api \
  -t quay.io/open-cluster-management/dashboard-api:latest \
  --push .

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.ui \
  -t quay.io/open-cluster-management/dashboard-ui:latest \
  --push .
```

### Available Make Targets

The following make targets are available:

**Development Targets:**

- `dev-ui`: Run UI in development mode
- `dev-uiserver`: Run UI server (serves built frontend)
- `dev-apiserver`: Run API server with development settings
- `dev-apiserver-real`: Run API server with real Kubernetes connection
- `debug-apiserver`: Run API server in debug mode

**Build Targets:**

- `build`: Build all components (UI, UI server, and API server)
- `build-ui`: Build UI only
- `build-uiserver`: Build UI server only
- `build-apiserver`: Build API server only

**Docker Targets:**

- `docker-build`: Build both API and UI Docker images
- `docker-build-api`: Build API Docker image
- `docker-build-ui`: Build UI Docker image
- `docker-push`: Push both API and UI Docker images
- `docker-push-api`: Push API Docker image only
- `docker-push-ui`: Push UI Docker image only
- `docker-build-push`: Build and push both images
- `docker-build-push-api`: Build and push API image
- `docker-build-push-ui`: Build and push UI image
- `docker-build-local`: Build both images for local use
- `docker-build-local-api`: Build API image for local use
- `docker-build-local-ui`: Build UI image for local use

**Test Targets:**

- `test`: Run all tests
- `test-frontend`: Run frontend tests
- `test-apiserver`: Run API server tests
- `test-uiserver`: Run UI server tests
- `test-uiserver-functionality`: Test UI server functionality
- `lint`: Run linters for all components

**Other Targets:**

- `setup-buildx`: Setup Docker buildx for multi-arch builds
- `clean`: Clean build artifacts
- `all`: Default target (alias for `build`)

---

## Deployment

### Helm Chart Deployment

Deploy using Helm chart:

```bash
# Add OCM Helm repository
helm repo add ocm https://open-cluster-management.io/helm-charts
helm repo update

# Install OCM Dashboard
helm install ocm-dashboard ocm/ocm-dashboard \
  --namespace ocm-dashboard \
  --create-namespace

# Install with custom values
helm install ocm-dashboard ocm/ocm-dashboard \
  --namespace ocm-dashboard \
  --create-namespace \
  --set image.tag=latest \
  --set dashboard.env.DASHBOARD_BYPASS_AUTH=true


# Install with custom image
helm install ocm-dashboard ./charts/ocm-dashboard \
  --set api.image.registry=quay.io \
  --set api.image.repository=zhaoxue/dashboard-api \
  --set api.image.tag=latest \
  --set api.image.pullPolicy=Always \
  --set ui.image.registry=quay.io \
  --set ui.image.repository=zhaoxue/dashboard-ui \
  --set ui.image.tag=latest \
  --set ui.image.pullPolicy=Always \
  --namespace open-cluster-management-dashboard \
  --create-namespace

# Upgrade existing installation
helm upgrade ocm-dashboard ocm/ocm-dashboard \
  --namespace ocm-dashboard

# Access via port-forward
kubectl port-forward -n open-cluster-management-dashboard service/ocm-dashboard 3000:80
```

For development with local chart:

```bash
# Install from local chart
helm install ocm-dashboard ./charts/ocm-dashboard \
  --namespace ocm-dashboard \
  --create-namespace

# Install with custom values file
helm install ocm-dashboard ./charts/ocm-dashboard \
  --namespace ocm-dashboard \
  --create-namespace \
  --values my-values.yaml
```

---

## Configuration

### Environment Variables

**Backend Configuration:**

- `DASHBOARD_USE_MOCK`: Enable mock data mode (default: `false`)
- `DASHBOARD_DEBUG`: Enable debug logging (default: `false`)
- `DASHBOARD_BYPASS_AUTH`: Bypass authentication (default: `false`)
- `PORT`: Server port (default: `8080`)
- `KUBECONFIG`: Path to kubeconfig file (for out-of-cluster access)

**Frontend Configuration:**

- `VITE_API_BASE_URL`: Backend API URL (default: `http://localhost:8080`)

---

## RBAC Requirements (For Backend)

For the backend to function correctly, it will need RBAC permissions to:

1. List, get, and watch all OCM resources (ManagedCluster, ManagedClusterSet, ManagedClusterSetBinding, Placement, ManifestWork, Addon, etc.)
2. Perform token reviews for authentication

<details>
<summary>Example RBAC configuration</summary>

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ocm-dashboard
  namespace: ocm-dashboard
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ocm-dashboard-reader
rules:
  - apiGroups: ["cluster.open-cluster-management.io"]
    resources:
      [
        "managedclusters",
        "managedclustersets",
        "managedclustersetbindings",
        "placements",
        "placementdecisions",
        "manifestworks",
        "managedclusteraddons",
      ]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["authentication.k8s.io"]
    resources: ["tokenreviews"]
    verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ocm-dashboard-reader-binding
subjects:
  - kind: ServiceAccount
    name: ocm-dashboard
    namespace: ocm-dashboard
roleRef:
  kind: ClusterRole
  name: ocm-dashboard-reader
  apiGroup: rbac.authorization.k8s.io
```

</details>

---

## Next Steps

1. Fully implement real-time updates using SSE with actual Kubernetes informers in the backend.
2. Implement robust TokenReview authentication in the backend.
3. Enhance error handling and user feedback in both frontend and backend.
4. Create a Helm chart for deployment.
5. Add comprehensive unit and integration tests for both frontend and backend.
6. Improve UI/UX, potentially adding more visualizations or actions.
7. Add support for more OCM resource types and actions as needed.
8. Optimize frontend testing and mock implementation.

---

## License

This project is licensed under the Apache License 2.0.
