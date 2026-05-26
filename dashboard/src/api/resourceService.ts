import { createHeaders } from './utils';
import type { Condition, StatusFeedbackResult } from './manifestWorkService';

export type { Condition };

export interface ManagedResource {
  id: string;
  kind: string;
  apiVersion: string;
  name: string;
  namespace?: string;
  cluster: string;
  manifestWorkName: string;
  ordinal: number;
  status: string; // "Applied" | "Available" | "Pending" | "Failed"
  conditions?: Condition[];
  statusFeedback?: StatusFeedbackResult;
  rawResource?: Record<string, unknown>;
}

export interface ManagedResourceList {
  resources: ManagedResource[];
  availableKinds: string[];
  clusters: string[];
  namespaces: string[];
}

export interface ResourceFilters {
  kind?: string;
  cluster?: string;
  namespace?: string;
}

const API_BASE = import.meta.env.VITE_API_BASE || (import.meta.env.PROD ? '' : 'http://localhost:8080');

const MOCK_RESOURCES: ManagedResource[] = [
  {
    id: 'cluster1/deploy-nginx/0',
    kind: 'Deployment',
    apiVersion: 'apps/v1',
    name: 'nginx',
    namespace: 'default',
    cluster: 'cluster1',
    manifestWorkName: 'deploy-nginx',
    ordinal: 0,
    status: 'Applied',
    conditions: [{ type: 'Applied', status: 'True', reason: 'AppliedManifestComplete', message: 'Resource applied', lastTransitionTime: new Date().toISOString() }],
    statusFeedback: { values: [{ name: 'readyReplicas', fieldValue: { type: 'Integer', integer: 2 } }, { name: 'replicas', fieldValue: { type: 'Integer', integer: 2 } }] },
    rawResource: { apiVersion: 'apps/v1', kind: 'Deployment', metadata: { name: 'nginx', namespace: 'default' }, spec: { replicas: 2, selector: { matchLabels: { app: 'nginx' } }, template: { metadata: { labels: { app: 'nginx' } }, spec: { containers: [{ name: 'nginx', image: 'nginx:1.21', ports: [{ containerPort: 80 }] }] } } } },
  },
  {
    id: 'cluster1/deploy-nginx/1',
    kind: 'Service',
    apiVersion: 'v1',
    name: 'nginx-svc',
    namespace: 'default',
    cluster: 'cluster1',
    manifestWorkName: 'deploy-nginx',
    ordinal: 1,
    status: 'Applied',
    conditions: [{ type: 'Applied', status: 'True', reason: 'AppliedManifestComplete', message: 'Resource applied', lastTransitionTime: new Date().toISOString() }],
    statusFeedback: { values: [{ name: 'clusterIP', fieldValue: { type: 'String', string: '10.96.0.42' } }] },
    rawResource: { apiVersion: 'v1', kind: 'Service', metadata: { name: 'nginx-svc', namespace: 'default' }, spec: { selector: { app: 'nginx' }, ports: [{ port: 80, targetPort: 80 }] } },
  },
  {
    id: 'cluster2/deploy-nginx/0',
    kind: 'Deployment',
    apiVersion: 'apps/v1',
    name: 'nginx',
    namespace: 'default',
    cluster: 'cluster2',
    manifestWorkName: 'deploy-nginx',
    ordinal: 0,
    status: 'Applied',
    conditions: [{ type: 'Applied', status: 'True', reason: 'AppliedManifestComplete', message: 'Resource applied', lastTransitionTime: new Date().toISOString() }],
    rawResource: { apiVersion: 'apps/v1', kind: 'Deployment', metadata: { name: 'nginx', namespace: 'default' }, spec: { replicas: 2, selector: { matchLabels: { app: 'nginx' } } } },
  },
  {
    id: 'cluster2/deploy-nginx/1',
    kind: 'Service',
    apiVersion: 'v1',
    name: 'nginx-svc',
    namespace: 'default',
    cluster: 'cluster2',
    manifestWorkName: 'deploy-nginx',
    ordinal: 1,
    status: 'Pending',
    conditions: [],
    rawResource: { apiVersion: 'v1', kind: 'Service', metadata: { name: 'nginx-svc', namespace: 'default' }, spec: { selector: { app: 'nginx' }, ports: [{ port: 80, targetPort: 80 }] } },
  },
  {
    id: 'cluster1/monitoring-stack/0',
    kind: 'Deployment',
    apiVersion: 'apps/v1',
    name: 'prometheus',
    namespace: 'monitoring',
    cluster: 'cluster1',
    manifestWorkName: 'monitoring-stack',
    ordinal: 0,
    status: 'Applied',
    conditions: [{ type: 'Applied', status: 'True', reason: 'AppliedManifestComplete', message: 'Resource applied', lastTransitionTime: new Date().toISOString() }],
    rawResource: { apiVersion: 'apps/v1', kind: 'Deployment', metadata: { name: 'prometheus', namespace: 'monitoring' }, spec: { replicas: 1 } },
  },
  {
    id: 'cluster1/monitoring-stack/1',
    kind: 'ConfigMap',
    apiVersion: 'v1',
    name: 'prometheus-config',
    namespace: 'monitoring',
    cluster: 'cluster1',
    manifestWorkName: 'monitoring-stack',
    ordinal: 1,
    status: 'Applied',
    conditions: [{ type: 'Applied', status: 'True', reason: 'AppliedManifestComplete', message: 'Resource applied', lastTransitionTime: new Date().toISOString() }],
    rawResource: { apiVersion: 'v1', kind: 'ConfigMap', metadata: { name: 'prometheus-config', namespace: 'monitoring' }, data: { 'prometheus.yml': 'global:\n  scrape_interval: 15s' } },
  },
  {
    id: 'cluster2/monitoring-stack/0',
    kind: 'Deployment',
    apiVersion: 'apps/v1',
    name: 'prometheus',
    namespace: 'monitoring',
    cluster: 'cluster2',
    manifestWorkName: 'monitoring-stack',
    ordinal: 0,
    status: 'Failed',
    conditions: [{ type: 'Applied', status: 'False', reason: 'AppliedManifestFailed', message: 'Failed to apply resource', lastTransitionTime: new Date().toISOString() }],
    rawResource: { apiVersion: 'apps/v1', kind: 'Deployment', metadata: { name: 'prometheus', namespace: 'monitoring' }, spec: { replicas: 1 } },
  },
];

