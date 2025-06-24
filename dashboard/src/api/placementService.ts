import type { Cluster } from './clusterService';
import { createHeaders } from './utils';

export interface PlacementDecision {
  name: string;
  namespace: string;
  decisions: {
    clusterName: string;
    reason: string;
  }[];
}

export interface Placement {
  id: string;
  name: string;
  namespace: string;
  creationTimestamp?: string;

  // Spec fields
  clusterSets: string[];
  numberOfClusters?: number;
  prioritizerPolicy?: {
    mode: string;
    configurations: {
      scoreCoordinate: {
        type: string;
        builtIn?: string;
        addOn?: {
          resourceName: string;
          scoreName: string;
        }
      };
      weight: number;
    }[];
  };
  predicates?: {
    requiredClusterSelector?: {
      labelSelector?: {
        matchLabels?: Record<string, string>;
        matchExpressions?: {
          key: string;
          operator: string;
          values: string[];
        }[];
      };
      claimSelector?: {
        matchExpressions?: {
          key: string;
          operator: string;
          values: string[];
        }[];
      };
      celSelector?: {
        celExpressions?: string[];
      };
    };
  }[];
  tolerations?: {
    key?: string;
    operator?: string;
    value?: string;
    effect?: string;
    tolerationSeconds?: number;
  }[];
  decisionStrategy?: {
    groupStrategy?: {
      decisionGroups?: {
        groupName: string;
        groupClusterSelector: {
          labelSelector?: {
            matchLabels?: Record<string, string>;
            matchExpressions?: {
              key: string;
              operator: string;
              values: string[];
            }[];
          };
        };
      }[];
      clustersPerDecisionGroup?: string;
    };
  };

  // Status fields
  numberOfSelectedClusters: number;
  decisionGroups?: {
    decisionGroupIndex: number;
    decisionGroupName: string;
    decisions: string[];
    clusterCount: number;
  }[];
  conditions?: {
    type: string;
    status: string;
    reason?: string;
    message?: string;
    lastTransitionTime?: string;
  }[];

  // Calculated fields
  succeeded: boolean; // Based on PlacementSatisfied condition status
  reasonMessage?: string;
  selectedClusters?: Cluster[];
}

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Helper to determine if a placement is succeeded based on PlacementSatisfied condition
const determineSucceededStatus = (conditions?: { type: string; status: string }[]): boolean => {
  if (!conditions || conditions.length === 0) return false;

  const satisfiedCondition = conditions.find(c => c.type === 'PlacementSatisfied');
  return satisfiedCondition?.status === 'True';
};

