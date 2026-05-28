# Dashboard Architecture

This document describes the architecture of the OCM Dashboard, a full-stack application for visualizing and monitoring Open Cluster Management resources.

## System Overview

```
┌─────────────────────────────────────────────────────────┐
│  Browser                                                │
│  ┌───────────────────────────────────────────────────┐  │
│  │  React 19 + MUI 7 + Vite SPA                     │  │
│  │  ┌─────────┐ ┌──────────┐ ┌────────────────────┐ │  │
│  │  │ API     │ │ Pages &  │ │ ReactFlow Charts   │ │  │
│  │  │Services │ │ Drawers  │ │ (MWRS/MW graphs)   │ │  │
│  │  └────┬────┘ └──────────┘ └────────────────────┘ │  │
│  └───────┼───────────────────────────────────────────┘  │
└──────────┼──────────────────────────────────────────────┘
           │ HTTP (Bearer token)
┌──────────┼──────────────────────────────────────────────┐
│  Kubernetes Cluster (Hub)                               │
│  ┌───────┴───────┐    ┌────────────────────────────┐   │
│  │  UI Server    │    │  API Server (Go + Gin)     │   │
│  │  (Gin, :3000) ├───►│  (:8080)                   │   │
│  │  static files │    │  ┌──────────────────────┐  │   │
│  │  + reverse    │    │  │ OCM Typed Clients    │  │   │
│  │    proxy      │    │  │ (cluster, work,      │  │   │
│  └───────────────┘    │  │  addon clientsets)   │  │   │
│                       │  └──────────┬───────────┘  │   │
│                       └─────────────┼──────────────┘   │
│                                     │                   │
│                       ┌─────────────┴───────────────┐  │
│                       │  Kubernetes API Server       │  │
│                       │  (OCM CRDs)                  │  │
│                       └──────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Backend (`apiserver/`)

### Package Structure

```
apiserver/
├── main.go                        # Entry point
├── pkg/
│   ├── client/
│   │   ├── ocm.go                 # OCMClient struct — wraps all typed clients
│   │   ├── kubernetes.go          # Kubeconfig loading (in-cluster or file)
│   │   ├── resources.go           # GVR constants for dynamic client usage
│   │   └── mocks.go               # Mock client for testing
│   ├── handlers/
│   │   ├── clusters.go            # ManagedCluster CRUD
│   │   ├── clustersets.go         # ManagedClusterSet CRUD
│   │   ├── clustersetbindings.go  # ManagedClusterSetBinding CRUD
│   │   ├── placements.go          # Placement CRUD
│   │   ├── placementdecisions.go  # PlacementDecision queries
│   │   ├── manifestwork.go        # ManifestWork CRUD (v1)
│   │   ├── manifestworkreplicasets.go  # ManifestWorkReplicaSet CRUD (v1alpha1)
│   │   ├── resources.go           # Managed resources (extracted from MW specs)
│   │   ├── addons.go              # ManagedClusterAddon CRUD
│   │   └── streaming.go           # SSE for cluster updates
│   ├── models/
│   │   ├── cluster.go             # Cluster model
│   │   ├── clusterset.go          # ClusterSet model
│   │   ├── clustersetbinding.go   # ClusterSetBinding model
│   │   ├── placement.go           # Placement model
│   │   ├── placementdecision.go   # PlacementDecision model
│   │   ├── manifestwork.go        # ManifestWork model
│   │   ├── manifestworkreplicaset.go  # MWRS, LocalPlacementReference, Summary models
│   │   ├── resource.go            # ManagedResource, ManagedResourceList models
│   │   ├── addon.go               # Addon model
│   │   └── common.go              # Shared types (Condition)
│   └── server/
│       └── server.go              # Gin router, middleware, route registration
```

### OCM Client

`OCMClient` in `pkg/client/ocm.go` wraps:

- **`ClusterClient`** (`cluster.open-cluster-management.io`) — ManagedClusters, ManagedClusterSets, Placements, PlacementDecisions
- **`WorkClient`** (`work.open-cluster-management.io`) — ManifestWorks (v1), ManifestWorkReplicaSets (v1alpha1)
- **`AddonClient`** (`addon.open-cluster-management.io`) — ManagedClusterAddons
- **`dynamic.Interface`** — backward-compatible dynamic client
- **`KubernetesClient`** — core client for TokenReview auth

All clients are created from a single `rest.Config` in `CreateOCMClient()`.

### API Endpoints

All endpoints are under `/api/` and require authentication (Bearer token validated via Kubernetes TokenReview, or bypassed with `DASHBOARD_BYPASS_AUTH=true`).

#### Clusters
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/clusters` | `GetClusters` | List all ManagedClusters |
| GET | `/api/clusters/:name` | `GetCluster` | Get a specific ManagedCluster |
| GET | `/api/clusters/:name/addons` | `GetClusterAddons` | List addons for a cluster |
| GET | `/api/clusters/:name/addons/:addonName` | `GetClusterAddon` | Get a specific addon |

