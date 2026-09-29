import { MessageOutlined, RocketOutlined } from '@ant-design/icons';
import { BookOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Select, Skeleton, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  instantiateAssistantTemplate,
  searchAssistantTemplates,
  searchWikiKbs
} from '@/service/api';

interface AssistantTemplateItem {
  id: string;
  name: string;
  title: string;
  category: string;
  icon: string;
  description: string;
  role_prompt: string;
  suggested_questions: string[];
  builtin: boolean;
  sort_order: number;
}

const CATEGORY_COLOR: Record<string, string> = {
  support: 'blue',
  sales: 'gold',
  hr: 'green',
  it: 'purple',
  productivity: 'cyan'
};

/**
 * Scenario template gallery: one click turns a preset (curated role prompt +
 * suggested questions) into a working assistant, optionally bound to a wiki
 * knowledge base — the KB's datasources become the assistant's data scope
 * and the KB records the binding back.
 */
export function Component() {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState<AssistantTemplateItem[]>([]);
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [loading, setLoading] = useState(true);
  const [instantiating, setInstantiating] = useState<AssistantTemplateItem | null>(null);
  const [creating, setCreating] = useState(false);
  const [form] = Form.useForm();

  const fetchTemplates = () => {
    setLoading(true);
    searchAssistantTemplates().then(list => {
      setTemplates((((list as any) || []) as AssistantTemplateItem[]).sort((a, b) => (a.sort_order || 0) - (b.sort_order || 0)));
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchTemplates();
  }, []);

  useEffect(() => {
    if (!instantiating) return;
    searchWikiKbs().then(res => setKbs(((res as any)?.data || []) as Api.Wiki.Kb[]));
    form.resetFields();
    form.setFieldsValue({ name: instantiating.title });
  }, [instantiating, form]);

  const onOk = () => {
    form.validateFields().then(values => {
      if (!instantiating) return;
      setCreating(true);
      instantiateAssistantTemplate(instantiating.id, {
        name: values.name,
        kb_id: values.kb_id || undefined
      })
        .then(res => {
          setCreating(false);
          const assistantId = (res as any)?._id as string;
          if (!assistantId) return; // request layer surfaced the error
          window.$message?.success(t('page.assistantTemplate.created'));
          setInstantiating(null);
          // land straight in the chat with the new assistant
          window.location.hash = `#/chat?mode=chat&assistant_id=${assistantId}`;
        })
        .catch(() => setCreating(false));
    });
  };

  return (
    <div className='min-h-500px'>
      <Card bordered={false} className='card-wrapper'>
        <div className='mb-4 mt-4 flex items-center justify-between'>
          <div>
            <div className='text-xl font-semibold'>
              <RocketOutlined className='mr-2' />
              {t('page.assistantTemplate.title')}
            </div>
            <div className='mt-1 text-gray-500'>{t('page.assistantTemplate.subtitle')}</div>
          </div>
        </div>

        {loading ? (
          <Card.Grid className='w-full'>
            <Skeleton active />
          </Card.Grid>
        ) : templates.length === 0 ? (
          <Empty description={t('page.assistantTemplate.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <div className='grid grid-cols-1 gap-16px sm:!grid-cols-2 xl:!grid-cols-3'>
            {templates.map(template => (
              <Card
                hoverable
                key={template.id}
                actions={[
                  <Button
                    key='use'
                    icon={<MessageOutlined />}
                    onClick={() => setInstantiating(template)}
                    type='primary'
                  >
                    {t('page.assistantTemplate.use')}
                  </Button>
                ]}
              >
                <Card.Meta
                  avatar={<span className='text-3xl'>{template.icon || '🤖'}</span>}
                  description={
                    <div className='h-56px overflow-hidden text-ellipsis-2'>{template.description}</div>
                  }
                  title={
                    <div className='flex items-center justify-between gap-2'>
                      <span className='truncate'>{template.title}</span>
                      {template.category && (
                        <Tag color={CATEGORY_COLOR[template.category] || 'default'}>
                          {t(`page.assistantTemplate.category.${template.category}`)}
                        </Tag>
                      )}
                    </div>
                  }
                />
                {template.suggested_questions?.length > 0 && (
                  <div className='mt-3 space-y-1'>
                    {template.suggested_questions.slice(0, 3).map(q => (
                      <div className='truncate text-xs text-gray-400' key={q}>
                        · {q}
                      </div>
                    ))}
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}
      </Card>

      <Modal
        destroyOnHidden
        okText={t('page.assistantTemplate.create')}
        open={!!instantiating}
        title={`${instantiating?.icon || ''} ${t('page.assistantTemplate.useTitle', { title: instantiating?.title || '' })}`}
        confirmLoading={creating}
        onCancel={() => {
          setInstantiating(null);
        }}
        onOk={onOk}
      >
        <Form className='my-2em' form={form} layout='vertical'>
          <Form.Item label={t('page.assistantTemplate.name')} name='name' rules={[{ required: true }]}>
            <Input maxLength={100} />
          </Form.Item>
          <Form.Item label={t('page.assistantTemplate.bindKb')} name='kb_id'>
            <Select
              allowClear
              options={kbs.map(kb => ({ value: kb.id, label: `${kb.icon || '📚'} ${kb.name}` }))}
              placeholder={t('page.assistantTemplate.bindKbPlaceholder')}
              suffixIcon={<BookOutlined />}
            />
          </Form.Item>
        </Form>
        <div className='text-12px text-gray-400'>{t('page.assistantTemplate.bindHint')}</div>
      </Modal>
    </div>
  );
}
