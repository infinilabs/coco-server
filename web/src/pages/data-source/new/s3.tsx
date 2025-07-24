import { QuestionCircleOutlined } from '@ant-design/icons';
import { Checkbox, Form, Input, Tooltip } from 'antd';
import { useTranslation } from 'react-i18next';

import { createExtensionValidator } from './validators';

const createEndpointValidator = (t: (key: string, defaultVal: string) => string) => async (_: any, value: string) => {
  if (!value) {
    // `required: true` will work with empty string
    return Promise.resolve();
  }

  // check whether prefix with `://`
  if (value.includes('://')) {
    return Promise.reject(
      new Error(t('page.datasource.s3.error.endpoint_prefix', 'Endpoint should not contain http:// or https:// prefix'))
    );
  }

  // check whether suffix with `/`
  if (value.endsWith('/')) {
    return Promise.reject(
      new Error(t('page.datasource.s3.error.endpoint_slash', 'Endpoint should not contain a trailing slash /'))
    );
  }

  // Regex
  //    supports:
  //    - hostname (e.g., my-minio.internal, s3.amazonaws.com)
  //    - IPv4 addr (e.g., 127.0.0.1)
  //    - IPv6 addr (e.g., [::1] or [2001:db8::1])
  //    - and all of the above with optional ports (e.g., localhost:9000 or [::1]:9000)
  const validEndpointRegex = /^(([a-zA-Z0-9.-]+)|(\[[a-fA-F0-9:]+\]))(:[0-9]{1,5})?$/;
  if (!validEndpointRegex.test(value)) {
    return Promise.reject(
      new Error(
        t(
          'page.datasource.s3.error.endpoint_format',
          'Invalid format, please use "host", "host:port", "[ipv6]" or "[ipv6]:port"'
        )
      )
    );
  }

  return Promise.resolve();
};

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default () => {
  const { t } = useTranslation();
  const endpointValidator = createEndpointValidator(t);
  const extensionValidator = createExtensionValidator(t);

  return (
    <>
      <Form.Item
        name={['config', 'endpoint']}
        label={
          <span>
            Endpoint&nbsp;
            <Tooltip
              title={t(
                'page.datasource.s3.tooltip.endpoint',
                'Endpoint of your S3 server, like：s3.amazonaws.com or localhost:9000'
              )}
            >
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        rules={[
          { message: t('page.datasource.s3.error.endpoint_required', 'Please input S3 Endpoint!'), required: true },
          { validator: endpointValidator }
        ]}
      >
        <Input
          placeholder="s3.us-east-1.amazonaws.com"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.s3.labels.bucket', 'Bucket Name')}
        name={['config', 'bucket']}
        rules={[
          { message: t('page.datasource.s3.error.bucket_required', 'Please input Bucket name!'), required: true }
        ]}
      >
        <Input
          placeholder="my-data-bucket"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.s3.labels.access_key_id', 'Access Key ID')}
        name={['config', 'access_key_id']}
        rules={[
          {
            message: t('page.datasource.s3.error.access_key_id_required', 'Please input Access Key ID!'),
            required: true
          }
        ]}
      >
        <Input
          autoComplete="off"
          placeholder="AKIAIOSFODNN7EXAMPLE"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.s3.labels.secret_access_key', 'Secret Access Key')}
        name={['config', 'secret_access_key']}
        rules={[
          {
            message: t('page.datasource.s3.error.secret_access_key_required', 'Please input Secret Access Key!'),
            required: true
          }
        ]}
      >
        <Input.Password
          autoComplete="new-password"
          placeholder="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.s3.labels.prefix', 'Object Prefix (optional)')}
        name={['config', 'prefix']}
        tooltip={t('page.datasource.s3.tooltip.prefix', 'Only index objects that begin with this prefix')}
      >
        <Input
          placeholder="documents/2024/"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        initialValue={true}
        label={t('page.datasource.s3.labels.ssl', 'SSL')}
        name={['config', 'use_ssl']}
        valuePropName="checked"
      >
        <Checkbox>{t('page.datasource.s3.labels.use_ssl', 'Use SSL (HTTPS)')}</Checkbox>
      </Form.Item>
      <Form.Item
        label={t('page.datasource.new.labels.file_extensions', 'File Extensions (optional)')}
        name={['config', 'extensions_str']}
        rules={[{ validator: extensionValidator }]}
        tooltip={t('page.datasource.new.tooltip.file_extensions', 'Comma-separated list. e.g., pdf, docx, txt')}
      >
        <Input
          placeholder="pdf, docx, md"
          style={{ width: 500 }}
        />
      </Form.Item>
    </>
  );
};
