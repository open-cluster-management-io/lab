import { createHeaders } from './utils';
import type { Condition, ManifestWork } from './manifestWorkService';

export interface ManifestWorkReplicaSetSummary {
  total: number;
  available: number;
  progressing: number;
  degraded: number;
  applied: number;
}

export interface LocalPlacementReference {
  name: string;
  rolloutStrategyType?: string;
}

export interface MWRSPlacementSummary {
  name: string;
  availableDecisionGroups: string;
  summary: ManifestWorkReplicaSetSummary;
}

export interface ManifestWorkReplicaSet {
  id: string;
  name: string;
  namespace: string;
  labels?: Record<string, string>;
  placementRefs?: LocalPlacementReference[];
  conditions?: Condition[];
  summary: ManifestWorkReplicaSetSummary;
  placementsSummary?: MWRSPlacementSummary[];
  creationTimestamp?: string;
  manifestCount: number;
}

const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Fetch all ManifestWorkReplicaSets across all namespaces
export const fetchManifestWorkReplicaSets = async (): Promise<ManifestWorkReplicaSet[]> => {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "mwrs-nginx-1",
            name: "nginx-deployment",
            namespace: "default",
            placementRefs: [{ name: "all-clusters", rolloutStrategyType: "All" }],
            conditions: [
              { type: "PlacementVerified", status: "True", reason: "AsExpected", message: "Placement verified", lastTransitionTime: "2025-05-14T09:35:54Z" },
              { type: "ManifestworkApplied", status: "True", reason: "AsExpected", message: "All ManifestWorks applied", lastTransitionTime: "2025-05-14T09:36:18Z" },
            ],
            summary: { total: 2, available: 2, progressing: 0, degraded: 0, applied: 2 },
            placementsSummary: [
              { name: "all-clusters", availableDecisionGroups: "1/1", summary: { total: 2, available: 2, progressing: 0, degraded: 0, applied: 2 } },
            ],
            creationTimestamp: "2025-05-14T09:35:54Z",
            manifestCount: 2,
          },
          {
            id: "mwrs-monitoring-1",
            name: "monitoring-stack",
            namespace: "monitoring",
            placementRefs: [{ name: "production-clusters", rolloutStrategyType: "Progressive" }],
            conditions: [
              { type: "PlacementVerified", status: "True", reason: "AsExpected", message: "Placement verified", lastTransitionTime: "2025-05-14T10:00:00Z" },
              { type: "ManifestworkApplied", status: "True", reason: "Processing", message: "ManifestWorks being applied", lastTransitionTime: "2025-05-14T10:01:00Z" },
            ],
            summary: { total: 3, available: 1, progressing: 2, degraded: 0, applied: 1 },
            placementsSummary: [
              { name: "production-clusters", availableDecisionGroups: "1/3", summary: { total: 3, available: 1, progressing: 2, degraded: 0, applied: 1 } },
            ],
            creationTimestamp: "2025-05-14T10:00:00Z",
            manifestCount: 4,
          },
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/manifestworkreplicasets`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching ManifestWorkReplicaSets:', error);
    return [];
  }
};

// Fetch a single ManifestWorkReplicaSet by namespace and name
export const fetchManifestWorkReplicaSet = async (namespace: string, name: string): Promise<ManifestWorkReplicaSet | null> => {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve({
          id: "mwrs-nginx-1",
          name: name,
          namespace: namespace,
          placementRefs: [{ name: "all-clusters", rolloutStrategyType: "All" }],
          conditions: [
            { type: "PlacementVerified", status: "True", reason: "AsExpected", message: "Placement verified", lastTransitionTime: "2025-05-14T09:35:54Z" },
            { type: "ManifestworkApplied", status: "True", reason: "AsExpected", message: "All ManifestWorks applied", lastTransitionTime: "2025-05-14T09:36:18Z" },
            { type: "PlacementRolledOut", status: "True", reason: "Complete", message: "Rollout complete", lastTransitionTime: "2025-05-14T09:37:00Z" },
          ],
          summary: { total: 2, available: 2, progressing: 0, degraded: 0, applied: 2 },
          placementsSummary: [
            { name: "all-clusters", availableDecisionGroups: "1/1", summary: { total: 2, available: 2, progressing: 0, degraded: 0, applied: 2 } },
          ],
          creationTimestamp: "2025-05-14T09:35:54Z",
          manifestCount: 2,
        });
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/manifestworkreplicasets/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching ManifestWorkReplicaSet ${namespace}/${name}:`, error);
    return null;
  }
};

// Fetch ManifestWorks created by a specific ManifestWorkReplicaSet
export const fetchManifestWorksByReplicaSet = async (
  namespace: string,
  name: string
): Promise<ManifestWork[]> => {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "mock-mw-1",
            name: name,
            namespace: "cluster1",
            creationTimestamp: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
            conditions: [
              { type: "Applied", status: "True", reason: "WorkApplied", message: "All resources applied", lastTransitionTime: new Date().toISOString() },
              { type: "Available", status: "True", reason: "ResourcesAvailable", message: "All resources available", lastTransitionTime: new Date().toISOString() },
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: { ordinal: 0, group: "apps", version: "v1", kind: "Deployment", resource: "deployments", name: "example-app", namespace: "default" },
                  conditions: [{ type: "Applied", status: "True", reason: "AppliedManifestComplete", message: "Resource applied", lastTransitionTime: new Date().toISOString() }],
                },
              ],
            },
          },
          {
            id: "mock-mw-2",
            name: name,
            namespace: "cluster2",
            creationTimestamp: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
            conditions: [
              { type: "Applied", status: "True", reason: "WorkApplied", message: "All resources applied", lastTransitionTime: new Date().toISOString() },
              { type: "Available", status: "True", reason: "ResourcesAvailable", message: "All resources available", lastTransitionTime: new Date().toISOString() },
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: { ordinal: 0, group: "apps", version: "v1", kind: "Deployment", resource: "deployments", name: "example-app", namespace: "default" },
                  conditions: [{ type: "Applied", status: "True", reason: "AppliedManifestComplete", message: "Resource applied", lastTransitionTime: new Date().toISOString() }],
                },
              ],
            },
          },
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(
      `${API_BASE}/api/namespaces/${namespace}/manifestworkreplicasets/${name}/manifestworks`,
      { headers: createHeaders() }
    );
    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }
    return await response.json();
  } catch (error) {
    console.error(`Error fetching ManifestWorks for MWRS ${namespace}/${name}:`, error);
    return [];
  }
};
