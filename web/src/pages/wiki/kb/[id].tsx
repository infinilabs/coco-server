import { ArrowLeftOutlined, DeleteOutlined, FileAddOutlined, ReloadOutlined, RobotOutlined, SearchOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag
} from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { createWikiArticle, deleteWikiArticle, getWikiKb, listWikiDatasources, searchWikiArticles } from '@/service/api';
import { GenerateModal } from '../components/GenerateModal';
import { KbGraph } from '../components/KbGraph';
import { WikiEntityManager } from '../components/WikiEntityManager';
import { KbOverview } from '../components/KbOverview';
import { KbSettings } from '../components/KbSettings';
import { WikiShell } from '../components/WikiShell';
import Shares from '@/components/Resource/Shares';
import useResource from '@/components/Resource/hooks/useResource';
import { selectUserInfo } from '@/store/slice/auth';
import { useAuth } from '@/hooks/business/auth';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  reviewed: 'processing',
  published: 'success',
  archived: 'warning'
};

const DS_STATUS_COLOR: Record<string, string> = {
  connected: 'success',
  syncing: 'processing',
  error: 'error',
  disconnected: 'default'
};

function NewArticleModal({
  kbId,
  open,
  onClose,
  onCreated
}: {
  kbId: string;
  open: boolean;
  onClose: () => void;
  onCreated: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const onOk = () => {
    form.validateFields().then(values => {
      setLoading(true);
      createWikiArticle({ ...values, kb_id: kbId }).then(res => {
        setLoading(false);
        const id = ((res as any)?._id as string) || ((res as any)?.id as string) || '';
        if (!id) return; // request layer already surfaced the error toast
        form.resetFields();
        window.$message?.success(t('common.addSuccess'));
        onCreated(id);
        onClose();
      });
    });
  };

  return (
    <Modal
      destroyOnHidden
      okText={t('common.create')}
      open={open}
      title={t('page.wiki.kb.newArticle')}
      onCancel={onClose}
      onOk={onOk}
    >
      <Form className='my-2em' form={form} layout='vertical'>
        <Form.Item label={t('page.wiki.createArticle.title')} name='title' rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label={t('page.wiki.createArticle.pageType')} name='page_type' initialValue='concept'>
          <Select
            options={[
              { value: 'concept', label: t('page.wiki.pageType.concept') },
              { value: 'entity', label: t('page.wiki.pageType.entity') },
              { value: 'source', label: t('page.wiki.pageType.source') }
            ]}
          />
        </Form.Item>
        <Form.Item label={t('page.wiki.createArticle.summary')} name='summary'>
          <Input.TextArea rows={2} />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export function Component() {
  const { t } = useTranslation();
  const { id } = useParams();
  const nav = useNavigate();
  const [kb, setKb] = useState<Api.Wiki.Kb | null>(null);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [newOpen, setNewOpen] = useState(false);
  const [generateOpen, setGenerateOpen] = useState(false);
  const [artKeyword, setArtKeyword] = useState('');
  const [artStatus, setArtStatus] = useState<string>('all');
  const [dsList, setDsList] = useState<Api.Wiki.DatasourceInfo[]>([]);
  // deep link from an article capsule: /wiki/kb/:id?tab=graph&entity=<id>
  // selects the entity and opens its card on the canvas
  const [searchParams] = useSearchParams();
  const deepTab = searchParams.get('tab') || 'overview';
  const focusEntityId = searchParams.get('entity') || undefined;
  const [activeTab, setActiveTab] = useState(deepTab);

  const { addSharesToData } = useResource();
  const userInfo = useAppSelector(selectUserInfo);
  const { hasAuth } = useAuth();
  const canCreateArticle = hasAuth('coco#wiki_article/create');
  const canDeleteArticle = hasAuth('coco#wiki_article/delete');
  const canUpdateKb = hasAuth('coco#wiki_kb/update');
  const canDeleteKb = hasAuth('coco#wiki_kb/delete');

  const fetchAll = () => {
    if (!id) return;
    setLoading(true);
    Promise.all([getWikiKb(id), searchWikiArticles({ kbId: id }), listWikiDatasources()]).then(async ([kbRes, artRes, dsRes]) => {
      const all = ((dsRes as any) || []) as Api.Wiki.DatasourceInfo[];
      setDsList(all.filter(ds => (kbRes as any)?.datasource_ids?.includes(ds.id)));
      let kbObj = (kbRes as any) as Api.Wiki.Kb | null;
      const kbAny = kbObj as any;
      if (kbAny) {
        // attach owner/shares so the collaboration popover has real data
        kbObj = ((await addSharesToData([kbAny], [{ resource_id: kbAny.id, resource_type: 'wiki_kb' }]))?.[0] as Api.Wiki.Kb) || kbObj;
        // the share popover keys off owner/editor identity — guarantee the
        // basics even when the label service is unavailable
        const enriched = kbObj as any;
        enriched.shares = enriched.shares || [];
        enriched.owner = enriched.owner || (enriched._system?.owner_id ? { id: enriched._system.owner_id } : undefined);
        enriched.editor = enriched.editor || (userInfo?.id ? { id: userInfo.id } : undefined);
      }
      setKb(kbObj);
      setArticles(((artRes as any)?.data || []) as Api.Wiki.Article[]);
      setLoading(false);
    });
  };

  useEffect(fetchAll, [id]);

  const goArticle = (articleId: string) => nav(`/wiki/article/${articleId}?kb=${id}`);

  const visibleArticles = articles.filter(
    a =>
      (artStatus === 'all' || a.status === artStatus) &&
      (!artKeyword || a.title.toLowerCase().includes(artKeyword.toLowerCase()) || a.summary?.toLowerCase().includes(artKeyword.toLowerCase()))
  );

  const onDeleteArticle = (articleId: string) => {
    deleteWikiArticle(articleId).then(() => {
      window.$message?.success(t('common.deleteSuccess'));
      fetchAll();
    });
  };

  if (!loading && !kb) {
    return <Empty description={t('page.wiki.kb.notFound')} />;
  }

  return (
    <WikiShell kbId={id}>
    <div className='min-h-500px'>
      <Card
        bordered={false}
        className='card-wrapper'
        title={
          <div className='flex items-center gap-3'>
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => {
                nav('/wiki/list');
              }}
            />
            {!loading && (
              <span className='text-xl'>
                {kb?.icon} {kb?.name}
              </span>
            )}
          </div>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchAll} />
            {canCreateArticle && (
              <Button icon={<RobotOutlined />} onClick={() => setGenerateOpen(true)}>
                {t('page.wiki.kb.aiGenerate')}
              </Button>
            )}
            {canCreateArticle && (
              <Button
                icon={<FileAddOutlined />}
                type="primary"
                onClick={() => {
                  setNewOpen(true);
                }}
              >
                {t('page.wiki.kb.newArticle')}
              </Button>
            )}
          </Space>
        }
      >
        {!loading && (
          <div className='min-w-0'>
            <div>
              <Tabs
                activeKey={activeTab}
                onChange={setActiveTab}
                items={[
                  {
                    key: 'overview',
                    label: t('page.wiki.kb.tabs.overview'),
                    children: (
                      <KbOverview
                        articles={articles}
                        kb={kb as Api.Wiki.Kb}
                        onAiGenerate={() => setGenerateOpen(true)}
                        onNewArticle={() => setNewOpen(true)}
                      />
                    )
                  },
                  {
                    key: 'articles',
                    label: t('page.wiki.kb.tabs.articles'),
                    children: (
                      <div>
                      <div className='mb-12px flex items-center gap-8px'>
                        <Input
                          allowClear
                          className='w-240px'
                          placeholder={t('page.wiki.articles.filterPlaceholder')}
                          prefix={<SearchOutlined />}
                          value={artKeyword}
                          onChange={e => setArtKeyword(e.target.value)}
                        />
                        <Select
                          className='w-140px'
                          value={artStatus}
                          onChange={setArtStatus}
                          options={[
                            { value: 'all', label: t('page.wiki.articles.allStatuses') },
                            ...['draft', 'reviewed', 'published', 'archived'].map(st => ({
                              value: st,
                              label: t(`page.wiki.status.${st}`)
                            }))
                          ]}
                        />
                      </div>
                      <List
                        dataSource={visibleArticles}
                        loading={loading}
                        pagination={{ pageSize: 10, hideOnSinglePage: true }}
                        renderItem={article => (
                          <List.Item
                            className='cursor-pointer'
                            actions={
                              canDeleteArticle
                                ? [
                                    <Popconfirm
                                      key='delete'
                                      title={t('page.wiki.articles.deleteConfirm')}
                                      onConfirm={() => onDeleteArticle(article.id)}
                                    >
                                      <Button
                                        danger
                                        icon={<DeleteOutlined />}
                                        onClick={e => e.stopPropagation()}
                                        size='small'
                                        type='text'
                                      />
                                    </Popconfirm>
                                  ]
                                : undefined
                            }
                            extra={
                              <Space>
                                <Tag color={STATUS_COLOR[article.status]}>
                                  {t(`page.wiki.status.${article.status}`)}
                                </Tag>
                                <span className='text-xs text-gray-400'>{article.updated_at}</span>
                              </Space>
                            }
                            onClick={() => {
                              goArticle(article.id);
                            }}
                          >
                            <List.Item.Meta
                              description={article.summary}
                              title={
                                <Space>
                                  <a>{article.title}</a>
                                  {article.page_type && <Tag>{t(`page.wiki.pageType.${article.page_type}`)}</Tag>}
                                </Space>
                              }
                            />
                          </List.Item>
                        )}
                      />
                      </div>
                    )
                  },
                  {
                    key: 'graph',
                    label: t('page.wiki.kb.tabs.graph'),
                    children: <KbGraph focusEntityId={focusEntityId} kbId={id || ''} />
                  },
                  {
                    key: 'entities',
                    label: t('page.wiki.kb.tabs.entities'),
                    children: <WikiEntityManager kbId={id || ''} />
                  },
                  {
                    key: 'datasources',
                    label: t('page.wiki.kb.tabs.datasources'),
                    children: (
                      <List
                        dataSource={dsList}
                        renderItem={ds => (
                          <List.Item
                            actions={[
                              <Tag key='status' color={DS_STATUS_COLOR[ds.status]}>
                                {ds.status}
                              </Tag>
                            ]}
                          >
                            <List.Item.Meta
                              description={`${ds.document_count} ${t('page.wiki.kb.docsUnit')} · ${t(
                                'page.wiki.kb.lastSynced'
                              )} ${ds.last_synced}`}
                              title={ds.name}
                            />
                          </List.Item>
                        )}
                      />
                    )
                  },
                  {
                    key: 'members',
                    label: t('page.wiki.kb.tabs.members'),
                    children: (
                      <div className="max-w-640px">
                        <div className="mb-12px text-13px color-[var(--ant-color-text-tertiary)]">
                          {t('page.wiki.members.hint')}
                        </div>
                        {kb && (
                          <>
                            <Shares
                              record={kb as any}
                              resource={{ resource_type: 'wiki_kb', resource_id: kb.id }}
                              title={kb.name}
                              onSuccess={fetchAll}
                            />
                            {(!(kb as any).owner || (kb as any).owner?.id === (kb as any).editor?.id) === false && (kb as any).shares?.length === 0 && (
                              <div className="text-13px color-[var(--ant-color-text-tertiary)]">
                                {t('page.wiki.members.ownerOnly', { owner: (kb as any).owner?.title || (kb as any).owner?.id || '' })}
                              </div>
                            )}
                          </>
                        )}
                      </div>
                    )
                  },
                  {
                    key: 'settings',
                    label: t('page.wiki.kb.tabs.settings'),
                    children: <KbSettings kb={kb as Api.Wiki.Kb} onSaved={fetchAll} />
                  }
                ]}
              />
            </div>
          </div>
        )}
      </Card>

      <NewArticleModal
        kbId={id || ''}
        onClose={() => {
          setNewOpen(false);
        }}
        onCreated={articleId => {
          if (articleId) goArticle(articleId);
          else fetchAll();
        }}
        open={newOpen}
      />

      <GenerateModal
        kbId={id || ''}
        open={generateOpen}
        onClose={() => {
          setGenerateOpen(false);
        }}
        onGenerated={fetchAll}
        onOpenArticle={goArticle}
      />
    </div>
    </WikiShell>
  );
}
