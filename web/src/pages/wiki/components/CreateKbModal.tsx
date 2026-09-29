import { Form, Input, Modal, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createWikiKb, listWikiAssistants, listWikiDatasources } from '@/service/api';

export function CreateKbModal({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: () => void }) {
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
      createWikiKb(values as Partial<Api.Wiki.Kb>).then(res => {
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
