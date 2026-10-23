# Scaffold a Template-based OCM Add-on

## Purpose

Turn a workload that runs on one Kubernetes cluster into an Open Cluster Management (OCM) add-on
built with the `AddOnTemplate` API, so the hub can roll it out to many managed clusters. The
output is a set of YAML files to apply on the hub, plus commands to verify the rollout.

## Context

A template add-on needs no code. It is four hub objects: `AddOnTemplate` (what to deploy),
`AddOnDeploymentConfig` (per-cluster variables), `ClusterManagementAddOn` (registration and
rollout) and `Placement` (which clusters). The add-on manager in OCM creates a
`ManagedClusterAddOn` and a `ManifestWork` for each selected cluster.

The full explanation is in the guide
[Build an OCM Add-on with AddOnTemplate](../../guides/getting-started/build-an-addon-with-addontemplate.md)
and the upstream
[add-on developer guide](https://open-cluster-management.io/docs/developer-guides/addon/#build-an-addon-with-addon-template).
The rules are repeated inside the prompt below, because assistants follow instructions more
reliably than links.

The output needs an OCM v1.1.0 or newer hub. Older hubs ignore `agentInstallNamespace: ""`.

Before using the prompt, have ready:

- The workload manifests (YAML) you want to ship.
- A name for the add-on.
- Which clusters should get it (all, or a label selector).
- Which values differ between clusters (image tag, log level, and so on).
- Whether the agent needs to access the hub cluster.

## Prompt

Copy the text below into your AI assistant, fill in the parts in angle brackets, and paste your
manifests at the end.

~~~text
You are helping me build an Open Cluster Management (OCM) add-on with the AddOnTemplate API.

Inputs:
- Add-on name: <addon-name, lowercase letters, digits and dashes>
- Target clusters: <"all clusters" or a label selector such as environment=production>
- Configurable values and their defaults for all clusters: <e.g. IMAGE_TAG=v1.2.3,
  LOG_LEVEL=info, or "none">
- Overrides for some clusters: <e.g. LOG_LEVEL=debug for clusters labeled environment=dev,
  or "none">
- Agent needs access to the hub: <no | yes, and what it needs to read or write>
- Placement namespace on the hub: <default>
- Rollout: <all at once | progressive>
- Workload manifests: pasted at the end of this message.

First, check fit. A template add-on is right only if the agent workloads are Deployments and/or
DaemonSets, no configuration API is needed beyond AddOnDeploymentConfig, nothing must run on the
hub besides managing the agent, and hosted mode is not needed. If any of these fails, stop and
tell me to use the addon-framework (Go) instead, and why.

Then generate these files, each as a separate YAML document with a short comment at the top:

1. 00-managedclustersetbinding.yaml: ManagedClusterSetBinding
   (cluster.open-cluster-management.io/v1beta2) named "global" in the Placement namespace,
   spec.clusterSet: global. Say that it can be skipped only if the global cluster set is
   already bound to that namespace; the Placement below selects from global only.
2. 01-addontemplate.yaml: AddOnTemplate (addon.open-cluster-management.io/v1alpha1).
   Cluster-scoped, no metadata.namespace. spec.addonName equals the add-on name. Put my
   manifests under spec.agentSpec.workload.manifests, unchanged except for parameterized values.
3. 02-addondeploymentconfig.yaml: AddOnDeploymentConfig (addon.open-cluster-management.io/v1alpha1)
   in namespace open-cluster-management, with spec.agentInstallNamespace: "" and one
   spec.customizedVariables entry per configurable value, set to its default. These values
   apply to every selected cluster. For each override group, add another AddOnDeploymentConfig
   with the full variable list and its own values, plus a Placement for that group. List that
   Placement under installStrategy.placements after the main Placement, with configs
   referencing its AddOnDeploymentConfig. When placements select the same cluster, only the
   last one listed applies its configs.
4. 03-clustermanagementaddon.yaml: ClusterManagementAddOn (addon.open-cluster-management.io/v1alpha1)
   with the annotation addon.open-cluster-management.io/lifecycle: "addon-manager" (added
   automatically since OCM v0.14.0; keep it so older hubs work too);
   spec.supportedConfigs listing addontemplates (defaultConfig.name = the AddOnTemplate name)
   and addondeploymentconfigs (defaultConfig = the AddOnDeploymentConfig name and namespace);
   spec.installStrategy.type: Placements pointing at the Placement. For progressive rollout use
   rolloutStrategy.type: Progressive with maxConcurrency, minSuccessTime, progressDeadline and
   maxFailures.
5. 04-placement.yaml: Placement (cluster.open-cluster-management.io/v1beta1) in the Placement
   namespace with spec.clusterSets: [global]. If I gave labels, put them under
   spec.predicates[].requiredClusterSelector.labelSelector.matchLabels (never directly under
   spec).

Rules you must follow:
- Variables in the template use bare double braces: {{NAME}}. This is plain string
  substitution, not Go templating. Never write {{ .ClusterName }}, {{ .Values.x }} or similar;
  they render as an empty string without any error.
- Built-in variables: {{CLUSTER_NAME}} (managed cluster name), {{INSTALL_NAMESPACE}} (only
  when agentInstallNamespace is non-empty) and {{HUB_KUBECONFIG}} (hub kubeconfig path, only
  when the template has a registration). Every other variable must be defined in
  AddOnDeploymentConfig.spec.customizedVariables. Names match ^[a-zA-Z_][_a-zA-Z0-9]*$ and
  values are strings.
- Never put a variable in metadata.namespace or in a label value; the API server rejects the
  AddOnTemplate. Use variables only in values such as image, args, env and annotations.
- Set metadata.namespace on every namespaced object in the template.
- OCM already adds CLUSTER_NAME (and HUB_KUBECONFIG with a registration) as environment
  variables to every container of the template's Deployments and DaemonSets. Do not add env
  entries with those names.
- Add a Namespace manifest to the template, first in the list, for every namespace my manifests
  use, unless it is certain to exist on every managed cluster.
- Keep agentInstallNamespace: "" in the AddOnDeploymentConfig. If it is non-empty (its default
  is open-cluster-management-agent-addon), OCM moves every namespaced object in the template to
  that namespace and rewrites RoleBinding and ClusterRoleBinding subjects.
- Only Deployments and DaemonSets count toward add-on health. If my manifests have neither,
  say that the add-on will be reported available without any workload being checked.
- Do not generate ManagedClusterAddOn objects; the add-on manager creates them.
- Do not use PlacementBinding or any policy API; add-on rollout is installStrategy plus Placement.
- Placement tolerations use the effects NoSelect, PreferNoSelect or NoSelectIfNew, never
  NoSchedule or NoExecute.
- If the agent needs hub access, add spec.registration with type KubeClient and hubPermissions
  (CurrentCluster or SingleNamespace) to the AddOnTemplate, use {{HUB_KUBECONFIG}} for the
  kubeconfig path, and list the hub Role or ClusterRole I must create and the binding the add-on
  manager (service account open-cluster-management-hub/addon-manager-controller-sa) needs.
- If my manifests contain RBAC or other kinds the work agent on the managed cluster may not be
  allowed to create, tell me which ones.
- Do not invent fields. If something is unclear, ask me before generating.

After the files, give me:
- The apply command: kubectl apply -f <dir>/
- A server-side dry run first: kubectl apply --dry-run=server --validate=strict -f <dir>/
  Say that it checks only the hub objects, not the fields of the embedded workload manifests,
  and give a second dry run of the workload manifests alone, with real values in place of the
  variables, against one managed cluster.
- Verification commands: kubectl get managedclusteraddon -A, kubectl get manifestwork -n
  <cluster-name>, and how to read the ManagedClusterAddOn conditions.
- A short list of what I should check by hand before applying.

Workload manifests:
<paste YAML here>
~~~

## Expected Output

- A fit check: template add-on or addon-framework, with the reason.
- Five YAML files: `00-managedclustersetbinding.yaml`, `01-addontemplate.yaml`,
  `02-addondeploymentconfig.yaml`, `03-clustermanagementaddon.yaml`, `04-placement.yaml`.
- Variables written as `{{NAME}}`, a `Namespace` manifest for each namespace the workload uses,
  and `agentInstallNamespace: ""`.
- Dry-run, apply and verification commands, and a short manual checklist.

Always review the output before applying it. Run the server-side dry run against your hub and
check that a `ManagedClusterAddOn` appears for each selected cluster and becomes `Available`.

## Example Usage

Inputs:

- Add-on name: `node-exporter`
- Target clusters: all clusters
- Configurable values: `IMAGE_TAG=v1.12.1`, `LOG_LEVEL=info`; overrides: none
- Agent needs access to the hub: no
- Placement namespace: `default`
- Rollout: progressive
- Workload: a `DaemonSet` named `node-exporter` in namespace `node-exporter` running
  `quay.io/prometheus/node-exporter`

Result, in short:

- `01-addontemplate.yaml` starts with a `Namespace` named `node-exporter`, followed by the
  DaemonSet with `image: quay.io/prometheus/node-exporter:{{IMAGE_TAG}}` and
  `--log.level={{LOG_LEVEL}}`.
- `02-addondeploymentconfig.yaml` defines `IMAGE_TAG` and `LOG_LEVEL` and sets
  `agentInstallNamespace: ""`.
- `03-clustermanagementaddon.yaml` registers `node-exporter` with a progressive rollout to the
  `node-exporter` Placement in `default`.
- After `kubectl apply -f node-exporter-addon/`, `kubectl get managedclusteraddon -A` lists
  `node-exporter` in every cluster namespace, and `AVAILABLE` becomes `True` when the DaemonSet is
  ready.

The complete files for this example are in the
[guide](../../guides/getting-started/build-an-addon-with-addontemplate.md#steps).