#### ClusterSets & Bindings
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/clustersets` | `GetClusterSets` | List all ManagedClusterSets |
| GET | `/api/clustersets/:name` | `GetClusterSet` | Get a specific ManagedClusterSet |
| GET | `/api/clustersetbindings` | `GetAllClusterSetBindings` | List all bindings cross-namespace |
| GET | `/api/namespaces/:namespace/clustersetbindings` | `GetClusterSetBindings` | List bindings in namespace |
| GET | `/api/namespaces/:namespace/clustersetbindings/:name` | `GetClusterSetBinding` | Get a specific binding |

#### Placements & Decisions
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/placements` | `GetPlacements` | List all Placements |
| GET | `/api/namespaces/:namespace/placements` | `GetPlacementsByNamespace` | List Placements in namespace |
| GET | `/api/namespaces/:namespace/placements/:name` | `GetPlacement` | Get a specific Placement |
| GET | `/api/namespaces/:namespace/placements/:name/decisions` | `GetPlacementDecisions` | Get decisions for a Placement |
| GET | `/api/placementdecisions` | `GetAllPlacementDecisions` | List all PlacementDecisions |
| GET | `/api/namespaces/:namespace/placementdecisions` | `GetPlacementDecisionsByNamespace` | List decisions in namespace |

#### ManifestWorks
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/manifestworks` | `GetAllManifestWorks` | List all ManifestWorks cross-namespace |
| GET | `/api/namespaces/:namespace/manifestworks` | `GetManifestWorks` | List ManifestWorks in namespace (cluster) |
| GET | `/api/namespaces/:namespace/manifestworks/:name` | `GetManifestWork` | Get a specific ManifestWork |

#### ManifestWorkReplicaSets
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/manifestworkreplicasets` | `GetAllManifestWorkReplicaSets` | List all MWRS cross-namespace |
| GET | `/api/namespaces/:namespace/manifestworkreplicasets` | `GetManifestWorkReplicaSets` | List MWRS in namespace |
| GET | `/api/namespaces/:namespace/manifestworkreplicasets/:name` | `GetManifestWorkReplicaSet` | Get a specific MWRS |
| GET | `/api/namespaces/:namespace/manifestworkreplicasets/:name/manifestworks` | `GetManifestWorksByReplicaSet` | List child ManifestWorks (by label selector) |

The "manifestworks by replica set" endpoint queries ManifestWorks using the OCM label `work.open-cluster-management.io/manifestworkreplicaset=<namespace>.<name>`.

#### Resources
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/resources` | `GetManagedResources` | List all resources (supports `kind`, `cluster`, `namespace` query filters) |
| GET | `/api/resources/:cluster/:manifestwork/:ordinal` | `GetManagedResource` | Get a specific resource (includes raw spec) |

Resources are not standalone Kubernetes objects — they are extracted from ManifestWork specs and enriched with status and StatusFeedback values from `resourceStatus.manifests[]`.

#### Streaming
| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/stream/clusters` | `StreamClusters` | SSE stream of ManagedCluster updates |

