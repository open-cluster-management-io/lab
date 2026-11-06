
# Validate AI-Generated OCM Add-ons

## Overview

This guide provides a repeatable process for validating
AI-generated Open Cluster Management (OCM) add-ons before
deploying them across managed Kubernetes clusters.

It focuses on AddOnTemplate-based add-ons and complements
the add-on development guide.

## Prerequisites

- An OCM hub with the required add-on APIs installed
- At least one registered managed cluster
- kubectl access to the hub and target managed clusters
- Generated AddOnTemplate-based manifests
- A dedicated test environment

## 1. Static Manifest Validation

Before applying manifests, check:

- AddOnTemplate.spec.addonName matches ClusterManagementAddOn.metadata.name.
- ClusterManagementAddOn references the correct AddOnTemplate.
- AddOnDeploymentConfig references are valid.
- Placement references point to the correct namespace.
- The required ManagedClusterSetBinding exists.
- Template variables use the supported syntax.
- All required namespaces and RBAC resources are included.
- agentInstallNamespace is explicitly configured.

Do not assume syntactically valid YAML is a valid OCM add-on.

## 2. Kubernetes API Validation

Run against a compatible OCM hub:

```bash
kubectl apply --dry-run=server --validate=strict -f addon/
```

Expected result: all resources pass validation.

Note: Server-side dry-run validates API admission but does
not prove that the add-on will deploy successfully.

## 3. Verify Cluster Readiness

```bash
kubectl get managedclusters
kubectl get placements -A
kubectl get placementdecisions -A
```

Confirm that target clusters are Joined and Available,
and that the Placement selects the expected clusters.

## 4. Deploy in a Test Environment

After reviewing the generated manifests:

```bash
kubectl apply -f addon/
```

Check the add-on resources:

```bash
kubectl get managedclusteraddon -A
kubectl get manifestwork -A
```

Inspect a specific add-on:

```bash
kubectl describe managedclusteraddon ADDON_NAME -n CLUSTER_NAME
```

Confirm the ManifestWork is applied and available, and
verify the actual workload on the managed cluster.

## 5. Negative Testing

Test these failure scenarios:

| Scenario | Expected behavior |
|---|---|
| Missing ManagedClusterSetBinding | Placement selects no eligible clusters |
| Invalid API field | Strict server-side validation rejects the resource |
| Undefined template variable | Detect through static checks or rendered-output inspection |
| Missing workload namespace | Workload application fails if the namespace does not exist |
| Incorrect namespace configuration | Workload may be installed in an unexpected namespace |
| Insufficient RBAC permissions | ManifestWork reports an application or permission failure |

## 6. Validation Test Matrix

Use a dedicated test cluster. Run each negative test independently
and restore the original manifests before starting the next test.

| ID | Test case | Expected result |
|---|---|---|
| TC-01 | Valid add-on manifests | Strict server-side validation succeeds |
| TC-02 | Unknown API field | API server rejects the invalid field |
| TC-03 | Missing cluster-set binding | Placement reports no eligible clusters |
| TC-04 | Invalid template variable in metadata.namespace | AddOnTemplate admission rejects the resource |
| TC-05 | Undefined customized variable | Rendered output must be inspected for incorrect substitution |
| TC-06 | Missing namespace | ManifestWork application fails if the namespace is absent |
| TC-07 | Incorrect installation namespace | Workload appears in an unexpected namespace |
| TC-08 | Insufficient work-agent RBAC | ManifestWork reports a permission error |
| TC-09 | Valid multi-cluster rollout | Add-on becomes Available on all selected clusters |
| TC-10 | Add-on deletion | Managed workloads are removed after reconciliation |

### TC-01: Valid manifests

```bash
kubectl apply --dry-run=server --validate=strict -f addon/
```

Expected: all generated resources pass API validation.

### TC-02: Unknown API field

In a disposable test manifest, add an unknown field to a
resource specification.

Run:

```bash
kubectl apply --dry-run=server --validate=strict -f invalid.yaml
```

