import { FileAddOutlined, RobotOutlined, SearchOutlined } from '@ant-design/icons';
import { Button, Empty, Space, Tag } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  reviewed: 'processing',
  published: 'success',
  archived: 'warning'
};

/**
 * KB landing tab: headline numbers, recently touched articles, quick actions — mrdoc-style divided stat band over a
 * dotted-leader recent list.
 */
export function KbOverview({
  kb,
  articles,
  onNewArticle,
  onAiGenerate
}: {
  readonly kb: Api.Wiki.Kb;
  readonly articles: Api.Wiki.Article[];
  readonly onNewArticle: () => void;
  readonly onAiGenerate: () => void;
}) {
  const { t } = useTranslation();
  const nav = useNavigate();

  const published = articles.filter(a => a.status === 'published').length;
  const aiGenerated = articles.filter(a => a.ai_generated).length;
  const recent = [...articles].sort((a, b) => (b.updated_at > a.updated_at ? 1 : -1)).slice(0, 6);

  return (
    // full container width: the header/tabs above span the whole column, so a
    // capped overview column read as "not adaptive" on wide screens
    <div className='w-full flex flex-col gap-14px'>
      <div className='wiki-statband'>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{articles.length}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.articles')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{published}</div>
          <div className='wiki-stat-label'>{t('page.wiki.status.published')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{aiGenerated}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.aiGenerated')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{kb.datasource_ids?.length ?? 0}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.datasources')}</div>
        </div>
      </div>

      <section className='wiki-panel'>
        <div className='wiki-panel-head'>
          <span className='wiki-panel-title'>{t('page.wiki.overview.recent')}</span>
          <Space>
            <Button
              icon={<SearchOutlined />}
              onClick={() => nav('/wiki/list')}
            >
              {t('page.wiki.overview.search')}
            </Button>
            <Button
              icon={<RobotOutlined />}
              onClick={onAiGenerate}
            >
              {t('page.wiki.kb.aiGenerate')}
            </Button>
            <Button
              icon={<FileAddOutlined />}
              type='primary'
              onClick={onNewArticle}
            >
              {t('page.wiki.kb.newArticle')}
            </Button>
          </Space>
        </div>
        <div className='p-16px'>
          {recent.length === 0 ? (
            <Empty
              description={t('page.wiki.hub.empty')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          ) : (
            <div className='flex flex-col'>
              {recent.map(a => (
                <div
                  className='flex cursor-pointer items-center gap-12px rounded-8px px-8px py-7px transition-colors -mx-8px hover:bg-[var(--wiki-panel-2)]'
                  key={a.id}
                  onClick={() => nav(`/wiki/article/${a.id}?kb=${kb.id}`)}
                >
                  <span className='min-w-0 truncate text-14px'>{a.title}</span>
                  {/* dotted leader line filling the gap between title and meta —
                      the classic catalog row */}
                  <span
                    className='h-1px flex-1 border-t border-dotted'
                    style={{ borderColor: 'var(--wiki-border)' }}
                  />
                  <Tag
                    className='shrink-0'
                    color={STATUS_COLOR[a.status]}
                  >
                    {t(`page.wiki.status.${a.status}`)}
                  </Tag>
                  <span className='shrink-0 text-xs color-[var(--wiki-text-3)]'>{a.updated_at}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
