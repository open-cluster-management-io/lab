import { describe, it, expect } from 'vitest';
import type { ManifestWork, StatusFeedbackResult } from '../api/manifestWorkService';
import {
  borderColor,
  chipColor,
  deriveMWStatus,
  deriveResStatus,
  deriveMWRSStatus,
  buildMWRSChildMap,
  getDegradedReason,
  getMWDegradedReasons,
} from './statusHelpers';

// --- Helpers to build test data ---

function fb(values: StatusFeedbackResult['values']): StatusFeedbackResult {
  return { values };
}

function intVal(name: string, value: number) {
  return { name, fieldValue: { type: 'Integer' as const, integer: value } };
}

function strVal(name: string, value: string) {
  return { name, fieldValue: { type: 'String' as const, string: value } };
}

function makeMW(overrides: Partial<ManifestWork> = {}): ManifestWork {
  return {
    id: 'test-id',
    name: 'test-mw',
    namespace: 'cluster1',
    ...overrides,
  } as ManifestWork;
}

// --- borderColor ---

describe('borderColor', () => {
  it('returns green for Applied', () => {
    expect(borderColor('Applied')).toBe('#4caf50');
  });

  it('returns green for Available', () => {
    expect(borderColor('Available')).toBe('#4caf50');
  });

  it('returns orange for Degraded', () => {
    expect(borderColor('Degraded')).toBe('#ff9800');
  });

  it('returns orange for Progressing', () => {
    expect(borderColor('Progressing')).toBe('#ff9800');
  });

  it('returns orange for Pending', () => {
    expect(borderColor('Pending')).toBe('#ff9800');
  });

  it('returns red for Failed', () => {
    expect(borderColor('Failed')).toBe('#f44336');
  });

  it('returns grey for unknown status', () => {
    expect(borderColor('Something')).toBe('#9e9e9e');
  });
});

// --- chipColor ---

describe('chipColor', () => {
  it('returns success for Applied/Available', () => {
    expect(chipColor('Applied')).toBe('success');
    expect(chipColor('Available')).toBe('success');
  });

  it('returns warning for Degraded/Progressing/Pending', () => {
    expect(chipColor('Degraded')).toBe('warning');
    expect(chipColor('Progressing')).toBe('warning');
    expect(chipColor('Pending')).toBe('warning');
  });

  it('returns error for Failed', () => {
    expect(chipColor('Failed')).toBe('error');
  });

  it('returns default for unknown', () => {
    expect(chipColor('Other')).toBe('default');
  });
});

// --- deriveMWStatus ---

describe('deriveMWStatus', () => {
  it('returns Failed when Applied is False', () => {
    const mw = makeMW({ conditions: [{ type: 'Applied', status: 'False' }] });
    expect(deriveMWStatus(mw)).toBe('Failed');
  });

  it('returns Applied when Applied is True and no degraded feedback', () => {
    const mw = makeMW({
      conditions: [{ type: 'Applied', status: 'True' }],
      resourceStatus: { manifests: [] },
    });
    expect(deriveMWStatus(mw)).toBe('Applied');
  });

  it('returns Degraded when Applied is True but feedback is degraded', () => {
    const mw = makeMW({
      conditions: [{ type: 'Applied', status: 'True' }],
      resourceStatus: {
        manifests: [{
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
          conditions: [],
          statusFeedback: fb([strVal('Available', 'False')]),
        }],
      },
    });
    expect(deriveMWStatus(mw)).toBe('Degraded');
  });

  it('returns Available when Available is True and no Applied', () => {
    const mw = makeMW({
      conditions: [{ type: 'Available', status: 'True' }],
      resourceStatus: { manifests: [] },
    });
    expect(deriveMWStatus(mw)).toBe('Available');
  });

  it('returns Degraded when Available is True but feedback is degraded', () => {
    const mw = makeMW({
      conditions: [{ type: 'Available', status: 'True' }],
      resourceStatus: {
        manifests: [{
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
          conditions: [],
          statusFeedback: fb([intVal('ReadyReplicas', 0), intVal('Replicas', 2)]),
        }],
      },
    });
    expect(deriveMWStatus(mw)).toBe('Degraded');
  });

  it('returns Pending when no conditions', () => {
    expect(deriveMWStatus(makeMW())).toBe('Pending');
  });
});

// --- deriveResStatus ---

