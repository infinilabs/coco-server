import { SearchBoxForm } from "./SearchBoxForm";
import { FullscreenForm } from "./FullscreenForm";
import { Button, Form, Input, Radio, Select, Spin, Switch } from "antd";
import { useLoading } from '@sa/hooks';
import { fetchDataSourceList } from "@/service/api";
import './EditForm.css';

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
  const { actionText, record, onSubmit } = props;
  const [type, setType] = useState('searchbox');
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();
  const [assistants, setAssistants] = useState([])
  const [enabledList, setEnabledList] = useState({})

  const [startPagelogos, setStartPagelogos] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    darkLoading: false,
    darkList: [],
    dark: undefined,
  })

  const [searchLogos, setSearchLogos] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    lightMobileLoading: false,
    lightMobileList: [],
    'light_mobile': undefined,
  })

  const [aiOverviewLogo, setAIOverviewLogo] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
  })

  const [widgetsLogo, setWidgetsLogo] = useState([])

  const {
    data: result,
    loading: dataSourceLoading,
    run
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  useEffect(() => {
    run({
      from: 0,
      size: 10000
    });
  }, []);

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map(item => ({ ...item._source })) || [];
  }, [JSON.stringify(result)]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { mode, cors = {}, enabled_module = {}, start_page = {}, payload = {} } = params;
    const { search = {}, ai_chat = {} } = enabled_module
    const { datasource = [] } = search
    const { assistants = [] } = ai_chat
    const { ai_overview = {}, ai_widgets = {} } = payload
    onSubmit(type === 'fullscreen' ? {
        ...params,
        type: 'fullscreen',
        enabled_module: {
          search: {
            ...search,
            enabled: true,
            datasource: datasource?.includes('*') ? ['*'] : datasource,
          }
        },
        payload: {
          ...payload,
          ai_overview: {
            ...ai_overview,
            assistant: ai_overview?.assistant?.id,
            logo: {
              "light": aiOverviewLogo?.light,
            }
          },
          ai_widgets: {
            ...ai_widgets,
            widgets: ai_widgets.widgets? ai_widgets.widgets.map((item, index) => ({
              ...item,
              assistant: item.assistant?.id,
              logo: {
                "light": widgetsLogo[index]?.light
              }
            })) : []
          },
          "logo": {
            "light": searchLogos?.light,
            "light_mobile": searchLogos?.light_mobile
          },
        },
        cors: {
          ...cors,
          allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
        },
      } : {
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
                "light": startPagelogos.light,
                "dark": startPagelogos.dark
              },
            }
          }
        },
        cors: {
          ...cors,
          allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
        },
        type: mode
      },
      startLoading,
      endLoading
    );
  };

  const initValue = (record) => {
    setType(record?.type === 'fullscreen' ? 'fullscreen': 'searchbox')
    if (record?.type === 'fullscreen') {
      if (record) {
        setSearchLogos((state) => ({ ...state, ...(record.payload?.logo || {}) }))
        setAIOverviewLogo((state) => ({ ...state, ...(record.payload?.ai_overview?.logo || {}) }))
        setWidgetsLogo(record.payload?.ai_widgets?.widgets ? record.payload?.ai_widgets?.widgets.map((item) => item.logo) : [])
      }
      const initValue = record
        ? {
            ...record,
            enabled_module: {
              ...(record.enabled_module || {}),
              search: record.enabled_module?.search ? {
                ...(record.enabled_module?.search || {}),
                enabled: true,
                datasource: record.enabled_module?.search?.datasource?.includes('*') ? ['*'] : record.enabled_module?.search?.datasource
              } : {
                enabled: true,
                datasource: ['*'],
                placeholder: 'Search whatever you want...'
              }
            },
            payload: {
              ...(record.payload || {}),
              ai_overview: record.payload?.ai_overview ? {
                ...record.payload?.ai_overview,
                assistant: { id: record.payload?.ai_overview.assistant }
              } : {
                enabled: true,
                title: 'AI Overview',
                height: 200,
                output: 'markdown'
              },
              ai_widgets: record.payload?.ai_widgets ? {
                ...record.payload.ai_widgets,
                widgets: record.payload?.ai_widgets.widgets ? record.payload?.ai_widgets.widgets.map((item) => ({
                  ...item,
                  assistant: { id: item.assistant }
                })) : []
              } : {
                enabled: true,
                widgets: []
              }
            },
            cors: {
              ...(record.cors || {}),
              allowed_origins: record.cors?.allowed_origins ? record.cors?.allowed_origins.join(',') : ''
            },
            type: 'fullscreen',
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
              search: {
                enabled: true,
                datasource: ['*'],
                placeholder: 'Search whatever you want...'
              }
            },
            payload: {
              ai_overview: {
                enabled: true,
                title: 'AI Overview',
                height: 200
              },
              ai_widgets: {
                enabled: true,
                widgets: []
              }
            },
            name: `widget-${generateRandomString(8)}`,
            enabled: true,
            type: 'fullscreen',
          }
      setEnabledList({
        search: true,
        ai_overview: initValue.payload?.ai_overview?.enabled,
        ai_widgets: initValue.payload?.ai_widgets?.enabled
      })
      form.setFieldsValue(initValue);
    } else {
      if (record) {
        setStartPagelogos((state) => ({ ...state, ...(record.enabled_module?.ai_chat?.start_page_config?.logo || {}) }))
        setAssistants(record?.enabled_module?.ai_chat?.assistants ? record?.enabled_module?.ai_chat?.assistants.map((item) => ({
          id: item
        })) : [])
      }
      const initValue = record
        ? {
            ...record,
            enabled_module: {
              ...(record.enabled_module || {}),
              search: record.enabled_module?.search ? {
                ...(record.enabled_module?.search || {}),
                datasource: record.enabled_module?.search?.datasource?.includes('*') ? ['*'] : record.enabled_module?.search?.datasource
              } : {
                enabled: true,
                datasource: ['*'],
                placeholder: 'Search whatever you want...'
              },
              ai_chat: record.enabled_module?.ai_chat ? {
                ...(record.enabled_module?.ai_chat || {}),
                assistants: record.enabled_module?.ai_chat?.assistants ? record.enabled_module?.ai_chat?.assistants.map((item) => ({
                  id: item
                })) : []
              } : {
                enabled: true,
                placeholder: 'Ask whatever you want...'
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
            },
            type: 'searchbox',
            mode: ['embedded', 'floating', 'all'].includes(record?.type) ? record?.type : 'embedded'
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
            type: 'searchbox',
            mode: 'embedded'
          };
      setEnabledList({
        search: initValue.enabled_module?.search?.enabled,
        ai_chat: initValue.enabled_module?.ai_chat?.enabled,
      })
      form.setFieldsValue(initValue);
    }
  }

  useEffect(() => {
    initValue(record)
  }, [record])

  const itemClassNames = '!w-496px';

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
          name={'type'}
          rules={[defaultRequiredRule]}
        >
          <Radio.Group
            className={itemClassNames}
            block
            options={[
              { label: t('page.integration.form.labels.type_searchbox'), value: 'searchbox' },
              { label: t('page.integration.form.labels.type_fullscreen'), value: 'fullscreen' },
            ]}
            optionType="button"
            onChange={(e) => {
              const type = e.target.value
              setType(type)
              if (type === 'fullscreen' && !enabledList?.search) {
                setEnabledList((state) => ({ ...state, search: true }))
              }
            }}
          />
        </Form.Item>
        {
          (
            type === 'searchbox' ? (
              <SearchBoxForm 
                {...props} 
                type={type} 
                setType={setType} 
                form={form} 
                loading={loading} 
                startLoading={startLoading} 
                endLoading={endLoading}
                startPagelogos={startPagelogos}
                setStartPagelogos={setStartPagelogos}
                assistants={assistants}
                setAssistants={setAssistants}
                dataSourceLoading={dataSourceLoading}
                dataSource={dataSource}
                enabledList={enabledList}
                setEnabledList={setEnabledList}
              />
            ) : (
              <FullscreenForm 
                {...props} 
                type={type} 
                setType={setType} 
                form={form} 
                loading={loading} 
                startLoading={startLoading}
                endLoading={endLoading}
                searchLogos={searchLogos}
                setSearchLogos={setSearchLogos}
                aiOverviewLogo={aiOverviewLogo}
                setAIOverviewLogo={setAIOverviewLogo}
                widgetsLogo={widgetsLogo}
                setWidgetsLogo={setWidgetsLogo}
                dataSourceLoading={dataSourceLoading}
                dataSource={dataSource}
                enabledList={enabledList}
                setEnabledList={setEnabledList}
              />
            )
          )
        }
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
  )
});
