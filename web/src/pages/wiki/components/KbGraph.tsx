import { AimOutlined, CloseOutlined, ExportOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import { Button, Empty, Spin, Tag, Tooltip } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchWikiKbGraph } from '@/service/api';
// the shared knowledge-graph canvas — the search panel's fullscreen graph
// renders through the exact same component, so both surfaces stay identical
import { KbGraphCanvas, type KbGraphCanvasHandle } from 'ui-search/source';

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

// the canvas fills the viewport below the toolbar instead of a fixed slab
const MIN_GRAPH_HEIGHT = 520;
const MAX_GRAPH_HEIGHT = 1200;

/**
 * Knowledge graph for one KB (ontology phase O3), rendered through the shared
 * KbGraphCanvas: articles, wikilink-resolved entities and one-hop typed
 * relations in a self-organizing layout — drag to orbit, wheel to zoom, click
 * an article to navigate, click an entity to open its card. The canvas stays
 * clean by default: hovering a node shows its name via the built-in tooltip,
 * clicking one paints overlay labels for it and its direct one-hop neighbors,
 * plus the relation titles on its adjacent edges, and flies the camera to the
 * neighborhood. Nodes without a direct edge to the selection stay unnamed.
 */
export function KbGraph({ kbId, focusEntityId }: { kbId: string; focusEntityId?: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<GNode[]>([]);
  const [edges, setEdges] = useState<GEdge[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [cardOpen, setCardOpen] = useState(false);
  const [canvasHeight, setCanvasHeight] = useState(MIN_GRAPH_HEIGHT);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<KbGraphCanvasHandle>(null);

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

  // the viewport below the toolbar — recomputed on window resizes; the canvas
  // tracks its own width/height inside the container
  useEffect(() => {
    const computeHeight = () => {
      const host = containerRef.current;
      if (!host) return;
      const next = Math.round(
        Math.min(
          Math.max(window.innerHeight - host.getBoundingClientRect().top - 24, MIN_GRAPH_HEIGHT),
          MAX_GRAPH_HEIGHT
        )
      );
      setCanvasHeight(prev => (prev === next ? prev : next));
    };
    computeHeight();
    window.addEventListener('resize', computeHeight);
    return () => window.removeEventListener('resize', computeHeight);
  }, [loading, nodes.length]);

  const nodeById = useMemo(() => new Map(nodes.map(n => [n.id, n])), [nodes]);
  const cardNode = cardOpen && selected ? nodeById.get(selected) : null;
  // wikilink edges are relations too (a page linking an entity) — the card
  // must list every edge the canvas draws, not only typed relations
  const cardOutgoing = useMemo(
    () => (cardNode ? edges.filter(e => e.source === cardNode.id) : []),
    [cardNode, edges]
  );
  const cardIncoming = useMemo(
    () => (cardNode ? edges.filter(e => e.target === cardNode.id) : []),
    [cardNode, edges]
  );

  const handleSelect = (id: string | null) => {
    setSelected(id);
    setCardOpen(Boolean(id));
  };

  const focusNode = (id: string) => {
    setSelected(id);
    setCardOpen(true);
    canvasRef.current?.flyTo(id);
  };

  // deep link from an article capsule: select the requested entity once the
  // graph has loaded (ignored when the id is not on canvas); the canvas flies
  // to it so the click-to-focus view (labels, relations) applies here too.
  // Callers pass the bare entity id while canvas nodes carry the "entity:"
  // prefix — accept both.
  useEffect(() => {
    if (!focusEntityId || loading || selected) return;
    const n = nodes.find(x => x.id === focusEntityId || x.id === `entity:${focusEntityId}`);
    if (!n) return;
    // select with the canvas node id (prefixed) — the bare param id matches
    // no node, which dimmed the whole canvas with no card and no labels
    setSelected(n.id);
    setCardOpen(true);
    // the engine assigns node coordinates a few frames after graphData —
    // the handle's flyTo polls until they exist
    canvasRef.current?.flyTo(n.id);
  }, [focusEntityId, loading, nodes, selected]);

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
          <Button icon={<ZoomOutOutlined />} onClick={() => canvasRef.current?.zoomBy(1.4)} size="small" />
          <Button icon={<ZoomInOutlined />} onClick={() => canvasRef.current?.zoomBy(1 / 1.4)} size="small" />
          <Tooltip title={t('page.wiki.graph.reset')}>
            <Button icon={<AimOutlined />} onClick={() => canvasRef.current?.fitAll()} size="small" />
          </Tooltip>
        </div>
      </div>
      <div className="relative w-full touch-none overflow-hidden rounded-4px" ref={containerRef} style={{ height: canvasHeight }}>
        <KbGraphCanvas
          ref={canvasRef}
          edges={edges}
          labels={{ fit: t('page.wiki.graph.reset') }}
          nodes={nodes}
          selected={selected}
          onSelect={handleSelect}
        />
      </div>

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
                    <span className="text-[var(--ant-color-text-tertiary)]">→ {e.label || e.relation || t('page.wiki.graph.wikilinkRelation')}</span>
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
                    <span className="text-[var(--ant-color-text-tertiary)]">← {e.inverseLabel || e.inverse || e.label || e.relation || t('page.wiki.graph.wikilinkRelation')}</span>
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
