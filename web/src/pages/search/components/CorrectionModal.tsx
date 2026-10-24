import { FlagOutlined } from '@ant-design/icons';
import { Form, Input, Modal, Select } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { reportWikiCorrection } from '@/service/api';

export interface CorrectionPayload {
  content: string;
  question: string;
  id: string;
}

/**
 * Work-side correction capture (D7): report a wrong or outdated answer.
 * The correction lands in the governance queue with the query, an answer
 * excerpt and a routing hint — a human reviews it before anything changes.
 */
export function CorrectionModal({ payload, onClose }: { payload: CorrectionPayload | null; onClose: () => void }) {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!payload) return;
    form.resetFields();
    form.setFieldsValue({ route_hint: 'fact_outdated' });
  }, [payload, form]);

  const onOk = () => {
    form.validateFields().then(values => {
      if (!payload) return;
      setSaving(true);
      reportWikiCorrection({
        query: payload.question,
        answer: payload.content,
        message_id: payload.id || undefined,
        route_hint: values.route_hint,
        comment: values.comment
      })
        .then(() => {
          setSaving(false);
          window.$message?.success(t('page.wiki.correction.success'));
          onClose();
        })
        .catch(() => setSaving(false));
    });
  };

  return (
    <Modal
      destroyOnHidden
      okText={t('page.wiki.correction.ok')}
      open={!!payload}
      title={
        <span>
          <FlagOutlined className='mr-2' />
          {t('page.wiki.correction.title')}
        </span>
      }
      confirmLoading={saving}
      onCancel={onClose}
      onOk={onOk}
    >
      <Form className='my-2em' form={form} layout='vertical'>
        <Form.Item label={t('page.wiki.correction.routeHint')} name='route_hint' rules={[{ required: true }]}>
          <Select
            options={[
              { value: 'fact_missing', label: t('page.wiki.correction.hints.fact_missing') },
              { value: 'fact_outdated', label: t('page.wiki.correction.hints.fact_outdated') },
              { value: 'preference', label: t('page.wiki.correction.hints.preference') },
              { value: 'technique', label: t('page.wiki.correction.hints.technique') },
              { value: 'source_conflict', label: t('page.wiki.correction.hints.source_conflict') }
            ]}
          />
        </Form.Item>
        <Form.Item label={t('page.wiki.correction.comment')} name='comment' rules={[{ required: true }]}>
          <Input.TextArea rows={4} maxLength={4000} placeholder={t('page.wiki.correction.commentPlaceholder')} />
        </Form.Item>
      </Form>
      <div className='text-12px text-gray-400'>{t('page.wiki.correction.reviewHint')}</div>
    </Modal>
  );
}
