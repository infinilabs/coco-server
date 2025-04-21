import { Button, Checkbox, Form, Input, Radio, Select, Spin, Switch } from 'antd';

import All from '@/assets/integration/all.png';
import Embedded from '@/assets/integration/embedded.png';
import Floating from '@/assets/integration/floating.png';

import './EditForm.css';
import { useLoading, useRequest } from '@sa/hooks';

import { fetchDataSourceList } from '@/service/api';
import { getDarkMode } from '@/store/slice/theme';

import { HotKeys } from './HotKeys';
import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import ChartStartPage from '@/pages/settings/modules/ChartStartPage';

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
  const [assistants, setAssistants] = useState([])

  const darkMode = useAppSelector(getDarkMode);

  const [type, setType] = useState();
  const [logo, setLogo] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    darkLoading: false,
    darkList: [],
    dark: undefined,
  })

  const {
    data: result,
    loading: dataSourceLoading,
    run
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { cors = {}, enabled_module = {}, start_page = {} } = params;
    const { search = {}, ai_chat = {} } = enabled_module
    const { datasource = [] } = search
    const { assistants = [] } = ai_chat
    onSubmit(
      {
        ...params,
        enabled_module: {
          ...enabled_module,
          search: {
            ...search,
            datasource: datasource?.includes('*') ? ['*'] : datasource
          },
          ai_chat: {
            ...ai_chat,
            assistants: assistants.map((item) => item.id),
            start_page_config: {
              ...start_page,
              "display_assistants": start_page?.display_assistants?.map((item) => item.id),
              "logo": {
                "light": logo.light,
                "dark": logo.dark
              },
            }
          }
        },
        cors: {
          ...cors,
          allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
        },
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
    if (record) {
      setLogo((state) => ({ ...state, ...(record.enabled_module?.ai_chat?.start_page_config?.logo || {}) }))
      setAssistants(record?.enabled_module?.ai_chat?.assistants ? record?.enabled_module?.ai_chat?.assistants.map((item) => ({
        id: item
      })) : [])
    }
    const initValue = record
      ? {
          ...record,
          enabled_module: {
            ...(record.enabled_module || {}),
            search: {
              ...(record.enabled_module?.search || {}),
              datasource: record.enabled_module?.search?.datasource?.includes('*') ? ['*'] : record.enabled_module?.search?.datasource
            },
            ai_chat: {
              ...(record.enabled_module?.ai_chat || {}),
              assistants: record.enabled_module?.ai_chat?.assistants ? record.enabled_module?.ai_chat?.assistants.map((item) => ({
                id: item
              })) : []
            }
          },
          cors: {
            ...(record.cors || {}),
            allowed_origins: record.cors?.allowed_origins ? record.cors?.allowed_origins.join(',') : ''
          },
          start_page: {
            ...(record.enabled_module?.ai_chat?.start_page_config || {}),
            display_assistants: record.enabled_module?.ai_chat?.start_page_config?.display_assistants ? record.enabled_module?.ai_chat?.start_page_config?.display_assistants.map((item) => ({
              id: item
            })) : []
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
          enabled_module: {
            ai_chat: {
              enabled: true,
              placeholder: 'Ask whatever you want...'
            },
            features: ['search_active', 'think_active'],
            search: {
              enabled: true,
              datasource: ['*'],
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
              <div className="mb-8px">
                {t('page.integration.form.labels.type_embedded_placeholder')}
              </div>
              <Form.Item
                name={['options', 'embedded_placeholder']}
                className="mb-0px"
              >
                <Input className={itemClassNames} />
              </Form.Item>
            </Form.Item>
            <Form.Item label=" ">
              <div className="mb-8px">
                {t('page.integration.form.labels.type_embedded_icon')}
              </div>
              <Form.Item
                className="mb-0px"
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
              <div className="mb-8px">
                {t('page.integration.form.labels.type_floating_placeholder')}
              </div>
              <Form.Item
                className="mb-0px"
                name={['options', 'floating_placeholder']}
              >
                <Input className={itemClassNames} />
              </Form.Item>
            </Form.Item>
            <Form.Item label=" ">
              <div className="mb-8px">
                {t('page.integration.form.labels.type_floating_icon')}
              </div>
              <Form.Item
                className="mb-0px"
                name={['options', 'floating_icon']}
              >
                <Input className={itemClassNames}  placeholder={`${window.location.origin}/icon.svg`}/>
              </Form.Item>
            </Form.Item>
          </>
        )}
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
            className="mb-0px"
            label={t('page.integration.form.labels.module_search')}
            name={['enabled_module', 'search', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
        </Form.Item>
        <Form.Item label=" " >
          <div className="mb-8px">
            {t('page.integration.form.labels.datasource')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['enabled_module', 'search', 'datasource']}
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
        </Form.Item>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.module_search_placeholder')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['enabled_module', 'search', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
          <Form.Item
            className="mb-0px"
            label={t('page.integration.form.labels.module_chat')}
            name={['enabled_module', 'ai_chat', 'enabled']}
          >
            <Switch size="small" />
          </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.module_chat_ai_assistant')}
          </div>
          <Form.Item
            name={['enabled_module', 'ai_chat', 'assistants']}
            rules={[defaultRequiredRule]}
            className="mb-0px"
          >
            <AIAssistantSelect mode="multiple" className={itemClassNames} onChange={(as) => {
              setAssistants(as)
              const startPageSettings = form.getFieldValue('start_page') || {}
              const { display_assistants = [] } = startPageSettings 
              form.setFieldValue('start_page', {
                ...startPageSettings,
                display_assistants: display_assistants.filter((item) => !!(as.find((a) => a.id === item.id)))
              })
            }}/>
          </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.module_chat_placeholder')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['enabled_module', 'ai_chat', 'placeholder']}
          >
            <Input className={itemClassNames} />
          </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.feature_Control')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['enabled_module', 'features']}
          >
            <Checkbox.Group
              options={[
                {
                  label: t('page.integration.form.labels.feature_chat_history'),
                  value: 'chat_history'
                }
              ]}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
          <ChartStartPage assistants={assistants} isSub={true} startPageSettings={record?.enabled_module?.ai_chat?.start_page_config} logo={logo} setLogo={setLogo}/>
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
          <div className="mb-8px">
            {t('page.integration.form.labels.theme')}
          </div>
          <Form.Item
            className="mb-0px"
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
          name={['cors', 'enabled']}
        >
          <Switch size="small" />
        </Form.Item>
        <Form.Item
          label=" "
          name="cors"
        >
          <div className="mb-8px">
            {t('page.integration.form.labels.allow_origin')}
          </div>
          <Form.Item
            className="mb-0px"
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
