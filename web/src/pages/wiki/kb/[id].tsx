import { ArrowLeftOutlined, FileAddOutlined, ReloadOutlined, RobotOutlined } from '@ant-design/icons';
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
  Tag
} from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { createWikiArticle, getWikiKb, searchWikiArticles } from '@/service/api';
import { GenerateModal } from '../components/GenerateModal';
import { KbGraph } from '../components/KbGraph';
import { KbOverview } from '../components/KbOverview';
import { KbSettings } from '../components/KbSettings';
import { WikiShell } from '../components/WikiShell';

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

  const fetchAll = () => {
    if (!id) return;
    setLoading(true);
    Promise.all([getWikiKb(id), searchWikiArticles({ kbId: id })]).then(([kbRes, artRes]) => {
      setKb((kbRes as any) as Api.Wiki.Kb);
      setArticles(((artRes as any)?.data || []) as Api.Wiki.Article[]);
      setLoading(false);
    });
  };

  useEffect(fetchAll, [id]);

  const goArticle = (articleId: string) => nav(`/wiki/article/${articleId}?kb=${id}`);

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
            <Button icon={<RobotOutlined />} onClick={() => setGenerateOpen(true)}>
              {t('page.wiki.kb.aiGenerate')}
            </Button>
            <Button
              icon={<FileAddOutlined />}
              type="primary"
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
          <div className='min-w-0'>
            <div>
              <Tabs
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
                    key: 'graph',
                    label: t('page.wiki.kb.tabs.graph'),
                    children: <KbGraph kbId={id || ''} />
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
