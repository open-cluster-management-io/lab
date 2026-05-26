import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  Handle,
  Position,
  MarkerType,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type NodeTypes,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { Box, Typography, Chip, CircularProgress, Alert } from '@mui/material';
import type { ManifestWorkReplicaSet } from '../api/manifestWorkReplicaSetService';
import { fetchManifestWorksByReplicaSet } from '../api/manifestWorkReplicaSetService';
import type { ManifestWork, StatusFeedbackResult } from '../api/manifestWorkService';
import StatusFeedbackDisplay from './StatusFeedbackDisplay';
import { useAutoLayout } from '../hooks/useAutoLayout';
import { borderColor, chipColor, deriveMWStatus, deriveResStatus } from '../utils/statusHelpers';

// ── Custom node components ──────────────────────────────────────────────────

type MWRSData = { name: string; namespace: string; status: string };

function MWRSNode({ data }: { data: MWRSData }) {
  return (
    <Box sx={{ background: '#fff', border: '2px solid #1976d2', borderRadius: 2, p: 1.5, minWidth: 210, boxShadow: 2 }}>
      <Handle type="source" position={Position.Right} isConnectable={false} style={{ background: '#1976d2' }} />
      <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.25 }}>
        ManifestWorkReplicaSet
      </Typography>
      <Typography variant="body2" fontWeight="bold">{data.name}</Typography>
      <Typography variant="caption" color="text.secondary" display="block">{data.namespace}</Typography>
      <Chip label={data.status} size="small" color={chipColor(data.status)} sx={{ mt: 0.75 }} />
    </Box>
  );
}

type MWData = { cluster: string; status: string };

function ManifestWorkNode({ data }: { data: MWData }) {
  const color = borderColor(data.status);
  return (
    <Box sx={{ background: '#fff', border: '1px solid #e0e0e0', borderLeft: `4px solid ${color}`, borderRadius: '0 8px 8px 0', p: 1.5, minWidth: 190, boxShadow: 1 }}>
      <Handle type="target" position={Position.Left} isConnectable={false} style={{ background: color }} />
      <Handle type="source" position={Position.Right} isConnectable={false} style={{ background: color }} />
      <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.25 }}>
        ManifestWork
      </Typography>
      <Typography variant="body2" fontWeight="bold">{data.cluster}</Typography>
      <Chip label={data.status} size="small" color={chipColor(data.status)} sx={{ mt: 0.75 }} />
    </Box>
  );
}

type ResData = { kind: string; name: string; namespace?: string; status: string; path: string; feedback?: StatusFeedbackResult };

function ResourceNode({ data }: { data: ResData }) {
  const navigate = useNavigate();
  const color = borderColor(data.status);
  return (
    <Box
      sx={{
        background: '#fff',
        border: '1px solid #e0e0e0',
        borderLeft: `4px solid ${color}`,
        borderRadius: '0 8px 8px 0',
        p: 1.25,
        minWidth: 170,
        boxShadow: 1,
        cursor: 'pointer',
        '&:hover': { boxShadow: 4, borderColor: color },
      }}
      onClick={() => navigate(data.path)}
    >
      <Handle type="target" position={Position.Left} isConnectable={false} style={{ background: color }} />
      <Typography variant="caption" sx={{ color: 'primary.main', fontWeight: 'bold' }} display="block">
        {data.kind}
      </Typography>
      <Typography variant="body2" noWrap title={data.name}>{data.name}</Typography>
      {data.namespace && (
        <Typography variant="caption" color="text.secondary" display="block">{data.namespace}</Typography>
      )}
      <Chip label={data.status} size="small" color={chipColor(data.status)} sx={{ mt: 0.75 }} />
      {data.feedback?.values?.length && (
        <Box sx={{ mt: 0.5 }}>
          <StatusFeedbackDisplay feedback={data.feedback} variant="inline" />
        </Box>
      )}
    </Box>
  );
}

const nodeTypes: NodeTypes = {
  mwrsNode: MWRSNode as never,
  manifestWorkNode: ManifestWorkNode as never,
  resourceNode: ResourceNode as never,
};

