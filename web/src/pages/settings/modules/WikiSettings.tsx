import { Button, Divider, Form, Select, Spin } from 'antd';
import '../index.scss';
import { useEffect, useState } from 'react';
import {
  BUILTIN_KM_ASSISTANT_ID,
  type KbAssistantOption,
  collectSearchBoundAssistantIds,
  fetchIntegrations,
  groupKbAssistantOptions,
  listWikiAssistants
} from '@/service/api';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';

/**
 * AI 知识库 system settings: the default assistant that curates the wiki. Knowledge bases can still bind their own
 * assistant (KB settings → AI 智能体) — that per-KB binding overrides this system default.
 */
const WikiSettings = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { hasAuth } = useAuth();

  const permissions = {
    update: hasAuth('coco#system/update')
  };

  const { endLoading, loading, startLoading } = useLoading();

  const [assistants, setAssistants] = useState<KbAssistantOption[]>([]);
  const [searchBoundIds, setSearchBoundIds] = useState<Set<string>>(new Set());

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  useEffect(() => {
    run();
    listWikiAssistants().then(list => {
      const arr = (list as KbAssistantOption[]) || [];
      // keep the builtin selectable even when the seed hasn't landed yet
      setAssistants(
        arr.some(a => a.id === BUILTIN_KM_ASSISTANT_ID)
          ? arr
          : [{ id: BUILTIN_KM_ASSISTANT_ID, name: '知识管理助手', type: 'data_processing' }, ...arr]
      );
    });
    fetchIntegrations({})
      .then((res: any) => {
        const sources = (res?.data?.hits?.hits || []).map((h: any) => h._source);
        setSearchBoundIds(collectSearchBoundAssistantIds(sources));
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (data?.wiki_settings?.assistant_id) {
      form.setFieldsValue({ assistant_id: data.wiki_settings.assistant_id });
    } else {
      form.setFieldsValue({ assistant_id: BUILTIN_KM_ASSISTANT_ID });
    }
  }, [JSON.stringify(data)]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();
    const wiki_settings = {
      assistant_id: params.assistant_id || ''
    };
    const result = await updateSettings({ wiki_settings });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className='settings-form py-24px'
          colon={false}
          form={form}
          labelAlign='left'
        >
          <Form.Item
            label={t('page.settings.wiki_settings.labels.assistant')}
            name='assistant_id'
            tooltip={t('page.settings.wiki_settings.labels.assistant_desc')}
          >
            <Select
              allowClear
              className='max-w-600px'
              options={groupKbAssistantOptions(assistants, t, searchBoundIds)}
              placeholder={t('page.settings.wiki_settings.labels.assistant_placeholder')}
            />
          </Form.Item>

          <Divider />

          <div className='mb-8px color-[var(--ant-color-text)] font-medium'>
            {t('page.settings.wiki_settings.labels.default_hint_title')}
          </div>
          <div className='settings-form-help mb-24px'>{t('page.settings.wiki_settings.labels.default_hint')}</div>
          {permissions.update && (
            <Button
              loading={loading}
              type='primary'
              onClick={handleSubmit}
            >
              {t('common.save')}
            </Button>
          )}
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default WikiSettings;
