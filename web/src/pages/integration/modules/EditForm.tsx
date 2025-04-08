import { Button, Checkbox, Form, Input, Radio, Select, Spin, Switch } from 'antd';

import All from "@/assets/integration/all.png";
import Embedded from "@/assets/integration/embedded.png"; 
import Floating from "@/assets/integration/floating.png";
import './EditForm.css';
import { useRequest , useLoading } from '@sa/hooks';
import { fetchDataSourceList } from '@/service/api';
import { HotKeys } from './HotKeys';

import { getDarkMode } from '@/store/slice/theme';

function generateRandomString(size) {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

export const EditForm = memo(props => {
  const { actionText, onSubmit, record } = props;
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();

  const darkMode = useAppSelector(getDarkMode);

  const [type, setType] = useState();

  const {
    data: result,
    run,
    loading: dataSourceLoading
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { cors = {}, datasource } = params;
    onSubmit(
      {
        ...params,
        cors: {
          ...cors,
          allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(",") : []
        },
        datasource: datasource?.includes('*') ? ['*'] : datasource
      },
      startLoading,
      endLoading
    );
  };

  useEffect(() => {
    run({
      from: 0,
      size: 10000
    });
  }, []);

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map(item => ({ ...item._source })) || [];
  }, [JSON.stringify(result)]);

  useEffect(() => {
    const initValue = record
      ? {
          ...record,
          cors: {
            ...(record.cors || {}),
            allowed_origins: record.cors?.allowed_origins ? record.cors?.allowed_origins.join(',') : ''
          }
        }
      : {
          name: `widget-${generateRandomString(8)}`,
          type: 'embedded',
          options: {
            placeholder: 'Search...'
          },
          hotkey: 'ctrl+/',
          datasource: ['*'],
          enabled_module: {
            search: {
              enabled: true,
              placeholder: 'Search whatever you want...'
            },
            ai_chat: {
              enabled: true,
              placeholder: 'Ask whatever you want...'
            },
            features: ['search_active', 'think_active']
          },
          access_control: {
            authentication: true,
            chat_history: true
          },
          appearance: {
            theme: 'auto'
          },
          cors: {
            enabled: true,
            allowed_origins: '*'
          }
        };
    form.setFieldsValue(initValue);
    setType(initValue.type);
  }, [record]);

  const itemClassNames = '!w-496px';
  const imageClassNames = darkMode ? 'brightness-30 saturate-0' : '';

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        colon={false}
          form={form}
          labelAlign="left"
          layout="horizontal"
          labelCol={{
            style: { textAlign: "left", minWidth: 200, maxWidth: 200 },
          }} 
          wrapperCol={{ 
            style: { textAlign: "left", minWidth: 528, maxWidth: 528 }, 
          }}
      >
        <Form.Item
          label={t('page.integration.form.labels.name')}
          rules={[defaultRequiredRule]}
          name="name"
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.type')}
          rules={[defaultRequiredRule]}
          name="type"
        >
          <Radio.Group
            className="custom-radio-group"
            options={[
              {
                value: 'embedded',
                label: (
                  <div>
                    {t('page.integration.form.labels.type_embedded')}
                    <img
                      src={Embedded}
                      className={imageClassNames}
                    />
                  </div>
                )
              },
              {
                label: (
                    <div>
                      {t('page.integration.form.labels.type_floating')}
                      <img src={Floating} className={imageClassNames}/>
                    </div>
                  ),
                  value: 'floating',
              },
              {
                label: (
                    <div>
                      {t('page.integration.form.labels.type_all')}
                      <img src={All} className={imageClassNames}/>
                    </div>
                  ),
                  value: 'all',
                )
              }
            ]}
            onChange={e => {
              setType(e.target.value);
            }}
          />
        </Form.Item>
        {['embedded', 'all'].includes(type) && (
          <Form.Item label=" ">
            <Form.Item
              className="mb-32px"
              layout="vertical"
              label={t('page.integration.form.labels.type_embedded_placeholder')}
              name={['options', 'placeholder']}
            >
              <Input className={itemClassNames} />
            </Form.Item>
          </Form.Item>
        )}
        <Form.Item
          label={t('page.integration.form.labels.datasource')}
          rules={[defaultRequiredRule]}
          name="datasource"
        >
          <Select
            allowClear
              className={itemClassNames}  
              loading={dataSourceLoading}
              mode="multiple"
              options={[{label: "*", value: "*"}].concat(dataSource.map((item) => ({
                label: item.name,
                value: item.id,
              })))}
          />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.hotkey')}
          name={'hotkey'}
        >
          <HotKeys
            className={itemClassNames}
            placeholder={t('page.integration.form.labels.hotkey_placeholder')}
          />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.enable_module')}
          name="enabled_module"
        >
          <Form.Item
            className="mb-12px"
            layout="horizontal"
            label={t('page.integration.form.labels.module_search')}
            name={['enabled_module', 'search', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-48px"
            layout="vertical"
            label={t('page.integration.form.labels.module_search_placeholder')}
            name={['enabled_module', 'search', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
          <Form.Item
            className="mb-12px"
            layout="horizontal"
            label={t('page.integration.form.labels.module_chat')}
            name={['enabled_module', 'ai_chat', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-48px"
            layout="vertical"
            label={t('page.integration.form.labels.module_chat_placeholder')}
            name={['enabled_module', 'ai_chat', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
          <Form.Item
            className="mb-44px"
            layout="vertical"
            label={t('page.integration.form.labels.feature_Control')}
            name={['enabled_module', 'features']}
          >
            <Checkbox.Group
              options={[
                { label: t('page.integration.form.labels.feature_think_active'), value: 'think_active' },
                { label: t('page.integration.form.labels.feature_think'), value: 'think' },
                { label: t('page.integration.form.labels.feature_search_active'), value: 'search_active' },
                { label: t('page.integration.form.labels.feature_search'), value: 'search' },
                { label: t('page.integration.form.labels.feature_chat_history'), value: 'chat_history' }
              ]}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.access_control')}
          name="access_control"
        >
          <Form.Item
            className="mb-0px"
            layout="horizontal"
            label={t('page.integration.form.labels.enable_auth')}
            name={['access_control', 'authentication']}
          >
            <Switch size="small" />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.appearance')}
          name="appearance"
        >
          <Form.Item
            className="mb-32px"
            layout="vertical"
            label={t('page.integration.form.labels.theme')}
            name={['appearance', 'theme']}
          >
            <Select
              className={itemClassNames}
              allowClear
              options={[
                { label: t('page.integration.form.labels.theme_auto'), value: 'auto' },
                { label: t('page.integration.form.labels.theme_light'), value: 'light' },
                { label: t('page.integration.form.labels.theme_dark'), value: 'dark' }
              ]}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.cors')}
          name="cors"
        >
          <Form.Item
            className="mb-12px"
            layout="horizontal"
            label=""
            name={['cors', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-98px"
            layout="vertical"
            label={t('page.integration.form.labels.allow_origin')}
            name={['cors', 'allowed_origins']}
          >
            <Input.TextArea
              placeholder={t('page.integration.form.labels.allow_origin_placeholder')}
              rows={4}
              className={itemClassNames}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.description')}
          name="description"
        >
          <Input.TextArea
            rows={4}
            className={itemClassNames}
          />
        </Form.Item>
        <Form.Item label=" ">
          <Button
            type="primary"
            onClick={() => handleSubmit()}
          >
            {actionText}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});