// Fetch all placements
export const fetchPlacements = async (): Promise<Placement[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "dd1845a6-4913-48ea-8784-24bf2fb1edf0",
            name: "placement-label-claim",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: ["default"],
            numberOfClusters: 1,
            numberOfSelectedClusters: 1,
            predicates: [
              {
                requiredClusterSelector: {
                  claimSelector: {
                    matchExpressions: [
                      {
                        key: "usage",
                        operator: "In",
                        values: ["dev"]
                      }
                    ]
                  },
                  labelSelector: {
                    matchLabels: {
                      "feature.open-cluster-management.io/addon-managed-serviceaccount": "available"
                    }
                  }
                }
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-label-claim-decision-1"],
                clusterCount: 1
              }
            ],
            conditions: [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ],
            succeeded: true
          },
          {
            id: "a5ca7369-0740-4b26-a8d5-77a097a3cfc9",
            name: "placement-priority",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: [],
            numberOfClusters: 1,
            numberOfSelectedClusters: 1,
            prioritizerPolicy: {
              mode: "Additive",
              configurations: [
                {
                  scoreCoordinate: {
                    type: "BuiltIn",
                    builtIn: "ResourceAllocatableMemory"
                  },
                  weight: 1
                }
              ]
            },
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-priority-decision-1"],
                clusterCount: 1
              }
            ],
            conditions: [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ],
            succeeded: true
          },
          {
            id: "0d39b430-8a78-46e1-b6fc-62b091196703",
            name: "placement-tolerations",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: [],
            numberOfSelectedClusters: 2,
            tolerations: [
              {
                key: "gpu",
                value: "true",
                operator: "Equal",
                tolerationSeconds: 300
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-tolerations-decision-1"],
                clusterCount: 2
              }
            ],
            conditions: [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ],
            succeeded: true
          },
          {
            id: "bcfd62a3-43a3-4717-9fce-3455631bbe82",
            name: "global",
            namespace: "open-cluster-management-addon",
            creationTimestamp: "2025-05-20T08:52:35Z",
            clusterSets: ["global"],
            numberOfSelectedClusters: 2,
            tolerations: [
              {
                key: "cluster.open-cluster-management.io/unreachable",
                operator: "Equal"
              },
              {
                key: "cluster.open-cluster-management.io/unavailable",
                operator: "Equal"
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["global-decision-1"],
                clusterCount: 2
              }
            ],
            conditions: [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              }
            ],
            succeeded: true
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/placements`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    const placements = await response.json();

    // Add succeeded status to each placement based on the PlacementSatisfied condition
    return placements.map((placement: Placement) => ({
      ...placement,
      succeeded: determineSucceededStatus(placement.conditions)
    }));
  } catch (error) {
    console.error('Error fetching placements:', error);
    return [];
  }
};

// Fetch a single placement by namespace and name
export const fetchPlacementByName = async (
  namespace: string,
  name: string
): Promise<Placement | null> => {
  console.log(`API fetchPlacementByName called with namespace=${namespace}, name=${name}`);

  // Handle the case where URL parameters might include namespace
  let actualNamespace = namespace;
  let actualName = name;

  // Check if name contains a slash, which means it might be in "namespace/name" format
  // But avoid re-parsing if it has already been processed in the component
  if (!namespace.includes('_PARSED_') && name && name.includes('/')) {
    const parts = name.split('/');
    if (parts.length === 2) {
      actualNamespace = parts[0];
      actualName = parts[1];
      console.log(`Parsed name from URL: namespace=${actualNamespace}, name=${actualName}`);
    }
  }

  // Remove marker
  if (actualNamespace.includes('_PARSED_')) {
    actualNamespace = actualNamespace.replace('_PARSED_', '');
  }

  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        console.log(`Checking mock data for ${actualNamespace}/${actualName}`);

        if (actualName === "placement-label-claim" && actualNamespace === "default") {
          console.log(`Returning mock data for ${actualNamespace}/${actualName}`);
          const conditions = [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ];

          resolve({
            id: "dd1845a6-4913-48ea-8784-24bf2fb1edf0",
            name: "placement-label-claim",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: ["default"],
            numberOfClusters: 1,
            numberOfSelectedClusters: 1,
            predicates: [
              {
                requiredClusterSelector: {
                  claimSelector: {
                    matchExpressions: [
                      {
                        key: "usage",
                        operator: "In",
                        values: ["dev"]
                      }
                    ]
                  },
                  labelSelector: {
                    matchLabels: {
                      "feature.open-cluster-management.io/addon-managed-serviceaccount": "available"
                    }
                  }
                }
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-label-claim-decision-1"],
                clusterCount: 1
              }
            ],
            conditions,
            succeeded: determineSucceededStatus(conditions),
            selectedClusters: [
              {
                id: "mock-cluster-1",
                name: "mock-cluster-1",
                status: "Online",
              }
            ],
          });
        } else if (actualName === "placement-priority" && actualNamespace === "default") {
          const priorityConditions = [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ];

          resolve({
            id: "a5ca7369-0740-4b26-a8d5-77a097a3cfc9",
            name: "placement-priority",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: [],
            numberOfClusters: 1,
            numberOfSelectedClusters: 1,
            prioritizerPolicy: {
              mode: "Additive",
              configurations: [
                {
                  scoreCoordinate: {
                    type: "BuiltIn",
                    builtIn: "ResourceAllocatableMemory"
                  },
                  weight: 1
                }
              ]
            },
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-priority-decision-1"],
                clusterCount: 1
              }
            ],
            conditions: priorityConditions,
            succeeded: determineSucceededStatus(priorityConditions),
            selectedClusters: [
              {
                id: "mock-cluster-2",
                name: "mock-cluster-2",
                status: "Online",
              }
            ],
          });
        } else if (actualName === "placement-tolerations" && actualNamespace === "default") {
          const tolerationsConditions = [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T13:42:49Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T13:51:37Z"
              }
            ];

          resolve({
            id: "0d39b430-8a78-46e1-b6fc-62b091196703",
            name: "placement-tolerations",
            namespace: "default",
            creationTimestamp: "2025-05-20T13:42:49Z",
            clusterSets: [],
            numberOfSelectedClusters: 2,
            tolerations: [
              {
                key: "gpu",
                value: "true",
                operator: "Equal",
                tolerationSeconds: 300
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["placement-tolerations-decision-1"],
                clusterCount: 2
              }
            ],
            conditions: tolerationsConditions,
            succeeded: determineSucceededStatus(tolerationsConditions),
            selectedClusters: [
              {
                id: "mock-cluster-3",
                name: "mock-cluster-3",
                status: "Online",
              },
              {
                id: "mock-cluster-4",
                name: "mock-cluster-4",
                status: "Online",
              }
            ],
          });
        } else if (actualName === "global" && actualNamespace === "open-cluster-management-addon") {
          const globalConditions = [
              {
                type: "PlacementMisconfigured",
                status: "False",
                reason: "Succeedconfigured",
                message: "Placement configurations check pass",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              },
              {
                type: "PlacementSatisfied",
                status: "True",
                reason: "AllDecisionsScheduled",
                message: "All cluster decisions scheduled",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              }
            ];

          resolve({
            id: "bcfd62a3-43a3-4717-9fce-3455631bbe82",
            name: "global",
            namespace: "open-cluster-management-addon",
            creationTimestamp: "2025-05-20T08:52:35Z",
            clusterSets: ["global"],
            numberOfSelectedClusters: 2,
            tolerations: [
              {
                key: "cluster.open-cluster-management.io/unreachable",
                operator: "Equal"
              },
              {
                key: "cluster.open-cluster-management.io/unavailable",
                operator: "Equal"
              }
            ],
            decisionGroups: [
              {
                decisionGroupIndex: 0,
                decisionGroupName: "",
                decisions: ["global-decision-1"],
                clusterCount: 2
              }
            ],
            conditions: globalConditions,
            succeeded: determineSucceededStatus(globalConditions),
            selectedClusters: [
              {
                id: "mock-cluster-5",
                name: "mock-cluster-5",
                status: "Online",
              },
              {
                id: "mock-cluster-6",
                name: "mock-cluster-6",
                status: "Online",
              }
            ],
          });
        } else {
          resolve(null);
        }
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${actualNamespace}/placements/${actualName}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    const placement = await response.json();

    // Add succeeded status based on the PlacementSatisfied condition
    return {
      ...placement,
      succeeded: determineSucceededStatus(placement.conditions)
    };
  } catch (error) {
    console.error(`Error fetching placement ${actualNamespace}/${actualName}:`, error);
    return null;
  }
};

// Fetch placement decisions for a placement
export const fetchPlacementDecisions = async (
  namespace: string,
  placementName: string
): Promise<PlacementDecision[]> => {
  console.log(`API fetchPlacementDecisions called with namespace=${namespace}, placementName=${placementName}`);

  // Handle the case where URL parameters might include namespace
  let actualNamespace = namespace;
  let actualName = placementName;

  // Check if placementName contains a slash, which means it might be in "namespace/name" format
  // But avoid re-parsing if it has already been processed in the component
  if (!namespace.includes('_PARSED_') && placementName && placementName.includes('/')) {
    const parts = placementName.split('/');
    if (parts.length === 2) {
      actualNamespace = parts[0];
      actualName = parts[1];
      console.log(`Parsed name from URL: namespace=${actualNamespace}, name=${actualName}`);
    }
  }

  // Remove marker
  if (actualNamespace.includes('_PARSED_')) {
    actualNamespace = actualNamespace.replace('_PARSED_', '');
  }

  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        console.log(`Checking mock decisions for ${actualNamespace}/${actualName}`);

        if (actualName === "placement-label-claim" && actualNamespace === "default") {
          console.log(`Returning mock decisions for ${actualNamespace}/${actualName}`);
          resolve([
            {
              name: "placement-label-claim-decision-1",
              namespace: "default",
              decisions: [
                {
                  clusterName: "mock-cluster-1",
                  reason: "Selected by placement"
                }
              ]
            }
          ]);
        } else if (actualName === "placement-priority" && actualNamespace === "default") {
          resolve([
            {
              name: "placement-priority-decision-1",
              namespace: "default",
              decisions: [
                {
                  clusterName: "mock-cluster-2",
                  reason: "Selected by placement (highest ResourceAllocatableMemory)"
                }
              ]
            }
          ]);
        } else if (actualName === "placement-tolerations" && actualNamespace === "default") {
          resolve([
            {
              name: "placement-tolerations-decision-1",
              namespace: "default",
              decisions: [
                {
                  clusterName: "mock-cluster-3",
                  reason: "Selected by placement"
                },
                {
                  clusterName: "mock-cluster-4",
                  reason: "Selected by placement"
                }
              ]
            }
          ]);
        } else if (actualName === "global" && actualNamespace === "open-cluster-management-addon") {
          resolve([
            {
              name: "global-decision-1",
              namespace: "open-cluster-management-addon",
              decisions: [
                {
                  clusterName: "mock-cluster-5",
                  reason: "Selected by placement"
                },
                {
                  clusterName: "mock-cluster-6",
                  reason: "Selected by placement"
                }
              ]
            }
          ]);
        } else {
          resolve([]);
        }
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${actualNamespace}/placements/${actualName}/placementdecisions`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching placement decisions for ${actualNamespace}/${actualName}:`, error);
    return [];
  }
};

// Fetch all placement decisions across all namespaces
export const fetchAllPlacementDecisions = async (): Promise<PlacementDecision[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            name: "placement-label-claim-decision-1",
            namespace: "default",
            decisions: [
              {
                clusterName: "mock-cluster-1",
                reason: "Selected by placement"
              }
            ]
          },
          {
            name: "placement-priority-decision-1",
            namespace: "default",
            decisions: [
              {
                clusterName: "mock-cluster-2",
                reason: "Selected by placement (highest ResourceAllocatableMemory)"
              }
            ]
          },
          {
            name: "placement-tolerations-decision-1",
            namespace: "default",
            decisions: [
              {
                clusterName: "mock-cluster-3",
                reason: "Selected by placement"
              },
              {
                clusterName: "mock-cluster-4",
                reason: "Selected by placement"
              }
            ]
          },
          {
            name: "global-decision-1",
            namespace: "open-cluster-management-addon",
            decisions: [
              {
                clusterName: "mock-cluster-5",
                reason: "Selected by placement"
              },
              {
                clusterName: "mock-cluster-6",
                reason: "Selected by placement"
              }
            ]
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/placementdecisions`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching all placement decisions:', error);
    return [];
  }
};

// Fetch placement decisions by namespace
export const fetchPlacementDecisionsByNamespace = async (namespace: string): Promise<PlacementDecision[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    // Filter mock data by namespace
    const allDecisions = await fetchAllPlacementDecisions();
    return allDecisions.filter(decision => decision.namespace === namespace);
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/placementdecisions`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching placement decisions for namespace ${namespace}:`, error);
    return [];
  }
};

// Fetch a specific placement decision by name and namespace
export const fetchPlacementDecisionByName = async (
  namespace: string,
  name: string
): Promise<PlacementDecision | null> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    const allDecisions = await fetchAllPlacementDecisions();
    return allDecisions.find(
      decision => decision.namespace === namespace && decision.name === name
    ) || null;
  }

  try {
    const response = await fetch(`${API_BASE}/api/namespaces/${namespace}/placementdecisions/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching placement decision ${namespace}/${name}:`, error);
    return null;
  }
};