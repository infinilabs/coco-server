import ForceGraph3D from '3d-force-graph';
import { AimOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import { Maximize2, Minimize2 } from 'lucide-react';
import { Button, Tooltip } from 'antd';
import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef } from 'react';

/**
 * Shared knowledge-graph 3D canvas — the single rendering path behind both
 * the wiki KB graph page (KbGraph) and the search-page graph panel's
 * fullscreen mode, so the two surfaces stay pixel- and behavior-identical:
 * same force layout, same selection dimming/swelling, same DOM overlay that
 * paints entity names for the selection and its direct one-hop neighbors,
 * and the same fly-to-neighborhood camera move.
 *
 * Presentational only: data, selection state and the detail card live with
 * the callers; this component owns the 3D scene, the overlay label declutter
 * and the camera. Tooltip strings come in via `labels` because widget and
 * host run separate i18n instances.
 */

export interface KbGraphNode {
  id: string;
  label: string;
  kind: 'article' | 'entity' | 'unresolved';
  color: string;
  type?: string;
  typeLabel?: string;
  status?: string;
  aliases?: string[];
  properties?: Record<string, unknown>;
  confidence?: number;
}

export interface KbGraphEdge {
  source: string;
  target: string;
  kind: 'wikilink' | 'relation';
  relation?: string;
  label?: string;
  inverse?: string;
  inverseLabel?: string;
}

export interface KbGraphCanvasHandle {
  /** select nothing; fly the camera to a node's one-hop neighborhood */
  flyTo: (id: string) => void;
  zoomBy: (factor: number) => void;
  fitAll: () => void;
}

export interface KbGraphCanvasLabels {
  zoomIn?: string;
  zoomOut?: string;
  fit?: string;
  fullscreen?: string;
  exitFullscreen?: string;
}

interface KbGraphCanvasProps {
  nodes: KbGraphNode[];
  edges: KbGraphEdge[];
  selected: string | null;
  onSelect: (id: string | null) => void;
  /** fullscreen toggle button hidden unless provided */
  onToggleExpand?: () => void;
  expanded?: boolean;
  labels?: KbGraphCanvasLabels;
  /** cap on overlay node names around the selection (nearest-first) */
  maxNodeLabels?: number;
  /** cap on relation titles on the selection's edges */
  maxRelationLabels?: number;
}

// world-space radius of a default node ball (three-forcegraph: cbrt(val) * nodeRelSize)
const NODE_WORLD_RADIUS = 4.2;

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

// the lib swaps string endpoints for node-object references on the links
// passed to graphData — comparisons against selection must normalize both
const endpointId = (v: unknown): string =>
  typeof v === 'object' && v !== null ? String((v as { id?: unknown }).id) : String(v);

type Positioned = KbGraphNode & { x?: number; y?: number; z?: number };

/**
 * Fly the camera so the selected node sits centered with its one-hop
 * neighborhood spread across the viewport: distance comes from a p70
 * percentile of neighbor distances TO THE SELECTED NODE — outliers neither
 * starve the view (fixed-distance bug) nor shrink the dense core into an
 * unreadable blob (centroid-max fit). Far neighbors may fall off-frame;
 * they stay lit and named on hover.
 */
