import { MinusCircleOutlined, PlusCircleOutlined } from '@ant-design/icons';
import { Button, Form, Input, Space, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default () => {
  const { t } = useTranslation();

  return (
    <>
      <Form.Item
        label={t('page.datasource.github.labels.token', 'Personal Access Token')}
        name={['config', 'token']}
        rules={[
          {
            message: t('page.datasource.github.error.token_required', 'Please input your Personal Access Token!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.github.tooltip.token',
          'A GitHub Personal Access Token (PAT) with `repo` scope is required.'
        )}
      >
        <Input.Password
          placeholder="YourPersonalAccessToken"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.github.labels.owner', 'Owner')}
        name={['config', 'owner']}
        rules={[
          {
            message: t('page.datasource.github.error.owner_required', 'Please input the repository owner!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.github.tooltip.owner',
          'The username or organization name that owns the repositories.'
        )}
      >
        <Input
          placeholder="e.g., infinilabs"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.github.labels.repos', 'Repositories (optional)')}
        tooltip={t(
          'page.datasource.github.tooltip.repos',
          'Specific repositories to index. Leave blank to index all repositories for the owner.'
        )}
      >
        <Form.List name={['config', 'repos']}>
          {(fields, { add, remove }) => (
            <div>
              {fields.map(({ key, name, ...restField }, index) => (
                <Space
                  align="baseline"
                  key={key}
                  style={{ display: 'flex', marginBottom: 8 }}
                >
                  <Form.Item
                    {...restField}
                    name={name}
                    style={{ margin: 0 }}
                    rules={[
                      {
                        message: t('page.datasource.github.error.repo_required', 'Repository name is required.'),
                        required: true
                      }
                    ]}
                  >
                    <Input
                      placeholder="coco-server"
                      style={{ width: 440 }}
                    />
                  </Form.Item>
                  <MinusCircleOutlined
                    style={{ color: 'red' }}
                    onClick={() => remove(name)}
                  />
                  {index === fields.length - 1 && (
                    <PlusCircleOutlined
                      style={{ color: 'blue' }}
                      onClick={() => add()}
                    />
                  )}
                </Space>
              ))}
              {fields.length === 0 && (
                <Button
                  icon={<PlusCircleOutlined />}
                  style={{ width: '500px' }}
                  type="dashed"
                  onClick={() => add('')}
                >
                  {t('page.datasource.github.buttons.add_repo', 'Add Repository')}
                </Button>
              )}
            </div>
          )}
        </Form.List>
      </Form.Item>

      <Form.Item
        initialValue={true}
        label={t('page.datasource.github.labels.index_issues', 'Index Issues')}
        name={['config', 'index_issues']}
        tooltip={t('page.datasource.github.tooltip.index_issues', 'Whether to index issues for the repositories.')}
        valuePropName="checked"
      >
        <Switch />
      </Form.Item>

      <Form.Item
        initialValue={true}
        label={t('page.datasource.github.labels.index_pull_requests', 'Index Pull Requests')}
        name={['config', 'index_pull_requests']}
        tooltip={t(
          'page.datasource.github.tooltip.index_pull_requests',
          'Whether to index pull requests for the repositories.'
        )}
        valuePropName="checked"
      >
        <Switch />
      </Form.Item>
    </>
  );
};

