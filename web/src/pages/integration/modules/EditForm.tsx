import { SearchBoxForm } from './SearchBoxForm';
import { FullscreenForm } from './FullscreenForm';
import { Button, Form, Input, Radio, Select, Spin, Switch } from 'antd';
import { useLoading, useRequest } from '@sa/hooks';
import { fetchDataSourceList } from '@/service/api';
import './EditForm.css';
import { generateRandomString } from '@/utils/common';
import { request } from '@/service/request';
import { formatESSearchResult } from '@/service/request/es';
import { FULLSCREEN_TYPES, SEARCHBOX_TYPES } from '../list';
import { getLocale, getLocaleOptions } from '@/store/slice/app';

export function isFullscreen(type) {
  return ['page', 'modal', 'fullscreen'].includes(type);
}

export const EditForm = memo(props => {
  const { defaultType, actionText, record, onSubmit } = props;
  const [type, setType] = useState('searchbox');
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();
  const [assistants, setAssistants] = useState([]);
  const [enabledList, setEnabledList] = useState({});
  const [guestEnabled, setGuestEnabled] = useState(false);

  const locale = useAppSelector(getLocale);
  const localeOptions = useAppSelector(getLocaleOptions);

  const { hasAuth } = useAuth();

  const permissions = {
    fetchDataSources: hasAuth('coco#datasource/search')
  };

  const [startPagelogos, setStartPagelogos] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    darkLoading: false,
    darkList: [],
    dark: undefined
  });

  const [searchLogos, setSearchLogos] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined,
    lightMobileLoading: false,
    lightMobileList: [],
    light_mobile: undefined
  });

  const [aiOverviewLogo, setAIOverviewLogo] = useState({
    lightLoading: false,
    lightList: [],
    light: undefined
  });

  const [widgetsLogo, setWidgetsLogo] = useState([]);

  const {
    data: result,
    loading: dataSourceLoading,
    run
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  useEffect(() => {
    if (defaultType) {
      setType(defaultType)
    }
  }, [defaultType])

  useEffect(() => {
    if (permissions.fetchDataSources) {
      run({
        from: 0,
        size: 10000
      });
    }
  }, [permissions.fetchDataSources]);

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map(item => ({ ...item._source })) || [];
  }, [JSON.stringify(result)]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { searchbox_mode, fullscreen_mode, cors = {}, enabled_module = {}, start_page = {}, payload = {}, guest = {} } = params;
    const { search = {}, ai_chat = {} } = enabled_module;
    const { datasource = [] } = search;
    const { assistants = [] } = ai_chat;
    const { ai_overview = {}, ai_widgets = {} } = payload;
    const formatGuest = {
      ...guest,
      run_as: guest.enabled && guest.run_as?.id ? guest.run_as?.id : undefined
    }
    onSubmit(
      type === 'fullscreen'
        ? {
          ...params,
          guest: formatGuest,
          enabled_module: {
            search: {
              ...search,
              enabled: true,
              datasource: datasource?.includes('*') ? ['*'] : datasource
            }
          },
          payload: {
            ...payload,
            ai_overview: {
              ...ai_overview,
              assistant: ai_overview?.assistant?.id,
              logo: {
                light: aiOverviewLogo?.light
              }
            },
            ai_widgets: {
              ...ai_widgets,
              widgets: ai_widgets.widgets
                ? ai_widgets.widgets.map((item, index) => ({
                  ...item,
                  assistant: item.assistant?.id,
                  logo: {
                    light: widgetsLogo[index]?.light
                  }
                }))
                : []
            },
            logo: {
              light: searchLogos?.light,
              light_mobile: searchLogos?.light_mobile
            }
          },
          cors: {
            ...cors,
            allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
          },
          type: fullscreen_mode
        }
        : {
          ...params,
          guest: formatGuest,
          enabled_module: {
            ...enabled_module,
            search: {
              ...search,
              datasource: datasource?.includes('*') ? ['*'] : datasource
            },
            ai_chat: {
              ...ai_chat,
              assistants: assistants.map(item => item.id),
              start_page_config: {
                ...start_page,
                display_assistants: start_page?.display_assistants?.map(item => item.id),
                logo: {
                  light: startPagelogos.light,
                  dark: startPagelogos.dark
                }
              }
            }
          },
          cors: {
            ...cors,
            allowed_origins: cors.allowed_origins?.trim() ? cors.allowed_origins.trim().split(',') : []
          },
          type: searchbox_mode
        },
      startLoading,
      endLoading
    );
  };

  const initValue = (record, locale) => {
    setType(isFullscreen(record?.type) ? 'fullscreen' : 'searchbox');
    setGuestEnabled(!!record.guest?.enabled)
    const commonValues = {
        cors: {
          ...(record.cors || {}),
          allowed_origins: record.cors?.allowed_origins ? record.cors?.allowed_origins.join(',') : ''
        },
        guest: {
          ...(record.guest || {}),
          run_as: record.guest?.enabled && record.guest?.run_as ? { id: record.guest?.run_as } : undefined
        },
        appearance: {
          ...(record.appearance || {}),
          theme: record.appearance?.theme || 'auto',
          language: record.appearance?.language || locale
        },
    }
    if (isFullscreen(record?.type)) {
      setSearchLogos(state => ({ ...state, ...(record.payload?.logo || {}) }));
      setAIOverviewLogo(state => ({ ...state, ...(record.payload?.ai_overview?.logo || {}) }));
      setWidgetsLogo(
        record.payload?.ai_widgets?.widgets ? record.payload?.ai_widgets?.widgets.map(item => item.logo) : []
      );
      const initValue = {
        ...record,
        ...commonValues,
        enabled_module: {
          ...(record.enabled_module || {}),
          search: record.enabled_module?.search
            ? {
              ...(record.enabled_module?.search || {}),
              enabled: true,
              datasource: record.enabled_module?.search?.datasource?.includes('*')
                ? ['*']
                : record.enabled_module?.search?.datasource
            }
            : {
              enabled: true,
              datasource: ['*'],
              placeholder: 'Search whatever you want...'
            }
        },
        payload: {
          ...(record.payload || {}),
          ai_overview: record.payload?.ai_overview
            ? {
              ...record.payload?.ai_overview,
              assistant: { id: record.payload?.ai_overview.assistant }
            }
            : {
              enabled: true,
              title: 'AI Overview',
              height: 200,
              output: 'markdown'
            },
          ai_widgets: record.payload?.ai_widgets
            ? {
              ...record.payload.ai_widgets,
              widgets: record.payload?.ai_widgets.widgets
                ? record.payload?.ai_widgets.widgets.map(item => ({
                  ...item,
                  assistant: { id: item.assistant }
                }))
                : []
            }
            : {
              enabled: true,
              widgets: []
            }
        },
        type: 'fullscreen',
        fullscreen_mode: FULLSCREEN_TYPES.includes(record?.type) ? record?.type : FULLSCREEN_TYPES[0],
      };
      setEnabledList({
        search: true,
        ai_overview: initValue.payload?.ai_overview?.enabled,
        ai_widgets: initValue.payload?.ai_widgets?.enabled
      });
      form.setFieldsValue(initValue);
    } else {
      setStartPagelogos(state => ({ ...state, ...(record.enabled_module?.ai_chat?.start_page_config?.logo || {}) }));
      setAssistants(
        record?.enabled_module?.ai_chat?.assistants
          ? record?.enabled_module?.ai_chat?.assistants.map(item => ({
            id: item
          }))
          : []
      );
      const initValue = {
        ...record,
        ...commonValues,
        enabled_module: {
          ...(record.enabled_module || {}),
          search: record.enabled_module?.search
            ? {
              ...(record.enabled_module?.search || {}),
              datasource: record.enabled_module?.search?.datasource?.includes('*')
                ? ['*']
                : record.enabled_module?.search?.datasource
            }
            : {
              enabled: true,
              datasource: ['*'],
              placeholder: 'Search whatever you want...'
            },
          ai_chat: record.enabled_module?.ai_chat
            ? {
              ...(record.enabled_module?.ai_chat || {}),
              assistants: record.enabled_module?.ai_chat?.assistants
                ? record.enabled_module?.ai_chat?.assistants.map(item => ({
                  id: item
                }))
                : []
            }
            : {
              enabled: true,
              placeholder: 'Ask whatever you want...'
            }
        },
        start_page: {
          ...(record.enabled_module?.ai_chat?.start_page_config || {}),
          display_assistants: record.enabled_module?.ai_chat?.start_page_config?.display_assistants
            ? record.enabled_module?.ai_chat?.start_page_config?.display_assistants.map(item => ({
              id: item
            }))
            : []
        },
        type: 'searchbox',
        searchbox_mode: SEARCHBOX_TYPES.includes(record?.type) ? record?.type : SEARCHBOX_TYPES[0],
      }
      setEnabledList({
        search: initValue.enabled_module?.search?.enabled,
        ai_chat: initValue.enabled_module?.ai_chat?.enabled
      });
      form.setFieldsValue(initValue);
    }
  };

  useEffect(() => {
    if (record) {
      initValue(record, locale);
    } else {
      if (type === 'fullscreen') {
        const initValue = {
          access_control: {
            authentication: true,
            chat_history: true
          },
          appearance: {
            theme: 'auto',
            language: locale
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
          fullscreen_mode: 'page'
        }
        setEnabledList({
          search: true,
          ai_overview: initValue.payload?.ai_overview?.enabled,
          ai_widgets: initValue.payload?.ai_widgets?.enabled
        });
        form.setFieldsValue(initValue);
      } else {
        const initValue = {
          access_control: {
            authentication: true,
            chat_history: true
          },
          appearance: {
            theme: 'auto',
            language: locale
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
          searchbox_mode: 'embedded'
        };
        setEnabledList({
          search: initValue.enabled_module?.search?.enabled,
          ai_chat: initValue.enabled_module?.ai_chat?.enabled
        });
        form.setFieldsValue(initValue);
      }
    }
  }, [record, type, locale]);

  const itemClassNames = '!w-496px';

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        colon={false}
        form={form}
        labelAlign='left'
        layout='horizontal'
        labelCol={{
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
        wrapperCol={{
          style: { maxWidth: 528, minWidth: 528, textAlign: 'left' }
        }}
      >
        <Form.Item
          label={t('page.integration.form.labels.name')}
          name='name'
          rules={[defaultRequiredRule]}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.enabled')}
          name='enabled'
          rules={[defaultRequiredRule]}
        >
          <Switch size='small' />
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.type')}
          name='type'
          rules={[defaultRequiredRule]}
        >
          <Radio.Group
            block
            className={itemClassNames}
            optionType='button'
            options={[
              { label: t('page.integration.form.labels.type_searchbox'), value: 'searchbox' },
              { label: t('page.integration.form.labels.type_fullscreen'), value: 'fullscreen' }
            ]}
            onChange={e => {
              const type = e.target.value;
              setType(type);
              if (type === 'fullscreen' && !enabledList?.search) {
                setEnabledList(state => ({ ...state, search: true }));
              }
            }}
          />
        </Form.Item>
        {type === 'searchbox' ? (
          <SearchBoxForm
            assistants={assistants}
            dataSource={dataSource}
            dataSourceLoading={dataSourceLoading}
            enabledList={enabledList}
            record={record}
            setAssistants={setAssistants}
            setEnabledList={setEnabledList}
            setStartPagelogos={setStartPagelogos}
            startPagelogos={startPagelogos}
          />
        ) : (
          <FullscreenForm
            aiOverviewLogo={aiOverviewLogo}
            dataSource={dataSource}
            dataSourceLoading={dataSourceLoading}
            enabledList={enabledList}
            searchLogos={searchLogos}
            setAIOverviewLogo={setAIOverviewLogo}
            setEnabledList={setEnabledList}
            setSearchLogos={setSearchLogos}
            setWidgetsLogo={setWidgetsLogo}
            widgetsLogo={widgetsLogo}
          />
        )}
        <Form.Item
          label={t('page.integration.form.labels.access_control')}
          name='guest'
        >
          <Form.Item
            className='mb-0px'
            label={t('page.integration.form.labels.tourist_mode')}
            layout='horizontal'
            name={['guest', 'enabled']}
          >
            <Switch size='small' onChange={(checked) => setGuestEnabled(checked)}/>
          </Form.Item>
          {
            guestEnabled && (
              <>
                <div className='pb-2 pt-1 text-[var(--ant-color-text-description)]'>{t('page.integration.form.hints.tourist_mode')}</div>
                <Form.Item
                  className='mb-0px'
                  layout='horizontal'
                  name={['guest', 'run_as']}
                >
                  <PrincipalSelect className={itemClassNames} />
                </Form.Item>
              </>
            )
          }
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.appearance')}
          name='appearance'
        >
          <div className='mb-8px'>{t('page.integration.form.labels.theme')}</div>
          <Form.Item
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
          <div className='mb-8px'>{t('page.integration.form.labels.language')}</div>
          <Form.Item
            className='mb-0px'
            name={['appearance', 'language']}
          >
            <Select
              allowClear
              className={itemClassNames}
              options={localeOptions.map((item) => ({ value: item.key, label: item.label }))}
            />
          </Form.Item>
        </Form.Item>
        <Form.Item
          label={t('page.integration.form.labels.cors')}
          name={['cors', 'enabled']}
        >
          <Switch size='small' />
        </Form.Item>
        <Form.Item
          label=' '
          name='cors'
        >
          <div className='mb-8px'>{t('page.integration.form.labels.allow_origin')}</div>
          <Form.Item
            className='mb-0px'
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
          name='description'
        >
          <Input.TextArea
            className={itemClassNames}
            rows={4}
          />
        </Form.Item>
        <Form.Item label=' '>
          <Button
            type='primary'
            onClick={() => handleSubmit()}
          >
            {actionText}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});
