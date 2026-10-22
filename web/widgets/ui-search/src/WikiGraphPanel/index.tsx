import { AimOutlined, CloseOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import { Maximize2, Minimize2 } from 'lucide-react';
import ForceGraph3D from '3d-force-graph';
import { Button, Empty, Spin, Tag, Tooltip } from 'antd';
import { createPortal } from 'react-dom';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

/**
 * Knowledge-graph side panel for search results: when the current hits include
 * curated knowledge-base articles/entities, this panel draws the KB subgraph
 * (articles, wikilink-resolved entities, one-hop typed relations) as an
 * interactive 3D force graph — the same dataset the wiki app's graph page
 * serves (GET /wiki/kb/:id/_graph), merged across the KBs present in the
 * results. Clicking a node dims everything unlinked and opens an info card;
 * article nodes offer a jump into the wiki article page. The panel expands to
 * a fullscreen overlay for detailed exploration.
 */

interface GNode {
  id: string;
  label: string;
  kind: 'article' | 'entity' | 'unresolved';
  color: string;
  kbId: string;
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
const STATUS_COLOR: Record<string, string> = {
  published: 'green',
  reviewed: 'blue',
  proposed: 'orange',
  draft: 'orange'
};

const typeColor = (type?: string) => {
  if (!type) return TYPE_PALETTE[0];
  let hash = 0;
  for (let i = 0; i < type.length; i++) hash = (hash * 31 + type.charCodeAt(i)) >>> 0;
  return TYPE_PALETTE[hash % TYPE_PALETTE.length];
};

const toGNode = (raw: any, kbId: string): GNode => ({
  id: raw.id,
  label: raw.label,
  kind: raw.kind,
  kbId,
  color:
    raw.kind === 'article' ? ARTICLE_COLOR : raw.kind === 'unresolved' ? UNRESOLVED_COLOR : typeColor(raw.type),
  articleId: raw.kind === 'article' ? String(raw.id || '').slice('article:'.length) : raw.article_id,
  type: raw.type,
  typeLabel: raw.type_label,
  status: raw.status,
  confidence: raw.confidence,
  aliases: raw.aliases,
  properties: raw.properties,
  viaRelation: raw.via_relation
});

/* ---------------- data loading ---------------- */

function useWikiGraphData(kbIds: string[], apiConfig?: Record<string, any>) {
  const [loading, setLoading] = useState(false);
  const [nodes, setNodes] = useState<GNode[]>([]);
  const [edges, setEdges] = useState<GEdge[]>([]);
  const key = kbIds.join(',');

  useEffect(() => {
    if (!key) {
      setNodes([]);
      setEdges([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const base = String(apiConfig?.BaseUrl || '').replace(/\/+$/, '');
    const headers: Record<string, string> = { ...(apiConfig?.headers || {}) };
    if (apiConfig?.Token) headers['X-API-TOKEN'] = String(apiConfig.Token);

    const loadOne = (kbId: string) =>
      fetch(`${base}/wiki/kb/${kbId}/_graph`, { headers, credentials: 'include' })
        .then(res => (res.ok ? res.json() : null))
        .then(body => {
          const data = body?.data ?? body ?? {};
          return {
            nodes: (data.nodes || []).map((n: any) => toGNode(n, kbId)),
            edges: (data.edges || []) as GEdge[]
          };
        })
        .catch(() => ({ nodes: [] as GNode[], edges: [] as GEdge[] }));

    Promise.resolve()
      .then(() => Promise.all(key.split(',').map(loadOne)))
      .then(results => {
        if (cancelled) return;
        const nodeMap = new Map<string, GNode>();
        const edgeMap = new Map<string, GEdge>();
        results.forEach(r => {
          (r.nodes as GNode[]).forEach(n => {
            if (!nodeMap.has(n.id)) nodeMap.set(n.id, n);
          });
          r.edges.forEach(e => {
            const k = `${e.source}\u0000${e.target}\u0000${e.relation || ''}`;
            if (!edgeMap.has(k)) edgeMap.set(k, e);
          });
        });
        setNodes([...nodeMap.values()]);
        setEdges([...edgeMap.values()]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [key, apiConfig]);

  return { loading, nodes, edges };
}

/* ---------------- canvas ---------------- */

interface GraphCanvasProps {
  nodes: GNode[];
  edges: GEdge[];
  selected: string | null;
  onSelect: (id: string | null) => void;
  expanded: boolean;
  onToggleExpanded: () => void;
  /** host-side navigation for the selected node's wiki article */
  onOpenArticle?: () => void;
}

function GraphCanvas({ nodes, edges, selected, onSelect, expanded, onToggleExpanded, onOpenArticle }: GraphCanvasProps) {
  const { t } = useTranslation();
  const hostRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  const selectedRef = useRef<string | null>(null);
  const neighborsRef = useRef<Set<string> | null>(null);

  const nodeById = useMemo(() => new Map(nodes.map(n => [n.id, n])), [nodes]);
  const neighborsOf = useMemo(() => {
    const map = new Map<string, Set<string>>();
    edges.forEach(e => {
      if (!map.has(e.source)) map.set(e.source, new Set());
      if (!map.has(e.target)) map.set(e.target, new Set());
      map.get(e.source)!.add(e.target);
      map.get(e.target)!.add(e.source);
    });
    return map;
  }, [edges]);

  // build the 3D scene once per mount (the panel remounts the canvas when the
  // expanded flag flips between sider card and fullscreen overlay)
  useEffect(() => {
    const host = hostRef.current;
    if (!host || nodes.length === 0 || graphRef.current) return;

    const nodeIds = new Set(nodes.map(n => n.id));
    const links = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));

    const graph = new ForceGraph3D(host)
      .backgroundColor('rgba(0,0,0,0)')
      .width(host.clientWidth)
      .height(host.clientHeight)
      .graphData({ nodes, links })
      .nodeColor((n: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== n.id && !neighborsRef.current?.has(n.id)) return 'rgba(140,140,140,0.22)';
        return n.color;
      })
      .nodeOpacity(0.92)
      .nodeRelSize(4.2)
      .nodeLabel((n: any) => {
        const bits = [n.label];
        if (n.typeLabel || n.type) bits.push(`<i>${n.typeLabel || n.type}</i>`);
        if (n.status) bits.push(n.status);
        return `<span style="font-size:13px">${bits.join(' · ')}</span>`;
      })
      .linkColor((l: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== l.source && sel !== l.target) return 'rgba(140,140,140,0.10)';
        return l.kind === 'relation' ? 'rgba(22,119,255,0.55)' : 'rgba(160,160,160,0.3)';
      })
      .linkWidth((l: any) =>
        selectedRef.current && (selectedRef.current === l.source || selectedRef.current === l.target) ? 1.6 : 0.5
      )
      .linkOpacity(0.5)
      .linkDirectionalParticles((l: any) =>
        selectedRef.current && (selectedRef.current === l.source || selectedRef.current === l.target) ? 2 : 0
      )
      .linkDirectionalParticleWidth(1.4)
      .onNodeClick((n: any) => {
        onSelect(n.id);
        const distRatio = 1 + 140 / Math.max(Math.hypot(n.x, n.y, n.z), 1);
        graphRef.current?.cameraPosition({ x: n.x * distRatio, y: n.y * distRatio, z: n.z * distRatio }, n, 700);
      })
      .onBackgroundClick(() => onSelect(null));

    graph.d3Force('charge')?.strength(-140);
    graph.d3Force('link')?.distance((l: any) => {
      const s = typeof l.source === 'object' ? l.source.id : l.source;
      const t2 = typeof l.target === 'object' ? l.target.id : l.target;
      return s?.startsWith('article:') && t2?.startsWith('article:') ? 120 : 60;
    });
    graphRef.current = graph;
    graph.onEngineStop(() => graph.zoomToFit(600, 40));

    const ro = new ResizeObserver(() => {
      if (host.clientWidth > 0 && host.clientHeight > 0) {
        graphRef.current?.width(host.clientWidth).height(host.clientHeight);
      }
    });
    ro.observe(host);
    return () => {
      ro.disconnect();
      if (graph._destructor) graph._destructor();
      graphRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [nodes, edges]);

  // selection changes repaint accessors; camera stays where the click left it
  useEffect(() => {
    selectedRef.current = selected;
    neighborsRef.current = selected ? neighborsOf.get(selected) || new Set([selected]) : null;
    graphRef.current?.refresh();
  }, [selected, neighborsOf]);

  const zoomBy = (factor: number) => {
    const g = graphRef.current;
    if (!g) return;
    const pos = g.cameraPosition();
    const dist = Math.max(Math.hypot(pos.x, pos.y, pos.z), 1);
    const next = Math.max(60, Math.min(2400, dist / factor));
    g.cameraPosition({ x: (pos.x / dist) * next, y: (pos.y / dist) * next, z: (pos.z / dist) * next });
  };

  const fitAll = () => graphRef.current?.zoomToFit(700, 40);

  const controlBtn = 'h-24px w-24px';

  return (
    <div className='relative h-full w-full'>
      <div className='absolute inset-0 touch-none' ref={hostRef} />
      <div className='absolute right-6px top-6px z-2 flex flex-col gap-4px'>
        <Tooltip title={t('labels.wikiGraphFullscreen')} placement='left'>
          <Button
            className={controlBtn}
            icon={expanded ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
            size='small'
            type='text'
            onClick={onToggleExpanded}
          />
        </Tooltip>
        <Tooltip title={t('labels.wikiGraphZoomIn')} placement='left'>
          <Button className={controlBtn} icon={<ZoomInOutlined />} size='small' type='text' onClick={() => zoomBy(1 / 1.4)} />
        </Tooltip>
        <Tooltip title={t('labels.wikiGraphZoomOut')} placement='left'>
          <Button className={controlBtn} icon={<ZoomOutOutlined />} size='small' type='text' onClick={() => zoomBy(1.4)} />
        </Tooltip>
        <Tooltip title={t('labels.wikiGraphFit')} placement='left'>
          <Button className={controlBtn} icon={<AimOutlined />} size='small' type='text' onClick={fitAll} />
        </Tooltip>
      </div>

      {/* selection card: the clicked node's summary + clickable neighborhood */}
      {selected && nodeById.get(selected) && (
        <div
          className='absolute left-6px top-6px z-2 w-220px rounded-6px border-1px border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)] p-10px shadow-md'
          onClick={e => e.stopPropagation()}
        >
          {(() => {
            const node = nodeById.get(selected)!;
            const out = edges.filter(e => e.source === node.id);
            const inc = edges.filter(e => e.target === node.id);
            const props = node.properties ? Object.entries(node.properties).slice(0, 5) : [];
            return (
              <>
                <div className='flex items-start justify-between gap-6px'>
                  <div className='flex min-w-0 items-center gap-6px'>
                    <span className='h-8px w-8px flex-none rounded-full' style={{ background: node.color }} />
                    <span className='truncate text-13px font-500'>{node.label}</span>
                  </div>
                  <Button icon={<CloseOutlined />} onClick={() => onSelect(null)} size='small' type='text' />
                </div>
                <div className='mt-4px flex flex-wrap items-center gap-4px'>
                  {(node.typeLabel || node.type) && (
                    <Tag variant='filled' color='purple'>
                      {node.typeLabel || node.type}
                    </Tag>
                  )}
                  {node.status && (
                    <Tag variant='filled' color={STATUS_COLOR[node.status] || 'default'}>
                      {node.status}
                    </Tag>
                  )}
                  {node.kind === 'unresolved' && (
                    <Tag variant='filled'>{t('labels.wikiGraphUnresolved')}</Tag>
                  )}
                </div>
                {!!node.aliases?.length && (
                  <div className='mt-4px text-11px text-[var(--ant-color-text-secondary)]'>{node.aliases.join(' / ')}</div>
                )}
                {props.length > 0 && (
                  <div className='mt-6px flex flex-col gap-2px'>
                    {props.map(([k, v]) => (
                      <div className='flex gap-6px text-11px' key={k}>
                        <span className='w-56px flex-none truncate text-[var(--ant-color-text-tertiary)]'>{k}</span>
                        <span className='min-w-0 flex-1 break-all'>{String(v)}</span>
                      </div>
                    ))}
                  </div>
                )}
                <div className='mt-6px'>
                  <div className='mb-2px text-11px font-500'>{t('labels.wikiGraphRelations')}</div>
                  {out.length === 0 && inc.length === 0 ? (
                    <div className='text-11px text-[var(--ant-color-text-tertiary)]'>{t('labels.wikiGraphNoRelations')}</div>
                  ) : (
                    <div className='flex flex-col gap-1px'>
                      {out.slice(0, 6).map((e, i) => (
                        <button
                          className='flex items-center gap-4px text-left text-11px'
                          key={`o${i}`}
                          onClick={() => onSelect(e.target)}
                          type='button'
                        >
                          <span className='flex-none text-[var(--ant-color-text-tertiary)]'>→</span>
                          <span className='truncate text-[var(--ant-color-link)]'>
                            {nodeById.get(e.target)?.label ?? e.target}
                          </span>
                        </button>
                      ))}
                      {inc.slice(0, 6).map((e, i) => (
                        <button
                          className='flex items-center gap-4px text-left text-11px'
                          key={`i${i}`}
                          onClick={() => onSelect(e.source)}
                          type='button'
                        >
                          <span className='flex-none text-[var(--ant-color-text-tertiary)]'>←</span>
                          <span className='truncate text-[var(--ant-color-link)]'>
                            {nodeById.get(e.source)?.label ?? e.source}
                          </span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
                {node.articleId && onOpenArticle && (
                  <Button
                    block
                    className='mt-6px'
                    size='small'
                    type='link'
                    onClick={onOpenArticle}
                  >
                    {t('labels.wikiGraphOpenArticle')}
                  </Button>
                )}
              </>
            );
          })()}
        </div>
      )}
    </div>
  );
}

/* ---------------- panel shell ---------------- */

interface WikiGraphPanelProps {
  kbIds: string[];
  apiConfig?: Record<string, any>;
  theme?: string;
  /** open the wiki article page for a graph node (host-side navigation) */
  onOpenArticle?: (articleId: string, kbId: string) => void;
  /** fixed height of the in-sider card; fullscreen covers the viewport */
  height?: number;
}

export function WikiGraphPanel({ kbIds, apiConfig, theme, onOpenArticle, height = 300 }: WikiGraphPanelProps) {
  const { t } = useTranslation();
  const { loading, nodes, edges } = useWikiGraphData(kbIds, apiConfig);
  const [selected, setSelected] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    setSelected(null);
  }, [kbIds.join(',')]);

  const openSelectedArticle = () => {
    const node = selected ? nodes.find(n => n.id === selected) : null;
    if (node?.articleId) onOpenArticle?.(node.articleId, node.kbId);
  };

  // clicking the card's open-article action routes to the wiki page — wired
  // here so the card stays inside GraphCanvas while navigation stays host-side
  const canvas = (
    <GraphCanvas
      edges={edges}
      expanded={expanded}
      nodes={nodes}
      selected={selected}
      onOpenArticle={openSelectedArticle}
      onSelect={setSelected}
      onToggleExpanded={() => setExpanded(v => !v)}
    />
  );

  const legend = (
    <div className='flex flex-wrap items-center gap-8px text-11px text-[var(--ant-color-text-tertiary)]'>
      <span className='flex items-center gap-4px'>
        <span className='h-7px w-7px rounded-full' style={{ background: ARTICLE_COLOR }} />
        {t('labels.wikiGraphArticles')}
      </span>
      <span className='flex items-center gap-4px'>
        <span className='h-7px w-7px rounded-full' style={{ background: TYPE_PALETTE[0] }} />
        {t('labels.wikiGraphEntities')}
      </span>
      <span className='hidden xl:inline'>{t('labels.wikiGraphHint')}</span>
    </div>
  );

  const header = (
    <div className='mb-6px flex items-center justify-between gap-8px'>
      <span className='flex items-center gap-6px text-13px font-550'>
        {t('labels.wikiGraph')}
        {loading && <Spin size='small' />}
      </span>
      {expanded && (
        <Button icon={<CloseOutlined />} size='small' type='text' onClick={() => setExpanded(false)} />
      )}
    </div>
  );

  if (expanded) {
    return createPortal(
      <div
        className='left-0 right-0 top-0 bottom-0 fixed z-1000 p-16px'
        style={{ background: theme === 'dark' ? '#141414' : '#ffffff' }}
        onClick={e => e.stopPropagation()}
      >
        <div className='mx-auto flex h-full max-w-1600px flex-col'>
          {header}
          {legend}
          <div className='mt-6px min-h-0 flex-1'>{canvas}</div>
        </div>
      </div>,
      document.body
    );
  }

  return (
    <div
      className='flex flex-col rounded-8px border-1px border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)] p-10px'
      style={{ height }}
    >
      {header}
      {legend}
      <div className='mt-6px min-h-0 flex-1 overflow-hidden rounded-4px'>
        {!loading && nodes.length === 0 ? (
          <Empty
            className='pt-40px'
            description={t('labels.wikiGraphEmpty')}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        ) : (
          canvas
        )}
      </div>
    </div>
  );
}

export default WikiGraphPanel;
