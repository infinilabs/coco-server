import { Divider, Empty, List, Spin, Tag, Typography } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { fetchWikiEntityBacklinks, fetchWikiEntityNeighbors, type WikiEntityBacklink } from '@/service/api';
import { wikiTypeColor } from '../shared/WikiLinkTag';

type NeighborData = Awaited<ReturnType<typeof fetchWikiEntityNeighbors>>;
type BacklinkList = WikiEntityBacklink[];

const STATUS_TAG_COLOR: Record<string, string> = {
  published: 'green',
  reviewed: 'blue',
  proposed: 'orange',
  draft: 'default'
};

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-5">
      <div className="mb-2 font-medium">{title}</div>
      {children}
    </div>
  );
}

/**
 * The entity face of a page_type=entity article (ontology O4): the entity's
 * typed properties, its relations in both directions (labels resolved from
 * the schema by the backend) and the wikilink back-references — the articles
 * that cite this entity.
 */
export function EntityArticleSections({ entityId, kbId }: { entityId: string; kbId?: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [data, setData] = useState<NeighborData | null>(null);
  const [backlinks, setBacklinks] = useState<BacklinkList>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!entityId) return;
    setLoading(true);
    Promise.all([fetchWikiEntityNeighbors(entityId), fetchWikiEntityBacklinks(entityId)])
      .then(([neighbors, links]) => {
        setData(neighbors);
        setBacklinks(links);
      })
      .finally(() => setLoading(false));
  }, [entityId]);

  if (loading) {
    return (
      <div className="flex justify-center py-6">
        <Spin />
      </div>
    );
  }

  const entity = data?.entity;
  const forward = data?.relations_detailed ?? [];
  const incoming = data?.incoming ?? [];
  const props = entity?.properties ?? {};
  const propKeys = Object.keys(props);

  return (
    <>
      <Divider />
      {entity && (
        <div className="mb-4 flex flex-wrap items-center gap-2 rounded-6px bg-[var(--ant-color-fill-tertiary)] px-12px py-8px">
          {entity.type && (
            <span className="font-mono text-12px" style={{ color: wikiTypeColor(entity.type) }}>
              {entity.type}
            </span>
          )}
          {entity.status && (
            <Tag className="m-0" color={STATUS_TAG_COLOR[entity.status]}>
              {entity.status}
            </Tag>
          )}
          {(entity.aliases || []).slice(0, 4).map(alias => (
            <Tag key={alias} className="m-0">
              {alias}
            </Tag>
          ))}
          <span className="ml-auto text-12px text-[var(--ant-color-text-tertiary)]">
            {t('page.wiki.entity.relationCount', {
              forward: String(forward.length),
              inverse: String(incoming.length)
            })}
          </span>
        </div>
      )}
      {propKeys.length > 0 && (
        <Section title={t('page.wiki.entity.properties')}>
          <div className="flex flex-col gap-1">
            {propKeys.map(key => (
              <div className="flex gap-3 text-13px" key={key}>
                <span className="w-40 shrink-0 text-[var(--ant-color-text-tertiary)]">{key}</span>
                <span className="min-w-0 break-all">{String(props[key])}</span>
              </div>
            ))}
          </div>
        </Section>
      )}
      <Section title={t('page.wiki.entity.relations')}>
        {forward.length + incoming.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.wiki.entity.noRelations')} />
        ) : (
          <div className="flex flex-col gap-2">
            {forward.map(rel => (
              <div className="flex items-center gap-2 text-13px" key={`f-${rel.target_id}-${rel.relation}`}>
                <span className="text-[var(--ant-color-text-tertiary)]">{t('page.wiki.entity.forward')}</span>
                <Tag className="m-0">{rel.label || rel.relation}</Tag>
                <a
                  className="cursor-pointer"
                  onClick={() => nav(`/wiki/kb/${kbId || ''}?tab=graph&entity=${rel.target_id}`)}
                >
                  {rel.target_name}
                </a>
                {rel.target_type && (
                  <span className="font-mono text-11px" style={{ color: wikiTypeColor(rel.target_type) }}>
                    {rel.target_type}
                  </span>
                )}
              </div>
            ))}
            {incoming.map(rel => (
              <div className="flex items-center gap-2 text-13px" key={`i-${rel.source_id}-${rel.relation}`}>
                <span className="text-[var(--ant-color-text-tertiary)]">{t('page.wiki.entity.inverse')}</span>
                <Tag className="m-0">{rel.inverse_label || rel.label || rel.relation}</Tag>
                <a
                  className="cursor-pointer"
                  onClick={() => nav(`/wiki/kb/${kbId || ''}?tab=graph&entity=${rel.source_id}`)}
                >
                  {rel.source_name}
                </a>
                {rel.source_type && (
                  <span className="font-mono text-11px" style={{ color: wikiTypeColor(rel.source_type) }}>
                    {rel.source_type}
                  </span>
                )}
              </div>
            ))}
          </div>
        )}
      </Section>
      <Section title={t('page.wiki.entity.backlinks')}>
        {backlinks.length === 0 ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.wiki.entity.noBacklinks')} />
        ) : (
          <List
            size="small"
            dataSource={backlinks}
            renderItem={bl => (
              <List.Item
                key={bl.article_id}
                className="cursor-pointer"
                onClick={() => nav(`/wiki/article/${bl.article_id}?kb=${bl.kb_id || kbId || ''}`)}
              >
                <List.Item.Meta
                  title={
                    <span className="flex items-center gap-2">
                      {bl.title}
                      {bl.status && <Tag className="m-0" color={STATUS_TAG_COLOR[bl.status]}>{bl.status}</Tag>}
                      {bl.link_type && (
                        <span className="font-mono text-11px" style={{ color: wikiTypeColor(bl.link_type) }}>
                          {bl.link_type}:{bl.link_name}
                        </span>
                      )}
                    </span>
                  }
                  description={
                    <Typography.Paragraph className="mb-0 text-12px text-[var(--ant-color-text-tertiary)]" ellipsis={{ rows: 2 }}>
                      {bl.snippet}
                    </Typography.Paragraph>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Section>
    </>
  );
}