## Frontend (`src/`)

### Structure

```
src/
├── api/                           # API service layer
│   ├── utils.ts                   # createHeaders(), API_BASE config
│   ├── clusterService.ts          # fetchClusters, fetchCluster
│   ├── clusterSetService.ts       # fetchClusterSets
│   ├── clusterSetBindingService.ts
│   ├── placementService.ts        # fetchPlacements
│   ├── manifestWorkService.ts     # fetchAllManifestWorks, fetchManifestWorks, fetchManifestWorkByName
│   ├── manifestWorkReplicaSetService.ts  # fetchManifestWorkReplicaSets, fetchManifestWorksByReplicaSet
│   ├── resourceService.ts         # fetchManagedResources, fetchManagedResource
│   └── addonService.ts
├── auth/
│   └── AuthContext.tsx             # React context for auth state
├── components/
│   ├── layout/
│   │   ├── AppShell.tsx            # Top-level layout with Outlet
│   │   ├── AppBar.tsx              # Top navigation bar
│   │   ├── Drawer.tsx              # Side navigation (Overview, Clusters, Clustersets, Placements, WorkReplicaSets, ManifestWorks, Resources)
│   │   ├── DrawerLayout.tsx        # Right-side detail drawer wrapper
│   │   └── PageLayout.tsx          # Full-page layout with back button
│   ├── OverviewPage.tsx            # KPI cards: Clusters, ClusterSets, Placements, WorkReplicaSets
│   ├── ClusterListPage.tsx         # Cluster table + detail drawer
│   ├── ClusterDetailPage.tsx       # Full cluster detail
│   ├── ClusterDetailContent.tsx    # Cluster detail content (compact/full)
│   ├── ClusterAddonsList.tsx       # Addon list for a cluster
│   ├── ClusterManifestWorksList.tsx # MW list grouped by cluster (used in MWRS detail)
│   ├── ClustersetList.tsx          # ClusterSet table + detail drawer
│   ├── PlacementListPage.tsx       # Placement table + detail drawer
│   ├── ManifestWorkReplicaSetListPage.tsx   # MWRS table + detail drawer
│   ├── ManifestWorkReplicaSetDetailPage.tsx # MWRS full detail (loads data, wraps content)
│   ├── ManifestWorkReplicaSetDetailContent.tsx # MWRS tabbed content (Overview, ManifestWorks, Graph)
│   ├── MWRSFlowChart.tsx           # ReactFlow: MWRS -> ManifestWorks -> Resources
│   ├── ManifestWorkListPage.tsx    # MW cross-namespace table + detail drawer
│   ├── ManifestWorkDetailPage.tsx  # MW full detail (loads data, wraps content)
│   ├── ManifestWorkDetailContent.tsx # MW tabbed content (Overview, Graph)
│   ├── MWFlowChart.tsx             # ReactFlow: ManifestWork -> Resources
│   ├── ResourceListPage.tsx        # Resource table with Kind/Cluster/Namespace filters
│   ├── ResourceDetailPage.tsx      # Resource full detail
│   ├── ResourceDetailContent.tsx   # Resource detail content (Overview + Spec tabs)
│   ├── StatusFeedbackDisplay.tsx   # Shared component for OCM StatusFeedback values (table/inline/compact variants)
│   └── Login.tsx                   # Login page
├── hooks/
│   └── useAutoLayout.ts            # Dagre-based auto-layout hook for ReactFlow graphs
├── utils/
│   └── statusHelpers.ts            # Shared status derivation (degraded detection, border/chip colors)
├── theme/
│   └── ThemeProvider.tsx           # MUI theme configuration
└── App.tsx                         # Router and route definitions
```

### Page Pattern

Most resource types follow the same pattern:

1. **ListPage** — table with search/filter, clicking a row opens a detail drawer (compact mode). A "View Full Details" chip navigates to the detail page.
2. **DetailPage** — `PageLayout` wrapper with back button, fetches resource data, renders `DetailContent`.
3. **DetailContent** — accepts `compact` prop (for drawer, overview only) or full mode (with tabs).

