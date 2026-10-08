import {
  CheckCircleFilled,
  CloseCircleFilled,
  ExclamationCircleFilled,
  ReloadOutlined
} from '@ant-design/icons';
import { Alert, Button, Form, Input, InputNumber, List, Spin, Tag } from 'antd';
import { memo, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import '../index.scss';
import { fetchEnvironmentCheck, fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { useAuth } from '@/hooks/business/auth';

const statusIcon = (status: Api.Environment.CheckStatus) => {
  if (status === 'ok') return <CheckCircleFilled className="text-18px color-[var(--ant-color-success)]" />;
  if (status === 'warning') return <ExclamationCircleFilled className="text-18px color-[var(--ant-color-warning)]" />;
  return <CloseCircleFilled className="text-18px color-[var(--ant-color-error)]" />;
};

const Environment = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    update: hasAuth('coco#system/update')
  };

  const { endLoading, loading, startLoading } = useLoading();
  const [checks, setChecks] = useState<Api.Environment.Check[]>([]);
  const [effectiveTika, setEffectiveTika] = useState<Api.Environment.EffectiveTika>();

  const {
    data: settingsData,
    loading: settingsLoading,
    run: runSettings
  } = useRequest(fetchSettings, { manual: true });

  const {
    data: checkData,
    loading: checkLoading,
    run: runCheck
  } = useRequest(fetchEnvironmentCheck, { manual: true });

  useEffect(() => {
    runSettings();
    runCheck();
  }, []);

  useEffect(() => {
    const cfg = (settingsData as any)?.document_processing;
    if (cfg) {
      form.setFieldsValue({
        tika_endpoint: cfg.tika_endpoint,
        tika_timeout_in_seconds: cfg.tika_timeout_in_seconds
      });
    }
  }, [JSON.stringify(settingsData)]);

  useEffect(() => {
    const result = checkData as unknown as Api.Environment.CheckResult | undefined;
    if (!result) return;
    setChecks(result.checks || []);
    setEffectiveTika(result.effective_tika);
  }, [JSON.stringify(checkData)]);

  const overriddenNames = useMemo(
    () => (effectiveTika?.overridden_by_pipelines || []).map(pipeline => pipeline.name || pipeline.id),
    [effectiveTika]
  );

  const handleSave = async () => {
    const params = await form.validateFields();
    startLoading();
    const result = await updateSettings({
      document_processing: {
        tika_endpoint: params.tika_endpoint || '',
        tika_timeout_in_seconds: params.tika_timeout_in_seconds || undefined
      }
    });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
      // the effective address just changed — refresh probes and overrides
      runCheck();
    }
    endLoading();
  };

  const renderCheck = (check: Api.Environment.Check) => (
    <List.Item>
      <List.Item.Meta
        avatar={statusIcon(check.status)}
        title={
          <span className="flex items-center gap-8px">
            {check.name}
            {check.version && <Tag>{check.version}</Tag>}
          </span>
        }
        description={
          <div className="flex flex-col gap-4px">
            {check.detail && (
              <span className="color-[var(--ant-color-text-tertiary)] text-12px break-all">{check.detail}</span>
            )}
            {check.hint && (
              <span className="text-12px">
                {t('page.settings.environment.labels.install_hint')}: <code>{check.hint}</code>
              </span>
            )}
          </div>
        }
      />
    </List.Item>
  );

  return (
    <ListContainer>
      <Spin spinning={settingsLoading || loading}>
        <div className="py-24px">
          <Form className="settings-form" colon={false} form={form} labelAlign="left">
            <Form.Item label={<span className="color-[var(--ant-color-text-tertiary)]">{t('page.settings.environment.labels.tika_service')}</span>}>
              <div className="m-b-8px color-[var(--ant-color-text-tertiary)]">
                {t('page.settings.environment.labels.tika_desc')}
              </div>
              <div className="flex items-start gap-8px">
                <Form.Item
                  className="flex-1"
                  name="tika_endpoint"
                  rules={[
                    {
                      validator: (_, value) => {
                        if (!value) return Promise.resolve();
                        try {
                          const url = new URL(value);
                          if (!['http:', 'https:'].includes(url.protocol) || !url.host) throw new Error();
                        } catch {
                          return Promise.reject(new Error(t('page.settings.environment.labels.tika_invalid')));
                        }
                        return Promise.resolve();
                      }
                    }
                  ]}
                >
                  <Input placeholder={effectiveTika?.endpoint} allowClear />
                </Form.Item>
                <Form.Item name="tika_timeout_in_seconds">
                  <InputNumber
                    min={1}
                    max={3600}
                    addonBefore={t('page.settings.environment.labels.tika_timeout')}
                    addonAfter="s"
                    style={{ width: 240 }}
                  />
                </Form.Item>
                {permissions.update && (
                  <Button type="primary" onClick={() => handleSave()}>
                    {t('common.update')}
                  </Button>
                )}
              </div>
              {overriddenNames.length > 0 && (
                <Alert
                  className="mt-8px"
                  showIcon
                  type="warning"
                  message={t('page.settings.environment.labels.overridden_title')}
                  description={t('page.settings.environment.labels.overridden_desc', {
                    pipelines: overriddenNames.join(', ')
                  })}
                />
              )}
            </Form.Item>
          </Form>

          <div className="m-b-8px flex items-center gap-8px">
            <span className="text-16px font-500">{t('page.settings.environment.labels.checks_title')}</span>
            <Button icon={<ReloadOutlined />} loading={checkLoading} size="small" onClick={() => runCheck()}>
              {t('page.settings.environment.labels.recheck')}
            </Button>
          </div>
          <List dataSource={checks} loading={checkLoading} renderItem={renderCheck} rowKey={check => check.key} />
        </div>
      </Spin>
    </ListContainer>
  );
});

export default Environment;
