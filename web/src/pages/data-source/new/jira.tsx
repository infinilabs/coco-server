import { QuestionCircleOutlined } from '@ant-design/icons';
import { Form, Input, Switch, Tooltip } from 'antd';
import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

// Custom validator for endpoint URL
const createEndpointValidator = (t: TFunction) => async (_: any, value: string) => {
  if (!value) return Promise.resolve();

  // Validate URL format
  if (!value.startsWith('http://') && !value.startsWith('https://')) {
    return Promise.reject(
      new Error(t('page.datasource.jira.error.endpoint_prefix', 'URL must start with http:// or https://'))
    );
  }

  try {
    new URL(value);
    return Promise.resolve();
  } catch (e) {
    return Promise.reject(
      new Error(t('page.datasource.jira.error.endpoint_invalid', 'Please enter a valid URL'))
    );
  }
};

export default () => {
  const { t } = useTranslation();
  const endpointValidator = createEndpointValidator(t);

  return (
    <>
      {/* Jira Server URL */}
      <Form.Item
        name={['config', 'endpoint']}
        label={
          <span>
            {t('page.datasource.jira.labels.endpoint', 'Jira Server URL')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.endpoint', 'Your Jira instance URL (e.g., https://your-domain.atlassian.net)')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        rules={[
          { required: true, message: t('page.datasource.jira.error.endpoint_required', 'Please input Jira URL!') },
          { validator: endpointValidator }
        ]}
      >
        <Input placeholder="https://your-domain.atlassian.net" style={{ width: 500 }} />
      </Form.Item>

      {/* Project Key */}
      <Form.Item
        label={
          <span>
            {t('page.datasource.jira.labels.project_key', 'Project Key')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.project_key', 'The Jira project key you want to index (e.g., "COCO")')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        name={['config', 'project_key']}
        rules={[
          { required: true, message: t('page.datasource.jira.error.project_key_required', 'Please input Project Key!') }
        ]}
      >
        <Input placeholder="COCO" style={{ width: 500 }} />
      </Form.Item>

      {/* Username */}
      <Form.Item
        label={
          <span>
            {t('page.datasource.jira.labels.username', 'Username (Optional)')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.username', 'Your Jira account username for authentication')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        name={['config', 'username']}
      >
        <Input autoComplete="off" placeholder="your-username" style={{ width: 500 }} />
      </Form.Item>

      {/* Password / Token */}
      <Form.Item
        label={
          <span>
            {t('page.datasource.jira.labels.token', 'Password / Token')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.token', 'Your password when username is provided (Basic Auth), or your Personal Access Token when username is empty (Bearer Auth)')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        name={['config', 'token']}
      >
        <Input.Password autoComplete="new-password" placeholder="Enter your password or token" style={{ width: 500 }} />
      </Form.Item>

      {/* Index Comments Switch */}
      <Form.Item
        initialValue={false}
        label={
          <span>
            {t('page.datasource.jira.labels.index_comments', 'Index Comments')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.index_comments', 'Whether to index issue comments')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        name={['config', 'index_comments']}
      >
        <Switch size="small" />
      </Form.Item>

      {/* Index Attachments Switch */}
      <Form.Item
        initialValue={false}
        label={
          <span>
            {t('page.datasource.jira.labels.index_attachments', 'Index Attachments')}&nbsp;
            <Tooltip title={t('page.datasource.jira.tooltip.index_attachments', 'Whether to index issue attachments')}>
              <QuestionCircleOutlined />
            </Tooltip>
          </span>
        }
        name={['config', 'index_attachments']}
      >
        <Switch size="small" />
      </Form.Item>
    </>
  );
};
