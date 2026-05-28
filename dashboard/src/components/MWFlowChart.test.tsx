import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { ManifestWork } from '../api/manifestWorkService';

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}));

// Mock @xyflow/react — provide minimal stubs so the component tree renders
vi.mock('@xyflow/react', () => ({
  ReactFlow: ({ nodes, edges }: { nodes: unknown[]; edges: unknown[] }) => (
    <div data-testid="reactflow" data-nodes={nodes.length} data-edges={edges.length} />
  ),
  ReactFlowProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  Background: () => null,
  Controls: () => null,
  Handle: () => null,
  Position: { Left: 'left', Right: 'right' },
  MarkerType: { ArrowClosed: 'arrowclosed' },
  useNodesInitialized: () => false,
  useReactFlow: () => ({ getNodes: () => [], setNodes: vi.fn(), fitView: vi.fn() }),
}));

// Mock useAutoLayout since it depends on ReactFlow context + dagre
vi.mock('../hooks/useAutoLayout', () => ({
  useAutoLayout: () => ({ requestLayout: vi.fn() }),
}));

// Import AFTER mocks are set up
import MWFlowChart from './MWFlowChart';

function makeMW(overrides: Partial<ManifestWork> = {}): ManifestWork {
  return {
    id: 'test-id',
    name: 'test-mw',
    namespace: 'cluster1',
    conditions: [{ type: 'Applied', status: 'True' }],
    ...overrides,
  } as ManifestWork;
}

describe('MWFlowChart', () => {
  it('shows fallback when no resourceStatus', () => {
    render(<MWFlowChart mw={makeMW()} />);
    expect(screen.getByText('No resource status available')).toBeInTheDocument();
  });

  it('shows fallback when resourceStatus has empty manifests', () => {
    render(<MWFlowChart mw={makeMW({ resourceStatus: { manifests: [] } })} />);
    expect(screen.getByText('No resource status available')).toBeInTheDocument();
  });

  it('renders ReactFlow when resourceStatus has manifests', () => {
    const mw = makeMW({
      resourceStatus: {
        manifests: [
          {
            resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx', namespace: 'default' },
            conditions: [{ type: 'Applied', status: 'True' }],
          },
        ],
      },
    });
    render(<MWFlowChart mw={mw} />);
    const flow = screen.getByTestId('reactflow');
    expect(flow).toBeInTheDocument();
    // 1 MW node + 1 resource node = 2 nodes
    expect(flow.getAttribute('data-nodes')).toBe('2');
    // 1 edge from MW to resource
    expect(flow.getAttribute('data-edges')).toBe('1');
  });

  it('creates correct node/edge counts for multiple resources', () => {
    const mw = makeMW({
      resourceStatus: {
        manifests: [
          {
            resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx', namespace: 'default' },
            conditions: [{ type: 'Applied', status: 'True' }],
          },
          {
            resourceMeta: { ordinal: 1, kind: 'Service', name: 'nginx-svc', namespace: 'default' },
            conditions: [{ type: 'Applied', status: 'True' }],
          },
          {
            resourceMeta: { ordinal: 2, kind: 'ConfigMap', name: 'config', namespace: 'default' },
            conditions: [],
          },
        ],
      },
    });
    render(<MWFlowChart mw={mw} />);
    const flow = screen.getByTestId('reactflow');
    // 1 MW node + 3 resource nodes = 4
    expect(flow.getAttribute('data-nodes')).toBe('4');
    // 3 edges from MW to each resource
    expect(flow.getAttribute('data-edges')).toBe('3');
  });

  it('does not render ReactFlow fallback text when resources exist', () => {
    const mw = makeMW({
      resourceStatus: {
        manifests: [
          {
            resourceMeta: { ordinal: 0, kind: 'Deployment', name: 'nginx' },
            conditions: [],
          },
        ],
      },
    });
    render(<MWFlowChart mw={mw} />);
    expect(screen.queryByText('No resource status available')).not.toBeInTheDocument();
  });
});
