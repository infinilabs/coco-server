import { AimOutlined, ZoomInOutlined, ZoomOutOutlined } from '@ant-design/icons';
import { Button, Empty, Spin, Tooltip } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { searchWikiArticles, searchWikiEntities } from '@/service/api';

interface GNode {
  id: string;
  label: string;
  kind: 'article' | 'entity';
  x: number;
  y: number;
  vx: number;
  vy: number;
  articleId?: string;
}

interface GEdge {
  source: string;
  target: string;
  label: string;
}

const W = 880;
const H = 560;
const ITERATIONS = 320;

/**
 * Knowledge graph for one KB: articles and the entities their wikilinks resolve to,
 * laid out with a small force simulation; typed entity relations become edges.
 * Clicking a node opens the linked article.
 */
export function KbGraph({ kbId }: { kbId: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<GNode[]>([]);
  const [edges, setEdges] = useState<GEdge[]>([]);
  const [scale, setScale] = useState(1);
  const [selected, setSelected] = useState<string | null>(null);
  const dragging = useRef<string | null>(null);
  const svgPoint = useRef({ x: 0, y: 0 });

  useEffect(() => {
    if (!kbId) return;
    setLoading(true);
    Promise.all([searchWikiArticles({ kbId }), searchWikiEntities()]).then(([artRes, entRes]) => {
      const articles = ((artRes as any).data || []) as Api.Wiki.Article[];
      const entities = ((entRes as any) || []) as Api.Wiki.EntityInfo[];
      const entById = new Map<string, Api.Wiki.EntityInfo>(entities.map(e => [e.id, e]));

      const gNodes: GNode[] = [];
      const gEdges: GEdge[] = [];
      const usedEntities = new Set<string>();

      articles.forEach((a, i) => {
        gNodes.push({
          id: `article:${a.id}`,
          label: a.title,
          kind: 'article',
          articleId: a.id,
          x: W / 2 + Math.cos((i / Math.max(articles.length, 1)) * Math.PI * 2) * 220,
          y: H / 2 + Math.sin((i / Math.max(articles.length, 1)) * Math.PI * 2) * 160,
          vx: 0,
          vy: 0
        });
        (a.linked_pages || []).forEach(lp => {
          const entId = lp.entity_id && entById.has(lp.entity_id) ? lp.entity_id : null;
          const nodeKey = entId ? `entity:${entId}` : `lp:${lp.type}:${lp.name}`;
          if (entId) usedEntities.add(entId);
          if (!gNodes.some(n => n.id === nodeKey)) {
            gNodes.push({
              id: nodeKey,
              label: lp.name,
              kind: 'entity',
              articleId: entId ? entById.get(entId)?.article_id : undefined,
              x: W / 2 + (Math.random() - 0.5) * 300,
              y: H / 2 + (Math.random() - 0.5) * 220,
              vx: 0,
              vy: 0
            });
          }
          gEdges.push({ source: `article:${a.id}`, target: nodeKey, label: lp.type });
        });
      });

      // typed entity relations, when both ends are on the canvas
      usedEntities.forEach(id => {
        const ent = entById.get(id);
        ent?.relations?.forEach(rel => {
          if (usedEntities.has(rel.target_id)) {
            gEdges.push({ source: `entity:${id}`, target: `entity:${rel.target_id}`, label: rel.relation });
          }
        });
      });

      // force relaxation
      const byId = new Map(gNodes.map(n => [n.id, n]));
      for (let step = 0; step < ITERATIONS; step++) {
        for (let i = 0; i < gNodes.length; i++) {
          for (let j = i + 1; j < gNodes.length; j++) {
            const a = gNodes[i];
            const b = gNodes[j];
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
        gEdges.forEach(e => {
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
        gNodes.forEach(n => {
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

      setNodes(gNodes);
      setEdges(gEdges.filter(e => byId.has(e.source) && byId.has(e.target)));
      setLoading(false);
    });
  }, [kbId]);

  const nodeById = useMemo(() => new Map(nodes.map(n => [n.id, n])), [nodes]);

  const onPointerMove = (e: React.PointerEvent<SVGSVGElement>) => {
    if (!dragging.current) return;
    const n = nodeById.get(dragging.current);
    if (!n) return;
    const rect = e.currentTarget.getBoundingClientRect();
    n.x = ((e.clientX - rect.left) / rect.width) * W;
    n.y = ((e.clientY - rect.top) / rect.height) * H;
    setNodes([...nodes]);
  };

  const onNodeClick = (n: GNode) => {
    setSelected(n.id);
    if (n.articleId) nav(`/wiki/article/${n.articleId}?kb=${kbId}`);
  };

  if (loading) return <div className="flex h-360px items-center justify-center"><Spin /></div>;
  if (nodes.length === 0) return <Empty description={t('page.wiki.graph.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />;

  return (
    <div>
      <div className="mb-8px flex items-center justify-between">
        <div className="flex items-center gap-12px text-xs color-[var(--ant-color-text-tertiary)]">
          <span>
            <span className="inline-block h-8px w-8px rounded-full" style={{ background: '#1677ff' }} />{' '}
            {t('page.wiki.graph.articles')}
          </span>
          <span>
            <span className="inline-block h-8px w-8px rounded-full" style={{ background: '#722ed1' }} />{' '}
            {t('page.wiki.graph.entities')}
          </span>
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
        <g transform={`scale(${scale}) translate(${(W * (1 - scale)) / (2 * scale)}, ${(H * (1 - scale)) / (2 * scale)})`}>
          {edges.map((e, i) => {
            const a = nodeById.get(e.source);
            const b = nodeById.get(e.target);
            if (!a || !b) return null;
            return (
              <g key={i}>
                <line stroke="var(--ant-color-border)" strokeWidth={1} x1={a.x} y1={a.y} x2={b.x} y2={b.y} />
                {scale > 1.4 && (
                  <text fill="var(--ant-color-text-tertiary)" fontSize={9} textAnchor="middle" x={(a.x + b.x) / 2} y={(a.y + b.y) / 2 - 3}>
                    {e.label}
                  </text>
                )}
              </g>
            );
          })}
          {nodes.map(n => (
            <g
              key={n.id}
              style={{ cursor: n.articleId ? 'pointer' : 'grab' }}
              onPointerDown={e => {
                e.stopPropagation();
                dragging.current = n.id;
                svgPoint.current = { x: n.x, y: n.y };
              }}
              onClick={() => onNodeClick(n)}
            >
              <circle
                cx={n.x}
                cy={n.y}
                fill={n.kind === 'article' ? '#1677ff' : '#722ed1'}
                opacity={selected && selected !== n.id ? 0.45 : 0.9}
                r={n.kind === 'article' ? 9 : 6}
                stroke={selected === n.id ? 'var(--ant-color-text)' : 'none'}
                strokeWidth={2}
              />
              <text dy={n.kind === 'article' ? 22 : 17} fill="var(--ant-color-text)" fontSize={11} textAnchor="middle">
                {n.label.length > 14 ? `${n.label.slice(0, 13)}…` : n.label}
              </text>
            </g>
          ))}
        </g>
      </svg>
    </div>
  );
}