describe('deriveResStatus', () => {
  it('returns Pending for empty conditions', () => {
    expect(deriveResStatus([])).toBe('Pending');
  });

  it('returns Failed when Applied is False', () => {
    expect(deriveResStatus([{ type: 'Applied', status: 'False' }])).toBe('Failed');
  });

  it('returns Applied when Applied is True and no feedback', () => {
    expect(deriveResStatus([{ type: 'Applied', status: 'True' }])).toBe('Applied');
  });

  it('returns Degraded when Applied is True but feedback is degraded', () => {
    const feedback = fb([strVal('Available', 'False')]);
    expect(deriveResStatus([{ type: 'Applied', status: 'True' }], feedback)).toBe('Degraded');
  });

  it('returns Applied when Applied is True and feedback is healthy', () => {
    const feedback = fb([strVal('Available', 'True')]);
    expect(deriveResStatus([{ type: 'Applied', status: 'True' }], feedback)).toBe('Applied');
  });

  it('returns Pending for conditions without Applied', () => {
    expect(deriveResStatus([{ type: 'Other', status: 'True' }])).toBe('Pending');
  });
});

// --- deriveMWRSStatus ---

describe('deriveMWRSStatus', () => {
  it('returns Failed when ManifestworkApplied is missing', () => {
    expect(deriveMWRSStatus({ conditions: [] })).toBe('Failed');
  });

  it('returns Failed when ManifestworkApplied is False', () => {
    expect(deriveMWRSStatus({
      conditions: [{ type: 'ManifestworkApplied', status: 'False' }],
    })).toBe('Failed');
  });

  it('returns Progressing when reason is Processing', () => {
    expect(deriveMWRSStatus({
      conditions: [{ type: 'ManifestworkApplied', status: 'True', reason: 'Processing' }],
    })).toBe('Progressing');
  });

  it('returns Applied when ManifestworkApplied is True and no degraded children', () => {
    expect(deriveMWRSStatus({
      conditions: [{ type: 'ManifestworkApplied', status: 'True' }],
    })).toBe('Applied');
  });

  it('returns Degraded when ManifestworkApplied is True but child MW is degraded', () => {
    const degradedMW = makeMW({
      conditions: [{ type: 'Applied', status: 'True' }],
      resourceStatus: {
        manifests: [{
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
          conditions: [],
          statusFeedback: fb([strVal('Available', 'False')]),
        }],
      },
    });
    expect(deriveMWRSStatus(
      { conditions: [{ type: 'ManifestworkApplied', status: 'True' }] },
      [degradedMW],
    )).toBe('Degraded');
  });

  it('returns Applied when child MWs are all healthy', () => {
    const healthyMW = makeMW({
      conditions: [{ type: 'Applied', status: 'True' }],
      resourceStatus: {
        manifests: [{
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
          conditions: [],
          statusFeedback: fb([intVal('ReadyReplicas', 2), intVal('Replicas', 2)]),
        }],
      },
    });
    expect(deriveMWRSStatus(
      { conditions: [{ type: 'ManifestworkApplied', status: 'True' }] },
      [healthyMW],
    )).toBe('Applied');
  });
});

// --- buildMWRSChildMap ---

describe('buildMWRSChildMap', () => {
  const MWRS_LABEL = 'work.open-cluster-management.io/manifestworkreplicaset';

  it('builds lookup from MWRS label', () => {
    const mw1 = makeMW({ name: 'mw1', labels: { [MWRS_LABEL]: 'default.deploy-nginx' } });
    const mw2 = makeMW({ name: 'mw2', labels: { [MWRS_LABEL]: 'default.deploy-nginx' } });
    const mw3 = makeMW({ name: 'mw3', labels: { [MWRS_LABEL]: 'monitoring.mon-config' } });

    const map = buildMWRSChildMap([mw1, mw2, mw3]);
    expect(map.get('default/deploy-nginx')).toHaveLength(2);
    expect(map.get('monitoring/mon-config')).toHaveLength(1);
  });

  it('skips ManifestWorks without the label', () => {
    const mw = makeMW({ name: 'orphan', labels: {} });
    const map = buildMWRSChildMap([mw]);
    expect(map.size).toBe(0);
  });

  it('skips labels without a dot separator', () => {
    const mw = makeMW({ name: 'bad', labels: { [MWRS_LABEL]: 'no-dot' } });
    const map = buildMWRSChildMap([mw]);
    expect(map.size).toBe(0);
  });

  it('returns empty map for empty input', () => {
    expect(buildMWRSChildMap([]).size).toBe(0);
  });
});

// --- isFeedbackDegraded (tested via getDegradedReason / deriveResStatus) ---

