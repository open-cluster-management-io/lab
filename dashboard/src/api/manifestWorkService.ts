import { createHeaders } from './utils';

export interface ManifestWork {
  id: string;
  name: string;
  namespace: string;
  labels?: Record<string, string>;
  creationTimestamp?: string;
  manifests?: Manifest[];
  conditions?: Condition[];
  resourceStatus?: ManifestResourceStatus;
}

export interface Manifest {
  rawExtension?: Record<string, unknown>;
}

export interface Condition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export interface ManifestResourceStatus {
  manifests?: ManifestCondition[];
}

export interface ManifestCondition {
  resourceMeta: ManifestResourceMeta;
  conditions: Condition[];
}

export interface ManifestResourceMeta {
  ordinal: number;
  group?: string;
  version?: string;
  kind?: string;
  resource?: string;
  name?: string;
  namespace?: string;
}

// Make sure we also export a type to avoid compiler issues
export type { ManifestWork as ManifestWorkType };

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Fetch all manifest works for a namespace
export const fetchManifestWorks = async (namespace: string): Promise<ManifestWork[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "app-deployment-work",
            name: "app-deployment",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            manifests: [
              {
                rawExtension: {
                  apiVersion: "apps/v1",
                  kind: "Deployment",
                  metadata: {
                    name: "example-app",
                    namespace: "default"
                  },
                  spec: {
                    replicas: 3
                  }
                }
              }
            ],
            conditions: [
              {
                type: "Applied",
                status: "True",
                reason: "WorkApplied",
                message: "All resources applied successfully",
                lastTransitionTime: "2025-05-14T09:37:25Z"
              }
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: {
                    ordinal: 0,
                    group: "apps",
                    version: "v1",
                    kind: "Deployment",
                    resource: "deployments",
                    name: "example-app",
                    namespace: "default"
                  },
                  conditions: [
                    {
                      type: "Applied",
                      status: "True",
                      reason: "AppliedManifestComplete",
                      message: "Resource has been applied",
                      lastTransitionTime: "2025-05-14T09:37:25Z"
                    }
                  ]
                }
              ]
            }
          },
          {
            id: "app-service-work",
            name: "app-service",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            manifests: [
              {
                rawExtension: {
                  apiVersion: "v1",
                  kind: "Service",
                  metadata: {
                    name: "example-service",
                    namespace: "default"
                  },
                  spec: {
                    ports: [
                      {
                        port: 80,
                        targetPort: 8080
                      }
                    ]
                  }
                }
              }
            ],
            conditions: [
              {
                type: "Applied",
                status: "True",
                reason: "WorkApplied",
                message: "All resources applied successfully",
                lastTransitionTime: "2025-05-14T09:36:18Z"
              }
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: {
                    ordinal: 0,
                    group: "",
                    version: "v1",
                    kind: "Service",
                    resource: "services",
                    name: "example-service",
                    namespace: "default"
                  },
                  conditions: [
                    {
                      type: "Applied",
                      status: "True",
                      reason: "AppliedManifestComplete",
                      message: "Resource has been applied",
                      lastTransitionTime: "2025-05-14T09:36:18Z"
                    }
                  ]
                }
              ]
            }
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/manifestworks`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching manifest works for namespace ${namespace}:`, error);
    return [];
  }
};

// Fetch a single manifest work by name and namespace
export const fetchManifestWorkByName = async (namespace: string, name: string): Promise<ManifestWork | null> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        if (name === "app-deployment") {
          resolve({
            id: "app-deployment-work",
            name: "app-deployment",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            manifests: [
              {
                rawExtension: {
                  apiVersion: "apps/v1",
                  kind: "Deployment",
                  metadata: {
                    name: "example-app",
                    namespace: "default"
                  },
                  spec: {
                    replicas: 3
                  }
                }
              }
            ],
            conditions: [
              {
                type: "Applied",
                status: "True",
                reason: "WorkApplied",
                message: "All resources applied successfully",
                lastTransitionTime: "2025-05-14T09:37:25Z"
              }
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: {
                    ordinal: 0,
                    group: "apps",
                    version: "v1",
                    kind: "Deployment",
                    resource: "deployments",
                    name: "example-app",
                    namespace: "default"
                  },
                  conditions: [
                    {
                      type: "Applied",
                      status: "True",
                      reason: "AppliedManifestComplete",
                      message: "Resource has been applied",
                      lastTransitionTime: "2025-05-14T09:37:25Z"
                    }
                  ]
                }
              ]
            }
          });
        } else if (name === "app-service") {
          resolve({
            id: "app-service-work",
            name: "app-service",
            namespace: namespace,
            creationTimestamp: "2025-05-14T09:35:54Z",
            manifests: [
              {
                rawExtension: {
                  apiVersion: "v1",
                  kind: "Service",
                  metadata: {
                    name: "example-service",
                    namespace: "default"
                  },
                  spec: {
                    ports: [
                      {
                        port: 80,
                        targetPort: 8080
                      }
                    ]
                  }
                }
              }
            ],
            conditions: [
              {
                type: "Applied",
                status: "True",
                reason: "WorkApplied",
                message: "All resources applied successfully",
                lastTransitionTime: "2025-05-14T09:36:18Z"
              }
            ],
            resourceStatus: {
              manifests: [
                {
                  resourceMeta: {
                    ordinal: 0,
                    group: "",
                    version: "v1",
                    kind: "Service",
                    resource: "services",
                    name: "example-service",
                    namespace: "default"
                  },
                  conditions: [
                    {
                      type: "Applied",
                      status: "True",
                      reason: "AppliedManifestComplete",
                      message: "Resource has been applied",
                      lastTransitionTime: "2025-05-14T09:36:18Z"
                    }
                  ]
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
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/manifestworks/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching manifest work ${namespace}/${name}:`, error);
    return null;
  }
};