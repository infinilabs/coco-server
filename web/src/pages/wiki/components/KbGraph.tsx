import { AimOutlined, CloseOutlined, ExportOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import { Button, Empty, Spin, Tag, Tooltip } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { fetchWikiKbGraph } from '@/service/api';

interface GNode {
  id: string;
  label: string;
  kind: 'article' | 'entity' | 'unresolved';
  x: number;
  y: number;
  vx: number;
  vy: number;
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

const W = 880;
const H = 560;
const ITERATIONS = 320;

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

/**
 * Knowledge graph for one KB (ontology phase O3): the backend assembles the
 * subgraph — articles, wikilink-resolved entities, one-hop relation
 * expansion with schema-resolved labels and inverse relation names.
 * Clicking an entity opens its card; article nodes navigate.
 */
export function KbGraph({ kbId }: { kbId: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<GNode[]>([]);
  const [edges, setEdges] = useState<GEdge[]>([]);
  const [scale, setScale] = useState(1);
  const [selected, setSelected] = useState<string | null>(null);
  const [cardOpen, setCardOpen] = useState(false);
  const dragging = useRef<string | null>(null);

  useEffect(() => {
    if (!kbId) return;
    setLoading(true);
    setSelected(null);
    setCardOpen(false);
    fetchWikiKbGraph(kbId)
      .then(({ nodes: gNodes, edges: gEdges }) => {
        const laid: GNode[] = gNodes.map((n, i) => ({
          id: n.id,
          label: n.label,
          kind: n.kind,
          x: W / 2 + Math.cos((i / Math.max(gNodes.length, 1)) * Math.PI * 2) * 240,
          y: H / 2 + Math.sin((i / Math.max(gNodes.length, 1)) * Math.PI * 2) * 170,
          vx: 0,
          vy: 0,
          color: n.kind === 'article' ? ARTICLE_COLOR : n.kind === 'unresolved' ? UNRESOLVED_COLOR : typeColor(n.type),
          articleId: n.article_id,
          type: n.type,
          typeLabel: n.type_label,
          status: n.status,
          confidence: n.confidence,
          aliases: n.aliases,
          properties: n.properties,
          viaRelation: n.via_relation
        }));
        const byId = new Map(laid.map(n => [n.id, n]));
        const kept = gEdges.filter(e => byId.has(e.source) && byId.has(e.target));

        // force relaxation
        for (let step = 0; step < ITERATIONS; step++) {
          for (let i = 0; i < laid.length; i++) {
            for (let j = i + 1; j < laid.length; j++) {
              const a = laid[i];
              const b = laid[j];
              let dx = b.x - a.x;
              let dy = b.y - a.y;
              let d2 = dx * dx + dy * dy;
              if (d2 < 1) {
                dx = Math.random() - 0.5;
                dy = Math.random() - 0.5;
                d2 = 1;
              }
              const f = 24000 / d2;
              const d = Math.sqrt(d2);
              a.vx -= (dx / d) * f;
              a.vy -= (dy / d) * f;
              b.vx += (dx / d) * f;
              b.vy += (dy / d) * f;
            }
          }
          kept.forEach(e => {
            const a = byId.get(e.source);
            const b = byId.get(e.target);
            if (!a || !b) return;
            const dx = b.x - a.x;
            const dy = b.y - a.y;
            const d = Math.max(Math.sqrt(dx * dx + dy * dy), 1);
            const target = a.kind === 'article' && b.kind === 'article' ? 190 : 110;
            const f = (d - target) * 0.02;
            a.vx += (dx / d) * f;
            a.vy += (dy / d) * f;
            b.vx -= (dx / d) * f;
            b.vy -= (dy / d) * f;
          });
          laid.forEach(n => {
            n.vx += (W / 2 - n.x) * 0.002;
            n.vy += (H / 2 - n.y) * 0.002;
            n.vx *= 0.82;
            n.vy *= 0.82;
            n.x += Math.max(-14, Math.min(14, n.vx));
            n.y += Math.max(-14, Math.min(14, n.vy));
            n.x = Math.max(40, Math.min(W - 40, n.x));
            n.y = Math.max(30, Math.min(H - 30, n.y));
          });
        }

        setNodes(laid);
        setEdges(
          kept.map(e => ({
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

  const nodeById = useMemo(() => new Map(nodes.map(n => [n.id, n])), [nodes]);

  // neighbours of the selected node stay bright, everything else dims
  const neighbors = useMemo(() => {
    if (!selected) return null;
    const set = new Set<string>([selected]);
    edges.forEach(e => {
      if (e.source === selected) set.add(e.target);
      if (e.target === selected) set.add(e.source);
    });
    return set;
  }, [selected, edges]);

  const cardNode = cardOpen && selected ? nodeById.get(selected) : null;
  const cardOutgoing = useMemo(
    () => (cardNode ? edges.filter(e => e.kind === 'relation' && e.source === cardNode.id) : []),
    [cardNode, edges]
  );
  const cardIncoming = useMemo(
    () => (cardNode ? edges.filter(e => e.kind === 'relation' && e.target === cardNode.id) : []),
    [cardNode, edges]
  );

  const onPointerMove = (e: React.PointerEvent<SVGSVGElement>) => {
    if (!dragging.current) return;
    const n = nodeById.get(dragging.current);
    if (!n) return;
    const rect = e.currentTarget.getBoundingClientRect();
    n.x = ((e.clientX - rect.left) / rect.width) * W;
    n.y = ((e.clientY - rect.top) / rect.height) * H;
    setNodes([...nodes]);
  };

  const focusNode = (id: string) => {
    setSelected(id);
    setCardOpen(true);
  };

  const onNodeClick = (n: GNode) => {
    if (n.kind === 'article') {
      nav(`/wiki/article/${n.articleId ?? n.id.replace(/^article:/, '')}?kb=${kbId}`);
      return;
    }
    focusNode(n.id);
  };

  if (loading) return <div className="flex h-360px items-center justify-center"><Spin /></div>;
  if (nodes.length === 0) return <Empty description={t('page.wiki.graph.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />;

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
          <span className="hidden md:inline">{t('page.wiki.graph.clickArticleHint')}</span>
        </div>
        <div className="flex gap-4px">
          <Button icon={<ZoomOutOutlined />} onClick={() => setScale(s => Math.max(0.4, s - 0.2))} size="small" />
          <Button icon={<ZoomInOutlined />} onClick={() => setScale(s => Math.min(2.5, s + 0.2))} size="small" />
          <Tooltip title={t('page.wiki.graph.reset')}>
            <Button icon={<AimOutlined />} onClick={() => setScale(1)} size="small" />
          </Tooltip>
        </div>
      </div>
      <svg
        className="w-full touch-none"
        viewBox={`0 0 ${W} ${H}`}
        style={{ height: 560, cursor: dragging.current ? 'grabbing' : 'default' }}
        onPointerMove={onPointerMove}
        onPointerUp={() => {
          dragging.current = null;
        }}
        onPointerLeave={() => {
          dragging.current = null;
        }}
      >
        <defs>
          <marker id="kg-arrow" markerHeight="6" markerWidth="6" orient="auto" refX="8" refY="3">
            <path d="M0,0 L0,6 L7,3 z" fill="var(--ant-color-border)" />
          </marker>
        </defs>
        <g transform={`scale(${scale}) translate(${(W * (1 - scale)) / (2 * scale)}, ${(H * (1 - scale)) / (2 * scale)})`}>
          {edges.map((e, i) => {
            const a = nodeById.get(e.source);
            const b = nodeById.get(e.target);
            if (!a || !b) return null;
            const dimmed = neighbors && !(neighbors.has(e.source) && neighbors.has(e.target));
            const isRelation = e.kind === 'relation';
            return (
              <g key={i} opacity={dimmed ? 0.18 : 1}>
                <line
                  markerEnd={isRelation ? 'url(#kg-arrow)' : undefined}
                  stroke={isRelation ? 'var(--ant-color-primary-4)' : 'var(--ant-color-border)'}
                  strokeDasharray={isRelation ? undefined : '4 3'}
                  strokeWidth={isRelation ? 1.4 : 1}
                  x1={a.x}
                  y1={a.y}
                  x2={b.x}
                  y2={b.y}
                />
                {(isRelation || scale > 1.4) && (
                  <text fill="var(--ant-color-text-tertiary)" fontSize={9} textAnchor="middle" x={(a.x + b.x) / 2} y={(a.y + b.y) / 2 - 3}>
                    {e.label || e.relation}
                  </text>
                )}
              </g>
            );
          })}
          {nodes.map(n => {
            const dimmed = neighbors ? !neighbors.has(n.id) : selected && selected !== n.id;
            return (
              <g
                key={n.id}
                opacity={dimmed ? 0.3 : 1}
                style={{ cursor: n.kind === 'article' ? 'pointer' : 'grab' }}
                onPointerDown={e => {
                  e.stopPropagation();
                  dragging.current = n.id;
                }}
                onClick={() => onNodeClick(n)}
              >
                {n.kind === 'unresolved' ? (
                  <circle cx={n.x} cy={n.y} fill="transparent" r={6} stroke={n.color} strokeDasharray="2 2" strokeWidth={1.5} />
                ) : (
                  <circle
                    cx={n.x}
                    cy={n.y}
                    fill={n.color}
                    r={n.kind === 'article' ? 9 : 6.5}
                    stroke={selected === n.id ? 'var(--ant-color-text)' : 'none'}
                    strokeWidth={2}
                  />
                )}
                <text dy={n.kind === 'article' ? 22 : 17} fill="var(--ant-color-text)" fontSize={11} textAnchor="middle">
                  {n.label.length > 14 ? `${n.label.slice(0, 13)}…` : n.label}
                </text>
              </g>
            );
          })}
        </g>
      </svg>

      {cardNode && (
        <div
          className="absolute top-40px right-0 z-10 w-280px rounded-6px border-1px border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)] p-12px shadow-md"
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
