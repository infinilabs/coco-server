import { AimOutlined, CloseOutlined, ExportOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import ForceGraph3D from '3d-force-graph';
import { Button, Empty, Spin, Tag, Tooltip } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchWikiKbGraph } from '@/service/api';

interface GNode {
  id: string;
  label: string;
  kind: 'article' | 'entity' | 'unresolved';
  color: string;
  articleId?: string;
  type?: string;
  typeLabel?: string;
  status?: string;
  confidence?: number;
  aliases?: string[];
  properties?: Record<string, unknown>;
  viaRelation?: boolean;
}

interface GEdge {
  source: string;
  target: string;
  kind: 'wikilink' | 'relation';
  relation?: string;
  label?: string;
  inverse?: string;
  inverseLabel?: string;
}

const ARTICLE_COLOR = '#1677ff';
const UNRESOLVED_COLOR = '#bfbfbf';
const TYPE_PALETTE = ['#722ed1', '#13c2c2', '#fa8c16', '#52c41a', '#eb2f96', '#2f54eb', '#faad14', '#a0d911'];

const typeColor = (type?: string) => {
  if (!type) return TYPE_PALETTE[0];
  let hash = 0;
  for (let i = 0; i < type.length; i++) hash = (hash * 31 + type.charCodeAt(i)) >>> 0;
  return TYPE_PALETTE[hash % TYPE_PALETTE.length];
};

const STATUS_COLOR: Record<string, string> = {
  published: 'green',
  reviewed: 'blue',
  proposed: 'orange',
  draft: 'orange'
};

const GRAPH_HEIGHT = 560;

/**
 * Knowledge graph for one KB (ontology phase O3), rendered as an interactive
 * 3D force graph: articles, wikilink-resolved entities and one-hop typed
 * relations in a self-organizing layout — drag to orbit, wheel to zoom,
 * click an article to navigate, click an entity to open its card.
 */
