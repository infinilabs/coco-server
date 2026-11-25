import { Empty, Spin, Tag, Tooltip } from 'antd';
import { ArrowDownLeft, ArrowUpRight, FileText, ListTree } from 'lucide-react';
import Markdown from '@infinilabs/markdown';

import { useEffect, useState, type ComponentType, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { wikiTypeColor } from './wikiTypeColor';

// mirrors the wiki console's status tag palette
const STATUS_TAG_COLOR: Record<string, string> = {
  published: 'green',
  reviewed: 'blue',
  proposed: 'orange',
  draft: 'default'
};

interface RelationDetailed {
  relation?: string;
  label?: string;
  target_id?: string;
  target_name?: string;
  target_type?: string;
  source_id?: string;
  source_name?: string;
  source_type?: string;
}

interface NeighborPayload {
  entity?: {
    id?: string;
    name?: string;
    type?: string;
    status?: string;
    aliases?: string[];
    description?: string;
    properties?: Record<string, unknown>;
  };
  display_description?: string;
  property_labels?: Record<string, string>;
  relations_detailed?: RelationDetailed[];
  incoming?: RelationDetailed[];
}

interface Backlink {
  article_id?: string;
  title?: string;
  kb_id?: string;
  snippet?: string;
}

function Section({
  title,
  icon: Icon,
  children
}: {
  title: string;
  icon: ComponentType<{ className?: string }>;
  children: ReactNode;
}) {
  return (
    <div className='mb-16px'>
      <div className='mb-6px inline-flex items-center gap-6px text-13px font-medium text-[#666] dark:text-white/70'>
        <Icon className='size-14px shrink-0 text-[#999] dark:text-white/45' />
        {title}
      </div>
      {children}
    </div>
  );
}

/**
 * Full entity face for the search result detail: identity chips, grounded
 * description, typed properties (schema labels resolved server-side), typed
 * relations in both directions and the wikilink citing articles — the same
 * content the wiki console's entity page shows, fetched from the
 * `neighbors` + `backlinks` endpoints.
 */
export function EntityDetail({
  entityId,
  apiConfig
}: {
  entityId?: string;
  apiConfig?: Record<string, any>;
}) {
  const { t } = useTranslation();
  const [data, setData] = useState<NeighborPayload | null>(null);
  const [backlinks, setBacklinks] = useState<Backlink[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!entityId) {
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const cfg = apiConfig || {};
    const base = String(cfg?.BaseUrl || '').replace(/\/+$/, '');
    const headers: Record<string, string> = { ...(cfg?.headers || {}) };
    if (cfg?.Token) headers['X-API-TOKEN'] = String(cfg.Token);

    Promise.all([
      fetch(`${base}/wiki/entity/${encodeURIComponent(entityId)}/neighbors`, { headers, credentials: 'include' })
        .then(res => (res.ok ? res.json() : null))
        .then(body => (body?.data ?? body ?? {}) as NeighborPayload)
        .catch(() => null),
      fetch(`${base}/wiki/entity/${encodeURIComponent(entityId)}/backlinks`, { headers, credentials: 'include' })
        .then(res => (res.ok ? res.json() : null))
        .then(body => ((body?.data ?? body ?? {}).items || []) as Backlink[])
        .catch(() => [] as Backlink[])
    ])
      .then(([neighbors, links]) => {
        if (cancelled) return;
        setData(neighbors);
        setBacklinks(links);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [entityId, apiConfig]);

  if (!entityId) return null;

  if (loading) {
    return (
      <div className='flex flex-1 items-center justify-center py-40px'>
        <Spin />
      </div>
    );
  }

  const entity = data?.entity;
  const forward = data?.relations_detailed ?? [];
  const incoming = data?.incoming ?? [];
  const props = entity?.properties ?? {};
  const propKeys = Object.keys(props);
  const propLabels = data?.property_labels ?? {};
  const description = data?.display_description || entity?.description || '';

  if (!entity && !description) {
    return (
      <div className='flex flex-1 items-center justify-center py-40px'>
        <Empty description={false} />
      </div>
    );
  }

  return (
    <div className='min-h-0 flex-1 overflow-auto pr-4px'>
      {/* identity chips: type (schema palette), review status, aliases */}
      <div className='mb-12px flex flex-wrap items-center gap-6px'>
        {entity?.type && (
          <span className='font-mono text-12px' style={{ color: wikiTypeColor(entity.type) }}>
            {entity.type}
          </span>
        )}
        {entity?.status && (
          <Tag className='m-0' color={STATUS_TAG_COLOR[entity.status] ?? 'default'}>
            {entity.status}
          </Tag>
        )}
        {(entity?.aliases || []).slice(0, 6).map(alias => (
          <Tag key={alias} className='m-0'>
            {alias}
          </Tag>
        ))}
      </div>

      {description && (
        <div className='mb-16px text-13px leading-22px text-[#333] dark:text-white/85'>
          <Markdown content={description} />
        </div>
      )}

      {propKeys.length > 0 && (
        <Section icon={ListTree} title={t('labels.entityProperties')}>
          <div className='grid grid-cols-1 gap-x-20px gap-y-4px rounded-6px bg-black/3 p-10px dark:bg-white/4'>
            {propKeys.map(key => (
              <div className='flex min-w-0 items-baseline gap-8px' key={key}>
                <span className='w-88px shrink-0 truncate text-12px text-[#999] dark:text-white/45'>
                  {propLabels[key] || key}
                </span>
                <span className='min-w-0 flex-1 break-words text-13px text-[#333] dark:text-white/85'>
                  {String(props[key] ?? '')}
                </span>
              </div>
            ))}
          </div>
        </Section>
      )}

      {(forward.length > 0 || incoming.length > 0) && (
        <Section icon={ArrowUpRight} title={t('labels.entityRelations')}>
          {forward.length === 0 && incoming.length === 0 ? (
            <div className='text-12px text-[#999] dark:text-white/45'>{t('labels.entityNoRelations')}</div>
          ) : (
            <div className='flex flex-col gap-4px'>
              {forward.map((rel, i) => (
                <div className='flex min-w-0 flex-wrap items-center gap-6px text-13px' key={`fw-${i}`}>
                  <ArrowUpRight className='shrink-0 text-[#999] dark:text-white/45' size={13} />
                  <Tooltip title={rel.relation}>
                    <span className='shrink-0 rounded-4px bg-black/4 px-6px py-1px text-12px text-[#666] dark:bg-white/8 dark:text-white/70'>
                      {rel.label || rel.relation}
                    </span>
                  </Tooltip>
                  {rel.target_type && (
                    <span className='font-mono text-11px' style={{ color: wikiTypeColor(rel.target_type) }}>
                      {rel.target_type}
                    </span>
                  )}
                  <span className='min-w-0 truncate text-[#333] dark:text-white/85'>{rel.target_name || rel.target_id}</span>
                </div>
              ))}
              {incoming.map((rel, i) => (
                <div className='flex min-w-0 flex-wrap items-center gap-6px text-13px' key={`in-${i}`}>
                  <ArrowDownLeft className='shrink-0 text-[#999] dark:text-white/45' size={13} />
                  <Tooltip title={rel.relation}>
                    <span className='shrink-0 rounded-4px bg-black/4 px-6px py-1px text-12px text-[#666] dark:bg-white/8 dark:text-white/70'>
                      {rel.label || rel.relation}
                    </span>
                  </Tooltip>
                  {rel.source_type && (
                    <span className='font-mono text-11px' style={{ color: wikiTypeColor(rel.source_type) }}>
                      {rel.source_type}
                    </span>
                  )}
                  <span className='min-w-0 truncate text-[#333] dark:text-white/85'>{rel.source_name || rel.source_id}</span>
                </div>
              ))}
            </div>
          )}
        </Section>
      )}

      {backlinks.length > 0 && (
        <Section icon={FileText} title={t('labels.entityCitedArticles')}>
          <div className='flex flex-col gap-4px'>
            {backlinks.slice(0, 10).map((b, i) => (
              <a
                className='min-w-0 rounded-6px px-8px py-4px text-13px text-[#333] no-underline hover:bg-black/4 hover:text-[--ant-color-primary] dark:text-white/85 dark:hover:bg-white/8'
                href={`#/wiki/article/${b.article_id}?kb=${b.kb_id || ''}`}
                key={`bl-${b.article_id || i}`}
                rel='noreferrer'
                target='_blank'
              >
                <div className='truncate font-medium'>{b.title || b.article_id}</div>
                {b.snippet && (
                  <div className='mt-2px truncate text-12px font-normal text-[#999] dark:text-white/45'>{b.snippet}</div>
                )}
              </a>
            ))}
            {backlinks.length > 10 && (
              <div className='text-12px text-[#999] dark:text-white/45'>+{backlinks.length - 10}</div>
            )}
          </div>
        </Section>
      )}
    </div>
  );
}

export default EntityDetail;
