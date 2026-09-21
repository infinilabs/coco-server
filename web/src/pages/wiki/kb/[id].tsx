import { ArrowLeftOutlined, FileAddOutlined, ReloadOutlined } from '@ant-design/icons';
import {
  Avatar,
  Button,
  Card,
  Empty,
  Form,
  Input,
  List,
  Modal,
  Select,
  Space,
  Tabs,
  Tag,
  Tree
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import { useEffect, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  createWikiArticle,
  getWikiKb,
  getWikiToc,
  searchWikiArticles,
  updateWikiToc
} from '@/service/api';
import { findNode, moveNode, toAntdTreeData, dropPositionFromAntd } from '../shared/toc';

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
        form.resetFields();
        window.$message?.success(t('common.addSuccess'));
        onCreated(((res as any)?.id as string) || '');
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
  const [searchParams] = useSearchParams();
  const [kb, setKb] = useState<Api.Wiki.Kb | null>(null);
  const [toc, setToc] = useState<Api.Wiki.TocNode[]>([]);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [newOpen, setNewOpen] = useState(false);
  const selectedTocId = searchParams.get('toc') || '';

  const fetchAll = () => {
    if (!id) return;
    setLoading(true);
    Promise.all([getWikiKb(id), getWikiToc(id), searchWikiArticles({ kbId: id })]).then(([kbRes, tocRes, artRes]) => {
      setKb((kbRes as any) as Api.Wiki.Kb);
      setToc(((tocRes as any) || []) as Api.Wiki.TocNode[]);
      setArticles(((artRes as any)?.data || []) as Api.Wiki.Article[]);
      setLoading(false);
    });
  };

  useEffect(fetchAll, [id]);

  const goArticle = (articleId: string) => nav(`/wiki/article/${articleId}?kb=${id}`);

  const onTreeSelect = (keys: React.Key[]) => {
    const key = String(keys[0] || '');
    const node = findNode(toc, key);
    if (node?.type === 'article' && node.article_id) goArticle(node.article_id);
  };

  const onTreeDrop = (info: {
    node: DataNode;
    dragNode: DataNode;
    dropToGap: boolean;
    dropPosition: number;
  }) => {
    const draggedId = String(info.dragNode.key);
    const targetId = String(info.node.key);
    const target = findNode(toc, targetId);
    const position = dropPositionFromAntd(info.dropToGap, info.dropPosition, target?.type === 'folder');
    const next = moveNode(toc, draggedId, targetId, position);
    setToc(next);
    if (id) updateWikiToc(id, next);
  };

  if (!loading && !kb) {
    return <Empty description={t('page.wiki.kb.notFound')} />;
  }

  return (
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
            <Button
              icon={<FileAddOutlined />}
              type='primary'
              onClick={() => {
                setNewOpen(true);
              }}
            >
              {t('page.wiki.kb.newArticle')}
            </Button>
          </Space>
        }
      >
        {!loading && (
          <div className='flex flex-col gap-4 lg:flex-row'>
            <Card
              className='w-full shrink-0 lg:w-260px'
              size='small'
              title={t('page.wiki.kb.toc')}
              styles={{ body: { maxHeight: 500, overflow: 'auto' } }}
            >
              <Tree
                blockNode
                defaultExpandAll
                draggable
                selectedKeys={selectedTocId ? [selectedTocId] : []}
                treeData={toAntdTreeData(toc)}
                onDrop={onTreeDrop as any}
                onSelect={onTreeSelect}
              />
            </Card>
            <div className='min-w-0 flex-1'>
              <Tabs
                items={[
                  {
                    key: 'articles',
                    label: t('page.wiki.kb.tabs.articles'),
                    children: (
                      <List
                        dataSource={articles}
                        loading={loading}
                        pagination={{ pageSize: 10, hideOnSinglePage: true }}
                        renderItem={article => (
                          <List.Item
                            className='cursor-pointer'
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
                    )
                  },
                  {
                    key: 'datasources',
                    label: t('page.wiki.kb.tabs.datasources'),
                    children: (
                      <List
                        dataSource={kb?.datasources || []}
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
                      <List
                        dataSource={kb?.members || []}
                        renderItem={member => (
                          <List.Item
                            actions={[
                              <Tag key='role' color={member.role === 'owner' ? 'gold' : member.role === 'agent' ? 'purple' : 'default'}>
                                {t(`page.wiki.role.${member.role}`)}
                              </Tag>
                            ]}
                          >
                            <List.Item.Meta
                              avatar={<Avatar>{member.avatar}</Avatar>}
                              description={member.email}
                              title={member.name}
                            />
                          </List.Item>
                        )}
                      />
                    )
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
    </div>
  );
}