const flyToNeighborhood = (graph: any, nodeList: Positioned[], edgeList: KbGraphEdge[], centerId: string) => {
  if (!graph) return;
  const ids = new Set<string>([centerId]);
  edgeList.forEach(e => {
    if (endpointId(e.source) === centerId) ids.add(endpointId(e.target));
    if (endpointId(e.target) === centerId) ids.add(endpointId(e.source));
  });
  const sel = nodeList.find(n => n.id === centerId);
  if (!sel || typeof sel.x !== 'number' || typeof sel.y !== 'number' || typeof sel.z !== 'number') return;
  const sx = sel.x;
  const sy = sel.y;
  const sz = sel.z;
  const dists: number[] = [];
  nodeList.forEach(n => {
    if (n.id === centerId || !ids.has(n.id)) return;
    if (typeof n.x === 'number' && typeof n.y === 'number' && typeof n.z === 'number')
      dists.push(Math.hypot(n.x - sx, n.y - sy, n.z - sz));
  });
  if (dists.length === 0) return;
  dists.sort((a, b) => a - b);
  // p70: the typical direct neighbor lands ~55% out from the center; the
  // crowded inner ring spreads wide enough for its name labels. The floor
  // is low on purpose — settled hub layouts pack neighbors tightly and
  // need a close camera, not a safe default distance.
  const p70 = dists[Math.min(dists.length - 1, Math.floor(dists.length * 0.7))] || 0;
  const dist = Math.max(50, (p70 / Math.tan((55 / 2) * (Math.PI / 180))) * 0.92);
  // keep the current viewing direction, just re-frame around the selection
  const pos = graph.cameraPosition();
  let dx = sx;
  let dy = sy;
  let dz = sz;
  if (pos && typeof pos.x === 'number') {
    dx = pos.x - sx;
    dy = pos.y - sy;
    dz = pos.z - sz;
  }
  const len = Math.max(Math.hypot(dx, dy, dz), 1);
  graph.cameraPosition(
    { x: sx + (dx / len) * dist, y: sy + (dy / len) * dist, z: sz + (dz / len) * dist },
    { x: sx, y: sy, z: sz },
    800
  );
};