### API Service Pattern

Each API service in `src/api/`:
- Exports TypeScript interfaces for the resource model
- Has mock data that runs when `import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API`
- Uses `createHeaders()` from `utils.ts` to attach the Bearer token
- Catches errors and returns empty arrays/null rather than throwing

### Flow Chart Visualizations

Two ReactFlow-based components provide graph visualizations:

- **`MWRSFlowChart`** — Three-column layout: MWRS node -> ManifestWork nodes (per cluster) -> Resource nodes. Fetches child ManifestWorks via the MWRS label selector endpoint.
- **`MWFlowChart`** — Two-column layout: ManifestWork node -> Resource nodes. Uses data already loaded by the parent page.

Both use custom node types with status-colored borders and chips. Resource nodes are clickable and navigate to `/resources/:cluster/:manifestwork/:ordinal`.

Layout is handled by `useAutoLayout` (`src/hooks/useAutoLayout.ts`), a shared hook that uses `@dagrejs/dagre` for directed-graph auto-layout. Nodes are created at position (0,0), ReactFlow measures their rendered dimensions, then dagre computes optimal positions and `fitView()` is called. This ensures nodes size correctly regardless of content (e.g., StatusFeedback chips of varying length).

### StatusFeedback Display

OCM's StatusFeedback mechanism allows spoke cluster work agents to sync specific status field values back to the hub via `ManifestCondition.StatusFeedbacks`. This is configured per-resource using `manifestConfigs[].feedbackRules` in the ManifestWorkReplicaSet or ManifestWork spec (e.g., JSONPaths for `.status.readyReplicas` or WellKnownStatus).

The dashboard extracts these feedback values in the backend (`convertManifestWork()` and `convertStatusFeedback()` in `handlers/manifestwork.go`) and exposes them via the API on `ManifestCondition.statusFeedback` and `ManagedResource.statusFeedback`.

**`StatusFeedbackDisplay`** (`src/components/StatusFeedbackDisplay.tsx`) is a shared component with three variants:

| Variant | Rendering | Used in |
|---------|-----------|---------|
| `table` | Full table with Name, Type, Value columns | Resource detail, MWRS Overview tab accordions, ClusterManifestWorksList accordions |
| `inline` | Horizontal `Chip` components (`name: value`) | ManifestWork resources table, ClusterManifestWorksList manifests table |
| `compact` | Single-line text (`name=value, name=value`) | Flow chart resource nodes (MWFlowChart, MWRSFlowChart) |

StatusFeedback is displayed in:
- **MWRS Overview tab** — collapsible accordions per cluster, showing feedback for all resources
- **MWRS ManifestWorks tab** — inline chips in the manifests table + collapsible detail sections per ManifestWork
- **ManifestWork detail** — "Status Feedback" column in the resources table
- **Resource detail** — full table in the Status Feedback section
- **Flow chart nodes** — compact summary below the status chip on resource nodes

### Degraded Status Detection

OCM provides two mechanisms for workload health on spoke clusters:

1. **ConditionRules** — CEL expressions or WellKnownConditions evaluated against resources, producing per-manifest `Progressing`, `Degraded`, and `Complete` conditions. These aggregate into work-level `WorkProgressing` and `WorkDegraded` conditions that the MWRS rollout controller uses for progressive rollout gating (e.g., `Progressing=False` → Succeeded, `Progressing=True + Degraded=True` → Failed).
2. **FeedbackRules** — sync specific status field values (e.g., `readyReplicas`, `clusterIP`) back to the hub for observability. These are displayed throughout the dashboard and also used to detect degraded workloads when ConditionRules are not configured.

The dashboard's `Applied`/`Available` conditions reflect manifest application and resource existence. The dashboard additionally inspects StatusFeedback values to detect degraded workloads — for example, a Deployment that is Applied but has `ReadyReplicas < Replicas` or `Available=False`.

Shared status logic lives in `src/utils/statusHelpers.ts`:

