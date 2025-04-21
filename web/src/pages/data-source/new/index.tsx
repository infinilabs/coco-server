import { Button, Checkbox, Form, Input, Modal, Spin, Switch, message } from 'antd';
import type { FormProps } from 'antd';
import { useForm } from 'antd/es/form/Form';

import { DataSync } from '@/components/datasource/data_sync';
import { TypeList, Types } from '@/components/datasource/type';
import { getConnector } from '@/service/api/connector';
import { createDatasource } from '@/service/api/data-source';

import GoogleDrive from './google_drive';
import HugoSite from './hugo_site';
import Notion from './notion';
import Yuque from './yuque';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const type = params.get('type') ?? Types.GoogleDrive;
  const [connector, setConnector] = useState<any>({});
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
          ...config,
          interval: values.sync_config.interval,
          sync_type: values.sync_config.sync_type || ''
        },
        id: type
      },
      enabled: Boolean(values.enabled),
      name: values.name,
      sync_enabled: values.sync_enabled,
      type: 'connector'
    };
    createDatasource(sValues).then(res => {
      if (res.data?.result == 'created') {
        message.success(t('common.addSuccess'));
        nav('/data-source/list', {});
      }
    });
  };

  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
  };
  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>
            {t('page.datasource.new.title', {
              connector: connectorType
            })}
          </div>
        </div>
        {type === Types.GoogleDrive ? (
          <GoogleDrive connector={connector} />
        ) : (
          <div>
            <Form
              autoComplete="off"
              colon={false}
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
            >
              <Form.Item
                label={t('page.datasource.new.labels.name')}
                name="name"
                rules={[{ message: 'Please input datasource name!', required: true }]}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              {type === Types.Yuque && <Yuque />}
              {type === Types.Notion && <Notion />}
              {type === Types.HugoSite && <HugoSite />}
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
              <Form.Item
                label={t('page.datasource.new.labels.enabled')}
                name="enabled"
              >
                <Switch />
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
