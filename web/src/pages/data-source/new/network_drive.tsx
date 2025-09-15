import { Form, Input } from 'antd';
import { useTranslation } from 'react-i18next';

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { MultiFilePathInput } from '@/components/datasource/type/file_paths';

import { createExtensionValidator } from './validators';

const createEndpointValidator = (t: (key: string, defaultVal: string) => string) => async (_: any, value: string) => {
  if (!value) {
    // `required: true` will work with empty string
    return Promise.resolve();
  }

  // Regex supports: ip:ports (e.g., localhost:445 or [::1]:445)
  const validEndpointRegex = /^(([a-zA-Z0-9.-]+)|(\[[a-fA-F0-9:]+]))(:[0-9]{1,5})$/;
  if (!validEndpointRegex.test(value)) {
    return Promise.reject(
      new Error(
        t(
          'page.datasource.network_drive.error.endpoint_format',
          'Invalid format, please use "host:port" or "[ipv6]:port"'
        )
      )
    );
  }

  return Promise.resolve();
};

const createPathValidator = (t: (key: string, defaultVal: string) => string) => async (_: any, value: string[]) => {
  // check is empty
  if (!value || value.length === 0 || value.every(item => !item)) {
    return Promise.reject(
      new Error(t('page.datasource.network_drive.error.folder_paths', 'Please input folder path!'))
    );
  }
  for (const path of value) {
    const startWithSlash = path.startsWith('/');
    if (startWithSlash) {
      return Promise.reject(
        new Error(t('page.datasource.network_drive.error.folder_paths_prefix', 'Invalid path, cannot start with /'))
      );
    }
  }
  return Promise.resolve();
};

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default () => {
  const { t } = useTranslation();
  const endpointValidator = createEndpointValidator(t);
  const pathValidator = createPathValidator(t);
  const extensionValidator = createExtensionValidator(t);
  return (
    <>
      <Form.Item
        label={t('page.datasource.network_drive.labels.endpoint', 'Endpoint')}
        name={['config', 'endpoint']}
        rules={[
          {
            message: t('page.datasource.network_drive.error.endpoint_required', 'Please input endpoint!'),
            required: true
          },
          { validator: endpointValidator }
        ]}
        tooltip={t(
          'page.datasource.network_drive.tooltip.endpoint',
          'The IP:port of the network drive server, e.g., 127.0.0.1:445.'
        )}
      >
        <Input
          placeholder="192.168.1.100:445"
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.network_drive.labels.share', 'Share')}
        name={['config', 'share']}
        tooltip={t('page.datasource.network_drive.tooltip.share', 'The name of the shared folder.')}
        rules={[
          { message: t('page.datasource.network_drive.error.share_required', 'Please input share!'), required: true }
        ]}
      >
        <Input
          placeholder="shared"
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.network_drive.labels.username', 'Username')}
        name={['config', 'username']}
        rules={[
          {
            message: t('page.datasource.network_drive.error.username_required', 'Please input username!'),
            required: true
          }
        ]}
      >
        <Input
          placeholder="user"
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.network_drive.labels.password', 'Password')}
        name={['config', 'password']}
      >
        <Input.Password style={{ width: 500 }} />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.network_drive.labels.domain', 'Domain (optional)')}
        name={['config', 'domain']}
        tooltip={t('page.datasource.network_drive.tooltip.domain', 'The domain of the user, e.g., WORKGROUP.')}
      >
        <Input
          placeholder="WORKGROUP"
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.network_drive.labels.folder_paths', 'Folder Paths')}
        name={['config', 'paths']}
        rules={[{ validator: pathValidator }]}
        tooltip={t(
          'page.datasource.network_drive.tooltip.folder_paths',
          'Relative paths to the folders you want to scan.'
        )}
      >
        <MultiFilePathInput
          addButtonText={t('page.datasource.file_paths_add', 'Add File Path')}
          placeholder="path/to/your/folder"
        />
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
