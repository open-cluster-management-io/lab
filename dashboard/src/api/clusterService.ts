export interface Cluster {
  id: string;
  name: string;
  status: string; // "Online" or "Offline" based on ManagedClusterConditionAvailable
  version?: string; // Kubernetes version from status.version.kubernetes
  conditions?: {
    type: string;
    status: string;
    reason?: string;
    message?: string;
    lastTransitionTime?: string;
  }[];
  labels?: Record<string, string>;
  hubAccepted?: boolean; // Based on spec.hubAcceptsClient or HubAcceptedManagedCluster condition
  capacity?: Record<string, string>; // From status.capacity
  allocatable?: Record<string, string>; // From status.allocatable
  clusterClaims?: { // From status.clusterClaims
    name: string;
    value: string;
  }[];
  managedClusterClientConfigs?: {
    url: string;
    caBundle?: string;
  }[]; // From spec.managedClusterClientConfigs
  taints?: {
    key: string;
    value?: string;
    effect: string;
  }[];
  creationTimestamp?: string; // From metadata.creationTimestamp
  // Addon info for list page
  addonCount?: number;
  addonNames?: string[];
  // Removing nodes field as it's not available in ManagedCluster
}

// Make sure we also export a type to avoid compiler issues
export type { Cluster as ClusterType };

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Import the shared header creation function
import { createHeaders } from './utils';

