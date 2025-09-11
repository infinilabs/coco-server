import type { FormProps } from 'antd';
import { Button, Form, Input, Modal, Spin, Switch, message } from 'antd';
import { useForm } from 'antd/es/form/Form';
import { useTranslation } from 'react-i18next';
import { useNavigate, useLocation } from 'react-router-dom';
import { useState, useEffect } from 'react';

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { DataSync } from '@/components/datasource/data_sync';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { Types } from '@/components/datasource/type';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { IconSelector } from '@/pages/connector/new/icon_selector';
import { getConnector, getConnectorIcons } from '@/service/api/connector';
import { createDatasource } from '@/service/api/data-source';

import Confluence from './confluence';
import Gitea from './gitea';
import GitHub from './github';
import GitLab from './gitlab';
import HugoSite from './hugo_site';
import LocalFS from './local_fs';
import { GiteaConfig, GithubConfig, GitlabConfig, NetworkDriveConfig, RdbmsConfig } from './models';
import NetworkDrive from './network_drive';
import Notion from './notion';
import Rdbms from './rdbms';
import Rss from './rss';
import S3 from './s3';
import Yuque from './yuque';
import OAuthConnect, { OAuthValidationPresets } from '@/components/oauth_connect';

// eslint-disable-next-line complexity
export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const type = params.get('type') ?? Types.GoogleDrive;
  const [connector, setConnector] = useState<any>({});
  const [form] = useForm();

  // Check if connector supports OAuth based on config
  const supportsOAuth = connector?.config?.auth_url !== undefined;

  // Choose appropriate validation rules based on connector type
  const getValidationRules = () => {
    // If connector specifies validation rules in metadata, use those
    if (connector?.validationRules) {
      return connector.validationRules;
    }

    // For known OAuth connectors, use appropriate validation
    switch (type) {
      case Types.GoogleDrive:
        // Google Drive needs all standard OAuth fields
        return OAuthValidationPresets.googleDrive;

      case Types.GitHub:
      case Types.GitLab:
      case Types.Gitea:
        // Standard OAuth 2.0 - all 5 fields required
        return OAuthValidationPresets.standard;

      case 'feishu':
      case 'lark':
        // Feishu/Lark have hardcoded endpoints in backend, only need credentials
        return OAuthValidationPresets.feishuLark;

      default:
        // For unknown connectors, default to standard validation
        // Users can override by providing validationRules prop
        return undefined; // Will use component's default (standard validation)
    }
  };

  const validationRules = getValidationRules();
  const getConnectorTypeName = () => {
    if (connector?.name) return connector.name;

    // Fallback to type-based names for backward compatibility
    switch (type) {
      case Types.Yuque:
        return 'Yuque';
      case Types.Notion:
        return 'Notion';
      case Types.HugoSite:
        return 'Hugo Site';
      case Types.RSS:
        return 'RSS';
      case Types.LocalFS:
        return 'Local FS';
      case Types.S3:
        return 'S3';
      case Types.Confluence:
        return 'Confluence';
      case Types.NetworkDrive:
        return 'Network Drive';
      case Types.Postgresql:
        return 'Postgresql';
      case Types.Mysql:
        return 'Mysql';
      case Types.GitHub:
        return 'Github';
      case Types.GitLab:
        return 'Gitlab';
      case Types.Gitea:
        return 'Gitea';
      default:
        return connector?.id || type || 'Unknown';
    }
  };

  useEffect(() => {
    getConnector(type).then(res => {
      if (res.data?.found === true) {
        setConnector(res.data._source || {});
      }
    });
  }, [type]);

  const [createState, setCreateState] = useState({
    isModalOpen: true,
    loading: false
  });

  const [modelForm] = useForm();
  const onModalOkClick = () => {
    modelForm.validateFields().then(values => {
      setCreateState(old => {
        return {
          ...old,
          loading: true
        };
      });
      createDatasource({
        connector: {
          id: connector.id
        },
        enabled: true,
        name: values.name,
        type: 'connector'
      })
        .then(({ data }) => {
          setCreateState(old => {
            nav(`/data-source/edit/${data._id}`, {});
            return {
              ...old,
              loading: false
            };
          });
        })
        .catch(() => {
          setCreateState(old => {
            return {
              ...old,
              loading: false
            };
          });
        });
    });
  };

  // Handle the default case for unknown connector types - show modal
  const shouldShowModal = !supportsOAuth && ![
    Types.Yuque,
    Types.Notion,
    Types.HugoSite,
    Types.GoogleDrive,
    Types.RSS,
    Types.LocalFS,
    Types.S3,
    Types.Confluence,
    Types.NetworkDrive,
    Types.Postgresql,
    Types.Mysql,
    Types.GitHub,
    Types.GitLab,
    Types.Gitea
  ].includes(type);

  const connectorTypeName = getConnectorTypeName();
  let connectorType = 'Google Drive';
  switch (type) {
    case Types.Yuque:
      connectorType = 'Yuque';
      break;
    case Types.Notion:
      connectorType = 'Notion';
      break;
    case Types.HugoSite:
      connectorType = 'Hugo Site';
      break;
    case Types.GoogleDrive:
      break;
    case Types.RSS:
      connectorType = 'RSS';
      break;
    case Types.LocalFS:
      connectorType = 'Local FS';
      break;
    case Types.S3:
      connectorType = 'S3';
      break;
    case Types.Confluence:
      connectorType = 'Confluence';
      break;
    case Types.NetworkDrive:
      connectorType = 'Network Drive';
      break;
    case Types.Postgresql:
      connectorType = 'Postgresql';
      break;
    case Types.Mysql:
      connectorType = 'Mysql';
      break;
    case Types.GitHub:
      connectorType = 'Github';
      break;
    case Types.GitLab:
      connectorType = 'Gitlab';
      break;
    case Types.Gitea:
      connectorType = 'Gitea';
      break;
    default:
      return (
        <Modal
          okText={t('common.save')}
          open={createState.isModalOpen}
          title={`${t('page.datasource.new.labels.connect')} '${connector.name}'`}
          onOk={onModalOkClick}
          onCancel={() => {
            nav('/data-source/new-first');
          }}
        >
          <Spin spinning={createState.loading}>
            <Form
              className="my-2em"
              form={modelForm}
              layout="vertical"
            >
              <Form.Item
                label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>}
                name="name"
                rules={[{ required: true }]}
              >
                <Input />
              </Form.Item>
            </Form>
          </Spin>
        </Modal>
      );
  }

  // eslint-disable-next-line complexity
  const onFinish: FormProps<any>['onFinish'] = values => {
    let config: any = {};
    // eslint-disable-next-line default-case
    switch (type) {
      case Types.Yuque:
        config = {
          ...(values.indexing_scope || {}),
          token: values.token || ''
        };
        break;
      case Types.Notion:
        config = {
          token: values.token || ''
        };
        break;
      case Types.HugoSite:
        config = {
          urls: values.urls || []
        };
        break;
      case Types.RSS:
        config = {
          urls: values.urls || []
        };
        break;
      case Types.LocalFS: {
        const extensions = values.config?.extensions_str
          ? values.config.extensions_str
              .split(',')
              .map((s: string) => s.trim())
              .filter(Boolean)
          : [];
        config = {
          extensions,
          paths: (values.config?.paths || []).filter(Boolean)
        };
        break;
      }
      case Types.S3: {
        const extensions: Array<string> = values.config?.extensions_str
          ? values.config.extensions_str
              .split(',')
              .map((s: string) => s.trim())
              .filter(Boolean)
          : [];
        config = {
          access_key_id: values.config?.access_key_id || '',
          bucket: values.config?.bucket || '',
          endpoint: values.config?.endpoint || '',
          extensions,
          prefix: values.config?.prefix || '',
          secret_access_key: values.config?.secret_access_key || '',
          use_ssl: values.config?.use_ssl || false
        };
        break;
      }
      case Types.Confluence: {
        config = {
          enable_attachments: values.config?.enable_attachments || false,
          enable_blogposts: values.config?.enable_blogposts || false,
          endpoint: values.config?.endpoint || '',
          space: values.config?.space || '',
          token: values.config?.token || '',
          username: values.config?.username || ''
        };
        break;
      }
      case Types.NetworkDrive: {
        config = NetworkDriveConfig(values);
        break;
      }
      case Types.Postgresql: {
        config = RdbmsConfig(values);
        break;
      }
      case Types.Mysql: {
        config = RdbmsConfig(values);
        break;
      }
      case Types.GitHub: {
        config = GithubConfig(values);
        break;
      }
      case Types.GitLab: {
        config = GitlabConfig(values);
        break;
      }
      case Types.Gitea: {
        config = GiteaConfig(values);
        break;
      }
    }
    const sValues = {
      connector: {
        config: {
          ...config,
          interval: values.sync_config.interval,
          sync_type: values.sync_config.sync_type || ''
        },
        id: type
      },
      enabled: Boolean(values.enabled),
      icon: values.icon,
      name: values.name,
      sync_enabled: values.sync_enabled,
      type: 'connector'
    };
    createDatasource(sValues).then(res => {
      if (res.data?.result === 'created') {
        message.success(t('common.addSuccess'));
        nav('/data-source/list', {});
      }
    });
  };

  // eslint-disable-next-line react-hooks/rules-of-hooks
  const [iconsMeta, setIconsMeta] = useState([]);
  // eslint-disable-next-line react-hooks/rules-of-hooks
  useEffect(() => {
    getConnectorIcons().then(res => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);

  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
  };
  // Early return for modal case to avoid hooks issues
  if (shouldShowModal) {
    return (
      <Modal
        okText={t('common.save')}
        open={createState.isModalOpen}
        title={`${t('page.datasource.new.labels.connect')} '${connector.name}'`}
        onOk={onModalOkClick}
        onCancel={() => {
          nav('/data-source/new-first');
        }}
      >
        <Spin spinning={createState.loading}>
          <Form
            className="my-2em"
            form={modelForm}
            layout="vertical"
          >
            <Form.Item
              label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>}
              name="name"
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
          </Form>
        </Spin>
      </Modal>
    );
  }

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>
            {t('page.datasource.new.title', {
              connector: connectorTypeName
            })}
          </div>
        </div>
        {supportsOAuth ? (
          <OAuthConnect
            connector={connector}
            validationRules={validationRules}
          />
        ) : (
          <div>
            <Form
              autoComplete="off"
              colon={false}
              form={form}
              labelCol={{ span: 4 }}
              layout="horizontal"
              wrapperCol={{ span: 18 }}
              initialValues={{
                connector: { config: {}, id: type },
                enabled: true,
                sync_config: { interval: '60s', sync_type: 'interval' },
                sync_enabled: true
              }}
              onFinish={onFinish}
              onFinishFailed={onFinishFailed}
              onValuesChange={(changedValues, allValues) => {
                if (changedValues.config?.field_mapping?.enabled === false) {
                  const config = allValues.config;
                  form.setFieldsValue({
                    config: {
                      ...config,
                      field_mapping: {
                        enabled: false,
                        mapping: {}
                      }
                    }
                  });
                }
              }}
            >
              <Form.Item
                label={t('page.datasource.new.labels.name')}
                name="name"
                rules={[
                  {
                    message: t(
                      'page.datasource.commons.error.datasource_name_required',
                      'Please input datasource name!'
                    ),
                    required: true
                  }
                ]}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              <Form.Item
                label={t('page.mcpserver.labels.icon')}
                name="icon"
              >
                <IconSelector
                  className="max-w-300px"
                  icons={iconsMeta}
                  type="connector"
                />
              </Form.Item>
              {type === Types.Yuque && <Yuque />}
              {type === Types.Notion && <Notion />}
              {type === Types.HugoSite && <HugoSite />}
              {type === Types.RSS && <Rss />}
              {type === Types.LocalFS && <LocalFS />}
              {type === Types.S3 && <S3 />}
              {type === Types.Confluence && <Confluence />}
              {type === Types.NetworkDrive && <NetworkDrive />}
              {type === Types.Postgresql && <Rdbms dbType="postgresql" />}
              {type === Types.Mysql && <Rdbms dbType="mysql" />}
              {type === Types.GitHub && <GitHub />}
              {type === Types.GitLab && <GitLab />}
              {type === Types.Gitea && <Gitea />}
              <Form.Item
                label={t('page.datasource.new.labels.data_sync')}
                name="sync_config"
              >
                <DataSync />
              </Form.Item>
              <Form.Item
                label={t('page.datasource.new.labels.sync_enabled')}
                name="sync_enabled"
              >
                <Switch size="small" />
              </Form.Item>
              <Form.Item
                label={t('page.datasource.new.labels.enabled')}
                name="enabled"
              >
                <Switch size="small" />
              </Form.Item>
              <Form.Item label=" ">
                <Button
                  htmlType="submit"
                  type="primary"
                >
                  {t('common.save')}
                </Button>
                {/* <div className='mt-10px'>
                <Checkbox className='mr-5px' />{t('page.datasource.new.labels.immediate_sync')}
              </div> */}
              </Form.Item>
            </Form>
          </div>
        )}
      </ACard>
    </div>
  );
}
