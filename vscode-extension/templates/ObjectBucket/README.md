# OCM ObjectBucket-type Application Lifecycle Project

Applying these resource files will create the following resources:

| Kind                     | Namespace      | Name                    |
| ------------------------ | -------------- | ----------------------- |
| Namespace                | x              | helloworld-chn          |
| Namespace                | x              | helloworld-app          |
| Channel                  | helloworld-chn | helloworld-channel      |
| ManagedClusterSet        | x              | helloworld-clusterset   |
| ManagedClusterSetBinding | helloworld-app | helloworld-clusterset   |
| Placement                | helloworld-app | helloworld-placement    |
| Subscription             | helloworld-app | helloworld-subscription |

## Prerequisites

The [Application Lifecycle Management Addon][0] should be installed and enabled in your hub/managed clusters.

## Make it your own

- Go into _channel.yaml_ and update the _pathname_ of your repository.
- Go into _subscription.yaml_ and update the configuration via annotations based on your needs.
- Label the _ManagedClusters_ which should be part of the _ManagedClusterSet_ with the following label:
  `cluster.open-cluster-management.io/clusterset=helloworld-clusterset`.
- Label the _ManagedClusters_ you want to be selected by the _Placement_ with the following label:
  `usage: development`.

It is also advised to modify the various CRs names,</br>
but if you're ok with _helloworld_, have at it. :v:

## Run and verify

When you're done with the _YAML_ files, apply them onto your hub cluster, and watch the magic takes place.

[0]: https://open-cluster-management.io/getting-started/integration/app-lifecycle/
