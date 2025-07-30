import { QuestionCircleOutlined } from '@ant-design/icons';
import { Form, Input, Switch, Tooltip } from 'antd';
import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

// Confluence Endpoint validator
const createEndpointValidator = (t: TFunction) => async (_: any, value: string) => {
  if (!value) {
    return Promise.resolve();
  }

  // Ensure Endpoint start with http:// or https://
  if (!value.startsWith('http://') && !value.startsWith('https://')) {
    return Promise.reject(
      new Error(t('page.datasource.confluence.error.endpoint_prefix', 'Endpoint must start with http:// or https://'))
    );
  }

  try {
    // use URL constructor to validate the endpoint
    // eslint-disable-next-line no-new
    new URL(value);
    return Promise.resolve();
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
  } catch (e) {
    return Promise.reject(
      new Error(t('page.datasource.confluence.error.endpoint_invalid', 'Please enter a valid URL'))
    );
  }
};

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default () => {
  const { t } = useTranslation();
  const endpointValidator = createEndpointValidator(t);

  return (
    <>
      <Form.Item
        name={['config', 'endpoint']}
        label={
          <span>
            Endpoint&nbsp;
            <Tooltip
              title={t(
                'page.datasource.confluence.tooltip.endpoint',
                'The base URL of your Confluence instance. e.g., https://your-domain.atlassian.net or http://confluence.example.com:8090/wiki'
              )}
            >
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        rules={[
          {
            message: t('page.datasource.confluence.error.endpoint_required', 'Please input Confluence Endpoint!'),
            required: true
          },
          { validator: endpointValidator }
        ]}
      >
        <Input
          placeholder="https://your-wiki.com"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.confluence.labels.space', 'Space Key')}
        name={['config', 'space']}
        rules={[
          {
            message: t('page.datasource.confluence.error.space_required', 'Please input Space Key!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.confluence.tooltip.space',
          'The key of the Confluence space you want to index (e.g., "DS" or "KB").'
        )}
      >
        <Input
          placeholder="MYSPACE"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.confluence.labels.username', 'Username (optional)')}
        name={['config', 'username']}
        tooltip={t(
          'page.datasource.confluence.tooltip.username',
          'Username for authentication. Can be left empty for anonymous access or if using a Personal Access Token.'
        )}
      >
        <Input
          autoComplete="off"
          placeholder="your-email@example.com"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.confluence.labels.token', 'API Token (optional)')}
        name={['config', 'token']}
        tooltip={t(
          'page.datasource.confluence.tooltip.token',
          'Your Confluence Personal Access Token (PAT). This is the recommended authentication method.'
        )}
      >
        <Input.Password
          autoComplete="new-password"
          placeholder="Enter your API or Personal Access Token"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        initialValue={false}
        label={t('page.datasource.confluence.labels.enable_blogposts', 'Index Blog Posts')}
        name={['config', 'enable_blogposts']}
        tooltip={t('page.datasource.confluence.tooltip.enable_blogposts', 'Whether to index blog posts')}
      >
        <Switch size="small" />
      </Form.Item>

      <Form.Item
        initialValue={false}
        label={t('page.datasource.confluence.labels.enable_attachments', 'Index Attachments')}
        name={['config', 'enable_attachments']}
        tooltip={t('page.datasource.confluence.tooltip.enable_attachments', 'Whether to index attachments')}
      >
        <Switch size="small" />
      </Form.Item>
    </>
  );
};
