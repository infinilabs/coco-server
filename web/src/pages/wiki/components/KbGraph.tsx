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

// the canvas fills the viewport below the toolbar instead of a fixed slab
const MIN_GRAPH_HEIGHT = 520;
const MAX_GRAPH_HEIGHT = 1200;

// world-space radius of a default node ball (three-forcegraph: cbrt(val) * nodeRelSize)
const NODE_WORLD_RADIUS = 4.2;
// caps on the overlay labels painted around a selected node: first-hop
// bright and nearest-first, second-hop dimmer
const MAX_VISIBLE_NODE_LABELS = 24;
const MAX_VISIBLE_SECOND_LABELS = 30;

const makeLabelEl = (text: string, kind: 'link' | 'node') => {
  const el = document.createElement('div');
  el.textContent = text;
  el.title = text;
  Object.assign(el.style, {
    position: 'absolute',
    left: '0',
    top: '0',
    display: 'none',
    pointerEvents: 'none',
    whiteSpace: 'nowrap',
    maxWidth: kind === 'node' ? '150px' : '120px',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    fontSize: kind === 'node' ? '11px' : '10px',
    lineHeight: kind === 'node' ? '17px' : '16px',
    padding: kind === 'node' ? '0 6px' : '0 4px',
    borderRadius: kind === 'node' ? '9px' : '6px',
    color: kind === 'node' ? 'var(--ant-color-text, rgba(0,0,0,0.88))' : 'var(--ant-color-text-secondary, rgba(0,0,0,0.45))',
    background: 'var(--ant-color-bg-elevated, rgba(255,255,255,0.92))',
    border: '1px solid var(--ant-color-border-secondary, rgba(5,5,5,0.06))',
    boxShadow: '0 1px 4px rgba(0,0,0,0.18)'
  } as Partial<CSSStyleDeclaration>);
  return el;
};

// world position → canvas pixels; null when the point sits behind the camera
interface ScreenPoint {
  sx: number;
  sy: number;
  depth: number;
}

const projectToScreen = (
  pos: { x: number; y: number; z: number },
  view: { inv: ArrayLike<number>; proj: ArrayLike<number>; width: number; height: number }
): ScreenPoint | null => {
  const viewInv = view.inv;
  const { proj, width, height } = view;
  const { x, y, z } = pos;
  const cx = viewInv[0] * x + viewInv[4] * y + viewInv[8] * z + viewInv[12];
  const cy = viewInv[1] * x + viewInv[5] * y + viewInv[9] * z + viewInv[13];
  const cz = viewInv[2] * x + viewInv[6] * y + viewInv[10] * z + viewInv[14];
  const dx = proj[0] * cx + proj[4] * cy + proj[8] * cz + proj[12];
  const dy = proj[1] * cx + proj[5] * cy + proj[9] * cz + proj[13];
  const dw = proj[3] * cx + proj[7] * cy + proj[11] * cz + proj[15];
  if (dw <= 0) return null;
  return {
    sx: ((dx / dw) * 0.5 + 0.5) * width,
    sy: (0.5 - (dy / dw) * 0.5) * height,
    depth: dw
  };
};

