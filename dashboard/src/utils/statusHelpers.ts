import type { ManifestWork, StatusFeedbackResult } from '../api/manifestWorkService';

/**
 * Status color for node borders. Maps derived status to a hex color.
 */
export const borderColor = (status: string) => {
  if (status === 'Applied' || status === 'Available') return '#4caf50';
  if (status === 'Degraded' || status === 'Progressing' || status === 'Pending') return '#ff9800';
  if (status === 'Failed') return '#f44336';
  return '#9e9e9e';
};

/**
 * MUI Chip color for status badges.
 */
export const chipColor = (status: string): 'success' | 'warning' | 'error' | 'default' => {
  if (status === 'Applied' || status === 'Available') return 'success';
  if (status === 'Degraded' || status === 'Progressing' || status === 'Pending') return 'warning';
  if (status === 'Failed') return 'error';
  return 'default';
};

/**
 * Derive ManifestWork status from its conditions.
 * If any child resource reports degraded feedback, the MW is Degraded.
 */
export const deriveMWStatus = (mw: ManifestWork): string => {
  const applied = mw.conditions?.find(c => c.type === 'Applied');
  if (applied?.status === 'False') return 'Failed';

  // Check if any resource has feedback indicating degraded health
  const hasDegraded = mw.resourceStatus?.manifests?.some(
    m => isFeedbackDegraded(m.statusFeedback),
  );

  if (applied?.status === 'True') return hasDegraded ? 'Degraded' : 'Applied';

  const available = mw.conditions?.find(c => c.type === 'Available');
  if (available?.status === 'True') return hasDegraded ? 'Degraded' : 'Available';

  return 'Pending';
};

/**
 * Derive resource status from OCM conditions and StatusFeedback.
 *
 * OCM's Applied condition reflects whether the manifest was applied to the spoke
 * cluster. OCM also supports ConditionRules (CEL/WellKnownConditions) for
 * Progressing/Degraded/Complete evaluation, which gate MWRS progressive rollout.
 *
 * StatusFeedback values (via FeedbackRules) provide additional observability into
 * workload health. The dashboard uses these to detect degraded workloads — e.g.,
 * when a Deployment is Applied but has Available=False or ReadyReplicas < Replicas.
 */
export const deriveResStatus = (
  conditions: { type: string; status: string }[],
  feedback?: StatusFeedbackResult,
): string => {
  if (!conditions?.length) return 'Pending';
  const applied = conditions.find(c => c.type === 'Applied');
  if (applied?.status === 'False') return 'Failed';

  if (applied?.status === 'True') {
    return isFeedbackDegraded(feedback) ? 'Degraded' : 'Applied';
  }

  return 'Pending';
};

/**
 * Derive MWRS status from its OCM conditions and child ManifestWorks' feedback.
 *
 * OCM's MWRS conditions (ManifestworkApplied) track whether child ManifestWorks
 * were applied. OCM also supports ConditionRules for rollout gating via
 * WorkProgressing/WorkDegraded conditions. The dashboard additionally checks
 * StatusFeedback from child ManifestWorks to surface degraded workload health.
 */
export const deriveMWRSStatus = (
  mwrs: { conditions?: { type: string; status: string; reason?: string }[] },
  childManifestWorks?: ManifestWork[],
): string => {
  const cond = mwrs.conditions?.find(c => c.type === 'ManifestworkApplied');
  if (!cond || cond.status === 'False') return 'Failed';
  if (cond.reason === 'Processing') return 'Progressing';

  if (cond.status === 'True') {
    const anyDegraded = childManifestWorks?.some(mw => deriveMWStatus(mw) === 'Degraded');
    return anyDegraded ? 'Degraded' : 'Applied';
  }

  return 'Pending';
};

/**
 * OCM label used to identify which MWRS created a ManifestWork.
 */
const MWRS_LABEL = 'work.open-cluster-management.io/manifestworkreplicaset';

/**
 * Build a lookup from "namespace/name" MWRS key to its child ManifestWorks.
 */
export function buildMWRSChildMap(manifestWorks: ManifestWork[]): Map<string, ManifestWork[]> {
  const map = new Map<string, ManifestWork[]>();
  for (const mw of manifestWorks) {
    const labelValue = mw.labels?.[MWRS_LABEL];
    if (!labelValue) continue;
    // Label format is "namespace.name" — convert to "namespace/name" for lookup
    const dotIdx = labelValue.indexOf('.');
    if (dotIdx === -1) continue;
    const key = `${labelValue.substring(0, dotIdx)}/${labelValue.substring(dotIdx + 1)}`;
    const existing = map.get(key);
    if (existing) {
      existing.push(mw);
    } else {
      map.set(key, [mw]);
    }
  }
  return map;
}

/**
 * Extract a human-readable degraded reason from StatusFeedback.
 * Returns the most informative message available, or undefined if not degraded.
 */
