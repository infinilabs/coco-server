import { MinusCircleOutlined, PlusCircleOutlined } from '@ant-design/icons';
import { Button, Form, Input, Space, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default ({ type }: { readonly type: 'gitea' | 'github' | 'gitlab' }) => {
  const { t } = useTranslation();

  const isGithub = type === 'github';
  const isGitLab = type === 'gitlab';
  const isGitea = type === 'gitea';

  const baseURLTooltips = {
    gitea: 'A Gitea Personal Access Token (PAT) is required.',
    github: 'A GitHub Personal Access Token (PAT) with `repo` scope is required.',
    gitlab: 'A GitLab Personal Access Token (PAT) with `api` scope is required.'
  };

  return (
    <>
      {(isGitLab || isGitea) && (
        <Form.Item
          label={t(`page.datasource.${type}.labels.base_url`, 'Base URL (optional)')}
          name={['config', 'base_url']}
          tooltip={t(
            `page.datasource.${type}.tooltip.base_url`,
            isGitLab
              ? 'The base URL of your self-hosted GitLab instance. Leave blank for GitLab.com.'
              : 'The base URL of your self-hosted Gitea instance. Leave blank for Gitea.com.'
          )}
        >
          <Input
            placeholder={isGitLab ? 'https://gitlab.example.com' : 'https://gitea.example.com'}
            style={{ width: 500 }}
          />
        </Form.Item>
      )}

      <Form.Item
        label={t(`page.datasource.git_commons.labels.token`, 'Personal Access Token')}
        name={['config', 'token']}
        tooltip={t(`page.datasource.${type}.tooltip.token`, baseURLTooltips[type])}
        rules={[
          {
            message: t(`page.datasource.git_commons.error.token_required`, 'Please input your Personal Access Token!'),
            required: true
          }
        ]}
      >
        <Input.Password
          placeholder="YourPersonalAccessToken"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t(`page.datasource.git_commons.labels.owner`, 'Owner')}
        name={['config', 'owner']}
        rules={[
          {
            message: t(
              `page.datasource.git_commons.error.owner_required`,
              isGitLab ? 'Please input the group or user!' : 'Please input the repository owner!'
            ),
            required: true
          }
        ]}
        tooltip={t(
          `page.datasource.git_commons.tooltip.owner`,
          'The username or organization name that owns the repositories.'
        )}
      >
        <Input
          placeholder="e.g., infinilabs"
          style={{ width: 500 }}
        />
      </Form.Item>

      <Form.Item
        label={t(`page.datasource.git_commons.labels.repos`, 'Repositories (optional)')}
        tooltip={t(
          `page.datasource.git_commons.tooltip.repos`,
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
                        message: t(`page.datasource.git_commons.error.repo_required`, 'Repository name is required.'),
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
                  {t(`page.datasource.git_commons.buttons.add_repo`, 'Add Repository')}
                </Button>
              )}
            </div>
          )}
        </Form.List>
      </Form.Item>

      <Form.Item
        initialValue={true}
        label={t(`page.datasource.git_commons.labels.index_issues`, 'Index Issues')}
        name={['config', 'index_issues']}
        tooltip={t(`page.datasource.git_commons.tooltip.index_issues`, 'Whether to index issues for the repositories.')}
        valuePropName="checked"
      >
        <Switch />
      </Form.Item>

      {(isGithub || isGitea) && (
        <Form.Item
          initialValue={true}
          label={t(`page.datasource.${type}.labels.index_pull_requests`, 'Index Pull Requests')}
          name={['config', 'index_pull_requests']}
          valuePropName="checked"
          tooltip={t(
            `page.datasource.${type}.tooltip.index_pull_requests`,
            `Whether to index pull requests for the repositories.`
          )}
        >
          <Switch />
        </Form.Item>
      )}

      {isGitLab && (
        <>
          <Form.Item
            initialValue={true}
            label={t(`page.datasource.gitlab.labels.index_merge_requests`, 'Index Merge Requests')}
            name={['config', 'index_merge_requests']}
            valuePropName="checked"
            tooltip={t(
              `page.datasource.gitlab.tooltip.index_merge_requests`,
              `Whether to index merge requests for the repositories.`
            )}
          >
            <Switch />
          </Form.Item>

          <Form.Item
            initialValue={true}
            label={t('page.datasource.gitlab.labels.index_wikis', 'Index Wikis')}
            name={['config', 'index_wikis']}
            tooltip={t('page.datasource.gitlab.tooltip.index_wikis', 'Whether to index wikis for the repositories.')}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>

          <Form.Item
            initialValue={true}
            label={t('page.datasource.gitlab.labels.index_snippets', 'Index Snippets')}
            name={['config', 'index_snippets']}
            valuePropName="checked"
            tooltip={t(
              'page.datasource.gitlab.tooltip.index_snippets',
              'Whether to index snippets for the repositories.'
            )}
          >
            <Switch />
          </Form.Item>
        </>
      )}
    </>
  );
};
