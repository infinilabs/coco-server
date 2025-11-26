import { Button, Form, Input, Select, message,Switch, Spin } from 'antd';
import type { FormProps } from 'antd';

import InfiniIcon from '@/components/common/icon';
import { Tags } from '@/components/common/tags';
import { getConnector, getConnectorCategory, getConnectorIcons, updateConnector } from '@/service/api/connector';
import { formatESSearchResult } from '@/service/request/es';

import { AssetsIcons } from '../new/assets_icons';
import { IconSelector } from '../new/icon_selector';
import { useRoute } from '@sa/simple-router';
import Processor from '../new/processor';
import { getServer } from '@/store/slice/server';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [form] = Form.useForm();

  const route = useRoute();
  const connectorID = route.params.id

  const [loading, setLoading] = useState(false);
  const [connector, setConnector] = useState<any>({});

  const onFinish: FormProps<any>['onFinish'] = values => {
    const category = typeof values.category === 'string' ? values.category : values.category[0] || '';
    const sValues = {
      assets: {
        icons: values.assets_icons
      },
      category,
      config: values.raw_config ? JSON.parse(values.raw_config) : {},
      description: values.description,
      icon: values.icon,
      name: values.name,
      path_hierarchy: values.path_hierarchy,
      tags: values.tags,
      processor: values.processor,
    };
    if (connector?.processor?.name === 'google_drive') {
      sValues.config = {
        auth_url: values.auth_url,
        client_id: values.client_id,
        client_secret: values.client_secret,
        redirect_url: values.redirect_url,
        token_url: values.token_url
      };
    }

    updateConnector(connectorID, sValues).then(res => {
      if (res.data?.result == 'updated') {
        message.success(t('common.updateSuccess'));
        nav('/settings?tab=connector', {});
      }
    });
  };
  const [iconsMeta, setIconsMeta] = useState([]);

  const fetchConnector = async (connectorID) => {
    if (!connectorID) return;
    setLoading(true)
    const res = await getConnector(connectorID);
    if (res.data?.found === true && res.data._source) {
      const newConnector = {
        ...res.data._source,
        assets_icons: res.data._source?.assets?.icons || {},
        ...(res.data._source?.config || {
          auth_url: "https://accounts.google.com/o/oauth2/auth",
          redirect_url: `${window.location.origin}${window.location.pathname}connector/${res.data._source.id}/oauth_redirect`,
          token_url: "https://oauth2.googleapis.com/token"
        }),
        raw_config: res.data._source?.config ? JSON.stringify(res.data._source?.config,null,4) : undefined
      }
      setConnector(newConnector)
      form.setFieldsValue(newConnector)
    }
    setLoading(false)
  }

  useEffect(() => {
    fetchConnector(connectorID)
  }, [connectorID]);

  useEffect(() => {
    getConnectorIcons().then(res => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);

  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getConnectorCategory().then(({ data }) => {
      if (!data?.error) {
        const newData = formatESSearchResult(data);
        const cates = newData?.aggregations?.categories?.buckets?.map((item: any) => {
          return item.key;
        });
        setCategories(cates||[]);
      }
    });
  }, []);

  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
  };
  const { defaultRequiredRule, formRules } = useFormRules();

  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        loading={loading}
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t('page.connector.edit.title')}</div>
        </div>
        
        <Spin spinning={loading}>
        <Form
          autoComplete="off"
          colon={false}
          form={form}
          initialValues={connector}
          labelCol={{ span: 4 }}
          layout="horizontal"
          wrapperCol={{ span: 18 }}
          onFinish={onFinish}
          onFinishFailed={onFinishFailed}
        >
          <Form.Item
            label={t('page.connector.new.labels.name')}
            name="name"
            rules={[{ message: 'Please input connector name!', required: true }]}
          >
            <Input className="max-w-600px" />
          </Form.Item>

           <Form.Item
            label={t('page.connector.new.labels.processor')}
            name="processor"
            rules={[{ message: 'Please input processor name!', required: true }, {
            validator: (_, value) => {
              if (!value) return Promise.resolve()
              if (value.name?.trim().length == 0 ) {
                return Promise.reject(new Error('name is required'))
              }
              return Promise.resolve()
            },
          },]}
          >
            <Processor className="max-w-600px" />
          </Form.Item>
          
          <Form.Item
            label={t('page.connector.new.labels.description')}
            name="description"
          >
            <Input.TextArea />
          </Form.Item>

          <Form.Item
            label={t('page.connector.new.labels.category')}
            tooltip={t('page.connector.new.tooltip.category', 'Please choose or input the category.')}
            name="category"
            rules={[{ required: true }]}
          >
            <Select
              className="max-w-600px"
              maxTagCount={1}
              mode="tags"
              options={categories.map(cate => {
                return { value: cate };
              })}
            />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.icon')}
            name="icon"
            rules={[{ required: true }]}
          >
            {connector?.builtin === true ? (
              <IconWrapper className="w-40px h-40px">
                <InfiniIcon
                  className="h-2em w-2em"
                  height="2em"
                  src={connector?.icon}
                  width="2em"
                />
              </IconWrapper>
            ) : (
              <IconSelector
                className="max-w-200px"
                icons={iconsMeta}
                readonly={connector?.builtin === true}
              />
            )}
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.assets_icons')}
            name="assets_icons"
          >
            {connector?.builtin === true ? (
              <AssetsIconsView />
            ) : (
              <AssetsIcons
                iconsMeta={iconsMeta}
                readonly={connector?.builtin === true}
              />
            )}
          </Form.Item>
          {connector?.processor?.name === 'google_drive' && (
            <>
              <Form.Item
                label={t('page.connector.new.labels.client_id')}
                name="client_id"
                rules={[defaultRequiredRule]}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              <Form.Item
                label={t('page.connector.new.labels.client_secret')}
                name="client_secret"
                rules={[defaultRequiredRule]}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              <Form.Item
                label={t('page.connector.new.labels.redirect_url')}
                name="redirect_url"
                rules={formRules.endpoint}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              <Form.Item
                label={t('page.connector.new.labels.auth_url')}
                name="auth_url"
                rules={formRules.endpoint}
              >
                <Input className="max-w-600px" />
              </Form.Item>
              <Form.Item
                label={t('page.connector.new.labels.token_url')}
                name="token_url"
                rules={formRules.endpoint}
              >
                <Input className="max-w-600px" />
              </Form.Item>
            </>
          )}

        <Form.Item
            label={t('page.connector.new.labels.config')}
            tooltip={t('page.connector.new.tooltip.config', 'Configurations in JSON format.')}
            name="raw_config"
         >
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 30 }} />
          </Form.Item>


         <Form.Item
            label={t('page.connector.new.labels.path_hierarchy')}
            name="path_hierarchy"
            tooltip={t('page.connector.new.tooltip.path_hierarchy', 'Whether to support access documents via path hierarchy manner.')}
            valuePropName="checked"
            className='w-0 h-0 m-0 overflow-hidden'
          >
          <Switch />
         </Form.Item>


      
          <Form.Item
            label={t('page.connector.new.labels.tags')}
            name="tags"
          >
            <Tags />
          </Form.Item>
          <Form.Item label=" ">
            <Button
              htmlType="submit"
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

const AssetsIconsView = ({ value = {} }) => {
  const { t } = useTranslation();
  const icons = Object.keys(value).map(key => {
    return {
      icon: value[key],
      type: key
    };
  });
  return (
    <div className="flex flex-col">
      <div className="flex flex-wrap gap-10px">
        {icons.map((icon, index) => {
          return (
            <div
              className="flex items-center gap-5px"
              key={index}
            >
              <IconWrapper className="w-20px h-20px">
                <InfiniIcon
                  className="h-1em w-1em"
                  height="1em"
                  src={icon.icon}
                  width="1em"
                />
              </IconWrapper>
              <span>{icon.type}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
