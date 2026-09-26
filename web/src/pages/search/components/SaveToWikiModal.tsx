import { BookOutlined } from '@ant-design/icons';
import { Form, Input, Modal, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createWikiArticleFromChat, searchWikiKbs } from '@/service/api';

export interface SaveToWikiPayload {
  content: string;
  question: string;
  id: string;
}

/**
 * Knowledge execution entry: persist an assistant answer into a knowledge
 * base as a draft article. The server stamps it AI-generated with the chat
 * provenance and the D1 review gate stays — the article opens as a draft.
 */
export function SaveToWikiModal({
  payload,
  onClose,
  onSaved
}: {
  payload: SaveToWikiPayload | null;
  onClose: () => void;
  onSaved: (articleId: string, kbId: string) => void;
}) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!payload) return;
    searchWikiKbs().then(res => setKbs(((res as any)?.data || []) as Api.Wiki.Kb[]));
    form.resetFields();
    form.setFieldsValue({ title: payload.question?.slice(0, 80) || '' });
  }, [payload, form]);

  const onOk = () => {
    form.validateFields().then(values => {
      if (!payload) return;
      setSaving(true);
      createWikiArticleFromChat({
        kb_id: values.kb_id,
        title: values.title,
        summary: values.summary,
        content: payload.content,
        message_id: payload.id || undefined
      })
        .then(res => {
          setSaving(false);
          // the request layer surfaces failures; a missing _id means rejected
          const articleId = (res as any)?._id as string;
          if (!articleId) return;
          window.$message?.success(t('page.wiki.saveFromChat.success'));
          onSaved(articleId, values.kb_id);
          onClose();
        })
        .catch(() => setSaving(false));
    });
  };

  return (
    <Modal
      destroyOnHidden
      okText={t('page.wiki.saveFromChat.ok')}
      open={!!payload}
      title={
        <span>
          <BookOutlined className='mr-2' />
          {t('page.wiki.saveFromChat.title')}
        </span>
      }
      confirmLoading={saving}
      onCancel={onClose}
      onOk={onOk}
    >
      <Form className='my-2em' form={form} layout='vertical'>
        <Form.Item label={t('page.wiki.saveFromChat.kb')} name='kb_id' rules={[{ required: true }]}>
          <Select
            options={kbs.map(kb => ({ value: kb.id, label: `${kb.icon || '📚'} ${kb.name}` }))}
            placeholder={t('page.wiki.saveFromChat.kbPlaceholder')}
          />
        </Form.Item>
        <Form.Item label={t('page.wiki.saveFromChat.articleTitle')} name='title' rules={[{ required: true }]}>
          <Input maxLength={200} />
        </Form.Item>
        <Form.Item label={t('page.wiki.saveFromChat.summary')} name='summary'>
          <Input.TextArea rows={2} maxLength={500} />
        </Form.Item>
      </Form>
      <div className='text-12px text-gray-400'>{t('page.wiki.saveFromChat.draftHint')}</div>
    </Modal>
  );
}
