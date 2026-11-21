import {
  BookOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  EllipsisOutlined,
  PlusOutlined,
  TeamOutlined
} from '@ant-design/icons';
import { Dropdown, Empty, Skeleton } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { deleteWikiKb, searchWikiArticles, searchWikiBookmarks, searchWikiKbs } from '@/service/api';
import { useAuth } from '@/hooks/business/auth';
import { getRecentArticles } from '../shared/recent';
import { CreateKbModal } from '../components/CreateKbModal';
import { WikiShell } from '../components/WikiShell';

const AI_STATUS_PILL: Record<string, { color: string }> = {
  ready: { color: '#27a644' },
  processing: { color: '#1990ff' },
  queued: { color: '#8b9199' },
  updating: { color: '#f2c94c' }
};

export function Component() {
  // the workspace switcher lives in the shell topbar; the hub only consumes
  // the selected workspace to filter its card wall (refreshKbs re-syncs the
  // shell tree + wall when a KB is created/deleted)
  return (
    <WikiShell>
      {({ kbVersion, refreshKbs, workspaceId }) => (
        <HubPage
          kbVersion={kbVersion}
          refreshKbs={refreshKbs}
          workspaceId={workspaceId}
        />
      )}
    </WikiShell>
  );
}

function HubPage({ kbVersion, refreshKbs, workspaceId }: { kbVersion: number; refreshKbs: () => void; workspaceId: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const { hasAuth } = useAuth();
  const canCreateKb = hasAuth('coco#wiki_kb/create');
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [bookmarks, setBookmarks] = useState<Api.Wiki.Bookmark[]>([]);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [recents] = useState(() => getRecentArticles());

  // KB search lives in the shell sidebar now (tree filter + global search
  // modal); the hub keeps browsing: cards, recents, bookmarks
  const fetchData = () => {
    setLoading(true);
    searchWikiKbs().then(res => {
      setKbs(((res as any).data || []) as Api.Wiki.Kb[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData();
  }, [kbVersion]);

  useEffect(() => {
    searchWikiBookmarks().then(res => setBookmarks(((res as any).data || []) as Api.Wiki.Bookmark[]));
    searchWikiArticles({}).then(res => setArticles(((res as any).data || []) as Api.Wiki.Article[]));
  }, []);

  const visibleKbs = useMemo(
    () => (workspaceId === 'all' ? kbs : kbs.filter(kb => kb.workspace_id === workspaceId)),
    [kbs, workspaceId]
  );

  const bookmarkedArticles = useMemo(() => {
    const ids = new Set(bookmarks.map(b => b.article_id));
    return articles.filter(a => ids.has(a.id));
  }, [bookmarks, articles]);

  const onDelete = (kb: Api.Wiki.Kb) => {
    window.$modal?.confirm({
      content: t('page.wiki.hub.deleteKbConfirm', { name: kb.name }),
      onOk: () => {
        deleteWikiKb(kb.id).then(() => {
          window.$message?.success(t('common.deleteSuccess'));
          fetchData();
        });
      }
    });
  };

  // dashed "new knowledge base" tile trailing the card wall (moved here from
  // the shell topbar); creation lands in the workspace picked in the topbar
  const createCard = (
    <div
      className='wiki-kb-card wiki-kb-card-create flex cursor-pointer flex-col items-center justify-center gap-8px border-dashed text-center !text-[var(--wiki-text-3)] transition-colors hover:border-[var(--ant-color-primary)] hover:!text-[var(--ant-color-primary)]'
      key='create-kb-card'
      onClick={() => setCreateOpen(true)}
    >
      <PlusOutlined className='text-22px' />
      <span className='text-14px font-500'>{t('page.wiki.hub.newKb')}</span>
    </div>
  );

  const visibilityLabel = (kb: Api.Wiki.Kb) =>
    kb.visibility === 'team'
      ? t('page.wiki.visibility.team')
      : kb.visibility === 'public'
        ? t('page.wiki.visibility.public')
        : t('page.wiki.visibility.private');

  return (
    <div className='wiki-hub-page pb-24px'>
      <header className='wiki-hero'>
        <span className='wiki-hero-eyebrow'>{t('page.wiki.hub.eyebrow')}</span>
        <h1 className='wiki-hero-title'>{t('page.wiki.hub.title')}</h1>
        <p className='wiki-hero-subtitle'>{t('page.wiki.hub.subtitle')}</p>
      </header>

        {loading ? (
          <div className='grid grid-cols-1 gap-14px 2xl:grid-cols-4 sm:grid-cols-2 xl:grid-cols-3'>
            {[0, 1, 2].map(i => (
              <div
                className='wiki-kb-card'
                key={i}
              >
                <Skeleton
                  active
                  paragraph={{ rows: 2 }}
                  title={{ width: '60%' }}
                />
              </div>
            ))}
          </div>
        ) : visibleKbs.length === 0 ? (
          <div className='wiki-panel mx-auto max-w-480px px-24px py-36px text-center'>
            <Empty
              description={t('page.wiki.hub.empty')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
            {canCreateKb && createCard}
          </div>
        ) : (
          <div className='grid grid-cols-1 gap-14px 2xl:grid-cols-4 sm:grid-cols-2 xl:grid-cols-3'>
            {visibleKbs.map(kb => {
              const ai = kb.ai_status ? AI_STATUS_PILL[kb.ai_status] : undefined;
              return (
                <div
                  className='wiki-kb-card group'
                  key={kb.id}
                  onClick={() => nav(`/wiki/kb/${kb.id}`)}
                >
                  <div className='wiki-kb-card-head'>
                    <div className='wiki-kb-icon-tile'>{kb.icon}</div>
                    <div className='min-w-0 flex-1'>
                      <div className='flex items-center justify-between gap-8px'>
                        <span className='truncate text-15px color-[var(--wiki-text)] font-600'>{kb.name}</span>
                        {ai && (
                          <span className='wiki-pill'>
                            <i
                              className='wiki-pill-dot'
                              style={{ background: ai.color, opacity: 1 }}
                            />
                            {t(`page.wiki.aiStatus.${kb.ai_status}`)}
                          </span>
                        )}
                      </div>
                      <div className='mt-4px flex items-center gap-6px text-12px color-[var(--wiki-text-3)]'>
                        <TeamOutlined />
                        {visibilityLabel(kb)}
                      </div>
                    </div>
                    <Dropdown
                      menu={{
                        items: [
                          {
                            key: 'edit',
                            label: t('common.edit'),
                            icon: <EditOutlined />,
                            onClick: () => nav(`/wiki/kb/${kb.id}`)
                          },
                          {
                            key: 'delete',
                            danger: true,
                            label: t('common.delete'),
                            icon: <DeleteOutlined />,
                            onClick: () => onDelete(kb)
                          }
                        ]
                      }}
                    >
                      <EllipsisOutlined
                        className='cursor-pointer color-[var(--wiki-text-3)] opacity-0 transition-opacity group-hover:opacity-100'
                        onClick={e => e.stopPropagation()}
                      />
                    </Dropdown>
                  </div>
                  <div className='wiki-kb-card-desc'>{kb.description}</div>
                  <div className='wiki-kb-card-foot'>
                    <span className='inline-flex items-center gap-5px'>
                      <BookOutlined />
                      {`${kb.article_count} ${t('page.wiki.hub.articlesUnit')}`}
                    </span>
                    <span className='inline-flex items-center gap-5px'>
                      <TeamOutlined />
                      {`${kb.members.length} ${t('page.wiki.hub.membersUnit')}`}
                    </span>
                    <span className='spacer' />
                    <span className='inline-flex items-center gap-5px'>
                      <ClockCircleOutlined />
                      {`${t('page.wiki.hub.lastUpdated')} ${kb.last_updated}`}
                    </span>
                  </div>
                </div>
              );
            })}
            {canCreateKb && createCard}
          </div>
        )}

        {recents.length > 0 && (
          <section className='wiki-panel mt-16px'>
            <div className='wiki-panel-head'>
              <span className='wiki-panel-title'>
                <ClockCircleOutlined className='color-[var(--wiki-accent)]' />
                {t('page.wiki.recent.title')}
              </span>
            </div>
            <div className='flex flex-wrap gap-8px p-16px'>
              {recents.map(r => (
                <button
                  className='wiki-pill wiki-pill-clickable'
                  key={r.id}
                  onClick={() => nav(`/wiki/article/${r.id}${r.kb_id ? `?kb=${r.kb_id}` : ''}`)}
                >
                  <BookOutlined />
                  {r.title}
                </button>
              ))}
            </div>
          </section>
        )}

        {bookmarkedArticles.length > 0 && (
          <section className='wiki-panel mt-16px'>
            <div className='wiki-panel-head'>
              <span className='wiki-panel-title'>{t('page.wiki.bookmarks.title')}</span>
            </div>
            <div className='grid grid-cols-1 gap-10px p-16px 2xl:grid-cols-4 sm:grid-cols-2 xl:grid-cols-3'>
              {bookmarkedArticles.map(article => (
                <div
                  className='wiki-kb-card !p-14px'
                  key={article.id}
                  onClick={() => nav(`/wiki/article/${article.id}`)}
                >
                  <div className='text-14px color-[var(--wiki-text)] font-550'>{article.title}</div>
                  <div className='wiki-kb-card-desc !mt-8px !min-h-36px'>{article.summary?.slice(0, 90)}</div>
                </div>
              ))}
            </div>
          </section>
        )}

      <CreateKbModal
        defaultWorkspaceId={workspaceId === 'all' ? undefined : workspaceId}
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => {
          fetchData();
          // re-sync the shell tree + bump kbVersion (this wall re-fetches too)
          refreshKbs();
        }}
      />
    </div>
  );
}
