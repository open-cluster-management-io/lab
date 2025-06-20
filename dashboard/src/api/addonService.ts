import { createHeaders } from './utils';

export interface ManagedClusterAddon {
  id: string;
  name: string;
  namespace: string;
  installNamespace: string;
  creationTimestamp?: string;
  conditions?: {
    type: string;
    status: string;
    reason?: string;
    message?: string;
    lastTransitionTime?: string;
  }[];
  registrations?: {
    signerName: string;
    subject: {
      groups: string[];
      user: string;
    };
  }[];
  supportedConfigs?: {
    group: string;
    resource: string;
  }[];
}

// Backend API base URL - configurable for production
// In production, use relative path so requests go through the same host/ingress
const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

// Fetch all addons for a specific cluster
export const fetchClusterAddons = async (clusterName: string): Promise<ManagedClusterAddon[]> => {
  // Use mock data in development mode unless specifically requested to use real API
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        resolve([
          {
            id: `${clusterName}-managed-serviceaccount`,
            name: 'managed-serviceaccount',
            namespace: clusterName,
            installNamespace: 'open-cluster-management-agent-addon',
            creationTimestamp: "2025-05-20T08:52:35Z",
            conditions: [
              {
                type: "Progressing",
                status: "False",
                reason: "Completed",
                message: "completed with no errors.",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              },
              {
                type: "Configured",
                status: "True",
                reason: "ConfigurationsConfigured",
                message: "Configurations configured",
                lastTransitionTime: "2025-05-20T08:52:35Z"
              },
              {
                type: "Available",
                status: "True",
                reason: "ManagedClusterAddOnLeaseUpdated",
                message: "managed-serviceaccount add-on is available.",
                lastTransitionTime: "2025-05-20T08:53:15Z"
              },
              {
                type: "RegistrationApplied",
                status: "True",
                reason: "SetPermissionApplied",
                message: "Registration of the addon agent is configured",
                lastTransitionTime: "2025-05-20T08:52:59Z"
              },
              {
                type: "ClusterCertificateRotated",
                status: "True",
                reason: "ClientCertificateUpdated",
                message: "client certificate rotated starting from 2025-05-20 08:47:59 +0000 UTC to 2026-05-20 08:47:59 +0000 UTC",
                lastTransitionTime: "2025-05-20T08:52:59Z"
              },
              {
                type: "ManifestApplied",
                status: "True",
                reason: "AddonManifestApplied",
                message: "manifests of addon are applied successfully",
                lastTransitionTime: "2025-05-20T08:52:59Z"
              }
            ],
            registrations: [
              {
                signerName: "kubernetes.io/kube-apiserver-client",
                subject: {
                  groups: [
                    `system:open-cluster-management:cluster:${clusterName}:addon:managed-serviceaccount`,
                    "system:open-cluster-management:addon:managed-serviceaccount",
                    "system:authenticated"
                  ],
                  user: `system:open-cluster-management:cluster:${clusterName}:addon:managed-serviceaccount:agent:addon-agent`
                }
              }
            ],
            supportedConfigs: [
              {
                group: "addon.open-cluster-management.io",
                resource: "addondeploymentconfigs"
              }
            ]
          },
          {
            id: `${clusterName}-application-manager`,
            name: 'application-manager',
            namespace: clusterName,
            installNamespace: 'open-cluster-management-agent-addon',
            creationTimestamp: "2025-05-20T08:52:35Z",
            conditions: [
              {
                type: "Available",
                status: "True",
                reason: "ManagedClusterAddOnLeaseUpdated",
                message: "application-manager add-on is available.",
                lastTransitionTime: "2025-05-20T08:53:15Z"
              }
            ]
          },
          {
            id: `${clusterName}-cert-policy-controller`,
            name: 'cert-policy-controller',
            namespace: clusterName,
            installNamespace: 'open-cluster-management-agent-addon',
            creationTimestamp: "2025-05-20T08:52:35Z",
            conditions: [
              {
                type: "Available",
                status: "True",
                reason: "ManagedClusterAddOnLeaseUpdated",
                message: "cert-policy-controller add-on is available.",
                lastTransitionTime: "2025-05-20T08:53:15Z"
              }
            ]
          }
        ]);
      }, 800);
    });
  }

  try {
    const response = await fetch(`${API_BASE}/api/clusters/${clusterName}/addons`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching addons for cluster ${clusterName}:`, error);
    return [];
  }
};

// Fetch a single addon by name for a specific cluster
export const fetchClusterAddonByName = async (clusterName: string, addonName: string): Promise<ManagedClusterAddon | null> => {
  try {
    const response = await fetch(`${API_BASE}/api/clusters/${clusterName}/addons/${addonName}`, {
      headers: createHeaders()
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error fetching addon ${addonName} for cluster ${clusterName}:`, error);
    return null;
  }
};