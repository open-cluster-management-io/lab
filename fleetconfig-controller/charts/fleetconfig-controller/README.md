# fleetconfig-controller helm chart

## TL;DR

```bash
helm repo add ocm https://open-cluster-management.io/helm-charts
helm repo update ocm
helm install fleetconfig-controller ocm/fleetconfig-controller -n fleetconfig-system --create-namespace
```

## Prerequisites

- Kubernetes >= v1.19
  
## Parameters

### fleetconfig-controller parameters

| Name                                                | Description                                                                                                                    | Value                                                    |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| `kubernetesProvider`                                | Kubernetes provider of the cluster that fleetconfig-controller will be installed on. Valid values are "Generic", "EKS", "GKE". | `Generic`                                                |
| `replicas`                                          | fleetconfig-controller replica count                                                                                           | `1`                                                      |
| `imageRegistry`                                     | Image registry                                                                                                                 | `""`                                                     |
| `image.repository`                                  | Image repository                                                                                                               | `quay.io/open-cluster-management/fleetconfig-controller` |
| `image.tag`                                         | x-release-please-version                                                                                                       | `v0.0.1`                                                 |
| `image.pullPolicy`                                  | Image pull policy                                                                                                              | `IfNotPresent`                                           |
| `imagePullSecrets`                                  | Image pull secrets                                                                                                             | `[]`                                                     |
| `serviceAccount.annotations`                        | Annotations to add to the service account                                                                                      | `{}`                                                     |
| `containerSecurityContext.allowPrivilegeEscalation` | allowPrivilegeEscalation                                                                                                       | `false`                                                  |
| `containerSecurityContext.capabilities.drop`        | capabilities to drop                                                                                                           | `["ALL"]`                                                |
| `containerSecurityContext.runAsNonRoot`             | runAsNonRoot                                                                                                                   | `true`                                                   |
| `resources.limits.cpu`                              | fleetconfig controller's cpu limit                                                                                             | `500m`                                                   |
| `resources.limits.memory`                           | fleetconfig controller's memory limit                                                                                          | `256Mi`                                                  |
| `resources.requests.cpu`                            | fleetconfig controller's cpu request                                                                                           | `100m`                                                   |
| `resources.requests.memory`                         | fleetconfig controller's memory request                                                                                        | `256Mi`                                                  |
| `healthCheck.port`                                  | port the liveness & readiness probes are bound to                                                                              | `9440`                                                   |
| `kubernetesClusterDomain`                           | kubernetes cluster domain                                                                                                      | `cluster.local`                                          |

### cert-manager

| Name                            | Description                               | Value  |
| ------------------------------- | ----------------------------------------- | ------ |
| `cert-manager.enabled`          | Whether to install cert-manager.          | `true` |
| `clusterIssuer.spec.selfSigned` | Default self-signed issuer configuration. | `{}`   |

### webhook parameters

| Name                                                 | Description                              | Value                    |
| ---------------------------------------------------- | ---------------------------------------- | ------------------------ |
| `admissionWebhooks.enabled`                          | enable admission webhooks                | `true`                   |
| `admissionWebhooks.failurePolicy`                    | admission webhook failure policy         | `Fail`                   |
| `admissionWebhooks.certificate.mountPath`            | admission webhook certificate mount path | `/etc/k8s-webhook-certs` |
| `admissionWebhooks.certManager.revisionHistoryLimit` | cert-manager revision history limit      | `3`                      |
| `webhookService.type`                                | webhook service type                     | `ClusterIP`              |
| `webhookService.port`                                | webhook service port                     | `9443`                   |

### dev parameters

| Name               | Description       | Value                    |
| ------------------ | ----------------- | ------------------------ |
| `devspaceEnabled`  | devspace enabled  | `false`                  |
| `fullnameOverride` | Fullname override | `fleetconfig-controller` |
