import { KubeObject } from '@kinvolk/headlamp-plugin/lib/k8s/cluster';

// --- Hub Operator ---

export class ClusterManager extends KubeObject {
  static kind = 'ClusterManager';
  static apiName = 'clustermanagers';
  static apiVersion = 'operator.open-cluster-management.io/v1';
  static isNamespaced = false;

  get featureGates(): Array<{ component: string; feature: string; mode: string }> {
    const gates: Array<{ component: string; feature: string; mode: string }> = [];
    const spec = this.jsonData.spec ?? {};
    const sections: Record<string, string> = {
      registrationConfiguration: 'Registration',
      workConfiguration: 'Work',
      addOnManagerConfiguration: 'Add-on Manager',
    };
    for (const [key, label] of Object.entries(sections)) {
      const fgs = spec[key]?.featureGates ?? [];
      for (const fg of fgs) {
        gates.push({ component: label, feature: fg.feature, mode: fg.mode });
      }
    }
    return gates;
  }
}

// --- Cluster Inventory ---

export class ManagedCluster extends KubeObject {
  static kind = 'ManagedCluster';
  static apiName = 'managedclusters';
  static apiVersion = 'cluster.open-cluster-management.io/v1';
  static isNamespaced = false;

  get kubernetesVersion(): string {
    return this.jsonData.status?.version?.kubernetes ?? '';
  }

  get cpuCapacity(): string {
    return this.jsonData.status?.capacity?.cpu ?? '';
  }

  get memoryCapacity(): string {
    return this.jsonData.status?.capacity?.memory ?? '';
  }

  get clusterClaims(): Array<{ name: string; value: string }> {
    return this.jsonData.status?.clusterClaims ?? [];
  }
}

export class ManagedClusterSet extends KubeObject {
  static kind = 'ManagedClusterSet';
  static apiName = 'managedclustersets';
  static apiVersion = 'cluster.open-cluster-management.io/v1beta2';
  static isNamespaced = false;
}

export class ManagedClusterSetBinding extends KubeObject {
  static kind = 'ManagedClusterSetBinding';
  static apiName = 'managedclustersetbindings';
  static apiVersion = 'cluster.open-cluster-management.io/v1beta2';
  static isNamespaced = true;
}

// --- Placements ---

export class Placement extends KubeObject {
  static kind = 'Placement';
  static apiName = 'placements';
  static apiVersion = 'cluster.open-cluster-management.io/v1beta1';
  static isNamespaced = true;

  get numberOfSelectedClusters(): number {
    return this.jsonData.status?.numberOfSelectedClusters ?? 0;
  }
}

export class PlacementDecision extends KubeObject {
  static kind = 'PlacementDecision';
  static apiName = 'placementdecisions';
  static apiVersion = 'cluster.open-cluster-management.io/v1beta1';
  static isNamespaced = true;

  get decisions(): Array<{ clusterName: string; reason: string }> {
    return this.jsonData.status?.decisions ?? [];
  }
}

// --- Addons ---

export class ManagedClusterAddOn extends KubeObject {
  static kind = 'ManagedClusterAddOn';
  static apiName = 'managedclusteraddons';
  static apiVersion = 'addon.open-cluster-management.io/v1beta1';
  static isNamespaced = true;

  get installNamespace(): string {
    return this.jsonData.status?.namespace ?? this.jsonData.spec?.installNamespace ?? '-';
  }
}

export class ClusterManagementAddOn extends KubeObject {
  static kind = 'ClusterManagementAddOn';
  static apiName = 'clustermanagementaddons';
  static apiVersion = 'addon.open-cluster-management.io/v1beta1';
  static isNamespaced = false;

  get displayName(): string {
    return this.jsonData.spec?.addOnMeta?.displayName ?? this.metadata.name;
  }

  get description(): string {
    return this.jsonData.spec?.addOnMeta?.description ?? '';
  }

  get installStrategy(): string {
    return this.jsonData.spec?.installStrategy?.type ?? 'Manual';
  }
}

// --- Governance / Policy ---

export class Policy extends KubeObject {
  static kind = 'Policy';
  static apiName = 'policies';
  static apiVersion = 'policy.open-cluster-management.io/v1';
  static isNamespaced = true;

  get complianceState(): string {
    return this.jsonData.status?.compliant ?? 'Pending';
  }
}

export class PlacementBinding extends KubeObject {
  static kind = 'PlacementBinding';
  static apiName = 'placementbindings';
  static apiVersion = 'policy.open-cluster-management.io/v1';
  static isNamespaced = true;
}

export class PolicySet extends KubeObject {
  static kind = 'PolicySet';
  static apiName = 'policysets';
  static apiVersion = 'policy.open-cluster-management.io/v1beta1';
  static isNamespaced = true;
}

// --- Work ---

export class ManifestWork extends KubeObject {
  static kind = 'ManifestWork';
  static apiName = 'manifestworks';
  static apiVersion = 'work.open-cluster-management.io/v1';
  static isNamespaced = true;

  get manifestCount(): number {
    return this.jsonData.spec?.workload?.manifests?.length ?? 0;
  }

  get manifests(): Array<{ apiVersion: string; kind: string; metadata: { name: string; namespace?: string } }> {
    return this.jsonData.spec?.workload?.manifests ?? [];
  }
}

// --- Managed Cluster (Spoke) Resources ---

export class Klusterlet extends KubeObject {
  static kind = 'Klusterlet';
  static apiName = 'klusterlets';
  static apiVersion = 'operator.open-cluster-management.io/v1';
  static isNamespaced = false;

  get deployMode(): string {
    return this.jsonData.spec?.deployOption?.mode ?? 'Default';
  }

  get clusterName(): string {
    return this.jsonData.spec?.clusterName ?? '';
  }

  get registrationNamespace(): string {
    return this.jsonData.spec?.namespace ?? 'open-cluster-management-agent';
  }
}

export class ClusterClaim extends KubeObject {
  static kind = 'ClusterClaim';
  static apiName = 'clusterclaims';
  static apiVersion = 'cluster.open-cluster-management.io/v1alpha1';
  static isNamespaced = false;

  get claimValue(): string {
    return this.jsonData.spec?.value ?? '';
  }
}

export class AppliedManifestWork extends KubeObject {
  static kind = 'AppliedManifestWork';
  static apiName = 'appliedmanifestworks';
  static apiVersion = 'work.open-cluster-management.io/v1';
  static isNamespaced = false;

  get hubHash(): string {
    return this.jsonData.spec?.hubHash ?? '';
  }

  get appliedResources(): Array<{ group: string; version: string; resource: string; name: string; namespace: string }> {
    return this.jsonData.status?.appliedResources ?? [];
  }
}