/**
 * Knowledge graph for one KB (ontology phase O3), rendered as an interactive
 * 3D force graph: articles, wikilink-resolved entities and one-hop typed
 * relations in a self-organizing layout — drag to orbit, wheel to zoom, click
 * an article to navigate, click an entity to open its card. The canvas stays
 * clean by default: hovering a node shows its name via the built-in tooltip,
 * clicking one paints overlay labels for it, its one-hop neighborhood (bright)
 * and second hop (dimmer), plus the relation titles on its adjacent edges.
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
  const overlayRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  // accessors read this ref — selection changes repaint without rebuilding
  const selectedRef = useRef<string | null>(null);
  const neighborsRef = useRef<Set<string> | null>(null);
  // node ids whose overlay label is currently on screen; the hover tooltip
  // reads this to avoid painting the same name twice
  const labeledNodeIdsRef = useRef<Set<string>>(new Set());

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

  // second-hop neighborhood of the selection: their titles render dimmer
  // instead of hiding, so the selected area reads two steps out
  const secondNeighbors = useMemo(() => {
    if (!selected || !neighbors) return null;
    const set = new Set<string>();
    edges.forEach(e => {
      if (neighbors.has(e.source) && !neighbors.has(e.target)) set.add(e.target);
      if (neighbors.has(e.target) && !neighbors.has(e.source)) set.add(e.source);
    });
    return set;
  }, [selected, neighbors, edges]);

  const hasNeighbor = (a: string, b: string) =>
    edges.some(e => (e.source === a && e.target === b) || (e.source === b && e.target === a));

  // build the 3D scene once data is in and the container is on screen
  useEffect(() => {
    if (loading || nodes.length === 0 || !containerRef.current || graphRef.current) return;

    let disposed = false;
    const host = containerRef.current;
    const nodeIds = new Set(nodes.map(n => n.id));
    const links = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target));
    const nodeByIdLocal = new Map(nodes.map(n => [n.id, n]));

    const computeHeight = () =>
      Math.round(
        Math.min(
          Math.max(window.innerHeight - host.getBoundingClientRect().top - 24, MIN_GRAPH_HEIGHT),
          MAX_GRAPH_HEIGHT
        )
      );
    const graphHeight = computeHeight();
    setCanvasHeight(graphHeight);

    const graph = new ForceGraph3D(host)
      .backgroundColor('rgba(0,0,0,0)')
      .width(host.clientWidth)
      .height(graphHeight)
      .graphData({ nodes, links })
      .nodeColor((n: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== n.id && !hasNeighbor(sel, n.id)) return 'rgba(140,140,140,0.25)';
        return n.color;
      })
      .nodeOpacity(0.92)
      .nodeLabel((n: any) => {
        if (labeledNodeIdsRef.current.has(n.id)) return '';
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

    // DOM overlay carrying node names and relation titles; visibility is
    // recomputed every frame from camera position and selection state
    const overlay = overlayRef.current;
    const nodeLabelEls = new Map<string, HTMLDivElement>();
    const linkLabelEls: Array<HTMLDivElement | undefined> = [];
    if (overlay) {
      overlay.innerHTML = '';
      nodes.forEach(n => {
        const el = makeLabelEl(n.label, 'node');
        overlay.appendChild(el);
        nodeLabelEls.set(n.id, el);
      });
      links.forEach((l, i) => {
        if (l.kind !== 'relation') return;
        const text = l.label || l.relation || '';
        if (!text) return;
        const el = makeLabelEl(text, 'link');
        overlay.appendChild(el);
        linkLabelEls[i] = el;
      });
    }

    const hideEl = (el?: HTMLDivElement) => {
      if (el && el.style.display !== 'none') el.style.display = 'none';
    };

    let rafId = 0;
    const paintLabels = () => {
      rafId = requestAnimationFrame(paintLabels);
      const g = graphRef.current;
      const overlayEl = overlay;
      const cam: any = g?.camera();
      const w = host.clientWidth;
      const h = host.clientHeight;
      if (!overlayEl || !cam?.projectionMatrix || !cam.matrixWorldInverse || !w || !h) return;

      const view = {
        inv: cam.matrixWorldInverse.elements as ArrayLike<number>,
        proj: cam.projectionMatrix.elements as ArrayLike<number>,
        width: w,
        height: h
      };
      const camPos = cam.position || { x: 0, y: 0, z: 0 };
      const target = (g.controls ? g.controls() : null)?.target || { x: 0, y: 0, z: 0 };
      const camDist = Math.max(Math.hypot(camPos.x - target.x, camPos.y - target.y, camPos.z - target.z), 1);
      const pxPerUnit = h / (2 * camDist * Math.tan((((cam.fov as number) || 55) / 2) * (Math.PI / 180)));
      const ballPx = Math.min(Math.max(NODE_WORLD_RADIUS * pxPerUnit, 3), 22);
      const focusRadius = 0.48 * Math.min(w, h);
      const halfW = w / 2;
      const halfH = h / 2;
      const sel = selectedRef.current;
      const nbrs = neighborsRef.current;
      const second = secondNeighborsRef.current;
      const firstCandidates: (ScreenPoint & { el: HTMLDivElement; centerDist: number; id: string; isSel: boolean })[] = [];
      const secondCandidates: (ScreenPoint & { el: HTMLDivElement; centerDist: number; id: string })[] = [];
      // labels hug the selected node on screen — near it or not at all;
      // fall back to screen center when the anchor is behind the camera
      let focusPt = { sx: halfW, sy: halfH };
      if (sel) {
        const sn: any = nodes.find(x => x.id === sel);
        if (sn && typeof sn.x === 'number' && typeof sn.y === 'number' && typeof sn.z === 'number') {
          const p = projectToScreen({ x: sn.x, y: sn.y, z: sn.z }, view);
          if (p) focusPt = { sx: p.sx, sy: p.sy };
        }
      }
      const distToFocus = (pt: ScreenPoint) => Math.hypot(pt.sx - focusPt.sx, pt.sy - focusPt.sy);
      const onScreen = (pt: ScreenPoint) => pt.sx >= 0 && pt.sx <= w && pt.sy >= 0 && pt.sy <= h;
      // ids whose overlay label is on screen right now; the hover tooltip
      // yields to them so the same name is never drawn twice
      const labeled = new Set<string>();

      nodes.forEach(raw => {
        const el = nodeLabelEls.get(raw.id);
        if (!el) return;
        const n = raw as typeof raw & { x?: number; y?: number; z?: number };
        const placed =
          typeof n.x === 'number' && typeof n.y === 'number' && typeof n.z === 'number'
            ? projectToScreen({ x: n.x, y: n.y, z: n.z }, view)
            : null;
        if (!placed) {
          hideEl(el);
          return;
        }

        if (sel) {
          // clicked: first-hop titles render bright nearest the anchor,
          // second-hop dimmer — far-flung neighbors label not at all
          if (n.id === sel) {
            labeled.add(n.id);
            el.style.display = '';
            el.style.opacity = '';
            el.style.zIndex = String(Math.max(1, Math.round(1000 - placed.depth)));
            el.style.transform = `translate(-50%, -50%) translate(${placed.sx.toFixed(1)}px, ${(placed.sy + ballPx + 5).toFixed(1)}px)`;
            el.style.fontWeight = '600';
            el.style.borderColor = n.color;
          } else if (nbrs?.has(n.id)) {
            // first hop: every on-screen neighbor is a candidate — the
            // nearest-capped render does the decluttering. A hard radius
            // here left scattered layouts with lines but no labels.
            if (!onScreen(placed)) {
              hideEl(el);
              return;
            }
            firstCandidates.push({ el, id: n.id, ...placed, centerDist: distToFocus(placed), isSel: false });
          } else if (second?.has(n.id)) {
            if (!onScreen(placed)) {
              hideEl(el);
              return;
            }
            const centerDist = distToFocus(placed);
            if (centerDist <= focusRadius * 1.15) secondCandidates.push({ el, id: n.id, ...placed, centerDist });
            else hideEl(el);
          } else hideEl(el);
          return;
        }

        // nothing selected: no ambient labels in the focus area —
        // names show on hover (tooltip) or on click (neighborhood labels)
        hideEl(el);
      });
      firstCandidates.sort((a, b) => a.centerDist - b.centerDist);
      firstCandidates.forEach((c, i) => {
        if (i >= MAX_VISIBLE_NODE_LABELS) {
          hideEl(c.el);
          return;
        }
        labeled.add(c.id);
        c.el.style.display = '';
        c.el.style.opacity = '';
        c.el.style.zIndex = String(Math.max(1, Math.round(1000 - c.depth)));
        c.el.style.transform = `translate(-50%, -50%) translate(${c.sx.toFixed(1)}px, ${(c.sy + ballPx + 5).toFixed(1)}px)`;
        c.el.style.fontWeight = '400';
        c.el.style.borderColor = '';
      });

      secondCandidates.sort((a, b) => a.centerDist - b.centerDist);
      secondCandidates.forEach((c, i) => {
        if (i >= MAX_VISIBLE_SECOND_LABELS) {
          hideEl(c.el);
          return;
        }
        labeled.add(c.id);
        c.el.style.display = '';
        c.el.style.opacity = '0.72';
        c.el.style.zIndex = String(Math.max(1, Math.round(900 - c.depth)));
        c.el.style.transform = `translate(-50%, -50%) translate(${c.sx.toFixed(1)}px, ${(c.sy + ballPx + 5).toFixed(1)}px)`;
        c.el.style.fontWeight = '400';
        c.el.style.borderColor = '';
      });

      links.forEach((l, i) => {
        const el = linkLabelEls[i];
        if (!el) return;
        const s = (typeof l.source === 'object' ? l.source : nodeByIdLocal.get(l.source)) as
          | { x?: number; y?: number; z?: number }
          | undefined;
        const t = (typeof l.target === 'object' ? l.target : nodeByIdLocal.get(l.target)) as
          | { x?: number; y?: number; z?: number }
          | undefined;
        if (!s || !t || typeof s.x !== 'number' || typeof t.x !== 'number') {
          hideEl(el);
          return;
        }
        const placed = projectToScreen(
          { x: (s.x + t.x) / 2, y: ((s.y ?? 0) + (t.y ?? 0)) / 2, z: ((s.z ?? 0) + (t.z ?? 0)) / 2 },
          view
        );
        if (!placed) {
          hideEl(el);
          return;
        }

        if (sel) {
          // the lib replaces string endpoints with node object references once settled
          const sid = typeof l.source === 'object' ? (l.source as unknown as { id: string }).id : l.source;
          const tid = typeof l.target === 'object' ? (l.target as unknown as { id: string }).id : l.target;
          if (sid === sel || tid === sel) {
            el.style.display = '';
            el.style.zIndex = String(Math.max(1, Math.round(1000 - placed.depth)));
            el.style.transform = `translate(-50%, -50%) translate(${placed.sx.toFixed(1)}px, ${placed.sy.toFixed(1)}px)`;
          } else hideEl(el);
          return;
        }

        // nothing selected: relation titles stay hidden too —
        // a click on either endpoint reveals them
        hideEl(el);
      });

      labeledNodeIdsRef.current = labeled;
    };
    rafId = requestAnimationFrame(paintLabels);

    const onResize = () => {
      if (containerRef.current && graphRef.current) {
        const next = computeHeight();
        graphRef.current.width(containerRef.current.clientWidth).height(next);
        setCanvasHeight(next);
      }
    };
    window.addEventListener('resize', onResize);
    return () => {
      disposed = true;
      cancelAnimationFrame(rafId);
      window.removeEventListener('resize', onResize);
      if (overlay) overlay.innerHTML = '';
      if (graph && graph._destructor) graph._destructor();
      graphRef.current = null;
    };
  }, [loading, nodes, edges]);

  // repaint accessors when the selection changes
  const secondNeighborsRef = useRef<Set<string> | null>(null);
  useEffect(() => {
    selectedRef.current = selected;
    neighborsRef.current = neighbors;
    secondNeighborsRef.current = secondNeighbors;
    graphRef.current?.refresh();
  }, [selected, neighbors, secondNeighbors]);

  // deep link from an article capsule: select the requested entity once the
  // graph has loaded (ignored when the id is not on canvas); fly the camera
  // to it so the click-to-focus view (labels, relations) applies here too
  useEffect(() => {
    if (!focusEntityId || loading || selected) return;
    const n = nodes.find(
      (x): x is GNode & { x?: number; y?: number; z?: number } => x.id === focusEntityId
    );
    if (!n) return;
    setSelected(focusEntityId);
    selectedRef.current = focusEntityId;
    setCardOpen(true);
    if (graphRef.current) {
      const x = n.x || 0;
      const y = n.y || 0;
      const z = n.z || 0;
      const distRatio = 1 + 140 / Math.max(Math.hypot(x, y, z), 1);
      graphRef.current.cameraPosition({ x: x * distRatio, y: y * distRatio, z: z * distRatio }, n, 800);
    }
  }, [focusEntityId, loading, nodes, selected]);

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
      <div className="relative w-full touch-none overflow-hidden rounded-4px" style={{ height: canvasHeight }}>
        <div className="absolute inset-0" ref={containerRef} />
        <div className="pointer-events-none absolute inset-0 overflow-hidden" ref={overlayRef} style={{ zIndex: 1 }} />
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