export function KbGraph({ kbId, focusEntityId }: { kbId: string; focusEntityId?: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<GNode[]>([]);
  const [edges, setEdges] = useState<GEdge[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [cardOpen, setCardOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  // accessors read this ref — selection changes repaint without rebuilding
  const selectedRef = useRef<string | null>(null);

  const toGNode = (n: Api.Wiki.GraphNode): GNode => ({
    id: n.id,
    label: n.label,
    kind: n.kind,
    color:
      n.kind === 'article'
        ? ARTICLE_COLOR
        : n.kind === 'unresolved'
          ? UNRESOLVED_COLOR
          : typeColor(n.type),
    articleId: n.article_id,
    type: n.type,
    typeLabel: n.type_label,
    status: n.status,
    confidence: n.confidence,
    aliases: n.aliases,
    properties: n.properties,
    viaRelation: n.via_relation
  });

  // fetch the subgraph once per KB
  useEffect(() => {
    if (!kbId) return;
    setLoading(true);
    setSelected(null);
    selectedRef.current = null;
    setCardOpen(false);
    setNodes([]);
    setEdges([]);

    fetchWikiKbGraph(kbId)
      .then(({ nodes: gNodes, edges: gEdges }) => {
        setNodes(gNodes.map(toGNode));
        setEdges(
          gEdges.map(e => ({
            source: e.source,
            target: e.target,
            kind: e.kind,
            relation: e.relation,
            label: e.label,
            inverse: e.inverse,
            inverseLabel: e.inverse_label
          }))
        );
      })
      .finally(() => setLoading(false));
  }, [kbId]);

  // neighbors of the selected node stay bright, everything else dims
  const neighbors = useMemo(() => {
    if (!selected) return null;
    const set = new Set<string>([selected]);
    edges.forEach(e => {
      if (e.source === selected) set.add(e.target);
      if (e.target === selected) set.add(e.source);
    });
    return set;
  }, [selected, edges]);

  const hasNeighbor = (a: string, b: string) =>
    edges.some(e => (e.source === a && e.target === b) || (e.source === b && e.target === a));

  // build the 3D scene once data is in and the container is on screen
  useEffect(() => {
    if (loading || nodes.length === 0 || !containerRef.current || graphRef.current) return;

    let disposed = false;
    const nodeIds = new Set(nodes.map(n => n.id));
    const links = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));

    const graph = new ForceGraph3D(containerRef.current)
      .backgroundColor('rgba(0,0,0,0)')
      .width(containerRef.current.clientWidth)
      .height(GRAPH_HEIGHT)
      .graphData({ nodes, links })
      .nodeColor((n: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== n.id && !hasNeighbor(sel, n.id)) return 'rgba(140,140,140,0.25)';
        return n.color;
      })
      .nodeOpacity(0.92)
      .nodeLabel((n: any) => {
        const bits = [n.label];
        if (n.typeLabel || n.type) bits.push(`<i>${n.typeLabel || n.type}</i>`);
        if (n.status) bits.push(n.status);
        return `<span style="font-size:13px">${bits.join(' · ')}</span>`;
      })
      .nodeRelSize(4.2)
      .linkColor((l: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== l.source && sel !== l.target) return 'rgba(140,140,140,0.12)';
        return l.kind === 'relation' ? 'rgba(22,119,255,0.55)' : 'rgba(160,160,160,0.3)';
      })
      .linkWidth((l: any) =>
        selectedRef.current && (selectedRef.current === l.source || selectedRef.current === l.target)
          ? 1.6
          : l.kind === 'relation'
            ? 0.9
            : 0.5
      )
      .linkOpacity(0.5)
      .linkDirectionalParticles((l: any) =>
        selectedRef.current && (selectedRef.current === l.source || selectedRef.current === l.target) ? 2 : 0
      )
      .linkDirectionalParticleWidth(1.4)
      .onNodeClick((n: any) => {
        selectedRef.current = n.id;
        setSelected(n.id);
        setCardOpen(true);
        graphRef.current?.refresh();
        // keep the clicked node in view
        const distance = 140;
        const distRatio = 1 + distance / Math.max(Math.hypot(n.x, n.y, n.z), 1);
        graphRef.current?.cameraPosition(
          { x: n.x * distRatio, y: n.y * distRatio, z: n.z * distRatio },
          n,
          800
        );
      })
      .onBackgroundClick(() => {
        selectedRef.current = null;
        setSelected(null);
        setCardOpen(false);
        graphRef.current?.refresh();
      });

    // spread the two families apart in 3D for a cleaner initial settle
    graph.d3Force('charge')?.strength(-160);
    graph.d3Force('link')?.distance((l: any) => {
      const s = typeof l.source === 'object' ? l.source.id : l.source;
      const t = typeof l.target === 'object' ? l.target.id : l.target;
      return s?.startsWith('article:') && t?.startsWith('article:') ? 120 : 60;
    });
    graphRef.current = graph;
    graph.onEngineStop(() => {
      if (!disposed) graph.zoomToFit(600, 40);
    });

    const onResize = () => {
      if (containerRef.current && graphRef.current) {
        graphRef.current.width(containerRef.current.clientWidth).height(GRAPH_HEIGHT);
      }
    };
    window.addEventListener('resize', onResize);
    return () => {
      disposed = true;
      window.removeEventListener('resize', onResize);
      if (graph && graph._destructor) graph._destructor();
      graphRef.current = null;
    };
  }, [loading, nodes, edges]);

  // repaint accessors when the selection changes
  useEffect(() => {
    selectedRef.current = selected;
    graphRef.current?.refresh();
  }, [selected]);

  // deep link from an article capsule: select the requested entity once the
  // graph has loaded (ignored when the id is not on canvas)
  useEffect(() => {
    if (!focusEntityId || loading || selected) return;
    if (nodes.some(n => n.id === focusEntityId)) {
      setSelected(focusEntityId);
      selectedRef.current = focusEntityId;
      setCardOpen(true);
    }
  }, [focusEntityId, loading, nodes, selected]);

  const nodeById = useMemo(() => new Map(nodes.map(n => [n.id, n])), [nodes]);
  const cardNode = cardOpen && selected ? nodeById.get(selected) : null;
  const cardOutgoing = useMemo(
    () => (cardNode ? edges.filter(e => e.kind === 'relation' && e.source === cardNode.id) : []),
    [cardNode, edges]
  );
  const cardIncoming = useMemo(
    () => (cardNode ? edges.filter(e => e.kind === 'relation' && e.target === cardNode.id) : []),
    [cardNode, edges]
  );

  const focusNode = (id: string) => {
    setSelected(id);
    selectedRef.current = id;
    setCardOpen(true);
    const n = nodeById.get(id) as (GNode & { x?: number; y?: number; z?: number }) | undefined;
    if (n && graphRef.current) {
      const x = n.x || 0;
      const y = n.y || 0;
      const z = n.z || 0;
      const distRatio = 1 + 140 / Math.max(Math.hypot(x, y, z), 1);
      graphRef.current.cameraPosition({ x: x * distRatio, y: y * distRatio, z: z * distRatio }, n, 800);
    }
    graphRef.current?.refresh();
  };

  const zoomBy = (factor: number) => {
    const g = graphRef.current;
    if (!g) return;
    const pos = g.cameraPosition();
    const dist = Math.max(Math.hypot(pos.x, pos.y, pos.z), 1);
    const next = Math.max(60, Math.min(2400, dist / factor));
    g.cameraPosition({ x: (pos.x / dist) * next, y: (pos.y / dist) * next, z: (pos.z / dist) * next });
  };

  const fitAll = () => graphRef.current?.zoomToFit(700, 40);

  if (loading && nodes.length === 0)
    return (
      <div className="flex h-560px items-center justify-center">
        <Spin />
      </div>
    );
  if (nodes.length === 0)
    return <Empty description={t('page.wiki.graph.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />;

  const propertyEntries = cardNode?.properties ? Object.entries(cardNode.properties).slice(0, 8) : [];

  return (
    <div className="relative">
      <div className="mb-8px flex items-center justify-between">
        <div className="flex flex-wrap items-center gap-12px text-xs color-[var(--ant-color-text-tertiary)]">
          <span>
            <span className="inline-block h-8px w-8px rounded-full" style={{ background: ARTICLE_COLOR }} />{' '}
            {t('page.wiki.graph.articles')}
          </span>
          <span>
            <span className="inline-block h-8px w-8px rounded-full" style={{ background: TYPE_PALETTE[0] }} />{' '}
            {t('page.wiki.graph.entities')}
          </span>
          <span>
            <span className="inline-block h-8px w-8px rounded-full border-1px" style={{ borderColor: UNRESOLVED_COLOR }} />{' '}
            {t('page.wiki.graph.unresolved')}
          </span>
          <span className="hidden md:inline">{t('page.wiki.graph.dragHint3d')}</span>
        </div>
        <div className="flex gap-4px">
          <Button icon={<ZoomOutOutlined />} onClick={() => zoomBy(1.4)} size="small" />
          <Button icon={<ZoomInOutlined />} onClick={() => zoomBy(1 / 1.4)} size="small" />
          <Tooltip title={t('page.wiki.graph.reset')}>
            <Button icon={<AimOutlined />} onClick={fitAll} size="small" />
          </Tooltip>
        </div>
      </div>
      <div
        ref={containerRef}
        className="w-full touch-none overflow-hidden rounded-4px"
        style={{ height: GRAPH_HEIGHT }}
      />

      {cardNode && (
        <div
          className="absolute top-0 right-0 z-10 w-280px rounded-6px border-1px border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)] p-12px shadow-md"
          onClick={e => e.stopPropagation()}
        >
          <div className="mb-4px flex items-start justify-between gap-8px">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-6px">
                <span className="inline-block h-8px w-8px flex-shrink-0 rounded-full" style={{ background: cardNode.color }} />
                <span className="truncate text-14px font-500">{cardNode.label}</span>
              </div>
              <div className="mt-2px flex flex-wrap items-center gap-4px">
                {(cardNode.typeLabel || cardNode.type) && (
                  <Tag bordered={false} color="purple">
                    {cardNode.typeLabel || cardNode.type}
                  </Tag>
                )}
                {cardNode.status && (
                  <Tag bordered={false} color={STATUS_COLOR[cardNode.status] || 'default'}>
                    {cardNode.status}
                  </Tag>
                )}
                {cardNode.viaRelation && <Tag bordered={false}>{t('page.wiki.graph.viaRelation')}</Tag>}
              </div>
            </div>
            <Button icon={<CloseOutlined />} onClick={() => setCardOpen(false)} size="small" type="text" />
          </div>

          {cardNode.kind === 'unresolved' && (
            <div className="mt-4px text-xs text-[var(--ant-color-text-tertiary)]">{t('page.wiki.graph.unresolved')}</div>
          )}

          {!!cardNode.aliases?.length && (
            <div className="mt-8px text-xs text-[var(--ant-color-text-secondary)]">{cardNode.aliases.join(' / ')}</div>
          )}

          {typeof cardNode.confidence === 'number' && cardNode.confidence > 0 && (
            <div className="mt-4px text-xs text-[var(--ant-color-text-tertiary)]">
              {t('page.wiki.graph.confidence')}: {(cardNode.confidence * 100).toFixed(0)}%
            </div>
          )}

          {propertyEntries.length > 0 && (
            <div className="mt-8px">
              <div className="mb-2px text-xs font-500">{t('page.wiki.graph.properties')}</div>
              <div className="flex flex-col gap-2px">
                {propertyEntries.map(([key, value]) => (
                  <div className="flex gap-8px text-xs" key={key}>
                    <span className="w-70px flex-shrink-0 truncate text-[var(--ant-color-text-tertiary)]">{key}</span>
                    <span className="min-w-0 flex-1 break-all">{String(value)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="mt-8px">
            <div className="mb-2px text-xs font-500">{t('page.wiki.graph.relations')}</div>
            {cardOutgoing.length === 0 && cardIncoming.length === 0 ? (
              <div className="text-xs text-[var(--ant-color-text-tertiary)]">{t('page.wiki.graph.noRelations')}</div>
            ) : (
              <div className="flex flex-col gap-2px">
                {cardOutgoing.map((e, i) => (
                  <button
                    className="flex items-center gap-6px text-left text-xs"
                    key={`out-${i}`}
                    onClick={() => focusNode(e.target)}
                    type="button"
                  >
                    <span className="text-[var(--ant-color-text-tertiary)]">→ {e.label || e.relation}</span>
                    <span className="truncate text-[var(--ant-color-link)]">{nodeById.get(e.target)?.label ?? e.target}</span>
                  </button>
                ))}
                {cardIncoming.map((e, i) => (
                  <button
                    className="flex items-center gap-6px text-left text-xs"
                    key={`in-${i}`}
                    onClick={() => focusNode(e.source)}
                    type="button"
                  >
                    <span className="text-[var(--ant-color-text-tertiary)]">← {e.inverseLabel || e.inverse || e.label || e.relation}</span>
                    <span className="truncate text-[var(--ant-color-link)]">{nodeById.get(e.source)?.label ?? e.source}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {cardNode.articleId && (
            <Button
              block
              className="mt-8px"
              icon={<ExportOutlined />}
              onClick={() => nav(`/wiki/article/${cardNode.articleId}?kb=${kbId}`)}
              size="small"
              type="link"
            >
              {t('page.wiki.graph.openArticle')}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
