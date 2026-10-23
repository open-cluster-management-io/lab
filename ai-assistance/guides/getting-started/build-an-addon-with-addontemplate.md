# Build an OCM Add-on with AddOnTemplate

## Overview

This guide walks through turning a workload that runs on one cluster into an Open Cluster
Management (OCM) add-on that the hub rolls out to many managed clusters, using the
`AddOnTemplate` API. No Go code and no hub-side controller are needed: the add-on is four YAML
objects, and the add-on manager that ships with OCM does the rest.

It is written to be read by people and by AI assistants. If you use an assistant, give it this
guide as context together with the companion prompt,
[`scaffold-template-addon.md`](../../prompts/addon-development/scaffold-template-addon.md).

The guide restates the rules that matter and links to the upstream documentation for the full
reference:

- [Add-on developer guide: Build an addon with addon template](https://open-cluster-management.io/docs/developer-guides/addon/#build-an-addon-with-addon-template)
- [Add-on concepts](https://open-cluster-management.io/docs/concepts/add-on-extensibility/addon/)
- [Add-on lifecycle management (install and rollout strategies)](https://open-cluster-management.io/docs/getting-started/installation/addon-management/)

## Prerequisites

- An OCM hub with at least one managed cluster joined. For a local setup, see
  [Quick start](https://open-cluster-management.io/docs/getting-started/quick-start/).
- `kubectl` access to the hub. `clusteradm` is optional but convenient.
- The `AddonManagement` feature gate must not be disabled in
  `ClusterManager.spec.addOnManagerConfiguration` or `Klusterlet.spec.registrationConfiguration`.
  It is enabled by default.
- The workload you want to ship, as plain Kubernetes manifests (`Deployment` and/or `DaemonSet`,
  plus whatever they need: `Namespace`, `ServiceAccount`, RBAC, `ConfigMap`, and so on).

## Step 0: Is a template add-on the right fit?

Use `AddOnTemplate` when **all** of these are true (from the upstream developer guide):

- The agent workloads that matter are `Deployment`s and/or `DaemonSet`s.
- No configuration API is needed beyond `AddOnDeploymentConfig`.
- Nothing needs to run on the hub other than managing the add-on agent.
- Hosted mode is not required.

If any of these is false (for example, you need a custom reconcile loop on the hub, manifests
computed from live cluster state, or your own configuration CRD), build the add-on with the
[addon-framework](https://github.com/open-cluster-management-io/addon-framework) instead. Most
"deploy this agent or operator to the fleet" cases fit the template path.

## Steps

The add-on is made of four objects on the hub. Create them in any order; the add-on manager
reconciles them as they appear. The example ships a `node-exporter` DaemonSet to every managed
cluster.

| Object | Scope | Answers |
|---|---|---|
| `AddOnTemplate` | cluster-scoped | **What** to deploy on each managed cluster |
| `AddOnDeploymentConfig` | namespaced | **How it varies**: variables, node placement, registries, proxy |
| `ClusterManagementAddOn` | cluster-scoped | **Registers** the add-on and says how to install and roll it out |
| `Placement` | namespaced | **Which** clusters get it |

### 1. Make sure the Placement can see clusters

A `Placement` only selects clusters from `ManagedClusterSet`s that are **bound to the
Placement's namespace**. OCM creates the `global` cluster set (all clusters) by default but does
not bind it to any namespace. Without a binding the Placement selects zero clusters and nothing
is installed; the only sign is a condition in the Placement's status
(`NoManagedClusterSetBindings`).

```yaml
apiVersion: cluster.open-cluster-management.io/v1beta2
kind: ManagedClusterSetBinding
metadata:
  name: global
  namespace: default        # the namespace your Placement lives in
spec:
  clusterSet: global
```

Equivalent: `clusteradm clusterset bind global --namespace default`.

### 2. AddOnTemplate: what to deploy

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnTemplate
metadata:
  name: node-exporter            # cluster-scoped: no namespace
spec:
  addonName: node-exporter       # must match the ClusterManagementAddOn name
  agentSpec:
    workload:
      manifests:
        # Ship every namespace your workload uses. Nothing else creates it.
        - apiVersion: v1
          kind: Namespace
          metadata:
            name: node-exporter
        - apiVersion: apps/v1
          kind: DaemonSet
          metadata:
            name: node-exporter
            namespace: node-exporter
            labels:
              app: node-exporter
            annotations:
              example.com/cluster: "{{CLUSTER_NAME}}"   # built-in variable
          spec:
            selector:
              matchLabels:
                app: node-exporter
            template:
              metadata:
                labels:
                  app: node-exporter
              spec:
                containers:
                  - name: node-exporter
                    image: quay.io/prometheus/node-exporter:{{IMAGE_TAG}}
                    args:
                      - "--log.level={{LOG_LEVEL}}"
                    ports:
                      - containerPort: 9100
                        name: metrics
```

Rules for the template:

- **Variables use bare double braces: `{{NAME}}`.** This is plain string substitution, not Go
  templating. `{{ .ClusterName }}` or `{{ .Values.x }}` match no variable and are replaced with an
  **empty string**, silently.
- Built-in variables: `{{CLUSTER_NAME}}` (the managed cluster name, cannot be overridden) and
  `{{HUB_KUBECONFIG}}` (path of the hub kubeconfig when the template has a `registration`;
  default `/managed/hub-kubeconfig/kubeconfig`). Everything else comes from
  `AddOnDeploymentConfig.spec.customizedVariables`.
- Variable names match `^[a-zA-Z_][_a-zA-Z0-9]*$`. Values are strings only.
- **Variables cannot be used in `metadata.namespace` or in label values.** The hub validates
  these fields in the raw template, so `{{...}}` there makes the `AddOnTemplate` invalid and it
  is rejected when you apply it. Use variables in values such as `image`, `args`, `env` and
  annotations; write namespaces and labels as fixed strings.
- The built-in values are also **added as environment variables** to every container of the
  template's Deployments and DaemonSets: `CLUSTER_NAME`, plus `HUB_KUBECONFIG` when there is a
  registration and `INSTALL_NAMESPACE` when `agentInstallNamespace` is set. Don't define these env
  vars yourself, or the container ends up with duplicates.
- **Include a `Namespace` manifest** for every namespace your objects use, unless it is certain to
  exist on every managed cluster. Otherwise the ManifestWork fails with
  `namespaces "<ns>" not found` and the add-on never becomes available.
- Health: the add-on manager reports the add-on `Available` from the status of the
  **`Deployment`s and `DaemonSet`s** in the template. Other kinds are deployed but do not count
  toward health, and Deployments scaled to 0 replicas are not probed. A template with no
  Deployment or DaemonSet has nothing to probe, so it is reported available without any workload
  being checked.
- The objects are applied on the managed cluster by the work agent. If they include RBAC or other
  kinds the work agent cannot create, grant it permission first; see
  [permission setting for work agent](https://open-cluster-management.io/docs/concepts/work-distribution/manifestwork/#permission-setting-for-work-agent).

### 3. AddOnDeploymentConfig: how it varies

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: node-exporter
  namespace: open-cluster-management
spec:
  agentInstallNamespace: ""      # keep the namespaces written in the template (see below)
  customizedVariables:
    - name: IMAGE_TAG
      value: v1.12.1
    - name: LOG_LEVEL
      value: info
  # Optional: schedule the agent pods on specific nodes
  # nodePlacement:
  #   nodeSelector:
  #     node-role.kubernetes.io/worker: ""
```

**Always set `agentInstallNamespace: ""` unless you want every object moved.** The field defaults
to `open-cluster-management-agent-addon`. When it is non-empty, the add-on manager rewrites the
namespace of **every namespaced object** in the template, renames the template's `Namespace`
objects, and rewrites `subjects[].namespace` in `RoleBinding`s and `ClusterRoleBinding`s to that
value. An object that must stay in a specific namespace (for example an OLM `Subscription`,
which must sit in the same namespace as its `OperatorGroup`) will break. With `""`, the
namespaces in the template are used as written. See
[Namespace configuration with AddOnDeploymentConfig](https://open-cluster-management.io/docs/developer-guides/addon/#namespace-configuration-with-addondeploymentconfig).

You can create additional `AddOnDeploymentConfig`s for groups of clusters (per placement) or for a
single cluster (on its `ManagedClusterAddOn`). Later scopes override earlier ones; `CLUSTER_NAME`
always wins. See
[Add-on configurations](https://open-cluster-management.io/docs/getting-started/installation/addon-management/#add-on-configurations).

### 4. ClusterManagementAddOn: register and roll out

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: ClusterManagementAddOn
metadata:
  name: node-exporter
  annotations:
    # The add-on manager owns the add-on's lifecycle. Added automatically since OCM v0.14.0;
    # keep it explicit so the file also works on older hubs.
    addon.open-cluster-management.io/lifecycle: "addon-manager"
spec:
  addOnMeta:
    displayName: node-exporter
    description: Prometheus node-exporter on every managed cluster
  supportedConfigs:
    # Listing addontemplates is what makes this a template add-on
    - group: addon.open-cluster-management.io
      resource: addontemplates
      defaultConfig:
        name: node-exporter
    - group: addon.open-cluster-management.io
      resource: addondeploymentconfigs
      defaultConfig:
        name: node-exporter
        namespace: open-cluster-management
  installStrategy:
    type: Placements
    placements:
      - name: node-exporter
        namespace: default
        rolloutStrategy:
          type: Progressive
          progressive:
            maxConcurrency: 25%
            minSuccessTime: 5m
            progressDeadline: 10m
            maxFailures: 2
```

- `supportedConfigs` with `resource: addontemplates` and a `defaultConfig` naming the
  `AddOnTemplate` is required. Without it there is nothing to deploy.
- `installStrategy.type: Placements` installs the add-on on every cluster the listed Placements
  select, and removes it when a cluster leaves the Placement. With `Manual` (the default), you
  create a `ManagedClusterAddOn` in each cluster namespace yourself.
- `rolloutStrategy` controls how changes to the add-on's configuration (template or deployment
  config) reach the fleet: `All` (default), `Progressive`, or `ProgressivePerGroup`.
- Do not create `ManagedClusterAddOn`s by hand when using `Placements`; the add-on manager owns
  them.

### 5. Placement: which clusters

```yaml
apiVersion: cluster.open-cluster-management.io/v1beta1
kind: Placement
metadata:
  name: node-exporter
  namespace: default
spec:
  clusterSets:
    - global
  # Narrow the selection with labels, for example:
  # predicates:
  #   - requiredClusterSelector:
  #       labelSelector:
  #         matchLabels:
  #           environment: production
```

Placement tolerations match **cluster taints**, so their effects are `NoSelect`,
`PreferNoSelect` and `NoSelectIfNew`, not the pod effects `NoSchedule`/`NoExecute`. See
[Placement](https://open-cluster-management.io/docs/concepts/content-placement/placement/).

### 6. Apply and verify

```bash
# Managed clusters should be JOINED=True and AVAILABLE=True
kubectl get managedclusters

# Validate first; --validate=strict rejects unknown or misspelled fields
kubectl apply --dry-run=server --validate=strict -f addon/
kubectl apply -f addon/                       # the files from steps 1-5

# One ManagedClusterAddOn per selected cluster, created by the add-on manager
kubectl get managedclusteraddon -A | grep node-exporter
# or: clusteradm get addon node-exporter

# The add-on is delivered as a ManifestWork in each cluster namespace
kubectl get manifestwork -n <cluster-name> | grep addon-node-exporter

# Check conditions when something is not Available
kubectl get managedclusteraddon node-exporter -n <cluster-name> -o yaml
kubectl get placementdecision -n default -l cluster.open-cluster-management.io/placement=node-exporter -o yaml
```

On a managed cluster, `kubectl get ds -n node-exporter` shows the agent. `AVAILABLE` on the
`ManagedClusterAddOn` turns `True` once the DaemonSet is ready.

### 7. Optional: give the agent access to the hub

If the agent needs to read or write hub resources, add `spec.registration` to the
`AddOnTemplate`. A `KubeClient` registration gives the agent a hub kubeconfig, mounted into the
template's Deployments and DaemonSets and referenced with `{{HUB_KUBECONFIG}}`. Hub permissions
are granted with `hubPermissions` (`CurrentCluster` for the cluster's own namespace,
`SingleNamespace` for a fixed one). The referenced `ClusterRole`/`Role` must exist on the hub,
and the add-on manager must be allowed to bind it. A `CustomSigner` registration issues a
certificate from your own signer instead. Full example and RBAC notes:
[Build an addon with addon template](https://open-cluster-management.io/docs/developer-guides/addon/#build-an-addon-with-addon-template).

## Best Practices

- Keep the workload manifests as close to what already runs on one cluster as possible.
  Parameterize only what really differs between clusters.
- Prefer `Progressive` rollout for anything that can break a cluster. Set `maxFailures` so a bad
  change stops instead of reaching the whole fleet.
- Pin image tags through a variable (`{{IMAGE_TAG}}`) rather than using `latest`, so a rollout is
  a config change you can see and revert.
- Give the agent the least RBAC it needs on the managed cluster and on the hub.
- Add the `addon.open-cluster-management.io/deletion-orphan` annotation only to objects that must
  survive add-on removal (for example a `PersistentVolumeClaim`).
- Check generated YAML against the hub before applying:
  `kubectl apply --dry-run=server --validate=strict -f addon/` catches schema mistakes and
  fields that don't exist, which is the most common error in AI-generated manifests.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| No `ManagedClusterAddOn` is created | Placement selects zero clusters: no `ManagedClusterSetBinding` in the Placement's namespace, or labels don't match | Bind a cluster set to the namespace (step 1); check the `PlacementDecision` |
| No `ManagedClusterAddOn` is created, Placement is fine | `installStrategy` is missing or `Manual`; on hubs older than OCM v0.14.0, also the `addon.open-cluster-management.io/lifecycle: "addon-manager"` annotation | Set `installStrategy.type: Placements` (and add the annotation on older hubs) |
| ManifestWork fails with `namespaces "<ns>" not found` | The template uses a namespace it doesn't ship | Add a `Namespace` manifest to the template |
| Objects land in `open-cluster-management-agent-addon` | `agentInstallNamespace` left unset in the `AddOnDeploymentConfig` | Set `agentInstallNamespace: ""` |
| `AddOnTemplate ... is invalid: ... metadata.namespace` (or `metadata.labels`) | A `{{VARIABLE}}` in a namespace or label value | Use fixed strings there; put variables in values such as `image`, `args`, `env` or annotations |
| A value renders empty | Go-style variable such as `{{ .ClusterName }}`, or a typo in the name | Use `{{CLUSTER_NAME}}` / `{{MY_VAR}}` and define custom variables in the `AddOnDeploymentConfig` |
| `AVAILABLE` stays `Unknown` | A progressive rollout has not reached this cluster yet, or there are no probe results yet (ManifestWork not applied, workload not reporting status) | Check rollout progress in the `ClusterManagementAddOn` status (`installProgressions`), then the ManifestWork conditions in the cluster namespace |
| `AVAILABLE` is `False` | The workload is not ready on the managed cluster | Inspect the pods on that cluster and the ManifestWork status conditions |
| ManifestWork reports `forbidden` | The work agent lacks permission for a kind in the template | Grant the work agent permission on the managed cluster |

## Next Steps

- Use the companion prompt,
  [`scaffold-template-addon.md`](../../prompts/addon-development/scaffold-template-addon.md),
  to generate these files for your own workload.
- Read [Add-on lifecycle management](https://open-cluster-management.io/docs/getting-started/installation/addon-management/)
  for per-placement and per-cluster configuration and rollout options.
- Browse working add-ons in
  [addon-contrib](https://github.com/open-cluster-management-io/addon-contrib).
- If you outgrow the template, see the
  [addon-framework](https://github.com/open-cluster-management-io/addon-framework) and the
  [add-on developer guide](https://open-cluster-management.io/docs/developer-guides/addon/).
