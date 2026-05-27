import type { FormInstance } from 'antd';
import type { NamePath } from 'antd/es/form/interface';

import { CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons';
import { Button, Flex, message } from 'antd';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { testDatasourceConnection } from '@/service/api/data-source';

interface TestConnectionButtonProps {
  readonly connectorId: string;
  readonly form: FormInstance;
  readonly configFields?: NamePath[];
}

type TestStatus = 'idle' | 'success' | 'error';

const TestConnectionButton: React.FC<TestConnectionButtonProps> = ({ connectorId, form, configFields }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<TestStatus>('idle');
  const [errorMsg, setErrorMsg] = useState('');

  const handleTest = async () => {
    try {
      if (configFields && configFields.length > 0) {
        await form.validateFields(configFields);
      }
    } catch {
      return;
    }

    setStatus('idle');
    setErrorMsg('');
    setLoading(true);
    try {
      const config = form.getFieldValue('config') || {};
      const res = await testDatasourceConnection({ config, connector_id: connectorId });
      const { data, error } = res as any;
      if (data?.success) {
        setStatus('success');
      } else {
        const errText = data?.error
          || (error?.message ? String(error.message) : null)
          || t('common.testConnectionFailed', 'Connection failed');
        setStatus('error');
        setErrorMsg(errText);
      }
    } catch (e: any) {
      setStatus('error');
      setErrorMsg(e?.message ? String(e.message) : t('common.testConnectionFailed', 'Connection failed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Flex align="center" gap="small">
      <Button loading={loading} onClick={handleTest}>
        {t('common.testConnection')}
      </Button>
      {status === 'success' && (
        <Flex align="center" gap={4}>
          <CheckCircleFilled style={{ color: '#52c41a', fontSize: 16 }} />
          <span style={{ color: '#52c41a', fontSize: 13 }}>
            {t('common.testConnectionSuccess', 'Connection successful')}
          </span>
        </Flex>
      )}
      {status === 'error' && (
        <Flex align="center" gap={4}>
          <CloseCircleFilled style={{ color: '#ff4d4f', fontSize: 16 }} />
          <span style={{ color: '#ff4d4f', fontSize: 13 }}>{errorMsg}</span>
        </Flex>
      )}
    </Flex>
  );
};

export default TestConnectionButton;