Expected: the API server rejects the unknown field.

Restore the original manifest after the test.

### TC-03: Missing cluster-set binding

Use an isolated Placement namespace without a
ManagedClusterSetBinding.

Check:

```bash
kubectl get placement -n TEST_NAMESPACE
kubectl describe placement PLACEMENT_NAME -n TEST_NAMESPACE
kubectl get placementdecision -n TEST_NAMESPACE
```

Expected: the Placement cannot select clusters from an
unbound ManagedClusterSet.

Do not delete shared cluster-set bindings to reproduce this test.

### TC-09: Multi-cluster rollout

On the hub:

```bash
kubectl get managedclusters
kubectl get managedclusteraddon -A
kubectl get manifestwork -A
```

For each selected managed cluster, verify:

1. The ManagedClusterAddOn exists.
2. The add-on reports Available=True.
3. The corresponding ManifestWork reports successful application.
4. The Deployment or DaemonSet is ready on the managed cluster.
5. The actual workload namespace matches the expected namespace.

A successful server-side dry-run alone is not a passing
result for this test.



## 7. Evidence Collection

For each test, record:

- Test case and input manifests
- OCM and Kubernetes versions
- Validation commands
- Expected result
- Actual result
- Relevant resource conditions and error messages
- Pass/fail outcome

Do not mark a test as passed without checking the
corresponding output.


## 8. Cleanup

Remove only resources created for the test.

```bash
kubectl delete -f addon/
```

Verify that the add-on and its managed workloads are removed.

Be careful with shared resources such as cluster-set bindings,
namespaces and existing add-ons.

## 9. Best Practices

- Use dedicated test namespaces for negative tests.
- Validate manifests using strict server-side dry-run.
- Verify Placement and PlacementDecision status.
- Check ManagedClusterAddOn and ManifestWork conditions.
- Confirm workloads are running on managed clusters.
- Preserve test evidence and actual error messages.
- Never modify shared production resources during testing.

## 10. Troubleshooting

### Placement selects no clusters

Check cluster-set bindings and Placement conditions:

    kubectl get managedclustersetbindings -A
    kubectl describe placement PLACEMENT_NAME -n TEST_NAMESPACE
    kubectl get placementdecision -n TEST_NAMESPACE -o yaml

If the Placement reports NoManagedClusterSetBindings,
create the required binding in the Placement namespace.

### Manifest validation fails

Use strict server-side validation:

    kubectl apply --dry-run=server --validate=strict -f addon/

Review the API version, resource schema, and reported field.

### Add-on is not available

Inspect ManagedClusterAddOn and ManifestWork conditions:

    kubectl describe managedclusteraddon ADDON_NAME -n CLUSTER_NAME
    kubectl get manifestwork -n CLUSTER_NAME

Check the actual workload, image-pull errors, RBAC,
and namespace configuration.

## 11. Local Validation Results

Tests were performed using a kind-based OCM environment
with one registered managed cluster named ocm-managed.

| Test | Observed result | Status |
|---|---|---|
| TC-01 | Valid AddOnTemplate accepted by server-side dry-run | Passed |
| TC-02 | Unknown field spec.invalidTestField rejected | Passed |
| TC-03 | Missing binding caused NoManagedClusterSetBindings | Passed |
| TC-03 recovery | Binding restored selection of ocm-managed | Passed |
| Reference rollout | node-exporter ManifestWork Applied=True, Available=True | Verified |
| Workload readiness | node-exporter DaemonSet 1 desired, 1 ready | Verified |

The reference rollout validates an existing add-on, not
the deployment of the temporary TC-01 manifest.

## 12. Next Steps

- Validate template-variable and namespace failure cases.
- Test insufficient RBAC permissions in an isolated environment.
- Test rollout to multiple registered managed clusters.
- Verify add-on deletion and managed workload cleanup.
- Test the guide with two different AI-assisted platforms.
- Keep the documented workflow vendor-neutral.