// Fetch all clusters
export const fetchClusters = async (): Promise<Cluster[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: "mock-cluster-1",
            name: "mock-cluster-1",
            status: "Online",
            version: "4.12.0",
            hubAccepted: true,
            creationTimestamp: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
            labels: {
              vendor: "OpenShift",
              region: "us-east-1",
              env: "development",
              tier: "gold"
            },
            clusterClaims: [
              {
                name: "usage",
                value: "dev"
              },
              {
                name: "platform.open-cluster-management.io",
                value: "AWS"
              },
              {
                name: "product.open-cluster-management.io",
                value: "OpenShift"
              }
            ],
            managedClusterClientConfigs: [
              {
                url: "https://cluster1-control-plane:6443",
                caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJWEZtWkR0bjdXM2N3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBMU1UUXdPVEk1TWpoYUZ3MHpOVEExTVRJd09UTTBNamhhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUUM2N0FXYSt2b1FQaE8xd05xUXdncjZxT0tuWW1hOWNTT0NCMHFTVW1VQUh0T29wSG1LWXArNzFMR1kKT0RXODB3M1FnMUJkTWw5Y0h1UVBjK043MTJsbzQwVVJMcDVCOEhoR2ZiZWlZOVhlWWZIYkRMdWpaV2tSaHI0agpOckNUcWRCN1JUYmhSY1NPKyszVVlGRG8ybVpSdmVBbGFyc25ldXJFNW5LL2RITU1Xb0hYL1VUcXBhc2RaTTZaCkVJaVNseldGUVYxWnpjTVBNVmZ4WjhlT1FWZjVqdHY4NnNhOTc1aFFhOG1WYXh6QTdjTzdiNTJYM200cXhuUWwKK1Voa1dTSC9GWXlEdE9vd3NFSDYvd25LRWY1Y3NiWFpJK2RGQ3EwWjU1b0JrbGcyMDlhSEJPOGUzYm1lZWE1dwpYQ1NBd2JpWm1wM0p1a203ODN5dkRyUWZodTRGQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJSd3NlVXh4cHNvOE1qNlZ4Wnl4RDUyYVU5K1pEQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQVVYZFFPN2NSMQpJWUhXVkxEZ0JFUTdJRUJqcjYrSS9MbCt0bzF1STZiQ3o0dmxmMEJ6ZnBaQllCQmFxdzM5dERtaGcwUys5ZnEvClFyL1ZMUHlLeUpuOC9zdmQzbjUzRy9pNC9HM2JGcVc4azc3M3hSK3hkV21TcnAybEFnRGFEU0cxZVlUUEZFN3UKZTQ4T01WcGNRaHNEbmRZY2ExNnJ6LzZ5WlpONkxiY0dXbUV6bEtxN1EyamVsaGNwZnpSWjlqMGJxRTRNSmg1Rgo5cEY2encyMnNKd2pvanhVQzMyVHNGczN4bndMMDRuUDREcHM2TkJBbTFXWmlxbTJJSDJHWHh4SVVFbVpOUWZmCmxET3hIdGJONU5yR2xRVUdrWDJQdUpyOXdFS0lOSWtYaHlYS0tvRngzUFg3d2VSWnB6TWZOR2UrU0JUVkFjTmkKbi9IMjdRODF0L3orCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
              }
            ],
            capacity: {
              cpu: "12",
              memory: "32Gi"
            },
            allocatable: {
              cpu: "10",
              memory: "28Gi"
            },

            conditions: [
              {
                type: "ManagedClusterConditionAvailable",
                status: "True",
                reason: "ClusterAvailable",
                message: "Cluster is available",
                lastTransitionTime: new Date().toISOString()
              },
              {
                type: "ManagedClusterJoined",
                status: "True",
                reason: "ClusterJoined",
                message: "Cluster has joined the hub",
                lastTransitionTime: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString()
              }
            ]
          },
          {
            id: "mock-cluster-2",
            name: "mock-cluster-2",
            status: "Offline",
            version: "4.11.0",
            hubAccepted: true,
            creationTimestamp: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
            labels: {
              vendor: "OpenShift",
              region: "us-west-1",
              env: "staging",
              tier: "silver"
            },
            clusterClaims: [
              {
                name: "usage",
                value: "staging"
              },
              {
                name: "platform.open-cluster-management.io",
                value: "GCP"
              },
              {
                name: "product.open-cluster-management.io",
                value: "OpenShift"
              }
            ],
            managedClusterClientConfigs: [
              {
                url: "https://cluster2-control-plane:6443",
                caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJWnhLblFMVFovaG93RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBMU1UUXdPVEk1TXpsYUZ3MHpOVEExTVRJd09UTTBNemxhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURndFVaM0JTT3pNWGZWZ3hZM3dpSGh5UGlqVU1Jb3JvYmRaY2FldDlLTnBqcU9RRHloQ05tTzAya1QKeGFkT1RtY0dJMmtPeDNvUE9PRGorWkd3cndXNjdtV0dTeTVHTGI5SlJJc1VydWZ4Rkt3cHk1L291dzBZU3lUVwphMkVNTmp1TS9TYmxHdE5lZHRaRkRVYXY5K015ejU2ZjBEZm1XdlRGNlNudEJLOGNLNEdYUXlPOGFzaC8xL1hOClRYQ2IxbjJldmllUlRiclp3aTR0d2kyQmFBVlc0dTArWmU0TWJaU3h1U01rL2t1UG02TXhVZUdHSXpUY1F2RXUKdjBzSDVPejRqeXRLbGsyR2Z1SXVwSXNQbGVrVWN5dS9wZnpvY0hmZlNpMVpla3YyNW1CMzlWN256TXZONWRjNAo2VXhPbjBjZGIvMU1xenhCVENjL0dDSGR1OHJaQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJTSDNmVWdXdG9UTEhYK2ZKUmZScnhuNVViUFd6QVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQ1hHd1Fzcjdpbgo3aXlqL3VCZTVPOTR6NVJMck0vZWR4U1M4ZkFIMzJrR2t6d0lzOFdoZUZJVHZuTC96UzBUY0Q4cll0Z3dmSThvCkE2WE1PaGxFVlJML0trQldFR2xLN0dyV0gva2orcjdpRjdTN2FoMzdRQUFSeTlCcGhPc1U1eERyaFAzN2gyMlYKeUVnQjhiWDJJcHJXdEwxZDhTeEVVRHFPMlV3a1VaVmIyK1RtV0lCMnpsT01CU0hjQ016VVNESWx4WTdPSzZXNgpTQ3djSmdtek1uWDFnMUQyZXRGM0p4eW5PU2k4VEoyejRLbFlZQk9tQ01uTHovaWIwVjNHMTNkRVVZamt0YXdxCmp2bHJuM2x5OVNDUThsZUdzNmVLTW4xYUNZZ2dpeUl6MllMbW45bEhHSUhJNU05Y0o1Z2lXcDlPVHM5MUt6d3EKb3FFcVRxaFBHZnhxCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
              }
            ],
            capacity: {
              cpu: "24",
              memory: "64Gi"
            },
            allocatable: {
              cpu: "20",
              memory: "56Gi"
            },

            conditions: [
              {
                type: "ManagedClusterConditionAvailable",
                status: "False",
                reason: "ClusterOffline",
                message: "Cluster is not responding",
                lastTransitionTime: new Date().toISOString()
              }
            ],
            taints: [
              {
                key: "cluster.open-cluster-management.io/unavailable",
                effect: "NoSelect"
              }
            ]
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clusters`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching clusters:', error);
    return [];
  }
};

// Fetch a single cluster by name
export const fetchClusterByName = async (name: string): Promise<Cluster | null> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        if (name === "mock-cluster-1") {
          resolve({
            id: "mock-cluster-1",
            name: "mock-cluster-1",
            status: "Online",
            version: "4.12.0",
            hubAccepted: true,
            creationTimestamp: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
            labels: {
              vendor: "OpenShift",
              region: "us-east-1",
              env: "development",
              tier: "gold"
            },
            clusterClaims: [
              {
                name: "usage",
                value: "dev"
              },
              {
                name: "platform.open-cluster-management.io",
                value: "AWS"
              },
              {
                name: "product.open-cluster-management.io",
                value: "OpenShift"
              }
            ],
            managedClusterClientConfigs: [
              {
                url: "https://cluster1-control-plane:6443",
                caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJWEZtWkR0bjdXM2N3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBMU1UUXdPVEk1TWpoYUZ3MHpOVEExTVRJd09UTTBNamhhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUUM2N0FXYSt2b1FQaE8xd05xUXdncjZxT0tuWW1hOWNTT0NCMHFTVW1VQUh0T29wSG1LWXArNzFMR1kKT0RXODB3M1FnMUJkTWw5Y0h1UVBjK043MTJsbzQwVVJMcDVCOEhoR2ZiZWlZOVhlWWZIYkRMdWpaV2tSaHI0agpOckNUcWRCN1JUYmhSY1NPKyszVVlGRG8ybVpSdmVBbGFyc25ldXJFNW5LL2RITU1Xb0hYL1VUcXBhc2RaTTZaCkVJaVNseldGUVYxWnpjTVBNVmZ4WjhlT1FWZjVqdHY4NnNhOTc1aFFhOG1WYXh6QTdjTzdiNTJYM200cXhuUWwKK1Voa1dTSC9GWXlEdE9vd3NFSDYvd25LRWY1Y3NiWFpJK2RGQ3EwWjU1b0JrbGcyMDlhSEJPOGUzYm1lZWE1dwpYQ1NBd2JpWm1wM0p1a203ODN5dkRyUWZodTRGQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJSd3NlVXh4cHNvOE1qNlZ4Wnl4RDUyYVU5K1pEQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQVVYZFFPN2NSMQpJWUhXVkxEZ0JFUTdJRUJqcjYrSS9MbCt0bzF1STZiQ3o0dmxmMEJ6ZnBaQllCQmFxdzM5dERtaGcwUys5ZnEvClFyL1ZMUHlLeUpuOC9zdmQzbjUzRy9pNC9HM2JGcVc4azc3M3hSK3hkV21TcnAybEFnRGFEU0cxZVlUUEZFN3UKZTQ4T01WcGNRaHNEbmRZY2ExNnJ6LzZ5WlpONkxiY0dXbUV6bEtxN1EyamVsaGNwZnpSWjlqMGJxRTRNSmg1Rgo5cEY2encyMnNKd2pvanhVQzMyVHNGczN4bndMMDRuUDREcHM2TkJBbTFXWmlxbTJJSDJHWHh4SVVFbVpOUWZmCmxET3hIdGJONU5yR2xRVUdrWDJQdUpyOXdFS0lOSWtYaHlYS0tvRngzUFg3d2VSWnB6TWZOR2UrU0JUVkFjTmkKbi9IMjdRODF0L3orCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
              }
            ],
            capacity: {
              cpu: "12",
              memory: "32Gi"
            },
            allocatable: {
              cpu: "10",
              memory: "28Gi"
            },

            conditions: [
              {
                type: "ManagedClusterConditionAvailable",
                status: "True",
                reason: "ClusterAvailable",
                message: "Cluster is available",
                lastTransitionTime: new Date().toISOString()
              },
              {
                type: "ManagedClusterJoined",
                status: "True",
                reason: "ClusterJoined",
                message: "Cluster has joined the hub",
                lastTransitionTime: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString()
              },
              {
                type: "HubAcceptedManagedCluster",
                status: "True",
                reason: "HubClusterAdminAccepted",
                message: "Cluster has been accepted by hub",
                lastTransitionTime: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString()
              }
            ]
          });
        } else if (name === "mock-cluster-2") {
          resolve({
            id: "mock-cluster-2",
            name: "mock-cluster-2",
            status: "Offline",
            version: "4.11.0",
            hubAccepted: true,
            creationTimestamp: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
            labels: {
              vendor: "OpenShift",
              region: "us-west-1",
              env: "staging",
              tier: "silver"
            },
            clusterClaims: [
              {
                name: "usage",
                value: "staging"
              },
              {
                name: "platform.open-cluster-management.io",
                value: "GCP"
              },
              {
                name: "product.open-cluster-management.io",
                value: "OpenShift"
              }
            ],
            managedClusterClientConfigs: [
              {
                url: "https://cluster2-control-plane:6443",
                caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJWnhLblFMVFovaG93RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBMU1UUXdPVEk1TXpsYUZ3MHpOVEExTVRJd09UTTBNemxhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURndFVaM0JTT3pNWGZWZ3hZM3dpSGh5UGlqVU1Jb3JvYmRaY2FldDlLTnBqcU9RRHloQ05tTzAya1QKeGFkT1RtY0dJMmtPeDNvUE9PRGorWkd3cndXNjdtV0dTeTVHTGI5SlJJc1VydWZ4Rkt3cHk1L291dzBZU3lUVwphMkVNTmp1TS9TYmxHdE5lZHRaRkRVYXY5K015ejU2ZjBEZm1XdlRGNlNudEJLOGNLNEdYUXlPOGFzaC8xL1hOClRYQ2IxbjJldmllUlRiclp3aTR0d2kyQmFBVlc0dTArWmU0TWJaU3h1U01rL2t1UG02TXhVZUdHSXpUY1F2RXUKdjBzSDVPejRqeXRLbGsyR2Z1SXVwSXNQbGVrVWN5dS9wZnpvY0hmZlNpMVpla3YyNW1CMzlWN256TXZONWRjNAo2VXhPbjBjZGIvMU1xenhCVENjL0dDSGR1OHJaQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJTSDNmVWdXdG9UTEhYK2ZKUmZScnhuNVViUFd6QVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQ1hHd1Fzcjdpbgo3aXlqL3VCZTVPOTR6NVJMck0vZWR4U1M4ZkFIMzJrR2t6d0lzOFdoZUZJVHZuTC96UzBUY0Q4cll0Z3dmSThvCkE2WE1PaGxFVlJML0trQldFR2xLN0dyV0gva2orcjdpRjdTN2FoMzdRQUFSeTlCcGhPc1U1eERyaFAzN2gyMlYKeUVnQjhiWDJJcHJXdEwxZDhTeEVVRHFPMlV3a1VaVmIyK1RtV0lCMnpsT01CU0hjQ016VVNESWx4WTdPSzZXNgpTQ3djSmdtek1uWDFnMUQyZXRGM0p4eW5PU2k4VEoyejRLbFlZQk9tQ01uTHovaWIwVjNHMTNkRVVZamt0YXdxCmp2bHJuM2x5OVNDUThsZUdzNmVLTW4xYUNZZ2dpeUl6MllMbW45bEhHSUhJNU05Y0o1Z2lXcDlPVHM5MUt6d3EKb3FFcVRxaFBHZnhxCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"
              }
            ],
            capacity: {
              cpu: "24",
              memory: "64Gi"
            },
            allocatable: {
              cpu: "20",
              memory: "56Gi"
            },

            conditions: [
              {
                type: "ManagedClusterConditionAvailable",
                status: "False",
                reason: "ClusterOffline",
                message: "Cluster is not responding",
                lastTransitionTime: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
              },
              {
                type: "ManagedClusterJoined",
                status: "True",
                reason: "ClusterJoined",
                message: "Cluster has joined the hub",
                lastTransitionTime: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString()
              }
            ],
            taints: [
              {
                key: "cluster.open-cluster-management.io/unavailable",
                effect: "NoSelect"
              }
            ]
          });
        } else {
          resolve(null);
        }
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clusters/${name}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching cluster ${name}:`, error);
    return null;
  }
};

