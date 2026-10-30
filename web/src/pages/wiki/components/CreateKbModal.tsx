import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { Button, Form, Input, Modal, Select, Space, Switch } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  createWikiKb,
  listWikiDatasources,
  searchWikiWorkspaces
} from '@/service/api';

export function CreateKbModal({
  open,
  onClose,
  onCreated,
  defaultWorkspaceId
}: {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onCreated: () => void;
  /** workspace preselected from the topbar switcher when the modal opens */
  readonly defaultWorkspaceId?: string;
}) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [datasources, setDatasources] = useState<Api.Wiki.DatasourceInfo[]>([]);
  const [workspaces, setWorkspaces] = useState<{ id: string; name: string }[]>([]);

  useEffect(() => {
    if (!open) return;
    // inherit the topbar switcher's workspace so a KB created inside a
    // workspace lands there without manual picking
    form.setFieldsValue({ workspace_id: defaultWorkspaceId || undefined });
    searchWikiWorkspaces().then(list => setWorkspaces(((list as any) || []) as { id: string; name: string }[]));
    listWikiDatasources().then(list => setDatasources(((list as any) || []) as Api.Wiki.DatasourceInfo[]));
  }, [open, defaultWorkspaceId, form]);

  const onOk = () => {
    form.validateFields().then(values => {
      setLoading(true);
      // layout rows carry prefixes/dimensions as comma strings in the form;
      // the API wants the array form — planning departments at creation time
      // means the first compile already lands a departmental tree
      // (dimensions slot order: 组织/产品/专题)
      const payload = {
        ...values,
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
      createWikiKb(payload as Partial<Api.Wiki.Kb>).then(res => {
        setLoading(false);
        // the request layer already surfaced the error toast; a failed
        // create resolves without an _id
        if (!(res as any)?._id) return;
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
      confirmLoading={loading}
      okText={t('common.create')}
      open={open}
      title={t('page.wiki.createKb.title')}
      onCancel={onClose}
      onOk={onOk}
    >
      <Form
        className='my-2em'
        form={form}
        layout='vertical'
      >
        <Form.Item
          label={t('page.wiki.createKb.name')}
          name='name'
          rules={[{ required: true }]}
        >
          <Input />
        </Form.Item>
        <Form.Item
          label={t('page.wiki.createKb.description')}
          name='description'
        >
          <Input.TextArea rows={2} />
        </Form.Item>
        <div className='flex gap-3'>
          <Form.Item
            className='flex-1'
            initialValue='📚'
            label={t('page.wiki.createKb.icon')}
            name='icon'
          >
            <Input />
          </Form.Item>
          <Form.Item
            className='flex-1'
            initialValue={false}
            label={t('page.wiki.settings.autoApprove')}
            name='auto_approve'
            tooltip={t('page.wiki.settings.autoApproveHint')}
            valuePropName='checked'
          >
            <Switch />
          </Form.Item>
          <Form.Item
            className='flex-1'
            initialValue='team'
            label={t('page.wiki.createKb.visibility')}
            name='visibility'
          >
            <Select
              options={[
                { value: 'public', label: t('page.wiki.visibility.public') },
                { value: 'private', label: t('page.wiki.visibility.private') },
                { value: 'team', label: t('page.wiki.visibility.team') }
              ]}
            />
          </Form.Item>
        </div>
        <Form.Item
          label={t('page.wiki.workspace.title')}
          name='workspace_id'
          tooltip={t('page.wiki.workspace.hint')}
        >
          <Select
            allowClear
            options={workspaces.map(w => ({ value: w.id, label: w.name }))}
            placeholder={t('page.wiki.workspace.all')}
          />
        </Form.Item>
        <Form.Item
          label={t('page.wiki.createKb.datasources')}
          name='datasource_ids'
        >
          <Select
            allowClear
            mode='multiple'
            options={datasources.map(ds => ({ value: ds.id, label: ds.name }))}
            placeholder={t('page.wiki.createKb.datasourcesPlaceholder')}
          />
        </Form.Item>
        <Form.Item
          label={t('page.wiki.settings.tocLayout')}
          tooltip={t('page.wiki.settings.tocLayoutHint')}
        >
          <Form.List name='toc_layout' initialValue={[]}>
            {(fields, { add, remove }) => (
              <div className='flex flex-col gap-6px'>
                {fields.map(field => (
                  <Space key={field.key} align='baseline' className='w-full'>
                    <Form.Item
                      className='mb-0'
                      name={[field.name, 'title']}
                      noStyle
                    >
                      <Input
                        placeholder={t('page.wiki.settings.tocLayoutDeptPlaceholder')}
                        style={{ width: 150 }}
                      />
                    </Form.Item>
                    <Form.Item className='mb-0' name={[field.name, 'prefixes']} noStyle>
                      <Input
                        placeholder={t('page.wiki.settings.tocLayoutPrefixPlaceholder')}
                        style={{ width: 230 }}
                      />
                    </Form.Item>
                    <Form.Item className='mb-0' name={[field.name, 'dimensions']} noStyle>
                      <Input
                        placeholder={t('page.wiki.settings.tocLayoutDimPlaceholder')}
                        style={{ width: 200 }}
                      />
                    </Form.Item>
                    <Button
                      icon={<DeleteOutlined />}
                      onClick={() => remove(field.name)}
                      type='text'
                    />
                  </Space>
                ))}
                <Button
                  className='w-fit'
                  icon={<PlusOutlined />}
                  onClick={() => add({ title: '', prefixes: '', dimensions: '' })}
                >
                  {t('page.wiki.settings.tocLayoutAdd')}
                </Button>
              </div>
            )}
          </Form.List>
        </Form.Item>
      </Form>
    </Modal>
  );
}