export const fetchManagedResources = async (filters?: ResourceFilters): Promise<ManagedResourceList> => {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    return new Promise((resolve) => {
      setTimeout(() => {
        let resources = MOCK_RESOURCES;
        if (filters?.kind) resources = resources.filter(r => r.kind === filters.kind);
        if (filters?.cluster) resources = resources.filter(r => r.cluster === filters.cluster);
        if (filters?.namespace) resources = resources.filter(r => r.namespace === filters.namespace);

        const availableKinds = [...new Set(MOCK_RESOURCES.map(r => r.kind))].sort();
        const clusters = [...new Set(MOCK_RESOURCES.map(r => r.cluster))].sort();
        const namespaces = [...new Set(MOCK_RESOURCES.map(r => r.namespace).filter(Boolean) as string[])].sort();

        resolve({ resources, availableKinds, clusters, namespaces });
      }, 500);
    });
  }

  const params = new URLSearchParams();
  if (filters?.kind) params.set('kind', filters.kind);
  if (filters?.cluster) params.set('cluster', filters.cluster);
  if (filters?.namespace) params.set('namespace', filters.namespace);

  try {
    const url = `${API_BASE}/api/resources${params.toString() ? '?' + params.toString() : ''}`;
    const response = await fetch(url, { headers: createHeaders() });
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return await response.json();
  } catch (error) {
    console.error('Error fetching managed resources:', error);
    return { resources: [], availableKinds: [], clusters: [], namespaces: [] };
  }
};

export const fetchManagedResource = async (
  cluster: string,
  manifestwork: string,
  ordinal: number
): Promise<ManagedResource | null> => {
  if (import.meta.env.DEV && !import.meta.env.VITE_USE_REAL_API) {
    const id = `${cluster}/${manifestwork}/${ordinal}`;
    return new Promise((resolve) => {
      setTimeout(() => resolve(MOCK_RESOURCES.find(r => r.id === id) || null), 300);
    });
  }

  try {
    const response = await fetch(
      `${API_BASE}/api/resources/${cluster}/${manifestwork}/${ordinal}`,
      { headers: createHeaders() }
    );
    if (!response.ok) throw new Error(`API error: ${response.status}`);
    return await response.json();
  } catch (error) {
    console.error(`Error fetching resource ${cluster}/${manifestwork}/${ordinal}:`, error);
    return null;
  }
};
