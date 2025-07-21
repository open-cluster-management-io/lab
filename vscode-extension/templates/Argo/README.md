# OCM argcd application Project

Applying these resource files will create the following resources:

| Kind                     | Namespace      | Name                            |
| ------------------------ | -------------- | ------------------------------- |
| Namespace                | x              | argocd                          |
| Role                     | argcd          | ocm-placement-consumer          |
| RoleBinding              | argcd          | ocm-placement-consumer:argocd   |
| ConfigMap                | argcd          | ocm-placement-generator         |
| Placement                | argcd          | guestbook-app-placement         |
| ApplicationSet           | argcd          | guestbook-app                   |

## Prerequisites

[ArgoCD][0] should be installed and configured.

## Make it your own

- Go into _application-set.yaml_ and update the _repoURL_ with your repository.
- in the _placement.yaml_ update the spec according to your needs. more example could be found [here][1]

## Run and verify
When you're done with the _YAML_ files, apply them onto your hub cluster, and watch the magic takes place.
```
kubectl get applications  -n argocd
```

you can also login to the argocd ui to watch deployment progress 
-  port forward argo
```
kubectl port-forward svc/argocd-server -n argocd 8080:443
```
-  get admin secret password 
```
kubectl -n argocd get secret argocd-initial-admin-secret -o=jsonpath='{.data.password}' | base64 -d
```
- login to the argoUI and user name _admin_ and the _password_ retrieved from previous step   
https://localhost:8080

for more details click [here][2]


[0]: https://github.com/open-cluster-management-io/ocm/tree/main/solutions/deploy-argocd-apps
[1]: https://open-cluster-management.io/concepts/placement/
[2]: https://open-cluster-management.io/scenarios/integration-with-argocd