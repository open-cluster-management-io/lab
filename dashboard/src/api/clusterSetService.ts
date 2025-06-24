import { createHeaders } from './utils';

export interface ClusterSet {
  id: string;
  name: string;
  labels?: Record<string, string>;
  creationTimestamp?: string;
  spec?: {
    clusterSelector: {
      selectorType: string;
      labelSelector?: {
        matchLabels?: Record<string, string>;
      };
    };
  };
  status?: {
    conditions?: {
      type: string;
      status: string;
      reason?: string;
      message?: string;
      lastTransitionTime?: string;
    }[];
  };
}

// Make sure we also export a type to avoid compiler issues
export type { ClusterSet as ClusterSetType };

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Fetch all cluster sets
export const fetchClusterSets = async (): Promise<ClusterSet[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "default",
            name: "default",
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSelector: {
                selectorType: "ExclusiveClusterSetLabel"
              }
            },
            status: {
              conditions: [
                {
                  type: "ClusterSetEmpty",
                  status: "False",
                  reason: "ClustersSelected",
                  message: "2 ManagedClusters selected",
                  lastTransitionTime: "2025-05-14T09:37:25Z"
                }
              ]
            }
          },
          {
            id: "global",
            name: "global",
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSelector: {
                selectorType: "LabelSelector",
                labelSelector: {}
              }
            },
            status: {
              conditions: [
                {
                  type: "ClusterSetEmpty",
                  status: "False",
                  reason: "ClustersSelected",
                  message: "2 ManagedClusters selected",
                  lastTransitionTime: "2025-05-14T09:36:18Z"
                }
              ]
            }
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clustersets`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching cluster sets:', error);
    return [];
  }
};

// Fetch a single cluster set by name
export const fetchClusterSetByName = async (name: string): Promise<ClusterSet | null> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        if (name === "default") {
          resolve({
            id: "default",
            name: "default",
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSelector: {
                selectorType: "ExclusiveClusterSetLabel"
              }
            },
            status: {
              conditions: [
                {
                  type: "ClusterSetEmpty",
                  status: "False",
                  reason: "ClustersSelected",
                  message: "2 ManagedClusters selected",
                  lastTransitionTime: "2025-05-14T09:37:25Z"
                }
              ]
            }
          });
        } else if (name === "global") {
          resolve({
            id: "global",
            name: "global",
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSelector: {
                selectorType: "LabelSelector",
                labelSelector: {}
              }
            },
            status: {
              conditions: [
                {
                  type: "ClusterSetEmpty",
                  status: "False",
                  reason: "ClustersSelected",
                  message: "2 ManagedClusters selected",
                  lastTransitionTime: "2025-05-14T09:36:18Z"
                }
              ]
            }
          });
        } else {
          resolve(null);
        }
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clustersets/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching cluster set ${name}:`, error);
    return null;
  }
};
