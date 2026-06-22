import { Avatar, Button, Form, Input, InputNumber, Radio, Select, Switch, Upload } from 'antd';

import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';

export const FullscreenForm = memo(props => {
  const { searchLogos, setSearchLogos, aiOverviewLogo, setAIOverviewLogo, dataSourceLoading, dataSource, enabledList, setEnabledList } = props;
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
      label={t('page.integration.form.labels.mode')}
      name={'fullscreen_mode'}
      rules={[defaultRequiredRule]}
    >
      <Radio.Group
        className={itemClassNames}
        block
        options={[
          { label: t('page.integration.form.labels.mode_page'), value: 'page' },
          { label: t('page.integration.form.labels.mode_modal'), value: 'modal' },
        ]}
        optionType="button"
      />
    </Form.Item>
    <Form.Item
        label={t('page.integration.form.labels.search_settings')}
    >
      <div className="mb-8px">
        {t('page.integration.form.labels.datasource')}
      </div>
      <Form.Item
        className="mb-8px"
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
      <div className="mb-8px">
        {t('page.integration.form.labels.module_search_placeholder')}
      </div>
      <Form.Item
        className="mb-8px"
        name={['enabled_module', 'search', 'placeholder']}
      >
        <Input className={itemClassNames} />
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.module_search_welcome')}
      </div>
      <Form.Item
        className="mb-8px"
        name={['payload', 'welcome']}
      >
        <Input.TextArea rows={3} className={itemClassNames} />
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo')}
      </div>
      <Form.Item className="mb-8px" name={['logo', 'light']}>
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
            light: ''
          }));
        }}>{t('common.reset')}</Button>
      </div>
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo_dark')}
      </div>
      <Form.Item className="mb-8px" name={['logo', 'dark']}>
      <div style={{ display: "flex", gap: 22 }}>
        {renderIcon(searchLogos.dark)}
        <Upload
          {...uploadProps}
          showUploadList={false}
          fileList={searchLogos.darkList}
          beforeUpload={(file) => {
            setSearchLogos((state) => ({
              ...state,
              darkList: [file],
              darkLoading: true,
            }))
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => {
              setSearchLogos((state) => ({
                ...state,
                darkLoading: false,
                dark: reader.result
              }))
            };
            return false
          }}
        >
          <Button loading={searchLogos.darkLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
        </Upload>
        <Button className="px-0" type="link" onClick={() => {
          setSearchLogos((state) => ({
            ...state,
            darkLoading: false,
            dark: ''
          }));
        }}>{t('common.reset')}</Button>
      </div>
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo_mobile')}
      </div>
      <Form.Item className="mb-8px" name={['logo', 'light_mobile']}>
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
            light_mobile: ''
          }))
        }}>{t('common.reset')}</Button>
      </div>
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.logo_mobile_dark')}
      </div>
      <Form.Item className="mb-8px" name={['logo', 'dark_mobile']}>
      <div style={{ display: "flex", gap: 22 }}>
        {renderIcon(searchLogos.dark_mobile)}
        <Upload
          {...uploadProps}
          showUploadList={false}
          fileList={searchLogos.darkMobileList}
          beforeUpload={(file) => {
            setSearchLogos((state) => ({
              ...state,
              darkMobileList: [file],
              darkMobileLoading: true,
            }))
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => {
              setSearchLogos((state) => ({
                ...state,
                darkMobileLoading: false,
                dark_mobile: reader.result
              }))
            };
            return false
          }}
        >
          <Button loading={searchLogos.darkMobileLoading} icon={<SvgIcon className="text-12px" icon="mdi:upload" />}>{t('common.upload')}</Button>
        </Upload>
        <Button className="px-0" type="link" onClick={() => {
          setSearchLogos((state) => ({
            ...state,
            darkMobileLoading: false,
            dark_mobile: ''
          }))
        }}>{t('common.reset')}</Button>
      </div>
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.module_ai_overview')}
      </div>
      <Form.Item
        className="mb-0px"
        name={['payload', 'ai_overview', 'enabled']}
        valuePropName="checked"
      >
        <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, ai_overview: checked }))}/>
      </Form.Item>
      {
        enabledList?.ai_overview && (
          <>
            <div className="mb-8px pt-8px">
              {t('page.integration.form.labels.module_ai_overview_title')}
            </div>
            <Form.Item
              name={['payload', 'ai_overview', 'title']}
              className="mb-8px"
            >
              <Input className={itemClassNames} />
            </Form.Item>
            <div className="mb-8px">
              {t('page.integration.form.labels.logo')}
            </div>
            <Form.Item className="mb-8px" name={['payload', 'ai_overview', 'logo']}>
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
                  light: ''
                }));
              }}>{t('common.reset')}</Button>
            </div>
            </Form.Item>
            <div className="mb-8px">
              {t('page.integration.form.labels.module_ai_overview_height')}
            </div>
            <Form.Item
              name={['payload', 'ai_overview', 'height']}
              className="mb-8px"
            >
              <InputNumber className={itemClassNames} min={0} step={1}/>
            </Form.Item>
            <div className="mb-8px">
              {t('page.integration.form.labels.module_chat_ai_assistant')}
            </div>
            <Form.Item
              name={['payload', 'ai_overview', 'assistant']}
              rules={enabledList?.ai_overview ? [defaultRequiredRule] : []}
              className="mb-8px"
            >
              <AIAssistantSelect className={itemClassNames}/>
            </Form.Item>
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
          </>
        )
      }
    </Form.Item>
    <Form.Item
      label={t('page.integration.form.labels.conversation_settings')}
    >
      <div className="mb-8px">
        {t('page.integration.form.labels.deep_think_assistant')}
      </div>
      <Form.Item
        className="mb-8px"
        name="deep_think_assistant"
      >
        <AIAssistantSelect allowClear className={itemClassNames} filter={{ type: ['deep_think'] }} />
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.deep_research_assistant')}
      </div>
      <Form.Item
        className="mb-0px"
        name="deep_research_assistant"
      >
        <AIAssistantSelect allowClear className={itemClassNames} filter={{ type: ['deep_research'] }} />
      </Form.Item>
    </Form.Item>
    </>
  );
});
