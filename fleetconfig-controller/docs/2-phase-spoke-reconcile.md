# Spoke Reconciler Walkthrough

For the purposes of this document, `Spoke Reconciler (Spoke)` refers to the `SpokeReconciler` running on the spoke cluster, and `Spoke Reconciler (Hub)` refers to the `SpokeReconciler` running on the hub cluster. This is different from the `HubReconciler`, which is a hub-only controller for reconciling the Hub resource.

The Spoke reconciler runs in two different modes depending on where it's deployed:

- **Hub mode**: Runs on a hub cluster, handles joining spokes and cleaning up hub resources
- **Spoke mode**: Runs on spoke clusters, handles klusterlet upgrades and local cleanup. Automatically installed as an [OCM addon](https://open-cluster-management.io/docs/concepts/add-on-extensibility/addon/) when a spoke is joined to a hub.

Note: *hub-as-spoke* clusters are a special case where the hub is registered as a spoke. A hub-as-spoke cluster is denoted by either the name `hub-as-spoke` or an InCluster kubeconfig.

## Reconciler Steps

### 1. Finalizer Setup

When a Spoke resource is created, the reconcilers add finalizers to control cleanup:

**Spoke Reconciler (Hub)** adds:
- `HubCleanupPreflightFinalizer` - removed when hub is ready for spoke to unjoin
- `HubCleanupFinalizer` - removed after hub finishes cleanup
- `SpokeCleanupFinalizer` - only for *hub-as-spoke* clusters

**Spoke Reconciler (Spoke)** adds:
- `SpokeCleanupFinalizer` - removed after spoke finishes local cleanup

### 2. Day 1 Operations

**Spoke Reconciler (Hub)**:
1. Check if spoke is already joined as a ManagedCluster
2. If not joined: run `clusteradm join` on the spoke, then `clusteradm accept` on the hub
3. Wait for ManagedClusterJoined condition
4. Set up addon deployment configs and enable addons
5. For hub-as-spoke: also do [day 2 operations](#3-day-2-operations)

### 3. Day 2 Operations

**Spoke Reconciler (Spoke)**:
1. Set PivotComplete condition (spoke agent is now managing itself)
2. Check if klusterlet needs upgrading by comparing config hash and bundle version
3. If upgrade needed: run `clusteradm upgrade klusterlet`

### 4. Cleanup Process

The cleanup process coordinates between hub and Spoke Reconciler (Spoke)s using finalizers.

**Spoke Reconciler (Hub)**:
1. Check for active ManifestWorks (can't cleanup if cluster is still in use)
2. Disable addons (but keep fleetconfig-controller-agent running so spoke can unjoin)
3. Remove `HubCleanupPreflightFinalizer` (signals spoke to start unjoin)

**Spoke Reconciler (Spoke)**:
1. Wait for hub to remove `HubCleanupPreflightFinalizer`
2. Run `clusteradm unjoin` to deregister from hub
3. Remove klusterlet and OCM namespaces
4. Remove `SpokeCleanupFinalizer` (signals hub that unjoin is done)
5. Clean up remaining AppliedManifestWorks (at this point, there is only 1 - the fleetconfig-controller-agent)

**Spoke Reconciler (Hub)**:
1. Wait for spoke to remove `SpokeCleanupFinalizer`
2. Clean up remaining AppliedManifestWorks (at this point, there is only 1 - the fleetconfig-controller-agent)
3. Remove `HubCleanupFinalizer`

In total, the cleanup process is completed in 3 reconciles - 2 on the hub and 1 on the spoke.

**Special Cases**:

There are 2 special cases to consider:

- **hub-as-spoke**: The Spoke Reconciler (Hub) will also run the spoke-side cleanup steps right after the hub pre-flight cleanup steps. Cleanup completes in 2 reconciles.
- **Failed Pivot**: The spoke agent never came up, so the hub will attempt the spoke-side cleanup steps right after the hub pre-flight cleanup steps. Cleanup completes in 2 reconciles.
This allows for an "escape hatch" if the spoke agent never came up. This is the only case where a hub will perform day 2 operations on a spoke. It will never attempt upgrades on a spoke.

## Key Points

- All configuration is done on the hub cluster by CRUD operation on Spoke resources.
- After the initial join, OCM addon framework is leveraged to install a fleetconfig-controller-agent inside the spoke cluster. The agent is responsible for the spoke's day 2 operations, including klusterlet upgrades and local cleanup.
- After the "pivot", the spoke kubeconfig secret can be safely deleted. The hub will no longer directly manage the spoke cluster. Instead, the agent will asynchronously pull updates from the hub and reconcile them locally.
- Leveraging finalizers to coordinate cleanup tasks allows the controllers to operate independently and avoid direct communication. Otherwise, API calls between the manager and agent would be required to coordinate cleanup.

## Sequence Diagrams

### Spoke Reconciles

#### Actors:
- User: End user
- HubK8s: Hub cluster Kubernetes API server
- Spoke Reconciler (Hub): Hub-side instance of the fleetconfig-controller-manager SpokeReconciler
- SpokeK8s: Spoke cluster Kubernetes API server
- Spoke Reconciler (Spoke): Spoke-side instance of the fleetconfig-controller-agent SpokeReconciler
- Klusterlet CR: Klusterlet resource in the spoke cluster
- RegAgent: OCM Klusterlet Registration Agent
- WorkAgent: OCM Klusterlet Work Agent

#### Day 1 - Join

```mermaid
sequenceDiagram
    participant User
    participant HubK8s as Hub Cluster API Server
    participant HubController as Spoke Reconciler (Hub)
    participant SpokeK8s as Spoke Cluster API Server
    participant KlusterletCtrl as Klusterlet Controllers
    participant RegAgent as Registration Agent
    participant WorkAgent as Work Agent

    Note over User, HubController: Initialization

    User->>HubK8s: Create Spoke resource
    HubK8s->>HubController: Spoke resource created
    HubController->>HubK8s: Add HubCleanupPreflightFinalizer
    HubController->>HubK8s: Add HubCleanupFinalizer
    HubController->>HubK8s: Add SpokeCleanupFinalizer (hub-as-spoke only)
    
    Note over HubController, KlusterletCtrl: Join Process
    HubController->>HubController: Check if ManagedCluster exists
    alt ManagedCluster does not exist
        HubController->>SpokeK8s: Create Klusterlet and controllers (clusteradm join)
        RegAgent->>HubK8s: Create CSR
        HubController->>HubK8s: Accept CSR (clusteradm accept)
        HubController->>HubK8s: Wait for ManagedClusterJoined condition
        HubController->>HubController: Spoke Joined
    end

    Note over HubK8s, SpokeK8s: Addon Flow
    HubController->>HubK8s: Set up AddOnDeploymentConfigs for FCC-agent
    HubController->>HubK8s: Enable addons
    WorkAgent->>HubK8s: Pull FCC-agent addon ManifestWork
    WorkAgent->>SpokeK8s: Deploy FCC-agent addon (initiates pivot)

```

#### Day 2 - Maintenance
```mermaid
sequenceDiagram
    participant HubK8s as Hub Cluster API Server
    participant SpokeController as Spoke Reconciler (Spoke)
    participant SpokeK8s as Spoke Cluster API Server
    participant Klusterlet as Klusterlet CR

    Note over HubK8s, Klusterlet: Mainenance Flow
    SpokeController->>HubK8s: Add SpokeCleanupFinalizer
    SpokeController->>HubK8s: Set PivotComplete condition
    SpokeController->>HubK8s: Get Hub, klusterlet helm values
    SpokeController->>Klusterlet: Check klusterlet upgrade needed
    alt Upgrade needed
        SpokeController->>SpokeK8s: Run clusteradm upgrade klusterlet
    end
```
#### Day 2 - Cleanup

```mermaid
sequenceDiagram
    participant User
    participant HubK8s as Hub Cluster API Server
    participant HubController as Spoke Reconciler (Hub)
    participant SpokeK8s as Spoke Cluster API Server
    participant SpokeController as Spoke Reconciler (Spoke)

    Note over User, SpokeController: Cleanup Flow

    User->>HubK8s: Delete Spoke resource
    HubK8s->>HubController: Spoke deletion requested
    HubController->>HubK8s: Set phase to "Deleting"

    Note over HubK8s, HubController: Hub Pre-Flight Cleanup Phase
    alt ForceClusterDrain is set
        HubController->>HubK8s: Set `workload-cleanup` taint on ManagedCluster
    end
    HubController->>HubK8s: Check for active, non-addon ManifestWorks
    alt Active ManifestWorks
        HubController->>HubController: Requeue with error
    end
    HubController->>HubK8s: Set `terminating` taint on ManagedCluster
    HubController->>HubK8s: Disable addons (except fleetconfig-controller-agent)
    HubController->>HubK8s: Remove HubCleanupPreflightFinalizer
    
    Note over HubK8s, SpokeController: Spoke Cleanup Phase
    SpokeController->>HubK8s: HubCleanupPreflightFinalizer removed?
    alt Not Removed
        SpokeController->>SpokeController: Requeue
    end
    SpokeController->>SpokeK8s: Remove Klusterlet and OCM namespaces (clusteradm unjoin)
    SpokeController->>HubK8s: Remove SpokeCleanupFinalizer

    Note over HubK8s, HubController: Final Hub Cleanup
    HubController->>HubK8s: SpokeCleanupFinalizer removed?
    alt Not Removed
        HubController->>HubController: Requeue
    end
    HubController->>HubK8s: Clean up CSR, FCC-agent AddOn ManifestWork Finalizer, ManagedCluster, namespace
    HubController->>HubK8s: Remove HubCleanupFinalizer
    
    HubK8s->>User: Spoke resource deleted
    SpokeController->>SpokeK8s: Remove AppliedManifestWork (which removes FCC-agent)

    Note over HubK8s, HubController: Special Cases
    Note right of HubK8s: Hub-as-spoke: Hub does both hub and spoke cleanup
    Note right of HubK8s: Failed Pivot: Hub does spoke cleanup if agent never came up
```

### Hub Deletion

#### Actors:
- User: End user
- HubK8s: Hub cluster Kubernetes API server
- Hub Reconciler: Controller responsible for reconciling the Hub resource
- ClusterManager: ClusterManager CR and related controllers installed on the hub cluster

#### Cleanup Flow
```mermaid
sequenceDiagram
    participant User
    participant HubK8s as Hub Cluster API Server
    participant HubReconciler as Hub Reconciler

    Note over User, HubReconciler: Hub Deletion

    User->>HubK8s: Delete Hub resource
    HubK8s->>HubReconciler: Hub deletion requested
    HubReconciler->>HubK8s: Set phase to "Deleting"
    
    Note over HubK8s, HubReconciler: Hub Pre-Flight Cleanup Phase
    HubReconciler->>HubK8s: Check for Spoke resources with HubRef.Name/Namespace == Hub.Name/Namespace
    alt Joined Spokes found 
        HubReconciler->>HubK8s: Mark Spokes for deletion (if not already marked)
        HubReconciler->>HubReconciler: Requeue until all Spokes are deleted
    end

    Note over HubK8s, ClusterManager: Hub Cleanup Phase
    HubReconciler->>HubK8s: Delete all AddOns managed by FCC
    HubReconciler->>HubK8s: Delete ClusterManager, OCM namespaces (clusteradm clean)
    HubReconciler->>HubK8s: Remove HubCleanupFinalizer

```