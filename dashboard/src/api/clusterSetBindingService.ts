import { createHeaders } from './utils';

export interface ClusterSetBinding {
  id: string;
  name: string;
  namespace: string;
  creationTimestamp?: string;
  spec: {
    clusterSet: string;
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

export interface ManagedClusterSetBinding {
  id: string;
  name: string;
  namespace: string;
  clusterSet: string;
  bound: boolean;
  creationTimestamp?: string;
}

// Raw API response interface
interface RawBindingData {
  id?: string;
  uid?: string;
  name: string;
  namespace: string;
  creationTimestamp?: string;
  spec?: {
    clusterSet?: string;
  };
  status?: {
    conditions?: Array<{
      type: string;
      status: string;
      reason?: string;
      message?: string;
      lastTransitionTime?: string;
    }>;
  };
}

// Make sure we also export a type to avoid compiler issues
export type { ClusterSetBinding as ClusterSetBindingType };

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Fetch all cluster set bindings for a namespace
export const fetchNamespaceClusterSetBindings = async (namespace: string): Promise<ClusterSetBinding[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "default-binding",
            name: "default-binding",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSet: "default"
            },
            status: {
              conditions: [
                {
                  type: "BindingValid",
                  status: "True",
                  reason: "BindingValid",
                  message: "Binding to ManagedClusterSet default is valid",
                  lastTransitionTime: "2025-05-14T09:37:25Z"
                }
              ]
            }
          },
          {
            id: "global-binding",
            name: "global-binding",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSet: "global"
            },
            status: {
              conditions: [
                {
                  type: "BindingValid",
                  status: "True",
                  reason: "BindingValid",
                  message: "Binding to ManagedClusterSet global is valid",
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
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/clustersetbindings`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching cluster set bindings:', error);
    return [];
  }
};

// Fetch a single cluster set binding by name and namespace
export const fetchClusterSetBindingByName = async (namespace: string, name: string): Promise<ClusterSetBinding | null> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        if (name === "default-binding") {
          resolve({
            id: "default-binding",
            name: "default-binding",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSet: "default"
            },
            status: {
              conditions: [
                {
                  type: "BindingValid",
                  status: "True",
                  reason: "BindingValid",
                  message: "Binding to ManagedClusterSet default is valid",
                  lastTransitionTime: "2025-05-14T09:37:25Z"
                }
              ]
            }
          });
        } else if (name === "global-binding") {
          resolve({
            id: "global-binding",
            name: "global-binding",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            spec: {
              clusterSet: "global"
            },
            status: {
              conditions: [
                {
                  type: "BindingValid",
                  status: "True",
                  reason: "BindingValid",
                  message: "Binding to ManagedClusterSet global is valid",
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
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/clustersetbindings/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching cluster set binding ${namespace}/${name}:`, error);
    return null;
  }
};

// Fetch all cluster set bindings across all namespaces
export const fetchClusterSetBindings = async (): Promise<ManagedClusterSetBinding[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "default-default",
            name: "default",
            namespace: "default",
            clusterSet: "default",
            bound: true,
            creationTimestamp: "2025-05-20T13:51:37Z"
          },
          {
            id: "open-cluster-management-addon-global",
            name: "global",
            namespace: "open-cluster-management-addon",
            clusterSet: "global",
            bound: true,
            creationTimestamp: "2025-05-20T08:52:35Z"
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clustersetbindings`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    const data = await response.json() as RawBindingData[];

    return data.map((binding: RawBindingData) => ({
      id: binding.id || binding.uid || `${binding.namespace}-${binding.name}`,
      name: binding.name,
      namespace: binding.namespace,
      clusterSet: binding.spec?.clusterSet || '',
      bound: binding.status?.conditions?.some((condition) =>
        condition.type === 'Bound' && condition.status === 'True'
      ) || false,
      creationTimestamp: binding.creationTimestamp
    }));
  } catch (error) {
    console.error('Error fetching all cluster set bindings:', error);
    return [];
  }
};

// Fetch all bindings for a specific cluster set
export const fetchClusterSetBindingsByClusterSet = async (clusterSetName: string): Promise<ManagedClusterSetBinding[]> => {
  try {
    const allBindings = await fetchClusterSetBindings();
    return allBindings.filter(binding => binding.clusterSet === clusterSetName);
  } catch (error) {
    console.error(`Error fetching bindings for cluster set ${clusterSetName}:`, error);
    return [];
  }
};