import { Button, Form, Input, Spin, Switch, message } from 'antd';
import type { FormProps } from 'antd';
import Clipboard from 'clipboard';
import { useLoaderData } from 'react-router-dom';
import { ReactSVG } from 'react-svg';

import LinkSVG from '@/assets/svg-icon/link.svg';
import { DataSync } from '@/components/datasource/data_sync';
import { Types } from '@/components/datasource/type';
import { getDatasource, updateDatasource } from '@/service/api/data-source';

import HugoSite from '../new/hugo_site';
import Notion from '../new/notion';
import Yuque from '../new/yuque';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const loaderData = useLoaderData();
  const datasourceID = loaderData?.id || '';
  const [loading, setLoading] = useState(false);
  const [datasource, setDatasource] = useState<any>({
    id: datasourceID
  });
  useEffect(() => {
    if (!datasourceID) return;
    getDatasource(datasourceID).then(res => {
      if (res.data?.found === true) {
        setDatasource(res.data._source || {});
      }
    });
  }, [datasourceID]);
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

  const onFinish: FormProps<any>['onFinish'] = values => {
    let config: any = {};
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
      name: values.name,
      sync_enabled: Boolean(values.sync_enabled),
      type: 'connector'
    };
    if (values.sync_config) {
      sValues.connector.config.interval = values.sync_config.interval;
      sValues.connector.config.sync_type = values.sync_config.sync_type || '';
    }
    updateDatasource(datasourceID, sValues).then(res => {
      if (res.data?.result == 'updated') {
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
      datasource.urls = datasource?.connector?.config?.urls || [];
      break;
    case Types.GoogleDrive:
      break;
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
        <div className="mb-4 ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          {t('page.datasource.edit.title')}
        </div>
        <Spin spinning={loading}>
          <Form
            autoComplete="off"
            colon={false}
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
            {type === Types.Yuque && <Yuque />}
            {type === Types.Notion && <Notion />}
            {type === Types.HugoSite && <HugoSite />}
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
                  <Switch />
                </Form.Item>
              </>
            ) : (
              <Form.Item
                label={t('page.datasource.new.labels.insert_doc')}
                name=""
              >
                <div className="max-w-660px rounded bg-gray-100 p-1em">
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
              <Switch />
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
