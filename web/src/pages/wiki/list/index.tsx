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
  Form,
  Input,
  List,
  Modal,
  Select,
  Skeleton,
  Tag,
  Tooltip
} from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  createWikiKb,
  deleteWikiKb,
  listWikiAssistants,
  listWikiDatasources,
  searchWikiKbs
} from '@/service/api';
import { SearchModal } from '../components/SearchModal';

const AI_STATUS_COLOR: Record<string, string> = {
  ready: 'success',
  processing: 'processing',
  queued: 'default',
  updating: 'warning'
};

function CreateKbModal({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [assistants, setAssistants] = useState<{ id: string; name: string }[]>([]);
  const [datasources, setDatasources] = useState<Api.Wiki.DatasourceInfo[]>([]);

  useEffect(() => {
    if (!open) return;
    listWikiAssistants().then(list => setAssistants(((list as any) || []).map((a: any) => ({ id: a.id, name: a.name }))));
    listWikiDatasources().then(list => setDatasources(((list as any) || []) as Api.Wiki.DatasourceInfo[]));
  }, [open]);

  const onOk = () => {
    form.validateFields().then(values => {
      setLoading(true);
      createWikiKb(values as Partial<Api.Wiki.Kb>).then(() => {
        setLoading(false);
        form.resetFields();
        window.$message?.success(t('common.addSuccess'));
        onCreated();
        onClose();
      });
    });
  };

  return (
    <Modal
      destroyOnHidden
      okText={t('common.create')}
      open={open}
      title={t('page.wiki.createKb.title')}
      onCancel={onClose}
      onOk={onOk}
    >
      <Form className='my-2em' form={form} layout='vertical'>
        <Form.Item label={t('page.wiki.createKb.name')} name='name' rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item label={t('page.wiki.createKb.description')} name='description'>
          <Input.TextArea rows={2} />
        </Form.Item>
        <div className='flex gap-3'>
          <Form.Item className='flex-1' label={t('page.wiki.createKb.icon')} name='icon' initialValue='📚'>
            <Input />
          </Form.Item>
          <Form.Item className='flex-1' label={t('page.wiki.createKb.visibility')} name='visibility' initialValue='team'>
            <Select
              options={[
                { value: 'public', label: t('page.wiki.visibility.public') },
                { value: 'private', label: t('page.wiki.visibility.private') },
                { value: 'team', label: t('page.wiki.visibility.team') }
              ]}
            />
          </Form.Item>
        </div>
        <Form.Item label={t('page.wiki.createKb.assistant')} name='assistant_id'>
          <Select allowClear options={assistants.map(a => ({ value: a.id, label: a.name }))} />
        </Form.Item>
        <Form.Item label={t('page.wiki.createKb.datasources')} name='datasource_ids'>
          <Select
            allowClear
            mode='multiple'
            options={datasources.map(ds => ({ value: ds.id, label: ds.name }))}
            placeholder={t('page.wiki.createKb.datasourcesPlaceholder')}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [loading, setLoading] = useState(true);
  const [keyword, setKeyword] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);

  const fetchData = (query?: string) => {
    setLoading(true);
    searchWikiKbs({ query }).then(res => {
      setKbs(((res as any).data || []) as Api.Wiki.Kb[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData();
  }, []);

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
    <div className='min-h-500px'>
      <Card bordered={false} className='card-wrapper'>
        <div className='mb-4 mt-4 flex items-center justify-between'>
          <div>
            <div className='text-xl font-semibold'>{t('page.wiki.hub.title')}</div>
            <div className='mt-1 text-gray-500'>{t('page.wiki.hub.subtitle')}</div>
          </div>
          <div className='flex items-center gap-3'>
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
        ) : kbs.length === 0 ? (
          <Empty description={t('page.wiki.hub.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <List
            grid={{ gutter: 16, column: 3, xs: 1, sm: 2 }}
            dataSource={kbs}
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

      <CreateKbModal onClose={() => setCreateOpen(false)} onCreated={() => fetchData(keyword)} open={createOpen} />
      <SearchModal onClose={() => setSearchOpen(false)} open={searchOpen} />
    </div>
  );
}
