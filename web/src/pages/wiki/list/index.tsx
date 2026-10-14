import {
  BookOutlined,
  ClockCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  EllipsisOutlined,
  PlusOutlined,
  SearchOutlined,
  TeamOutlined
} from '@ant-design/icons';
import { Button, Dropdown, Empty, Input, Modal, Select, Skeleton, Tooltip } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  createWikiWorkspace,
  deleteWikiKb,
  searchWikiArticles,
  searchWikiBookmarks,
  searchWikiKbs,
  searchWikiWorkspaces
} from '@/service/api';
import { CreateKbModal } from '../components/CreateKbModal';
import { SearchModal } from '../components/SearchModal';
import { getRecentArticles } from '../shared/recent';
import { WikiShell } from '../components/WikiShell';

const AI_STATUS_PILL: Record<string, { color: string }> = {
  ready: { color: '#27a644' },
  processing: { color: '#1990ff' },
  queued: { color: '#8b9199' },
  updating: { color: '#f2c94c' }
};

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [workspaces, setWorkspaces] = useState<{ id: string; name: string }[]>([]);
  const [workspaceId, setWorkspaceId] = useState<string>('all');
  const [wsCreateOpen, setWsCreateOpen] = useState(false);
  const [wsName, setWsName] = useState('');
  const [bookmarks, setBookmarks] = useState<Api.Wiki.Bookmark[]>([]);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [recents] = useState(() => getRecentArticles());

  const fetchData = (query?: string) => {
    setLoading(true);
    searchWikiKbs({ query }).then(res => {
      setKbs(((res as any).data || []) as Api.Wiki.Kb[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData();
    searchWikiWorkspaces().then(res => setWorkspaces((res as any) || []));
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

  const onCreateWorkspace = () => {
    if (!wsName.trim()) return;
    createWikiWorkspace(wsName.trim()).then(res => {
      if ((res as any)?._id) {
        setWsName('');
        setWsCreateOpen(false);
        searchWikiWorkspaces().then(r => setWorkspaces((r as any) || []));
      }
    });
  };

  const onSearch = (value: string) => {
    setKeyword(value);
    fetchData(value);
  };

  const onDelete = (kb: Api.Wiki.Kb) => {
    window.$modal?.confirm({
      content: t('page.wiki.hub.deleteKbConfirm', { name: kb.name }),
      onOk: () => {
        deleteWikiKb(kb.id).then(() => {
          window.$message?.success(t('common.deleteSuccess'));
          fetchData(keyword);
        });
      }
    });
  };

  const visibilityLabel = (kb: Api.Wiki.Kb) =>
    kb.visibility === 'team'
      ? t('page.wiki.visibility.team')
      : kb.visibility === 'public'
        ? t('page.wiki.visibility.public')
        : t('page.wiki.visibility.private');

  return (
    <WikiShell>
      <div className='wiki-hub-page pb-24px'>
        <header className='wiki-hero'>
          <span className='wiki-hero-eyebrow'>{t('page.wiki.hub.eyebrow')}</span>
          <h1 className='wiki-hero-title'>{t('page.wiki.hub.title')}</h1>
          <p className='wiki-hero-subtitle'>{t('page.wiki.hub.subtitle')}</p>
          <div className='wiki-hero-actions'>
            <Select
              className='w-170px'
              showSearch={false}
              value={workspaceId}
              options={[
                { value: 'all', label: t('page.wiki.workspace.all') },
                ...workspaces.map(w => ({ value: w.id, label: w.name }))
              ]}
              suffixIcon={
                <PlusOutlined
                  onClick={e => {
                    e.stopPropagation();
                    setWsCreateOpen(true);
                  }}
                />
              }
              onChange={setWorkspaceId}
            />
            <Input.Search
              allowClear
              className='w-240px'
              placeholder={t('page.wiki.articles.filterPlaceholder')}
              value={keyword}
              onChange={e => setKeyword(e.target.value)}
              onSearch={onSearch}
            />
            <Tooltip title={t('page.wiki.hub.search')}>
              <Button
                icon={<SearchOutlined />}
                onClick={() => setSearchOpen(true)}
              />
            </Tooltip>
            <Button
              icon={<PlusOutlined />}
              type='primary'
              onClick={() => setCreateOpen(true)}
            >
              {t('page.wiki.hub.newKb')}
            </Button>
          </div>
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
      </div>

      <Modal
        cancelText={t('common.cancel')}
        okText={t('common.create')}
        open={wsCreateOpen}
        title={t('page.wiki.workspace.create')}
        onCancel={() => setWsCreateOpen(false)}
        onOk={onCreateWorkspace}
      >
        <Input
          placeholder={t('page.wiki.workspace.namePlaceholder')}
          value={wsName}
          onChange={e => setWsName(e.target.value)}
        />
      </Modal>

      <CreateKbModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={() => fetchData(keyword)}
      />
      <SearchModal
        open={searchOpen}
        onClose={() => setSearchOpen(false)}
      />
    </WikiShell>
  );
}
