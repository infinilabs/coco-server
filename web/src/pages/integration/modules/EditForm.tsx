import { Button, Checkbox, Form, Input, Radio, Select, Spin, Switch } from 'antd';

import All from '@/assets/integration/all.png';
import Embedded from '@/assets/integration/embedded.png';
import Floating from '@/assets/integration/floating.png';

import './EditForm.css';
import { useLoading, useRequest } from '@sa/hooks';

import { fetchDataSourceList } from '@/service/api';
import { getDarkMode } from '@/store/slice/theme';

import { HotKeys } from './HotKeys';

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
    loading: dataSourceLoading,
    run
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
          allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
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
          access_control: {
            authentication: true,
            chat_history: true
          },
          appearance: {
            theme: 'auto'
          },
          cors: {
            allowed_origins: '*',
            enabled: true
          },
          datasource: ['*'],
          enabled_module: {
            ai_chat: {
              enabled: true,
              placeholder: 'Ask whatever you want...'
            },
            features: ['search_active', 'think_active'],
            search: {
              enabled: true,
              placeholder: 'Search whatever you want...'
            }
          },
          hotkey: 'ctrl+/',
          name: `widget-${generateRandomString(8)}`,
          enabled: true,
          options: {
            embedded_placeholder: 'Search...',
            floating_placeholder: 'Ask AI'
          },
          type: 'embedded'
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
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
        wrapperCol={{
          style: { maxWidth: 528, minWidth: 528, textAlign: 'left' }
        }}
      >
        <Form.Item
          label={t('page.integration.form.labels.name')}
          name="name"
          rules={[defaultRequiredRule]}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.enabled')}
          name="enabled"
          rules={[defaultRequiredRule]}
        >
          <Switch size='small'/>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.type')}
          name="type"
          rules={[defaultRequiredRule]}
        >
          <Radio.Group
            className="custom-radio-group"
            options={[
              {
                label: (
                  <div>
                    {t('page.integration.form.labels.type_embedded')}
                    <img
                      className={imageClassNames}
                      src={Embedded}
                    />
                  </div>
                ),
                value: 'embedded'
              },
              {
                label: (
                  <div>
                    {t('page.integration.form.labels.type_floating')}
                    <img
                      className={imageClassNames}
                      src={Floating}
                    />
                  </div>
                ),
                value: 'floating'
              },
              {
                label: (
                  <div>
                    {t('page.integration.form.labels.type_all')}
                    <img
                      className={imageClassNames}
                      src={All}
                    />
                  </div>
                ),
                value: 'all'
              }
            ]}
            onChange={e => {
              setType(e.target.value);
            }}
          />
        </Form.Item>
        {['embedded', 'all'].includes(type) && (
          <>
            <Form.Item label=" ">
              <Form.Item
                className="mb-32px"
                label={t('page.integration.form.labels.type_embedded_placeholder')}
                layout="vertical"
                name={['options', 'embedded_placeholder']}
              >
                <Input className={itemClassNames} />
              </Form.Item>
            </Form.Item>
            <Form.Item label=" ">
              <Form.Item
                className="mb-32px"
                label={t('page.integration.form.labels.type_embedded_icon')}
                layout="vertical"
                name={['options', 'embedded_icon']}
              >
                <Input className={itemClassNames} placeholder={`${window.location.origin}/icon.svg`}/>
              </Form.Item>
            </Form.Item>
          </>
        )}
        {['floating', 'all'].includes(type) && (
          <>
            <Form.Item label=" ">
              <Form.Item
                className="mb-32px"
                label={t('page.integration.form.labels.type_floating_placeholder')}
                layout="vertical"
                name={['options', 'floating_placeholder']}
              >
                <Input className={itemClassNames} />
              </Form.Item>
            </Form.Item>
            <Form.Item label=" ">
              <Form.Item
                className="mb-32px"
                label={t('page.integration.form.labels.type_floating_icon')}
                layout="vertical"
                name={['options', 'floating_icon']}
              >
                <Input className={itemClassNames}  placeholder={`${window.location.origin}/icon.svg`}/>
              </Form.Item>
            </Form.Item>
          </>
        )}
        <Form.Item
          label={t('page.integration.form.labels.datasource')}
          name="datasource"
          rules={[defaultRequiredRule]}
        >
          <Select
            allowClear
            className={itemClassNames}
            loading={dataSourceLoading}
            mode="multiple"
            options={[{ label: '*', value: '*' }].concat(
              dataSource.map(item => ({
                label: item.name,
                value: item.id
              }))
            )}
          />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.hotkey')}
          name="hotkey"
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
            label={t('page.integration.form.labels.module_search')}
            layout="horizontal"
            name={['enabled_module', 'search', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-48px"
            label={t('page.integration.form.labels.module_search_placeholder')}
            layout="vertical"
            name={['enabled_module', 'search', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
          <Form.Item
            className="mb-12px"
            label={t('page.integration.form.labels.module_chat')}
            layout="horizontal"
            name={['enabled_module', 'ai_chat', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-48px"
            label={t('page.integration.form.labels.module_chat_placeholder')}
            layout="vertical"
            name={['enabled_module', 'ai_chat', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
          <Form.Item
            className="mb-44px"
            label={t('page.integration.form.labels.feature_Control')}
            layout="vertical"
            name={['enabled_module', 'features']}
          >
            <Checkbox.Group
              options={[
                {
                  label: t('page.integration.form.labels.feature_think_active'),
                  value: 'think_active'
                },
                {
                  label: t('page.integration.form.labels.feature_think'),
                  value: 'think'
                },
                {
                  label: t('page.integration.form.labels.feature_search_active'),
                  value: 'search_active'
                },
                {
                  label: t('page.integration.form.labels.feature_search'),
                  value: 'search'
                },
                {
                  label: t('page.integration.form.labels.feature_chat_history'),
                  value: 'chat_history'
                }
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
            label={t('page.integration.form.labels.enable_auth')}
            layout="horizontal"
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
            label={t('page.integration.form.labels.theme')}
            layout="vertical"
            name={['appearance', 'theme']}
          >
            <Select
              allowClear
              className={itemClassNames}
              options={[
                {
                  label: t('page.integration.form.labels.theme_auto'),
                  value: 'auto'
                },
                {
                  label: t('page.integration.form.labels.theme_light'),
                  value: 'light'
                },
                {
                  label: t('page.integration.form.labels.theme_dark'),
                  value: 'dark'
                }
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
            label=""
            layout="horizontal"
            name={['cors', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
          <Form.Item
            className="mb-98px"
            label={t('page.integration.form.labels.allow_origin')}
            layout="vertical"
            name={['cors', 'allowed_origins']}
          >
            <Input.TextArea
              className={itemClassNames}
              placeholder={t('page.integration.form.labels.allow_origin_placeholder')}
              rows={4}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.description')}
          name="description"
        >
          <Input.TextArea
            className={itemClassNames}
            rows={4}
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