describe('degraded detection', () => {
  describe('JSONPaths pattern', () => {
    it('detects Available=False', () => {
      const feedback = fb([strVal('Available', 'False')]);
      expect(getDegradedReason(feedback)).toBeDefined();
    });

    it('not degraded when Available=True', () => {
      const feedback = fb([strVal('Available', 'True')]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });
  });

  describe('WellKnownStatus — Deployment', () => {
    it('detects ReadyReplicas < Replicas', () => {
      const feedback = fb([intVal('ReadyReplicas', 1), intVal('Replicas', 3)]);
      expect(getDegradedReason(feedback)).toBe('1/3 replicas ready');
    });

    it('not degraded when ReadyReplicas == Replicas', () => {
      const feedback = fb([intVal('ReadyReplicas', 2), intVal('Replicas', 2)]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });

    it('detects zero ready replicas', () => {
      const feedback = fb([intVal('ReadyReplicas', 0), intVal('Replicas', 2)]);
      expect(getDegradedReason(feedback)).toBe('0/2 replicas ready');
    });
  });

  describe('WellKnownStatus — DaemonSet', () => {
    it('detects NumberReady < DesiredNumberScheduled', () => {
      const feedback = fb([intVal('NumberReady', 1), intVal('DesiredNumberScheduled', 3)]);
      expect(getDegradedReason(feedback)).toBe('1/3 pods ready');
    });

    it('not degraded when NumberReady == DesiredNumberScheduled', () => {
      const feedback = fb([intVal('NumberReady', 3), intVal('DesiredNumberScheduled', 3)]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });
  });

  describe('WellKnownStatus — Job', () => {
    it('detects JobComplete=False', () => {
      const feedback = fb([strVal('JobComplete', 'False'), intVal('JobSucceeded', 0)]);
      expect(getDegradedReason(feedback)).toBe('Job has not completed');
    });

    it('not degraded when JobComplete=True', () => {
      const feedback = fb([strVal('JobComplete', 'True'), intVal('JobSucceeded', 1)]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });
  });

  describe('WellKnownStatus — Pod', () => {
    it('detects PodReady=False', () => {
      const feedback = fb([strVal('PodReady', 'False'), strVal('PodPhase', 'Running')]);
      expect(getDegradedReason(feedback)).toBe('Pod not ready (phase: Running)');
    });

    it('detects PodPhase=Failed', () => {
      const feedback = fb([strVal('PodReady', 'False'), strVal('PodPhase', 'Failed')]);
      expect(getDegradedReason(feedback)).toBe('Pod failed');
    });

    it('not degraded when PodReady=True', () => {
      const feedback = fb([strVal('PodReady', 'True'), strVal('PodPhase', 'Running')]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });
  });

  describe('edge cases', () => {
    it('returns undefined for undefined feedback', () => {
      expect(getDegradedReason(undefined)).toBeUndefined();
    });

    it('returns undefined for empty values', () => {
      expect(getDegradedReason({ values: [] })).toBeUndefined();
    });

    it('returns undefined for unrelated feedback values', () => {
      const feedback = fb([strVal('clusterIP', '10.96.0.1')]);
      expect(getDegradedReason(feedback)).toBeUndefined();
    });
  });
});

// --- getDegradedReason message priority ---

describe('getDegradedReason message priority', () => {
  it('prefers ProgressingMessage over replica counts', () => {
    const feedback = fb([
      strVal('Available', 'False'),
      strVal('ProgressingMessage', 'ReplicaSet "nginx-abc" is progressing'),
      intVal('ReadyReplicas', 0),
      intVal('Replicas', 2),
    ]);
    expect(getDegradedReason(feedback)).toBe('ReplicaSet "nginx-abc" is progressing');
  });

  it('falls back to AvailableMessage', () => {
    const feedback = fb([
      strVal('Available', 'False'),
      strVal('AvailableMessage', 'Deployment does not have minimum availability'),
    ]);
    expect(getDegradedReason(feedback)).toBe('Deployment does not have minimum availability');
  });

  it('falls back to ProgressingReason', () => {
    const feedback = fb([
      strVal('Available', 'False'),
      strVal('ProgressingReason', 'ProgressDeadlineExceeded'),
    ]);
    expect(getDegradedReason(feedback)).toBe('ProgressDeadlineExceeded');
  });
});

// --- getMWDegradedReasons ---

describe('getMWDegradedReasons', () => {
  it('collects reasons from degraded resources', () => {
    const mw = makeMW({
      resourceStatus: {
        manifests: [
          {
            resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
            conditions: [],
            statusFeedback: fb([strVal('Available', 'False'), strVal('AvailableMessage', 'Not ready')]),
          },
          {
            resourceMeta: { ordinal: 1, kind: 'Service', name: 'nginx-svc' },
            conditions: [],
            statusFeedback: fb([strVal('clusterIP', '10.96.0.1')]),
          },
        ],
      },
    });

    const reasons = getMWDegradedReasons(mw);
    expect(reasons).toHaveLength(1);
    expect(reasons[0].resource).toBe('Deployment/nginx');
    expect(reasons[0].reason).toBe('Not ready');
  });

  it('returns empty array when no resources are degraded', () => {
    const mw = makeMW({
      resourceStatus: {
        manifests: [{
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
          conditions: [],
          statusFeedback: fb([intVal('ReadyReplicas', 2), intVal('Replicas', 2)]),
        }],
      },
    });
    expect(getMWDegradedReasons(mw)).toHaveLength(0);
  });

  it('returns empty array when no resourceStatus', () => {
    expect(getMWDegradedReasons(makeMW())).toHaveLength(0);
  });
});
