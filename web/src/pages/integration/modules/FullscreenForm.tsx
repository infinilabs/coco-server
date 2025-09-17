import { Avatar, Button, Form, Input, InputNumber, Select, Switch, Upload } from 'antd';

import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import { PlusOutlined } from '@ant-design/icons';
import { cloneDeep } from 'lodash';

export const FullscreenForm = memo(props => {
  const { searchLogos, setSearchLogos, aiOverviewLogo, setAIOverviewLogo, widgetsLogo, setWidgetsLogo, dataSourceLoading, dataSource, enabledList, setEnabledList } = props;
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();

  const renderIcon = (base64) => {
    if (base64) {
      return (
        <div className="chart-start-page-image css-var-r0 ant-btn">
          <Avatar shape="square" src={base64} />
        </div>
      );
    }
    return null;
  }

  const uploadProps = {
    name: "file",
    action: "",
    accept: "image/*,.svg",
  };

  const itemClassNames = '!w-496px';

  return (
    <>
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
        <div className="mb-8px">
            {t('page.integration.form.labels.module_search_welcome')}
        </div>
        <Form.Item
            className="mb-0px"
            name={['payload', 'welcome']}
        >
            <Input.TextArea rows={3} className={itemClassNames} />
        </Form.Item>
    </Form.Item>
    <Form.Item label=" " name={['logo', 'light']}>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo')}
      </div>
      <div style={{ display: "flex", gap: 22 }}>
        {renderIcon(searchLogos.light)}
        <Upload
          {...uploadProps}
          showUploadList={false}
          fileList={searchLogos.lightList}
          beforeUpload={(file) => {
            setSearchLogos((state) => ({
              ...state,
              lightList: [file],
              lightLoading: true,
            }))
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => {
              setSearchLogos((state) => ({
                ...state,
                lightLoading: false,
                light: reader.result
              }))
            };
            return false
          }}
        >
          <Button loading={searchLogos.lightLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
        </Upload>
        <Button className="px-0" type="link" onClick={() => {
          setSearchLogos((state) => ({
            ...state,
            lightLoading: false,
            light: undefined
          }));
        }}>{t('common.reset')}</Button>
      </div>
    </Form.Item>
    <Form.Item label=" " name={['logo', 'logo-mobile']}>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo_mobile')}
      </div>
      <div style={{ display: "flex", gap: 22 }}>
        {renderIcon(searchLogos.light_mobile)}
        <Upload
          {...uploadProps}
          showUploadList={false}
          fileList={searchLogos.lightMobileList}
          beforeUpload={(file) => {
            setSearchLogos((state) => ({
              ...state,
              lightMobileList: [file],
              lightMobileLoading: true,
            }))
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => {
              setSearchLogos((state) => ({
                ...state,
                lightMobileLoading: false,
                light_mobile: reader.result
              }))
            };
            return false
          }}
        >
          <Button loading={searchLogos.lightMobileLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
        </Upload>
        <Button className="px-0" type="link" onClick={() => {
          setSearchLogos((state) => ({
            ...state,
            lightMobileLoading: false,
            light_mobile: undefined
          }))
        }}>{t('common.reset')}</Button>
      </div>
    </Form.Item>
    <Form.Item label=" ">
        <Form.Item
            className="mb-0px"
            label={t('page.integration.form.labels.module_ai_overview')}
            name={['payload', 'ai_overview', 'enabled']}
        >
            <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, ai_overview: checked }))}/>
        </Form.Item>
    </Form.Item>
    {
      enabledList?.ai_overview && (
        <>
          <Form.Item label=" ">
              <div className="mb-8px">
                  {t('page.integration.form.labels.module_ai_overview_title')}
              </div>
              <Form.Item
                  name={['payload', 'ai_overview', 'title']}
                  className="mb-0px"
              >
                  <Input className={itemClassNames} />
              </Form.Item>
          </Form.Item>
          <Form.Item label=" " name={['payload', 'ai_overview', 'logo']}>
            <div className="mb-8px">
              {t('page.integration.form.labels.logo')}
            </div>
            <div style={{ display: "flex", gap: 22 }}>
              {renderIcon(aiOverviewLogo?.light)}
              <Upload
                {...uploadProps}
                showUploadList={false}
                fileList={aiOverviewLogo.lightList}
                beforeUpload={(file) => {
                  setAIOverviewLogo((state) => ({
                    ...state,
                    lightList: [file],
                    lightLoading: true,
                  }))
                  const reader = new FileReader();
                  reader.readAsDataURL(file);
                  reader.onload = () => {
                    setAIOverviewLogo((state) => ({
                      ...state,
                      lightLoading: false,
                      light: reader.result
                    }))
                  };
                  return false
                }}
              >
                <Button loading={aiOverviewLogo?.lightLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
              </Upload>
              <Button className="px-0" type="link" onClick={() => {
                setAIOverviewLogo((state) => ({
                  ...state,
                  lightLoading: false,
                  light: undefined
                }));
              }}>{t('common.reset')}</Button>
            </div>
          </Form.Item>
          <Form.Item label=" ">
              <div className="mb-8px">
                  {t('page.integration.form.labels.module_ai_overview_height')}
              </div>
              <Form.Item
                  name={['payload', 'ai_overview', 'height']}
                  className="mb-8px"
              >
                  <InputNumber className={itemClassNames} min={0} step={1}/>
              </Form.Item>
          </Form.Item>                
          <Form.Item label=" ">
              <div className="mb-8px">
                  {t('page.integration.form.labels.module_chat_ai_assistant')}
              </div>
              <Form.Item
                  name={['payload', 'ai_overview', 'assistant']}
                  rules={enabledList?.ai_overview ? [defaultRequiredRule] : []}
                  className="mb-0px"
              >
                  <AIAssistantSelect className={itemClassNames}/>
              </Form.Item>
          </Form.Item>
          <Form.Item label=" ">
            <div className="mb-8px">
                {t('page.integration.form.labels.module_ai_overview_output')}
            </div>
            <Form.Item
                name={['payload', 'ai_overview', 'output']}
                className="mb-0px"
            >
                <Select className={itemClassNames}>
                  <Select.Option value="markdown">Markdown</Select.Option>
                  <Select.Option value="html">HTML</Select.Option>
                  <Select.Option value="text">Text</Select.Option>
                </Select>
            </Form.Item>
          </Form.Item>
        </>
      )
    }
    <Form.Item label=" ">
        <Form.Item
            className="mb-0px"
            label={t('page.integration.form.labels.module_ai_widgets')}
            name={['payload', 'ai_widgets', 'enabled']}
        >
            <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, ai_widgets: checked }))}/>
        </Form.Item>
    </Form.Item>
    {
      enabledList?.ai_widgets && (
        <>
          <Form.Item label=" ">
              <Form.Item className="mb-0px">
                  <Form.List name={['payload', 'ai_widgets', 'widgets']}>
                      {(fields, { add, remove }) => (
                        <>
                            {fields.map((field, index) => {
                              const { key, name, ...restField } = field;
                              return (
                                  <div key={index} >
                                      <div className="mb-8px">
                                          {t('page.integration.form.labels.module_ai_widgets_title')} {` ${index + 1}`}
                                      </div>
                                      <div className="mb-8px">
                                          {t('page.integration.form.labels.module_ai_overview_title')}
                                      </div>
                                      <Form.Item
                                          name={[name, 'title']}
                                          className="mb-8px"
                                      >
                                          <Input className={itemClassNames} />
                                      </Form.Item>
                                      <div className="mb-8px">
                                        {t('page.integration.form.labels.logo')}
                                      </div>
                                      <div style={{ display: "flex", gap: 22 }} className="mb-8px">
                                        {renderIcon(widgetsLogo[name]?.light)}
                                        <Upload
                                          {...uploadProps}
                                          showUploadList={false}
                                          fileList={widgetsLogo[name]?.lightList}
                                          beforeUpload={(file) => {
                                            setWidgetsLogo((logos) => {
                                              const newLogos = cloneDeep(logos)
                                              newLogos[name] = {
                                                ...(newLogos[name] || {}),
                                                lightList: [file],
                                                lightLoading: true,
                                              }
                                              return newLogos
                                            })
                                            const reader = new FileReader();
                                            reader.readAsDataURL(file);
                                            reader.onload = () => {
                                              setWidgetsLogo((logos) => {
                                                const newLogos = cloneDeep(logos)
                                                newLogos[name] = {
                                                  ...(newLogos[name] || {}),
                                                  lightLoading: false,
                                                  light: reader.result
                                                }
                                                return newLogos
                                              })
                                            };
                                            return false
                                          }}
                                        >
                                          <Button loading={widgetsLogo[name]?.lightLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
                                        </Upload>
                                        <Button className="px-0" type="link" onClick={() => {
                                          setWidgetsLogo((logos) => {
                                            const newLogos = cloneDeep(logos)
                                            newLogos[name] = {
                                              ...(newLogos[name] || {}),
                                              lightLoading: false,
                                              light: undefined
                                            }
                                            return newLogos
                                          })
                                        }}>{t('common.reset')}</Button>
                                      </div>
                                      <div className="mb-8px">
                                          {t('page.integration.form.labels.module_ai_overview_height')}
                                      </div>
                                      <Form.Item
                                          name={[name, 'height']}
                                          className="mb-8px"
                                      >
                                          <InputNumber className={itemClassNames} min={0} step={1}/>
                                      </Form.Item>
                                      <div className="mb-8px">
                                          {t('page.integration.form.labels.module_chat_ai_assistant')}
                                      </div>
                                      <Form.Item className="mb-8px">
                                        <div className="flex gap-6px">
                                            <Form.Item
                                                {...restField}
                                                name={[name, 'assistant']}
                                                rules={enabledList?.ai_widgets?.[name]? [defaultRequiredRule] : []}
                                                className="flex-1 mb-8px"
                                            >
                                              <AIAssistantSelect className={itemClassNames}/>
                                            </Form.Item>
                                            <Form.Item className="mb-8px">
                                                <span onClick={() => remove(field.name)}><SvgIcon className="text-16px cursor-pointer" icon="mdi:minus-circle-outline" /></span>
                                            </Form.Item>
                                        </div>
                                      </Form.Item>
                                      <div className="mb-8px">
                                        输出类型
                                      </div>
                                      <Form.Item
                                          name={[name, 'output']}
                                          className="mb-8px"
                                      >
                                          <Select className={itemClassNames}>
                                            <Select.Option value="markdown">Markdown</Select.Option>
                                            <Select.Option value="html">HTML</Select.Option>
                                            <Select.Option value="text">Text</Select.Option>
                                          </Select>
                                      </Form.Item>
                                  </div>
                                  
                              )
                            })}
                            <Form.Item className="mb-0px">
                                <Button className="!w-80px" type="primary" disabled={fields.length >= 8} icon={<PlusOutlined />} onClick={() => {
                                  add({ title: '', height: 200, output: 'markdown' })
                                  setWidgetsLogo((logos) => {
                                    const newLogos = cloneDeep(logos)
                                    newLogos.push({
                                      lightLoading: false,
                                      lightList: [],
                                      light: undefined,
                                    }) 
                                    return newLogos
                                  })
                                }}></Button>
                            </Form.Item>
                        </>
                      )}
                  </Form.List>
              </Form.Item>
          </Form.Item>
        </>
      )
    }
    </>
  );
});
