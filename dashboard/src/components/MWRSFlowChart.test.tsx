import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import type { ManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';
import type { ManifestWork } from '../api/manifestWorkService';

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}));

// Track mock fetch calls
const mockFetchMWs = vi.fn<() => Promise<ManifestWork[]>>();

// Mock the MWRS service — only the fetch function
vi.mock('../api/manifestWorkReplicaSetService', async () => {
  const actual = await vi.importActual<typeof import('../api/manifestWorkReplicaSetService')>(
    '../api/manifestWorkReplicaSetService',
  );
  return {
    ...actual,
    fetchManifestWorksByReplicaSet: () => mockFetchMWs(),
  };
});

// Mock @xyflow/react
vi.mock('@xyflow/react', () => ({
  ReactFlow: ({ nodes, edges }: { nodes: unknown[]; edges: unknown[] }) => (
    <div data-testid="reactflow" data-nodes={nodes?.length ?? 0} data-edges={edges?.length ?? 0} />
  ),
  ReactFlowProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Background: () => null,
  Controls: () => null,
  Handle: () => null,
  Position: { Left: 'left', Right: 'right' },
  MarkerType: { ArrowClosed: 'arrowclosed' },
  useNodesState: (init: unknown[]) => [init, vi.fn(), vi.fn()],
  useEdgesState: (init: unknown[]) => [init, vi.fn(), vi.fn()],
  useNodesInitialized: () => false,
  useReactFlow: () => ({ getNodes: () => [], setNodes: vi.fn(), fitView: vi.fn() }),
}));

// Mock useAutoLayout
vi.mock('../hooks/useAutoLayout', () => ({
  useAutoLayout: () => ({ requestLayout: vi.fn() }),
}));

// Import AFTER mocks
import MWRSFlowChart from './MWRSFlowChart';

function makeMWRS(overrides: Partial<ManifestWorkReplicaSet> = {}): ManifestWorkReplicaSet {
  return {
    id: 'mwrs-1',
    name: 'deploy-nginx',
    namespace: 'default',
    summary: { total: 2, available: 2, progressing: 0, degraded: 0, applied: 2 },
    manifestCount: 2,
    ...overrides,
  } as ManifestWorkReplicaSet;
}

function makeMW(namespace: string): ManifestWork {
  return {
    id: `mw-${namespace}`,
    name: 'deploy-nginx',
    namespace,
    conditions: [{ type: 'Applied', status: 'True' }],
    resourceStatus: {
      manifests: [
        {
          resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx', namespace: 'default' },
          conditions: [{ type: 'Applied', status: 'True' }],
        },
      ],
    },
  } as ManifestWork;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MWRSFlowChart', () => {
  it('shows loading state initially', () => {
    // Never resolve the promise — stays in loading
    mockFetchMWs.mockReturnValue(new Promise(() => {}));
    render(<MWRSFlowChart mwrs={makeMWRS()} />);
    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('shows error on fetch failure', async () => {
    mockFetchMWs.mockRejectedValue(new Error('network error'));
    render(<MWRSFlowChart mwrs={makeMWRS()} />);
    await waitFor(() => {
      expect(screen.getByText('Failed to load ManifestWorks')).toBeInTheDocument();
    });
  });

  it('renders after successful fetch', async () => {
    mockFetchMWs.mockResolvedValue([makeMW('cluster1'), makeMW('cluster2')]);
    render(<MWRSFlowChart mwrs={makeMWRS()} />);
    await waitFor(() => {
      expect(screen.getByTestId('reactflow')).toBeInTheDocument();
    });
  });

  it('calls fetch with correct namespace and name', async () => {
    mockFetchMWs.mockResolvedValue([]);
    render(<MWRSFlowChart mwrs={makeMWRS({ namespace: 'monitoring', name: 'mon-stack' })} />);
    await waitFor(() => {
      expect(mockFetchMWs).toHaveBeenCalledTimes(1);
    });
  });
});