export function getDegradedReason(feedback?: StatusFeedbackResult): string | undefined {
  if (!isFeedbackDegraded(feedback)) return undefined;
  const values = feedback!.values!;

  // Prefer the most specific message available (JSONPaths pattern)
  const progressingMsg = values.find(v => v.name === 'ProgressingMessage')?.fieldValue.string;
  const availableMsg = values.find(v => v.name === 'AvailableMessage')?.fieldValue.string;
  const progressingReason = values.find(v => v.name === 'ProgressingReason')?.fieldValue.string;

  if (progressingMsg || availableMsg || progressingReason) {
    return progressingMsg || availableMsg || progressingReason;
  }

  // WellKnownStatus — Deployment: ReadyReplicas vs Replicas
  const replicas = values.find(v => v.name === 'Replicas')?.fieldValue.integer;
  const ready = values.find(v => v.name === 'ReadyReplicas')?.fieldValue.integer;
  if (replicas != null && ready != null && ready < replicas) {
    return `${ready}/${replicas} replicas ready`;
  }

  // WellKnownStatus — DaemonSet: NumberReady vs DesiredNumberScheduled
  const desired = values.find(v => v.name === 'DesiredNumberScheduled')?.fieldValue.integer;
  const numReady = values.find(v => v.name === 'NumberReady')?.fieldValue.integer;
  if (desired != null && numReady != null && numReady < desired) {
    return `${numReady}/${desired} pods ready`;
  }

  // WellKnownStatus — Job: explicit failure
  const jobComplete = values.find(v => v.name === 'JobComplete')?.fieldValue.string;
  if (jobComplete === 'False') {
    return 'Job has not completed';
  }

  // WellKnownStatus — Pod: not ready or failed phase
  const podPhase = values.find(v => v.name === 'PodPhase')?.fieldValue.string;
  const podReady = values.find(v => v.name === 'PodReady')?.fieldValue.string;
  if (podPhase === 'Failed') {
    return 'Pod failed';
  }
  if (podReady === 'False') {
    return `Pod not ready (phase: ${podPhase ?? 'Unknown'})`;
  }

  return 'Workload is not available';
}

/**
 * Collect all degraded reasons across a ManifestWork's resources.
 * Returns an array of { resource, reason } objects.
 */
export function getMWDegradedReasons(mw: ManifestWork): { resource: string; reason: string }[] {
  const results: { resource: string; reason: string }[] = [];
  for (const m of mw.resourceStatus?.manifests ?? []) {
    const reason = getDegradedReason(m.statusFeedback);
    if (reason) {
      const kind = m.resourceMeta.kind ?? 'Resource';
      const name = m.resourceMeta.name ?? '';
      results.push({ resource: `${kind}/${name}`, reason });
    }
  }
  return results;
}

/**
 * Check whether StatusFeedback values indicate a degraded workload.
 *
 * Supports both JSONPaths and WellKnownStatus patterns for the four resource
 * types OCM defines WellKnownStatus rules for:
 *   - Deployment: ReadyReplicas < Replicas
 *   - DaemonSet:  NumberReady < DesiredNumberScheduled
 *   - Job:        JobComplete = "False"
 *   - Pod:        PodReady = "False" or PodPhase = "Failed"
 *
 * See open-cluster-management-io/ocm pkg/work/spoke/statusfeedback/rules/rule.go
 */
function isFeedbackDegraded(feedback?: StatusFeedbackResult): boolean {
  if (!feedback?.values?.length) return false;

  // JSONPaths pattern: explicit Available condition
  const available = feedback.values.find(v => v.name === 'Available');
  if (available?.fieldValue.type === 'String' && available.fieldValue.string === 'False') {
    return true;
  }

  // WellKnownStatus — Deployment: ReadyReplicas < Replicas
  const replicas = feedback.values.find(v => v.name === 'Replicas');
  const readyReplicas = feedback.values.find(v => v.name === 'ReadyReplicas');
  if (replicas?.fieldValue.type === 'Integer' && replicas.fieldValue.integer != null) {
    const ready = readyReplicas?.fieldValue.type === 'Integer' ? (readyReplicas.fieldValue.integer ?? 0) : 0;
    if (ready < replicas.fieldValue.integer) return true;
  }

  // WellKnownStatus — DaemonSet: NumberReady < DesiredNumberScheduled
  const desired = feedback.values.find(v => v.name === 'DesiredNumberScheduled');
  const numReady = feedback.values.find(v => v.name === 'NumberReady');
  if (desired?.fieldValue.type === 'Integer' && desired.fieldValue.integer != null) {
    const ready = numReady?.fieldValue.type === 'Integer' ? (numReady.fieldValue.integer ?? 0) : 0;
    if (ready < desired.fieldValue.integer) return true;
  }

  // WellKnownStatus — Job: JobComplete = "False"
  const jobComplete = feedback.values.find(v => v.name === 'JobComplete');
  if (jobComplete?.fieldValue.type === 'String' && jobComplete.fieldValue.string === 'False') {
    return true;
  }

  // WellKnownStatus — Pod: PodReady = "False" or PodPhase = "Failed"
  const podReady = feedback.values.find(v => v.name === 'PodReady');
  if (podReady?.fieldValue.type === 'String' && podReady.fieldValue.string === 'False') {
    return true;
  }
  const podPhase = feedback.values.find(v => v.name === 'PodPhase');
  if (podPhase?.fieldValue.type === 'String' && podPhase.fieldValue.string === 'Failed') {
    return true;
  }

  return false;
}
