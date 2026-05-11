import { K8s } from '@kinvolk/headlamp-plugin/lib';

export type OCMContext =
  | { type: 'loading' }
  | { type: 'hub' }
  | { type: 'managed' }
  | { type: 'none' };

export function useOCMContext(): OCMContext {
  const [crds] = K8s.ResourceClasses.CustomResourceDefinition.useList();

  if (!crds) return { type: 'loading' };

  const crdNames = new Set(crds.map((crd: { metadata: { name: string } }) => crd.metadata.name));

  if (crdNames.has('managedclusters.cluster.open-cluster-management.io')) {
    return { type: 'hub' };
  }
  if (crdNames.has('klusterlets.operator.open-cluster-management.io')) {
    return { type: 'managed' };
  }
  return { type: 'none' };
}

export function useOCMInstalled(): boolean | null {
  const ctx = useOCMContext();
  if (ctx.type === 'loading') return null;
  return ctx.type === 'hub';
}

export function getConditionStatus(
  conditions: Array<{ type: string; status: string; message?: string; reason?: string }> | undefined,
  conditionType: string
): { status: string; message: string; reason: string } | null {
  if (!conditions) return null;
  const condition = conditions.find(c => c.type === conditionType);
  if (!condition) return null;
  return {
    status: condition.status,
    message: condition.message ?? '',
    reason: condition.reason ?? '',
  };
}