export const KbGraphCanvas = forwardRef<KbGraphCanvasHandle, KbGraphCanvasProps>(function KbGraphCanvas(
  {
    nodes,
    edges,
    selected,
    onSelect,
    onToggleExpand,
    expanded = false,
    labels = {},
    maxNodeLabels = 48,
    maxRelationLabels = 18
  },
  ref
) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const overlayRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<any>(null);
  // accessors read these refs — selection changes repaint without rebuilding
  const selectedRef = useRef<string | null>(null);
  const neighborsRef = useRef<Set<string> | null>(null);
  // node ids whose overlay label is currently on screen; the hover tooltip
  // reads this to avoid painting the same name twice
  const labeledNodeIdsRef = useRef<Set<string>>(new Set());
  // latest props for effects that must not re-run on every render
  const propsRef = useRef({ nodes, edges, selected, onSelect, maxNodeLabels, maxRelationLabels });
  propsRef.current = { nodes, edges, selected, onSelect, maxNodeLabels, maxRelationLabels };

  // overlay label elements live in refs so a dataset swap can rebuild them
  // in place without tearing down the whole scene
  const nodeLabelElsRef = useRef<Map<string, HTMLDivElement>>(new Map());
  const linkLabelElsRef = useRef<Array<HTMLDivElement | undefined>>([]);
  // the exact data the engine holds: node/edge props plus the link copies
  // passed to graphData (whose endpoints the engine mutates into objects)
  const dataRef = useRef<{ nodes: KbGraphNode[]; edges: KbGraphEdge[]; links: any[]; nodeById: Map<string, KbGraphNode> }>({
    nodes: [],
    edges: [],
    links: [],
    nodeById: new Map()
  });

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

  // build the 3D scene once per mount; datasets flow in through the effect
  // below so a KB switch / filter change re-lays the graph in place (the old
  // build-once-dataset behavior froze the canvas on its first graph forever)
  useEffect(() => {
    const host = hostRef.current;
    if (!host || graphRef.current) return;

    const graph = new ForceGraph3D(host)
      .backgroundColor('rgba(0,0,0,0)')
      .width(host.clientWidth)
      .height(host.clientHeight)
      .graphData({ nodes: [], links: [] })
      .nodeColor((n: any) => {
        const sel = selectedRef.current;
        if (sel && sel !== n.id && !neighborsRef.current?.has(n.id)) return 'rgba(140,140,140,0.25)';
        return n.color;
      })
      .nodeOpacity(0.92)
      // selection emphasis: the picked node and its direct neighbors swell,
      // everything else keeps the base size — "lit up" reads even when the
      // neighbor's own label was decluttered away
      .nodeVal((n: any) => {
        const sel = selectedRef.current;
        if (!sel) return 1;
        if (n.id === sel) return 5;
        return neighborsRef.current?.has(n.id) ? 2.2 : 0.5;
      })
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
        const sid = endpointId(l.source);
        const tid = endpointId(l.target);
        if (sel && sel !== sid && sel !== tid) return 'rgba(140,140,140,0.12)';
        return l.kind === 'relation' ? 'rgba(22,119,255,0.55)' : 'rgba(160,160,160,0.3)';
      })
      .linkWidth((l: any) => {
        const sel = selectedRef.current;
        const touches = sel && (sel === endpointId(l.source) || sel === endpointId(l.target));
        return touches ? 1.6 : l.kind === 'relation' ? 0.9 : 0.5;
      })
      .linkOpacity(0.5)
      .linkDirectionalParticles((l: any) => {
        const sel = selectedRef.current;
        const touches = sel && (sel === endpointId(l.source) || sel === endpointId(l.target));
        return touches ? 2 : 0;
      })
      .linkDirectionalParticleWidth(1.4)
      .onNodeClick((n: any) => {
        propsRef.current.onSelect(n.id);
        const { nodes: curNodes, edges: curEdges } = propsRef.current;
        flyToNeighborhood(graph, curNodes as Positioned[], curEdges, n.id);
      })
      .onBackgroundClick(() => propsRef.current.onSelect(null));

    // spread the two families apart in 3D for a cleaner initial settle
    graph.d3Force('charge')?.strength(-160);
    graph.d3Force('link')?.distance((l: any) => {
      const s = typeof l.source === 'object' ? l.source.id : l.source;
      const t = typeof l.target === 'object' ? l.target.id : l.target;
      return s?.startsWith('article:') && t?.startsWith('article:') ? 120 : 60;
    });
    graphRef.current = graph;
    graph.onEngineStop(() => {
      // a selection made while the engine was still settling (deep link)
      // had its camera fly overwritten by this fit — re-frame around the
      // selection's neighborhood instead of zooming out to the whole graph
      const sel = selectedRef.current;
      const { nodes: curNodes, edges: curEdges } = propsRef.current;
      if (sel) flyToNeighborhood(graph, curNodes as Positioned[], curEdges, sel);
      else graph.zoomToFit(600, 40);
    });

    const hideEl = (el?: HTMLDivElement) => {
      if (el && el.style.display !== 'none') el.style.display = 'none';
    };

    let rafId = 0;
    // DOM overlay carrying node names and relation titles; visibility is
    // recomputed every frame from camera position and selection state
    const paintLabels = () => {
      rafId = requestAnimationFrame(paintLabels);
      const g = graphRef.current;
      const overlayEl = overlayRef.current;
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
      const halfW = w / 2;
      const halfH = h / 2;
      const sel = selectedRef.current;
      const nbrs = neighborsRef.current;
      const { nodes: curNodes, maxNodeLabels: nodeBudget, maxRelationLabels: relationBudget } = propsRef.current;
      const firstCandidates: (ScreenPoint & {
        el: HTMLDivElement;
        centerDist: number;
        id: string;
        color: string;
      })[] = [];
      const relationCandidates: (ScreenPoint & { el: HTMLDivElement; centerDist: number })[] = [];
      // labels hug the selected node on screen — near it or not at all;
      // fall back to screen center when the anchor is behind the camera
      let focusPt = { sx: halfW, sy: halfH };
      if (sel) {
        const sn: any = curNodes.find(x => x.id === sel);
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

      curNodes.forEach(raw => {
        const el = nodeLabelElsRef.current.get(raw.id);
        if (!el) return;
        const n = raw as Positioned;
        const placed =
          typeof n.x === 'number' && typeof n.y === 'number' && typeof n.z === 'number'
            ? projectToScreen({ x: n.x, y: n.y, z: n.z }, view)
            : null;
        if (!placed) {
          hideEl(el);
          return;
        }

        if (sel) {
          // clicked: only the selection and its direct (one-hop) neighbors
          // get names — anything without a direct edge stays unnamed
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
            firstCandidates.push({
              el,
              id: n.id,
              color: n.color,
              ...placed,
              centerDist: distToFocus(placed)
            });
          } else hideEl(el);
          return;
        }

        // nothing selected: no ambient labels in the focus area —
        // names show on hover (tooltip) or on click (neighborhood labels)
        hideEl(el);
      });
      firstCandidates.sort((a, b) => a.centerDist - b.centerDist);
      firstCandidates.forEach((c, i) => {
        if (i >= nodeBudget) {
          hideEl(c.el);
          return;
        }
        labeled.add(c.id);
        c.el.style.display = '';
        c.el.style.opacity = '';
        c.el.style.zIndex = String(Math.max(1, Math.round(1000 - c.depth)));
        c.el.style.transform = `translate(-50%, -50%) translate(${c.sx.toFixed(1)}px, ${(c.sy + ballPx + 5).toFixed(1)}px)`;
        c.el.style.fontWeight = '500';
        // label border matches the neighbor's ball color — the name visually
        // anchors to its node even in a crowded first ring
        c.el.style.borderColor = c.color;
      });

      dataRef.current.links.forEach((l, i) => {
        const el = linkLabelElsRef.current[i];
        if (!el) return;
        const s = (typeof l.source === 'object' ? l.source : dataRef.current.nodeById.get(String(l.source))) as Positioned | undefined;
        const t = (typeof l.target === 'object' ? l.target : dataRef.current.nodeById.get(String(l.target))) as Positioned | undefined;
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
          const sid = typeof l.source === 'object' ? (l.source as unknown as { id: string }).id : l.source;
          const tid = typeof l.target === 'object' ? (l.target as unknown as { id: string }).id : l.target;
          if (sid === sel || tid === sel) {
            if (!onScreen(placed)) {
              hideEl(el);
              return;
            }
            relationCandidates.push({ el, centerDist: distToFocus(placed), ...placed });
          } else hideEl(el);
          return;
        }

        // nothing selected: relation titles stay hidden too —
        // a click on either endpoint reveals them
        hideEl(el);
      });

      // candidates are populated by the links scan above — paint after it
      relationCandidates.sort((a, b) => a.centerDist - b.centerDist);
      relationCandidates.forEach((c, i) => {
        if (i >= relationBudget) {
          hideEl(c.el);
          return;
        }
        c.el.style.display = '';
        c.el.style.zIndex = String(Math.max(1, Math.round(1000 - c.depth)));
        c.el.style.transform = `translate(-50%, -50%) translate(${c.sx.toFixed(1)}px, ${c.sy.toFixed(1)}px)`;
      });

      labeledNodeIdsRef.current = labeled;
    };
    rafId = requestAnimationFrame(paintLabels);

    const ro = new ResizeObserver(() => {
      if (host.clientWidth > 0 && host.clientHeight > 0) {
        graphRef.current?.width(host.clientWidth).height(host.clientHeight);
      }
    });
    ro.observe(host);
    return () => {
      cancelAnimationFrame(rafId);
      ro.disconnect();
      if (overlayRef.current) overlayRef.current.innerHTML = '';
      if (graph._destructor) graph._destructor();
      graphRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // feed dataset changes into the live scene: swap graphData and rebuild the
  // overlay labels — tearing down ForceGraph3D per change would flicker,
  // while the old build-once behavior froze the canvas on its first dataset
  useEffect(() => {
    const graph = graphRef.current;
    if (!graph) return;
    const nodeIds = new Set(nodes.map(n => n.id));
    // copies, not the state objects: the engine mutates each link's string
    // endpoints into node-object references in place, which would silently
    // break every e.source === selection comparison held elsewhere against
    // the state edges
    const links = edges.filter(e => nodeIds.has(e.source) && nodeIds.has(e.target)).map(e => ({ ...e }));
    const nodeById = new Map(nodes.map(n => [n.id, n]));
    dataRef.current = { nodes, edges, links, nodeById };
    graph.graphData({ nodes, links });

    const overlay = overlayRef.current;
    if (overlay) {
      overlay.innerHTML = '';
      const nodeLabelEls = new Map<string, HTMLDivElement>();
      const linkLabelEls: Array<HTMLDivElement | undefined> = [];
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
      nodeLabelElsRef.current = nodeLabelEls;
      linkLabelElsRef.current = linkLabelEls;
    }
  }, [nodes, edges]);

  // repaint accessors when the selection changes; camera stays where the
  // click left it
  useEffect(() => {
    selectedRef.current = selected;
    neighborsRef.current = selected ? neighborsOf.get(selected) || new Set([selected]) : null;
    graphRef.current?.refresh();
  }, [selected, neighborsOf]);

  useImperativeHandle(ref, () => ({
    flyTo: (id: string) => {
      const { nodes: curNodes, edges: curEdges } = propsRef.current;
      // selection state lives with the caller — route through the callback
      propsRef.current.onSelect(id);
      // the engine assigns node coordinates a few frames after graphData —
      // poll briefly until they exist, then fly
      let tries = 0;
      const tryFly = () => {
        const node = (curNodes as Positioned[]).find(x => x.id === id);
        if (!node) return;
        if (typeof node.x !== 'number' && tries++ < 300) {
          requestAnimationFrame(tryFly);
          return;
        }
        flyToNeighborhood(graphRef.current, curNodes as Positioned[], curEdges, id);
      };
      tryFly();
    },
    zoomBy: (factor: number) => {
      const g = graphRef.current;
      if (!g) return;
      const pos = g.cameraPosition();
      const dist = Math.max(Math.hypot(pos.x, pos.y, pos.z), 1);
      const next = Math.max(60, Math.min(2400, dist / factor));
      g.cameraPosition({ x: (pos.x / dist) * next, y: (pos.y / dist) * next, z: (pos.z / dist) * next });
    },
    fitAll: () => graphRef.current?.zoomToFit(700, 40)
  }));

  const controlBtn = 'h-24px w-24px';

  return (
    <div className='relative h-full w-full'>
      <div className='absolute inset-0 touch-none' ref={hostRef} />
      <div className='pointer-events-none absolute inset-0 overflow-hidden' ref={overlayRef} style={{ zIndex: 1 }} />
      <div className='absolute right-6px top-6px z-2 flex flex-col gap-4px'>
        {onToggleExpand && (
          <Tooltip title={expanded ? labels.exitFullscreen || 'Exit fullscreen' : labels.fullscreen || 'Fullscreen'} placement='left'>
            <Button
              className={controlBtn}
              icon={expanded ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
              size='small'
              type='text'
              onClick={onToggleExpand}
            />
          </Tooltip>
        )}
        <Tooltip title={labels.zoomIn || 'Zoom in'} placement='left'>
          <Button className={controlBtn} icon={<ZoomInOutlined />} size='small' type='text' onClick={() => graphRef.current && zoomCamera(graphRef.current, 1 / 1.4)} />
        </Tooltip>
        <Tooltip title={labels.zoomOut || 'Zoom out'} placement='left'>
          <Button className={controlBtn} icon={<ZoomOutOutlined />} size='small' type='text' onClick={() => graphRef.current && zoomCamera(graphRef.current, 1.4)} />
        </Tooltip>
        <Tooltip title={labels.fit || 'Fit view'} placement='left'>
          <Button className={controlBtn} icon={<AimOutlined />} size='small' type='text' onClick={() => graphRef.current?.zoomToFit(700, 40)} />
        </Tooltip>
      </div>
    </div>
  );
});

// zoom helper shared by the toolbar buttons — camera distance clamp mirrors
// the imperative handle so both paths stay in sync
function zoomCamera(g: any, factor: number) {
  const pos = g.cameraPosition();
  const dist = Math.max(Math.hypot(pos.x, pos.y, pos.z), 1);
  const next = Math.max(60, Math.min(2400, dist / factor));
  g.cameraPosition({ x: (pos.x / dist) * next, y: (pos.y / dist) * next, z: (pos.z / dist) * next });
}

export default KbGraphCanvas;
