import { Button, Form, Input, Radio, Select, Spin, Switch } from 'antd';

import './EditForm.css';
import { useLoading, useRequest } from '@sa/hooks';

import { fetchDataSourceList } from '@/service/api';

import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import { PlusOutlined } from '@ant-design/icons';

function generateRandomString(size) {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

export const SearchPageForm = memo(props => {
  const { actionText, onSubmit, record, type, setType } = props;
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();

  const {
    data: result,
    loading: dataSourceLoading,
    run
  } = useRequest(fetchDataSourceList, {
    manual: true
  });

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { cors = {}, payload = {} } = params;
    const { search = {}, ai_overview = {}, ai_widgets = {} } = payload
    const { datasource = [] } = search
    onSubmit(
      {
        ...params,
        type: 'searchpage',
        payload: {
          ...payload,
          search: {
            ...search,
            datasource: datasource?.includes('*') ? ['*'] : datasource
          },
          ai_overview: {
            ...ai_overview,
            assistant: ai_overview?.assistant?.id
          },
          ai_widgets: {
            ...ai_widgets,
            widgets: ai_widgets.widgets? ai_widgets.widgets.map((item) => ({
              ...item,
              assistant: item.assistant?.id
            })) : []
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

    const initValue = record
      ? {
          ...record,
          payload: {
            ...(record.payload || {}),
            search: record.payload?.search ? {
              ...(record.payload?.search || {}),
              datasource: record.payload?.search?.datasource?.includes('*') ? ['*'] : record.payload?.search?.datasource
            } : {
              enabled: true,
              datasource: ['*'],
              placeholder: 'Search whatever you want...'
            },
            ai_overview: record.payload?.ai_overview ? {
              ...record.payload?.ai_overview,
              assistant: { id: record.payload?.ai_overview.assistant }
            } : {
              enabled: true,
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
          type: 'searchpage',
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
          payload: {
            search: {
              enabled: true,
              datasource: ['*'],
              placeholder: 'Search whatever you want...'
            },
            ai_overview: {
              enabled: true,
            },
            ai_widgets: {
              enabled: true,
              widgets: []
            }
          },
          name: `widget-${generateRandomString(8)}`,
          enabled: true,
          type: 'searchpage',
        };
    form.setFieldsValue(initValue);
  }, [record]);

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
              { label: t('page.integration.form.labels.type_searchpage'), value: 'searchpage' },
            ]}
            optionType="button"
            onChange={(e) => setType(e.target.value)}
          />
        </Form.Item>
        <Form.Item
            label={t('page.integration.form.labels.enable_module')}
            name="payload"
        >
            <Form.Item
                className="mb-0px"
                label={t('page.integration.form.labels.module_search')}
            >
            </Form.Item>
        </Form.Item>
        <Form.Item label=" " >
            <div className="mb-8px">
                {t('page.integration.form.labels.datasource')}
            </div>
            <Form.Item
                className="mb-0px"
                name={['payload', 'search', 'datasource']}
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
                name={['payload', 'search', 'placeholder']}
            >
                <Input className={itemClassNames} />
            </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
            <Form.Item
                className="mb-0px"
                label={t('page.integration.form.labels.module_ai_overview')}
                name={['payload', 'ai_overview', 'enabled']}
            >
                <Switch size="small" />
            </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
            <div className="mb-8px">
                {t('page.integration.form.labels.module_chat_ai_assistant')}
            </div>
            <Form.Item
                name={['payload', 'ai_overview', 'assistant']}
                rules={[defaultRequiredRule]}
                className="mb-0px"
            >
                <AIAssistantSelect className={itemClassNames}/>
            </Form.Item>
        </Form.Item>
        
        <Form.Item label=" ">
            <Form.Item
                className="mb-0px"
                label={t('page.integration.form.labels.module_ai_widgets')}
                name={['payload', 'ai_widgets', 'enabled']}
            >
                <Switch size="small" />
            </Form.Item>
        </Form.Item>
        <Form.Item label=" ">
            <div className="mb-8px">
                {t('page.settings.app_settings.chat_settings.labels.assistant')}
            </div>
            <Form.Item className="mb-0px">
                <Form.List name={['payload', 'ai_widgets', 'widgets']}>
                    {(fields, { add, remove }) => (
                      <>
                          {fields.map((field, index) => {
                            const { key, name, ...restField } = field;
                            return (
                                <Form.Item key={index} className="mb-0px">
                                    <div className="flex gap-6px">
                                        <Form.Item
                                            {...restField}
                                            name={[name, 'assistant']}
                                            rules={[defaultRequiredRule]}
                                            className="flex-1 mb-8px"
                                        >
                                          <AIAssistantSelect className={itemClassNames}/>
                                        </Form.Item>
                                        <Form.Item className="mb-8px">
                                            <span onClick={() => remove(field.name)}><SvgIcon className="text-16px cursor-pointer" icon="mdi:minus-circle-outline" /></span>
                                        </Form.Item>
                                    </div>
                                </Form.Item>
                            )
                          })}
                          <Form.Item className="mb-0px">
                              <Button className="!w-80px" type="primary" disabled={fields.length >= 8} icon={<PlusOutlined />} onClick={() => add()}></Button>
                          </Form.Item>
                      </>
                    )}
                </Form.List>
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
