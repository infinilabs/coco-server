import { useEffect, useState } from 'react';
import { useAuth } from '@/hooks/business/auth';
import { Button, Card, Form, Input, Modal, Select, Space, message,
  Switch
} from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import ModelSelect from '@/pages/ai-assistant/modules/ModelSelect';
import { getEnabledModelProviders } from '@/service/api';
import { formatESSearchResult } from '@/service/request/es';
import {
  deleteWikiKb,
  listWikiDatasources,
  searchWikiWorkspaces,
  updateWikiKb
} from '@/service/api';

/** KB settings: basic info, AI generation (guidance + answering model), datasources and the danger zone. */
export function KbSettings({ kb, onSaved }: { kb: Api.Wiki.Kb; onSaved: () => void }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);
  const [datasources, setDatasources] = useState<Api.Wiki.DatasourceInfo[]>([]);
  const [workspaces, setWorkspaces] = useState<{ id: string; name: string }[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [modelProviders, setModelProviders] = useState<any[]>([]);
  const { hasAuth } = useAuth();
  const canUpdate = hasAuth('coco#wiki_kb/update');
  const canDelete = hasAuth('coco#wiki_kb/delete');

  // the parent refetches after save — re-sync the form with fresh kb data
  // (initialValues only apply on mount)
  useEffect(() => {
    if (!kb) return;
    form.setFieldsValue({
      name: kb.name,
      description: kb.description,
      icon: kb.icon,
      visibility: kb.visibility,
      workspace_id: kb.workspace_id || undefined,
      auto_approve: !!kb.auto_approve,
      experience_capture: !!kb.experience_capture,
      auto_organize_toc: !!kb.auto_organize_toc,
      ai_guidance: kb.ai_guidance || '',
      ai_model: (kb as any).ai_model || undefined,
      toc_layout: (kb.toc_layout || []).map(f => ({
        title: f.title,
        prefixes: (f.path_prefixes || []).join(', '),
        dimensions: (f.dimensions || []).join(', ')
      }))
    });
  }, [kb, form]);

  useEffect(() => {
    listWikiDatasources().then(list => setDatasources(((list as any) || []) as Api.Wiki.DatasourceInfo[]));
    searchWikiWorkspaces().then(list => setWorkspaces(((list as any) || []) as { id: string; name: string }[]));
    getEnabledModelProviders()
      .then((res: any) => {
        // the flat request resolves to a {data, error, response} wrapper —
        // the ES body lives in .data (same unwrap EditForm's useRequest does)
        const es = formatESSearchResult(res?.data);
        setModelProviders(
          (((es.data || []) as any[]) || []).map(item => ({
            ...item,
            // language pool includes vision-capable models: legacy type=vision
            // and the new support_vision flag both mark a vision-capable model
            models: item.models?.filter((m: any) => m.type === 'language' || m.type === 'vision') || []
          }))
        );
      })
      .catch(() => {});
    // mount-only: without deps this refetches (and re-renders) forever
  }, []);

  const onSave = () => {
    form.validateFields().then(values => {
      setSaving(true);
      // layout rows carry prefixes/dimensions as comma strings in the form;
      // the API wants the array form (dimensions slot order: 组织/产品/专题)
      const payload = {
        ...values,
        // an undefined (cleared) model must serialize explicitly — a missing
        // key would leave the stored value alive in the partial update
        ai_model: values.ai_model || { provider_id: '', name: '' },
        toc_layout: ((values.toc_layout || []) as any[])
          .filter(row => row?.title?.trim())
          .map(row => ({
            title: row.title.trim(),
            path_prefixes: (row.prefixes || '')
              .split(/[,，]/)
              .map((s: string) => s.trim())
              .filter(Boolean),
            dimensions: (row.dimensions || '')
              .split(/[,，]/)
              .map((s: string) => s.trim())
              .filter(Boolean)
          }))
      };
      updateWikiKb(kb.id, payload as Partial<Api.Wiki.Kb>).then(res => {
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
      {/* one form spans both cards: the save button (end of the AI-guidance
          card) persists everything below in a single PUT */}
      <Form
        className="pt-12px"
        form={form}
        initialValues={{
          name: kb.name,
          description: kb.description,
          icon: kb.icon,
          visibility: kb.visibility,
          auto_approve: !!kb.auto_approve,
          auto_organize_toc: !!kb.auto_organize_toc,
          datasource_ids: kb.datasource_ids,
          ai_guidance: kb.ai_guidance || '',
          ai_model: (kb as any).ai_model || undefined,
          toc_layout: (kb.toc_layout || []).map(f => ({
            title: f.title,
            prefixes: (f.path_prefixes || []).join(', '),
            dimensions: (f.dimensions || []).join(', ')
          }))
        }}
        labelCol={{ span: 5 }}
        layout="horizontal"
        wrapperCol={{ span: 17 }}
      >
        <Card bordered={false} className="wiki-panel mb-12px" title={t('page.wiki.settings.basic')}>
          <Form.Item label={t('page.wiki.createKb.name')} name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.description')} name="description">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.icon')} name="icon">
            <Input className="w-80px" />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.autoApprove')}
            tooltip={t('page.wiki.settings.autoApproveHint')}
            name="auto_approve"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.experienceCapture')}
            tooltip={t('page.wiki.settings.experienceCaptureHint')}
            name="experience_capture"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.autoOrganizeToc')}
            tooltip={t('page.wiki.settings.autoOrganizeTocHint')}
            name="auto_organize_toc"
            valuePropName="checked"
          >
            <Switch />
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
            label={t('page.wiki.workspace.title')}
            name="workspace_id"
            tooltip={t('page.wiki.workspace.hint')}
          >
            <Select
              allowClear
              options={workspaces.map(w => ({ value: w.id, label: w.name }))}
              placeholder={t('page.wiki.workspace.all')}
            />
          </Form.Item>
          <Form.Item label={t('page.wiki.createKb.datasources')} name="datasource_ids">
            <Select
              allowClear
              mode="multiple"
              options={datasources.map(ds => ({ value: ds.id, label: ds.name }))}
            />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.tocLayout')}
            tooltip={t('page.wiki.settings.tocLayoutHint')}
          >
            <Form.List name="toc_layout">
              {(fields, { add, remove }) => (
                <div className="flex flex-col gap-6px">
                  {fields.map(field => (
                    <Space key={field.key} align="baseline" className="w-full">
                      <Form.Item
                        className="mb-0"
                        name={[field.name, 'title']}
                        rules={[{ required: true, message: t('page.wiki.settings.tocLayoutTitleRequired') }]}
                        noStyle
                      >
                        <Input
                          placeholder={t('page.wiki.settings.tocLayoutDeptPlaceholder')}
                          style={{ width: 160 }}
                        />
                      </Form.Item>
                      <Form.Item className="mb-0" name={[field.name, 'prefixes']} noStyle>
                        <Input
                          placeholder={t('page.wiki.settings.tocLayoutPrefixPlaceholder')}
                          style={{ width: 260 }}
                        />
                      </Form.Item>
                      <Form.Item className="mb-0" name={[field.name, 'dimensions']} noStyle>
                        <Input
                          placeholder={t('page.wiki.settings.tocLayoutDimPlaceholder')}
                          style={{ width: 220 }}
                        />
                      </Form.Item>
                      <Button
                        icon={<DeleteOutlined />}
                        onClick={() => remove(field.name)}
                        type="text"
                      />
                    </Space>
                  ))}
                  <Button
                    block={false}
                    className="w-fit"
                    icon={<PlusOutlined />}
                    onClick={() => add({ title: '', prefixes: '', dimensions: '' })}
                  >
                    {t('page.wiki.settings.tocLayoutAdd')}
                  </Button>
                </div>
              )}
            </Form.List>
          </Form.Item>
        </Card>

        <Card bordered={false} className="wiki-panel mb-12px" title={t('page.wiki.settings.aiGuidance')}>
          <Form.Item
            label={t('page.wiki.settings.aiGuidance')}
            name="ai_guidance"
            tooltip={t('page.wiki.settings.aiGuidanceHint')}
          >
            <Input.TextArea
              placeholder={t('page.wiki.settings.aiGuidancePlaceholder')}
              rows={6}
            />
          </Form.Item>
          <Form.Item
            label={t('page.wiki.settings.aiModel')}
            name="ai_model"
            tooltip={t('page.wiki.settings.aiModelHint')}
          >
            <ModelSelect
              allowClear
              modelType='answering_model'
              namePrefix={['ai_model']}
              placeholder={t('page.wiki.settings.aiModelPlaceholder')}
              providers={modelProviders}
              showSettings={false}
              showTemplate={false}
            />
          </Form.Item>
          <Form.Item className="mb-0" wrapperCol={{ offset: 5 }}>
            {canUpdate && (
              <Button loading={saving} type="primary" onClick={onSave}>
                {t('common.save')}
              </Button>
            )}
          </Form.Item>
        </Card>
      </Form>

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
