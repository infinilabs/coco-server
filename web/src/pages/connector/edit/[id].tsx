import { Button, Form, Input, Select, message } from 'antd';
import type { FormProps } from 'antd';

import InfiniIcon from '@/components/common/icon';
import { Tags } from '@/components/common/tags';
import { getConnectorCategory, getConnectorIcons, updateConnector } from '@/service/api/connector';
import { formatESSearchResult } from '@/service/request/es';

import { AssetsIcons } from '../new/assets_icons';
import { IconSelector } from '../new/icon_selector';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  let { state: initialConnector } = useLocation();
  const connectorID = initialConnector?.id || '';
  initialConnector = {
    ...initialConnector,
    assets_icons: initialConnector.assets?.icons || {},
    ...(initialConnector.config || {
      auth_url: "https://accounts.google.com/o/oauth2/auth",
      redirect_url: "http://localhost:9000/connector/google_drive/oauth_redirect",
      token_url: "https://oauth2.googleapis.com/token"
    })
  };
  const [loading, setLoading] = useState(false);

  const onFinish: FormProps<any>['onFinish'] = values => {
    const category = typeof values.category === 'string' ? values.category : values.category[0] || '';
    const sValues = {
      assets: {
        icons: values.assets_icons
      },
      category,
      config: {},
      description: values.description,
      icon: values.icon,
      name: values.name,
      tags: values.tags
    };
    if (connectorID === 'google_drive') {
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
        const cates = newData.aggregations.categories.buckets.map((item: any) => {
          return item.key;
        });
        setCategories(cates);
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
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t('page.connector.edit.title')}</div>
        </div>
        <Form
          autoComplete="off"
          colon={false}
          initialValues={initialConnector}
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
            label={t('page.connector.new.labels.category')}
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
            {initialConnector.builtin === true ? (
              <IconWrapper className="w-40px h-40px">
                <InfiniIcon
                  className="h-2em w-2em"
                  height="2em"
                  src={initialConnector.icon}
                  width="2em"
                />
              </IconWrapper>
            ) : (
              <IconSelector
                className="max-w-200px"
                icons={iconsMeta}
                readonly={initialConnector.builtin === true}
              />
            )}
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.assets_icons')}
            name="assets_icons"
          >
            {initialConnector.builtin === true ? (
              <AssetsIconsView />
            ) : (
              <AssetsIcons
                iconsMeta={iconsMeta}
                readonly={initialConnector.builtin === true}
              />
            )}
          </Form.Item>
          {connectorID === 'google_drive' && (
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
            label={t('page.connector.new.labels.description')}
            name="description"
          >
            <Input.TextArea />
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
