import { useCallback, useEffect, useRef } from 'react';
import { useNodesInitialized, useReactFlow, type Edge } from '@xyflow/react';
import dagre from '@dagrejs/dagre';

export interface AutoLayoutOptions {
  direction?: 'TB' | 'LR';
  nodeSpacing?: number;
  rankSpacing?: number;
}

/**
 * Applies dagre directed-graph layout to ReactFlow nodes once they have been
 * rendered and measured. Call `requestLayout()` after setting new nodes/edges
 * to trigger a layout pass on the next measurement cycle.
 *
 * Must be used inside a `<ReactFlowProvider>`.
 */
export function useAutoLayout(edges: Edge[], options: AutoLayoutOptions = {}) {
  const { direction = 'LR', nodeSpacing = 30, rankSpacing = 60 } = options;
  const { getNodes, setNodes, fitView } = useReactFlow();
  const nodesInitialized = useNodesInitialized();
  const pendingLayout = useRef(true);

  useEffect(() => {
    if (!nodesInitialized || !pendingLayout.current) return;

    const nodes = getNodes();
    if (nodes.length === 0) return;
    if (nodes.some(n => !n.measured?.width || !n.measured?.height)) return;

    const g = new dagre.graphlib.Graph();
    g.setDefaultEdgeLabel(() => ({}));
    g.setGraph({ rankdir: direction, nodesep: nodeSpacing, ranksep: rankSpacing });

    for (const node of nodes) {
      g.setNode(node.id, { width: node.measured!.width!, height: node.measured!.height! });
    }
    for (const edge of edges) {
      g.setEdge(edge.source, edge.target);
    }

    dagre.layout(g);

    setNodes(nodes.map(node => {
      const pos = g.node(node.id);
      const w = node.measured!.width!;
      const h = node.measured!.height!;
      return { ...node, position: { x: pos.x - w / 2, y: pos.y - h / 2 } };
    }));

    pendingLayout.current = false;
    // Allow fitView to run after the position update renders
    requestAnimationFrame(() => fitView({ padding: 0.25 }));
  }, [nodesInitialized, edges, getNodes, setNodes, fitView, direction, nodeSpacing, rankSpacing]);

  const requestLayout = useCallback(() => {
    pendingLayout.current = true;
  }, []);

  return { requestLayout };
}
