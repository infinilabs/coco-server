import {
  BookOutlined,
  DeleteOutlined,
  EditOutlined,
  EllipsisOutlined,
  PlusOutlined,
  SearchOutlined,
  TeamOutlined
} from '@ant-design/icons';
import {
  Avatar,
  Button,
  Card,
  Dropdown,
  Empty,
  Input,
  List,
  Modal,
  Select,
  Skeleton,
  Tag,
  Tooltip
} from 'antd';
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

const AI_STATUS_COLOR: Record<string, string> = {
  ready: 'success',
  processing: 'processing',
  queued: 'default',
  updating: 'warning'
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

  return (
    <WikiShell>
    <div className='min-h-500px'>
      <Card bordered={false} className='card-wrapper'>
        <div className='mb-4 mt-4 flex items-center justify-between'>
          <div>
            <div className='text-xl font-semibold'>{t('page.wiki.hub.title')}</div>
            <div className='mt-1 text-gray-500'>{t('page.wiki.hub.subtitle')}</div>
          </div>
          <div className='flex items-center gap-3'>
            <Select
              className="w-180px"
              showSearch={false}
              value={workspaceId}
              onChange={setWorkspaceId}
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
            />
            <Input.Search
              allowClear
              className='w-260px'
              enterButton={t('common.refresh')}
              value={keyword}
              onChange={e => setKeyword(e.target.value)}
              onSearch={onSearch}
            />
            <Tooltip title={t('page.wiki.hub.search')}>
              <Button icon={<SearchOutlined />} onClick={() => setSearchOpen(true)} />
            </Tooltip>
            <Button icon={<PlusOutlined />} type='primary' onClick={() => setCreateOpen(true)}>
              {t('page.wiki.hub.newKb')}
            </Button>
          </div>
        </div>

        {loading ? (
          <Card.Grid className='w-full'>
            <Skeleton active />
          </Card.Grid>
        ) : visibleKbs.length === 0 ? (
          <Empty description={t('page.wiki.hub.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <List
            grid={{ gutter: 16, column: 3, xs: 1, sm: 2 }}
            dataSource={visibleKbs}
            renderItem={kb => (
              <List.Item>
                <Card
                  hoverable
                  actions={[
                    <span key='articles'>{`${kb.article_count} ${t('page.wiki.hub.articlesUnit')}`}</span>,
                    <span key='members'>{`${kb.members.length} ${t('page.wiki.hub.membersUnit')}`}</span>,
                    <Dropdown
                      key='more'
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
                      <EllipsisOutlined className='cursor-pointer' />
                    </Dropdown>
                  ]}
                  onClick={() => nav(`/wiki/kb/${kb.id}`)}
                >
                  <Card.Meta
                    avatar={<Avatar className='flex items-center justify-center text-xl' src=''>{kb.icon}</Avatar>}
                    description={
                      <div className='h-40px overflow-hidden text-ellipsis-2'>{kb.description}</div>
                    }
                    title={
                      <div className='flex items-center justify-between gap-2'>
                        <span className='truncate'>{kb.name}</span>
                        {kb.ai_status && (
                          <Tag color={AI_STATUS_COLOR[kb.ai_status] || 'default'}>
                            {t(`page.wiki.aiStatus.${kb.ai_status}`)}
                          </Tag>
                        )}
                      </div>
                    }
                  />
                  <div className='mt-3 flex items-center justify-between text-xs text-gray-400'>
                    <span>
                      <TeamOutlined className='mr-1' />
                      {kb.visibility === 'team'
                        ? t('page.wiki.visibility.team')
                        : kb.visibility === 'public'
                          ? t('page.wiki.visibility.public')
                          : t('page.wiki.visibility.private')}
                    </span>
                    <span>
                      <BookOutlined className='mr-1' />
                      {`${t('page.wiki.hub.lastUpdated')} ${kb.last_updated}`}
                    </span>
                  </div>
                </Card>
              </List.Item>
            )}
          />
        )}
      </Card>

      {recents.length > 0 && (
        <Card bordered={false} className='card-wrapper mt-12px' title={t('page.wiki.recent.title')}>
          <div className='flex flex-wrap gap-8px'>
            {recents.map(r => (
              <Button key={r.id} onClick={() => nav(`/wiki/article/${r.id}${r.kb_id ? `?kb=${r.kb_id}` : ''}`)}>
                {r.title}
              </Button>
            ))}
          </div>
        </Card>
      )}

      {bookmarkedArticles.length > 0 && (
        <Card bordered={false} className='card-wrapper mt-12px' title={t('page.wiki.bookmarks.title')}>
          <List
            dataSource={bookmarkedArticles}
            grid={{ gutter: 16, column: 3, xs: 1, sm: 2 }}
            renderItem={article => (
              <List.Item>
                <Card hoverable onClick={() => nav(`/wiki/article/${article.id}`)} size="small">
                  <Card.Meta description={article.summary?.slice(0, 60)} title={article.title} />
                </Card>
              </List.Item>
            )}
          />
        </Card>
      )}

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

      <CreateKbModal onClose={() => setCreateOpen(false)} onCreated={() => fetchData(keyword)} open={createOpen} />
      <SearchModal onClose={() => setSearchOpen(false)} open={searchOpen} />
    </div>
    </WikiShell>
  );
}
