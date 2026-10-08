import { useEffect, useState } from 'react';
import { useAuth } from '@/hooks/business/auth';
import { Button, Card, Form, Input, Modal, Select, Space, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  collectSearchBoundAssistantIds,
  deleteWikiKb,
  fetchIntegrations,
  groupKbAssistantOptions,
  listWikiAssistants,
  listWikiDatasources,
  updateWikiKb,
  type KbAssistantOption
} from '@/service/api';

/** KB settings: basic info, AI agent, datasources and the danger zone. */
export function KbSettings({ kb, onSaved }: { kb: Api.Wiki.Kb; onSaved: () => void }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [assistants, setAssistants] = useState<KbAssistantOption[]>([]);
  const [datasources, setDatasources] = useState<Api.Wiki.DatasourceInfo[]>([]);
  const [searchBoundIds, setSearchBoundIds] = useState<Set<string>>(new Set());
  const [deleteOpen, setDeleteOpen] = useState(false);
  const { hasAuth } = useAuth();
  const canUpdate = hasAuth('coco#wiki_kb/update');
  const canDelete = hasAuth('coco#wiki_kb/delete');

  useEffect(() => {
    listWikiAssistants().then(list => setAssistants((list as KbAssistantOption[]) || []));
    listWikiDatasources().then(list => setDatasources(((list as any) || []) as Api.Wiki.DatasourceInfo[]));
    // assistants referenced by search integrations get their own AI 搜索 group
    fetchIntegrations({})
      .then((res: any) => {
        const sources = (res?.data?.hits?.hits || []).map((h: any) => h._source);
        setSearchBoundIds(collectSearchBoundAssistantIds(sources));
      })
      .catch(() => {});
    // mount-only: without deps this refetches (and re-renders) forever
  }, []);

  const onSave = () => {
    form.validateFields().then(values => {
      setSaving(true);
      updateWikiKb(kb.id, values as Partial<Api.Wiki.Kb>).then(res => {
        setSaving(false);
        if ((res as any)?.id || res !== undefined) {
          message.success(t('page.wiki.settings.saved'));
          onSaved();
        }
      });
    });
  };

  const onDelete = () => {
    deleteWikiKb(kb.id).then(() => {
      message.success(t('common.deleteSuccess'));
      nav('/wiki/list');
    });
  };

  return (
    <div className="w-full">
      <Card bordered={false} className="wiki-panel mb-12px" title={t('page.wiki.settings.basic')}>
        <Form
          className="pt-12px"
          form={form}
          initialValues={{
            name: kb.name,
            description: kb.description,
            icon: kb.icon,
            visibility: kb.visibility,
            assistant_id: kb.assistant_id,
            datasource_ids: kb.datasource_ids
          }}
          labelCol={{ span: 5 }}
          layout="horizontal"
          wrapperCol={{ span: 17 }}
        >
          <Form.Item label={t('page.wiki.createKb.name')} name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.description')} name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.icon')} name="icon">
            <Input className="w-80px" />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.visibility')} name="visibility">
            <Select
              options={[
                { value: 'public', label: t('page.wiki.visibility.public') },
                { value: 'private', label: t('page.wiki.visibility.private') },
                { value: 'team', label: t('page.wiki.visibility.team') }
              ]}
            />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.agent')}
            name="assistant_id"
            tooltip={t('page.wiki.settings.agentTooltip')}
          >
            <Select allowClear options={groupKbAssistantOptions(assistants, t, searchBoundIds)} />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.datasources')} name="datasource_ids">
            <Select
              allowClear
              mode="multiple"
              options={datasources.map(ds => ({ value: ds.id, label: ds.name }))}
            />
          </Form.Item>
          <Form.Item className="mb-0" wrapperCol={{ offset: 5 }}>
            {canUpdate && (
              <Button loading={saving} type="primary" onClick={onSave}>
                {t('common.save')}
              </Button>
            )}
          </Form.Item>
        </Form>
      </Card>

      <Card bordered={false} className="wiki-panel" title={t('page.wiki.settings.danger')}>
        <Space>
          {canDelete && (
            <Button danger onClick={() => setDeleteOpen(true)}>
              {t('page.wiki.settings.deleteKb')}
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        cancelText={t('common.cancel')}
        okText={t('common.delete')}
        okButtonProps={{ danger: true }}
        open={deleteOpen}
        title={t('page.wiki.settings.deleteKb')}
        onCancel={() => setDeleteOpen(false)}
        onOk={onDelete}
      >
        {t('page.wiki.hub.deleteKbConfirm', { name: kb.name })}
      </Modal>
    </div>
  );
}
