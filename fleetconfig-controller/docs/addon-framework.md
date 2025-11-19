# Addon Configuration using FleetConfig Controller

## Overview

FleetConfig Controller (FCC) provides a declarative wrapper around the Open Cluster Management (OCM) [addon framework](https://open-cluster-management.io/docs/concepts/add-on-extensibility/addon/), simplifying the process of creating and deploying custom addons across your fleet of managed clusters. Instead of manually running `clusteradm` commands, you can define addons in your Hub and Spoke resources, and FCC handles the lifecycle management automatically.

The addon framework in FCC supports two types of addons:
1. **Custom Addons** - User-defined addons configured via `AddOnConfig` in the Hub spec
2. **Built-in Hub Addons** - Pre-configured addons (ArgoCD and Governance Policy Framework) via `HubAddOn`

## Custom Addons

### How It Works

FCC wraps the `clusteradm addon create` command, allowing you to define custom addon templates declaratively. When you specify an `AddOnConfig` in your Hub resource, FCC:

1. Looks for a ConfigMap containing the addon manifests
2. Creates an `AddOnTemplate` resource in the hub cluster using clusteradm
3. Makes the addon available for installation on spoke clusters
4. Manages the lifecycle (create/update/delete) based on your configuration

### Configuring a Custom Addon

#### Step 1: Create a ConfigMap with Addon Manifests

First, create a ConfigMap containing your addon manifests. The ConfigMap must be named following the pattern: `fleet-addon-<name>-<version>`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fleet-addon-myapp-v1.0.0
  namespace: <hub-namespace>
data:
  # Option 1: Provide raw manifests directly
  manifestsRaw: |
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: myapp-agent
      namespace: open-cluster-management-agent-addon
    spec:
      # ... your deployment spec
    ---
    # Additional resources...
  
  # Option 2: Provide a URL to manifests (mutually exclusive with manifestsRaw)
  # manifestsURL: "https://example.com/myapp/manifests.yaml"
```

**Note**: You must provide either `manifestsRaw` or `manifestsURL`, but not both.

#### Step 2: Define AddOnConfig in Hub Resource

Add the addon configuration to your Hub resource:

```yaml
apiVersion: fleet.ocm.io/v1beta1
kind: Hub
metadata:
  name: my-hub
spec:
  # ... other hub configuration
  
  addOnConfigs:
    - name: myapp
      version: v1.0.0
      
      # Optional: Enable hub registration for the addon agent
      # Allows the addon agent to register and communicate with the hub
      hubRegistration: false
      
      # Optional: Overwrite if addon already exists
      overwrite: false
      
      # Optional: ClusterRole binding for addon agent in the cluster namespace on the Hub cluster
      clusterRoleBinding: "cluster-admin"
```

#### AddOnConfig Fields

- **name** (required): The name of the addon
- **version** (optional): The addon version, defaults to "v0.0.1"
- **clusterRoleBinding** (optional): RoleBinding to a ClusterRole in the cluster namespace for the addon agent
- **hubRegistration** (optional): Enable agent registration to the hub cluster (default: false)
- **overwrite** (optional): Overwrite the addon if it already exists (default: false)

### Enabling Custom Addons on Spoke Clusters

Once you've defined an `AddOnConfig` in your Hub, you can enable it on specific spoke clusters by referencing it in the Spoke resource:

```yaml
apiVersion: fleet.ocm.io/v1beta1
kind: Spoke
metadata:
  name: my-spoke
spec:
  # ... other spoke configuration
  
  addOns:
    - configName: myapp  # Must match an AddOnConfig.name or HubAddOn.name
      
      # Optional: Add annotations to the addon
      annotations:
        example.com/type: "custom"
      
      # Optional: Provide deployment configuration
      deploymentConfig:
        nodePlacement:
          nodeSelector:
            node-role.kubernetes.io/worker: ""
        customizedVariables:
          - name: IMAGE
            value: "quay.io/myorg/myapp:v1.0.0"
```

#### AddOn Fields

- **configName** (required): The name of the addon, must match an `AddOnConfig.name` or `HubAddOn.name`
- **annotations** (optional): Annotations to apply to the addon
- **deploymentConfig** (optional): Additional configuration for addon deployment, creates an `AddOnDeploymentConfig` resource

The `deploymentConfig` field accepts the full [AddOnDeploymentConfigSpec](https://github.com/open-cluster-management-io/api/blob/main/addon/v1alpha1/types_addondeploymentconfig.go) from OCM, allowing you to configure:
- Node placement (node selectors, tolerations)
- Replicas and availability policy
- Custom variables for manifest templating
- Resource requirements

## Built-in Hub Addons

FCC includes two built-in addons that can be easily enabled without requiring a ConfigMap:

1. **argocd** - ArgoCD GitOps controller
2. **governance-policy-framework** - OCM governance policy framework

### Configuring Built-in Hub Addons

Add `HubAddOn` entries to your Hub resource:

```yaml
apiVersion: fleet.ocm.io/v1beta1
kind: Hub
metadata:
  name: my-hub
spec:
  # ... other hub configuration
  
  hubAddOns:
    - name: argocd
      createNamespace: true
    
    - name: governance-policy-framework
      installNamespace: open-cluster-management-addon
      createNamespace: false
```

#### HubAddOn Fields

- **name** (required): The name of the built-in addon. Must be one of: `argocd`, `governance-policy-framework`
- **installNamespace** (optional): The namespace to install the addon in. Defaults to "open-cluster-management-addon". Not supported for `argocd`.
- **createNamespace** (optional): Whether to create the namespace if it doesn't exist (default: false)

### Enabling Built-in Addons on Spoke Clusters

Built-in hub addons can be enabled on spoke clusters the same way as custom addons:

```yaml
apiVersion: fleet.ocm.io/v1beta1
kind: Spoke
metadata:
  name: my-spoke
spec:
  addOns:
    - configName: argocd
    - configName: governance-policy-framework
```

## Implementation Details

The addon creation logic in FCC (found in `internal/controller/v1beta1/addon.go`) performs the following operations:

### For Custom Addons (`AddOnConfig`)

1. **List existing addons**: Queries for `AddOnTemplate` resources with the `addon.open-cluster-management.io/managedBy` label
2. **Compare requested vs created**: Determines which addons need to be created or deleted
3. **Delete obsolete addons**: Removes addons no longer in the spec
4. **Create new addons**: For each new addon:
   - Loads the ConfigMap containing manifests (`fleet-addon-<name>-<version>`)
   - Validates manifest configuration (either raw or URL)
   - Constructs the `clusteradm addon create` command with appropriate flags
   - Executes the command to create the `AddOnTemplate`
   - Applies the `addon.open-cluster-management.io/managedBy` label for lifecycle tracking

### Command Construction

The controller builds `clusteradm` commands like:

```bash
clusteradm addon create <name> \
  --version=<version> \
  --labels=addon.open-cluster-management.io/managedBy=fleetconfig-controller \
  --filename=<manifests-path-or-url> \
  [--hub-registration] \
  [--overwrite] \
  [--cluster-role-bind=<role>] \
  --kubeconfig=<path> \
  --context=<context>
```

### For Built-in Hub Addons

Built-in hub addons follow a similar pattern but use pre-configured manifest sources from the `clusteradm` tool itself.

## Best Practices

1. **Version Management**: Always specify explicit versions for your custom addons to ensure predictable deployments. [Semantic versioning](https://semver.org/) is recommended, but not required.
2. **ConfigMap Naming**: Follow the naming convention strictly: `fleet-addon-<name>-<version>`
3. **Manifest Sources**: Use `manifestsURL` for large manifests or those stored in git; use `manifestsRaw` for simple, small manifests
4. **Hub Registration**: Only enable `hubRegistration` when your addon agent needs to communicate back to the hub
5. **Namespace Management**: For built-in hub addons, set `createNamespace: true` if you're unsure whether the namespace exists
6. **Labels**: FCC automatically applies the `addon.open-cluster-management.io/managedBy` label to track resources it manages
7. **Cleanup**: When removing an addon from the Hub spec, FCC will automatically delete the corresponding `AddOnTemplate`

## Troubleshooting

### Addon Not Created

- Verify the ConfigMap exists with the correct name format
- Check the Hub controller logs for clusteradm command errors
- Ensure the manifest source (raw or URL) is valid

### Addon Not Installing on Spoke

- Confirm the `configName` matches an existing `AddOnConfig` or `HubAddOn` name
- Check the spoke controller logs for addon reconciliation errors
- Verify the addon template exists in the hub cluster

### Version Conflicts

- If you need to update an addon version, create a new ConfigMap with the new version
- Update the `AddOnConfig` version in the Hub spec
- The old addon template will be automatically removed

