import type { FormProps } from 'antd';
import { Button, Form, Input, Spin, Switch, message } from 'antd';
import { useForm } from 'antd/es/form/Form';
import { useEffect, useState } from 'react';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
import { useLoaderData } from 'react-router-dom';

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { DataSync } from '@/components/datasource/data_sync';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { Types } from '@/components/datasource/type';
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error
import { IconSelector } from '@/pages/connector/new/icon_selector';
import { getConnectorIcons } from '@/service/api/connector';
import { getDatasource, updateDatasource } from '@/service/api/data-source';

import Confluence from '../new/confluence';
import GitHub from '../new/github';
import HugoSite from '../new/hugo_site';
import LocalFS from '../new/local_fs';
import { GithubConfig, NetworkDriveConfig, RdbmsConfig } from '../new/models';
import NetworkDrive from '../new/network_drive';
import Notion from '../new/notion';
import Rdbms from '../new/rdbms';
import Rss from '../new/rss';
import S3 from '../new/s3';
import Yuque from '../new/yuque';

// eslint-disable-next-line complexity
export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const loaderData = useLoaderData();
  const datasourceID = loaderData?.id || '';
  const [loading, setLoading] = useState(false);
  const [datasource, setDatasource] = useState<any>({
    id: datasourceID
  });
  const [form] = useForm();

  useEffect(() => {
    if (!datasourceID) return;
    // eslint-disable-next-line complexity
    getDatasource(datasourceID).then(res => {
      if (res.data?.found === true) {
        // eslint-disable-next-line @typescript-eslint/no-shadow
        const datasource = res.data._source;
        const type = datasource?.connector?.id;
        switch (type) {
          case Types.Yuque:
            datasource.indexing_scope = datasource?.connector?.config || {};
            datasource.token = datasource?.connector?.config?.token || '';
            break;
          case Types.Notion:
            datasource.token = datasource?.connector?.config?.token || '';
            break;
          // Use separate cases
          case Types.HugoSite:
          case Types.RSS:
            datasource.urls = datasource?.connector?.config?.urls || [''];
            break;
          case Types.GoogleDrive:
            break;
          case Types.Postgresql:
          case Types.Mysql:
            if (datasource.connector?.config) {
              datasource.config = datasource.connector.config;
            }
            break;
          default:
            break;
        }
        setDatasource({
          ...(datasource || {}),
          sync_config: {
            interval: datasource?.connector?.config?.interval || '1h',
            sync_type: datasource?.connector?.config?.sync_type || ''
          },
          urls: datasource?.connector?.config?.urls || ['']
        });
      }
    });
  }, [datasourceID]);
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then(res => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);
  const copyRef = useRef<HTMLSpanElement | null>(null);
  const insertDocCmd = `curl -H'X-API-TOKEN: REPLACE_YOUR_API_TOKEN_HERE'  -H 'Content-Type: application/json' -XPOST ${location.origin}/datasource/${datasourceID}/_doc -d'
  {
    "title": "I am just a Coco doc that you can search",
    "summary": "Nothing but great start",
    "content": "Coco is a unified private search engien that you can trust.",
    "url":"http://coco.rs/",
    "icon": "default"
  }'`;
  const [copyRefUpdated, setCopyRefUpdated] = useState(false);
  useEffect(() => {
    if (!copyRef.current) return;
    const clipboard = new Clipboard(copyRef.current as any, {
      text: () => {
        return insertDocCmd;
      }
    });
    clipboard.on('success', function (e) {
      message.success(t('common.copySuccess'));
    });
    return () => {
      clipboard.destroy();
    };
  }, [copyRefUpdated, insertDocCmd]);

  // eslint-disable-next-line complexity
  const onFinish: FormProps<any>['onFinish'] = values => {
    let config: any = {};
    // eslint-disable-next-line default-case,@typescript-eslint/no-use-before-define
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
        const extensions = values.config?.extensions_str
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
      case Types.Postgresql:
      case Types.Mysql: {
        config = RdbmsConfig(values);
        break;
      }
      case Types.GitHub: {
        config = GithubConfig(values);
        break;
      }
    }
    const sValues = {
      connector: {
        config: {
          ...(datasource?.connector?.config || {}),
          ...config
        },
        id: type
      },
      enabled: Boolean(values.enabled),
      icon: values.icon,
      name: values.name,
      sync_enabled: Boolean(values?.sync_enabled),
      type: 'connector'
    };
    if (values.sync_config) {
      sValues.connector.config.interval = values.sync_config.interval;
      sValues.connector.config.sync_type = values.sync_config.sync_type || '';
    }
    updateDatasource(datasourceID, sValues).then(res => {
      if (res.data?.result === 'updated') {
        setLoading(false);
        message.success(t('common.modifySuccess'));
        nav('/data-source/list', {});
      }
    });
  };
  datasource.sync_config = {
    interval: datasource?.connector?.config?.interval || '1h',
    sync_type: datasource?.connector?.config?.sync_type || ''
  };
  const type = datasource?.connector?.id;
  if (!type) {
    return null;
  }
  let isCustom = false;
  switch (type) {
    case Types.Yuque:
      datasource.indexing_scope = datasource?.connector?.config || {};
      datasource.token = datasource?.connector?.config?.token || '';
      break;
    case Types.Notion:
      datasource.token = datasource?.connector?.config?.token || '';
      break;
    case Types.HugoSite:
    case Types.RSS:
      datasource.urls = datasource?.connector?.config?.urls || [''];
      break;
    case Types.LocalFS:
      if (datasource.connector?.config) {
        datasource.config = {
          extensions_str: (datasource.connector.config?.extensions || []).join(', '),
          paths: datasource.connector.config.paths || ['']
        };
      }
      break;
    case Types.GoogleDrive:
      break;
    case Types.S3:
      if (datasource.connector?.config) {
        datasource.config = {
          access_key_id: datasource.connector.config?.access_key_id || '',
          bucket: datasource.connector.config?.bucket || '',
          endpoint: datasource.connector.config?.endpoint || '',
          extensions_str: (datasource.connector.config?.extensions || []).join(', '),
          prefix: datasource.connector.config?.prefix || '',
          secret_access_key: datasource.connector.config?.secret_access_key || '',
          use_ssl: datasource.connector.config?.use_ssl || false
        };
      }
      break;
    case Types.Confluence:
      if (datasource.connector?.config) {
        const values = datasource.connector;
        datasource.config = {
          enable_attachments: values.config?.enable_attachments || false,
          enable_blogposts: values.config?.enable_blogposts || false,
          endpoint: values.config?.endpoint || '',
          space: values.config?.space || '',
          token: values.config?.token || '',
          username: values.config?.username || ''
        };
      }
      break;
    case Types.NetworkDrive: {
      if (datasource.connector?.config) {
        const values = datasource.connector;
        datasource.config = {
          domain: values.config?.domain || '',
          endpoint: values.config?.endpoint || '',
          extensions_str: (values.config?.extensions || []).join(', '),
          password: values.config?.password || '',
          paths: (values.config?.paths || []).filter(Boolean),
          share: values.config?.share || '',
          username: values.config?.username || ''
        };
      }
      break;
    }
    case Types.Postgresql:
    case Types.Mysql: {
      if (datasource.connector?.config) {
        datasource.config = RdbmsConfig(datasource.connector);
      }
      break;
    }
    case Types.GitHub: {
      if (datasource.connector?.config) {
        datasource.config = GithubConfig(datasource.connector);
      }
      break;
    }
    default:
      isCustom = true;
  }
  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
    setLoading(false);
  };

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          {t('page.datasource.edit.title')}
        </div>
        <Spin spinning={loading}>
          <Form
            autoComplete="off"
            colon={false}
            form={form}
            initialValues={datasource || {}}
            labelCol={{ span: 4 }}
            layout="horizontal"
            wrapperCol={{ span: 18 }}
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item
              label={t('page.datasource.new.labels.name')}
              name="name"
              rules={[{ message: 'Please input datasource name!', required: true }]}
            >
              <Input className="max-w-660px" />
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
            {!isCustom ? (
              <>
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
              </>
            ) : (
              <Form.Item
                label={t('page.datasource.new.labels.insert_doc')}
                name=""
              >
                <div className="max-w-660px rounded-[var(--ant-border-radius)] bg-[var(--ant-color-border)] p-1em">
                  <div>
                    <pre
                      className="whitespace-pre-wrap break-words"
                      dangerouslySetInnerHTML={{ __html: insertDocCmd }}
                    />
                  </div>
                  <div className="flex justify-end">
                    <span
                      className="flex cursor-pointer items-center gap-1 text-blue-500"
                      ref={inst => {
                        copyRef.current = inst;
                        setCopyRefUpdated(true);
                      }}
                    >
                      <SvgIcon
                        className="text-18px"
                        icon="mdi:content-copy"
                      />
                      Copy
                    </span>
                  </div>
                </div>
                <div>
                  <a
                    className="my-10px inline-flex items-center text-blue-500"
                    href="https://docs.infinilabs.com/coco-server/main/docs/tutorials/howto_create_your_own_datasource/"
                    rel="noreferrer"
                    target="_blank"
                  >
                    <span>How to create a data source</span>
                    <ReactSVG
                      className="m-l-4px"
                      src={LinkSVG}
                    />
                  </a>
                </div>
              </Form.Item>
            )}
            <Form.Item
              label={t('page.datasource.new.labels.enabled')}
              name="enabled"
            >
              <Switch size="small" />
            </Form.Item>
            <Form.Item label=" ">
              <Button
                htmlType="submit"
                loading={loading}
                type="primary"
              >
                {t('common.save')}
              </Button>
            </Form.Item>
          </Form>
        </Spin>
      </ACard>
    </div>
  );
}

export async function loader({ params }: LoaderFunctionArgs) {
  return params;
}
