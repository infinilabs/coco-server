import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import { Button, Form, Input, Modal, Select, Space } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  BUILTIN_KM_ASSISTANT_ID,
  type KbAssistantOption,
  collectSearchBoundAssistantIds,
  createWikiKb,
  fetchIntegrations,
  groupKbAssistantOptions,
  listWikiAssistants,
  listWikiDatasources
} from '@/service/api';

export function CreateKbModal({
  open,
  onClose,
  onCreated
}: {
  readonly open: boolean;
  readonly onClose: () => void;
  readonly onCreated: () => void;
}) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [assistants, setAssistants] = useState<KbAssistantOption[]>([]);
  const [datasources, setDatasources] = useState<Api.Wiki.DatasourceInfo[]>([]);
  const [searchBoundIds, setSearchBoundIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!open) return;
    listWikiAssistants().then(list => {
      const arr = (list as KbAssistantOption[]) || [];
      // new KBs default to the builtin knowledge-management assistant —
      // keep it selectable even when the seed hasn't landed yet
      setAssistants(
        arr.some(a => a.id === BUILTIN_KM_ASSISTANT_ID)
          ? arr
          : [{ id: BUILTIN_KM_ASSISTANT_ID, name: '知识管理助手', type: 'data_processing' }, ...arr]
      );
    });
    listWikiDatasources().then(list => setDatasources(((list as any) || []) as Api.Wiki.DatasourceInfo[]));
    fetchIntegrations({})
      .then((res: any) => {
        const sources = (res?.data?.hits?.hits || []).map((h: any) => h._source);
        setSearchBoundIds(collectSearchBoundAssistantIds(sources));
      })
      .catch(() => {});
  }, [open]);

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
          initialValue={BUILTIN_KM_ASSISTANT_ID}
          label={t('page.wiki.createKb.assistant')}
          name='assistant_id'
          tooltip={t('page.wiki.settings.agentTooltip')}
        >
          <Select
            allowClear
            options={groupKbAssistantOptions(assistants, t, searchBoundIds)}
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