// ── Graph builder ─────────────────────────────────────────────────────────────

function buildGraph(mwrs: ManifestWorkReplicaSet, manifestWorks: ManifestWork[]) {
  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const edgeStyle = { stroke: '#bdbdbd' };
  const marker = { type: MarkerType.ArrowClosed, color: '#bdbdbd' };

  // Derive MWRS status: check if any child MW is degraded
  const mwrsCond = mwrs.conditions?.find(c => c.type === 'ManifestworkApplied');
  let mwrsStatus: string;
  if (mwrsCond?.status !== 'True') {
    mwrsStatus = mwrsCond?.reason === 'Processing' ? 'Progressing' : 'Pending';
  } else {
    const anyDegraded = manifestWorks.some(mw => deriveMWStatus(mw) === 'Degraded');
    mwrsStatus = anyDegraded ? 'Degraded' : 'Applied';
  }

  nodes.push({
    id: 'mwrs',
    type: 'mwrsNode',
    position: { x: 0, y: 0 },
    data: { name: mwrs.name, namespace: mwrs.namespace, status: mwrsStatus },
  });

  for (const mw of manifestWorks) {
    const mwId = `mw-${mw.namespace}`;
    nodes.push({
      id: mwId,
      type: 'manifestWorkNode',
      position: { x: 0, y: 0 },
      data: { cluster: mw.namespace, status: deriveMWStatus(mw) },
    });
    edges.push({
      id: `e-mwrs-${mw.namespace}`,
      source: 'mwrs',
      target: mwId,
      type: 'smoothstep',
      markerEnd: marker,
      style: edgeStyle,
    });

    const resources = mw.resourceStatus?.manifests ?? [];
    for (const res of resources) {
      const nodeId = `res-${mw.namespace}-${res.resourceMeta.ordinal}`;
      nodes.push({
        id: nodeId,
        type: 'resourceNode',
        position: { x: 0, y: 0 },
        data: {
          kind: res.resourceMeta.kind ?? 'Resource',
          name: res.resourceMeta.name ?? '-',
          namespace: res.resourceMeta.namespace,
          status: deriveResStatus(res.conditions ?? [], res.statusFeedback),
          path: `/resources/${mw.namespace}/${mw.name}/${res.resourceMeta.ordinal}`,
          feedback: res.statusFeedback,
        },
      });
      edges.push({
        id: `e-${mw.namespace}-${nodeId}`,
        source: mwId,
        target: nodeId,
        type: 'smoothstep',
        markerEnd: marker,
        style: edgeStyle,
      });
    }
  }

  return { nodes, edges };
}

// ── Inner component (needs ReactFlowProvider context) ─────────────────────────

interface InnerProps {
  mwrs: ManifestWorkReplicaSet;
}

function MWRSFlowChartInner({ mwrs }: InnerProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { requestLayout } = useAutoLayout(edges, { direction: 'LR', nodeSpacing: 30, rankSpacing: 80 });

  useEffect(() => {
    setIsLoading(true);
    setError(null);
    fetchManifestWorksByReplicaSet(mwrs.namespace, mwrs.name)
      .then((mws) => {
        const { nodes: n, edges: e } = buildGraph(mwrs, mws);
        setNodes(n);
        setEdges(e);
        requestLayout();
      })
      .catch(() => setError('Failed to load ManifestWorks'))
      .finally(() => setIsLoading(false));
  }, [mwrs.namespace, mwrs.name]);

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.25 }}
      nodesConnectable={false}
      deleteKeyCode={null}
    >
      <Background color="#e0e0e0" gap={20} />
      <Controls />
    </ReactFlow>
  );
}

// ── Component ──────────────────────────────────────────────────────────────────

interface Props {
  mwrs: ManifestWorkReplicaSet;
}

export default function MWRSFlowChart({ mwrs }: Props) {
  return (
    <Box sx={{ height: 'calc(100vh - 320px)', minHeight: 420, border: '1px solid', borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
      <ReactFlowProvider>
        <MWRSFlowChartInner mwrs={mwrs} />
      </ReactFlowProvider>
    </Box>
  );
}