// SSE for real-time cluster updates
export const setupClusterEventSource = (
  onAdd: (cluster: Cluster) => void,
  onUpdate: (cluster: Cluster) => void,
  onDelete: (clusterId: string) => void,
  onError: (error: Event) => void
): () => void => {
  // Use no-op in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return () => {}; // Return no-op cleanup function
  }

  // Create EventSource for SSE
  const token = localStorage.getItem('authToken');
  // Extract the token part without 'Bearer ' prefix for URL parameter
  const tokenParam = token ? token.replace('Bearer ', '') : '';
  const eventSource = new EventSource(
    `${API_BASE}/api/stream/clusters${tokenParam ? `?token=${tokenParam}` : ''}`
  );

  // Set up event listeners
  eventSource.addEventListener('ADDED', (event) => {
    const cluster = JSON.parse(event.data);
    onAdd(cluster);
  });

  eventSource.addEventListener('MODIFIED', (event) => {
    const cluster = JSON.parse(event.data);
    onUpdate(cluster);
  });

  eventSource.addEventListener('DELETED', (event) => {
    const clusterId = JSON.parse(event.data).id;
    onDelete(clusterId);
  });

  eventSource.addEventListener('error', onError);

  // Return cleanup function to close the SSE connection
  return () => {
    eventSource.close();
  };
};