- **`deriveMWStatus(mw)`** — derives ManifestWork status, checking child resource feedback for degraded health
- **`deriveResStatus(conditions, feedback?)`** — per-resource status, returns "Degraded" when Applied but feedback indicates unhealthy
- **`deriveMWRSStatus(mwrs, childManifestWorks?)`** — rolls up degraded state from child ManifestWorks
- **`isFeedbackDegraded(feedback)`** — detects degraded workloads via both JSONPaths (Available=False) and WellKnownStatus patterns for all four OCM resource types:

| Resource | WellKnownStatus Fields | Degraded When |
|----------|----------------------|---------------|
| Deployment | `ReadyReplicas`, `Replicas`, `AvailableReplicas` | `ReadyReplicas < Replicas` |
| DaemonSet | `NumberReady`, `DesiredNumberScheduled`, `NumberAvailable` | `NumberReady < DesiredNumberScheduled` |
| Job | `JobComplete`, `JobSucceeded` | `JobComplete = "False"` |
| Pod | `PodReady`, `PodPhase` | `PodReady = "False"` or `PodPhase = "Failed"` |

- **`getDegradedReason(feedback)`** / **`getMWDegradedReasons(mw)`** — extract human-readable diagnostic messages (deployment condition messages, replica counts, pod phase, etc.)

Degraded status is surfaced in:
- **Overview page** — KPI cards show degraded MWRS count alongside failed count
- **ManifestWork list** — ManifestWorks with Applied conditions but degraded feedback show as "Degraded"
- **WorkReplicaSets list** — fetches child ManifestWorks to derive feedback-aware status per MWRS
- **MWRS detail** — warning Alert with per-cluster, per-resource degraded reasons
- **ManifestWork detail** — warning Alert listing degraded resources with reasons
- **Flow chart nodes** — status-colored borders reflect degraded state

## Deployment

### Container Images

Two images, built from the `dashboard/` directory:

| Image | Dockerfile | Port | Purpose |
|-------|-----------|------|---------|
| `dashboard-api` | `Dockerfile.api` | 8080 | Go API server |
| `dashboard-ui` | `Dockerfile.ui` | 3000 | Gin static server + reverse proxy to API |

### Helm Chart

Located at `charts/ocm-dashboard/`. Deploys a single Pod with two containers (api + ui) behind a Service exposing ports 80 (UI) and 8080 (API).

Key values:
- `api.image.repository` / `ui.image.repository` — container image refs
- `dashboard.env.DASHBOARD_BYPASS_AUTH` — skip token validation
- RBAC templates grant read access to `cluster.open-cluster-management.io`, `work.open-cluster-management.io`, and `addon.open-cluster-management.io` API groups

### Example Manifests

`examples/mwrs/setup.yaml` provides sample ManifestWorkReplicaSet resources for testing:
- ManagedClusterSetBindings for `default` and `monitoring` namespaces
- Placements selecting all clusters
- Three MWRS: nginx deployment+service (with JSONPaths FeedbackRules and CEL ConditionRules for rollout gating), monitoring agent+config (with WellKnownStatus feedback and CEL ConditionRules), app configmaps
- Both Deployment MWRS include `conditionRules` for `Progressing` (Available != True) and `Degraded` (Progressing = False, i.e., ProgressDeadlineExceeded), which gate MWRS progressive rollout on actual workload health

Apply with: `kubectl apply -f examples/mwrs/setup.yaml`

## Key Dependencies

| Dependency | Version | Purpose |
|-----------|---------|---------|
| `open-cluster-management.io/api` | v1.2.0 | OCM typed clients and API types |
| `k8s.io/client-go` | v0.34.1 | Kubernetes client |
| `github.com/gin-gonic/gin` | v1.9.1 | HTTP framework |
| `@xyflow/react` | ^12.10.2 | Flow chart visualization |
| `@dagrejs/dagre` | ^3.0.0 | Directed graph auto-layout for flow charts |
| `@mui/material` | ^7.0.1 | UI component library |
| `react` | ^19.1.0 | UI framework |
| `vite` | ^6.3.5 | Build tool and dev server |
