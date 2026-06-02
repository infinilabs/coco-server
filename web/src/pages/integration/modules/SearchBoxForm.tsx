import { Form, Input, Radio, Select, Switch } from 'antd';

import All from '@/assets/integration/all.png';
import Embedded from '@/assets/integration/embedded.png';
import Floating from '@/assets/integration/floating.png';

import { getDarkMode } from '@/store/slice/theme';

import { HotKeys } from './HotKeys';
import ChatStartPage from '@/pages/settings/modules/ChatStartPage';
import { SEARCHBOX_TYPES } from '../list';
import AIAssistantSelect from '@/pages/ai-assistant/modules/AIAssistantSelect';
import { getServer } from '@/store/slice/server';
import normalizeUrl from 'normalize-url';

export const SearchBoxForm = memo(props => {
  const { form, record, startPagelogos, setStartPagelogos, assistants, setAssistants, dataSourceLoading, dataSource, enabledList, setEnabledList } = props;
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();

  const darkMode = useAppSelector(getDarkMode);

  const [mode, setMode] = useState();

  const server = useAppSelector(getServer);

  useEffect(() => {
    const mode = record
      ? SEARCHBOX_TYPES.includes(record?.type) ? record?.type : SEARCHBOX_TYPES[0]
      : SEARCHBOX_TYPES[0]
    setMode(mode);
  }, [record]);

  const itemClassNames = '!w-496px';
  const imageClassNames = darkMode ? 'brightness-30 saturate-0' : '';

  return (
    <>
    <Form.Item
      label={t('page.integration.form.labels.mode')}
      name="searchbox_mode"
      rules={[defaultRequiredRule]}
    >
      <Radio.Group
        className="custom-radio-group"
        options={[
          {
            label: (
              <div>
                {t('page.integration.form.labels.mode_embedded')}
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
                {t('page.integration.form.labels.mode_floating')}
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
                {t('page.integration.form.labels.mode_all')}
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
          setMode(e.target.value);
        }}
      />
    </Form.Item>
    {['embedded', 'all'].includes(mode) && (
      <>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.mode_embedded_placeholder')}
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
            {t('page.integration.form.labels.mode_embedded_icon')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['options', 'embedded_icon']}
          >
            <Input className={itemClassNames} placeholder={`${normalizeUrl(`${server}/icon.svg`)}`}/>
          </Form.Item>
        </Form.Item>
      </>
    )}
    {['floating', 'all'].includes(mode) && (
      <>
        <Form.Item label=" ">
          <div className="mb-8px">
            {t('page.integration.form.labels.mode_floating_placeholder')}
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
            {t('page.integration.form.labels.mode_floating_icon')}
          </div>
          <Form.Item
            className="mb-0px"
            name={['options', 'floating_icon']}
          >
            <Input className={itemClassNames}  placeholder={`${normalizeUrl(`${server}/icon.svg`)}`}/>
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
      label={t('page.integration.form.labels.search_settings')}
    >
      <div className="mb-8px">
        {t('page.integration.form.labels.enabled')}
      </div>
      <Form.Item
        name={['enabled_module', 'search', 'enabled']}
        valuePropName="checked"
        className="mb-0px"
      >
        <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, search: checked }))}/>
      </Form.Item>
      {
        enabledList?.search && (
          <>
            <div className="mb-8px pt-8px">
              {t('page.integration.form.labels.datasource')}
            </div>
            <Form.Item
              name={['enabled_module', 'search', 'datasource']}
              rules={[defaultRequiredRule]}
              className="mb-8px"
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
              name={['enabled_module', 'search', 'placeholder']}
              className="mb-0px"
            >
              <Input className={itemClassNames} />
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
        className="mb-8px"
        name="deep_research_assistant"
      >
        <AIAssistantSelect allowClear className={itemClassNames} filter={{ type: ['deep_research'] }} />
      </Form.Item>
      <div className="mb-8px">
        {t('page.integration.form.labels.module_chat')}
      </div>
      <Form.Item
        className="mb-0px"
        name={['enabled_module', 'ai_chat', 'enabled']}
        valuePropName="checked"
      >
        <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, ai_chat: checked }))}/>
      </Form.Item>
      {
        enabledList?.ai_chat && (
          <>
            <div className="mb-8px pt-8px">
              {t('page.integration.form.labels.module_chat_ai_assistant')}
            </div>
            <Form.Item
              name={['enabled_module', 'ai_chat', 'assistants']}
              rules={[defaultRequiredRule]}
              className="mb-8px"
            >
              <AIAssistantSelect mode="multiple" className={itemClassNames} onChange={(as) => {
                setAssistants(as)
                const startPageSettings = form.getFieldValue('start_page') || {}
                const { display_assistants = [] } = startPageSettings 
                form.setFieldValue('start_page', {
                  ...startPageSettings,
                  display_assistants: display_assistants.filter((item) => !!item && !!(as.find((a) => a.id === item.id)))
                })
              }}/>
            </Form.Item>
            <div className="mb-8px">
              {t('page.integration.form.labels.module_chat_placeholder')}
            </div>
            <Form.Item
              className="mb-8px"
              name={['enabled_module', 'ai_chat', 'placeholder']}
            >
              <Input className={itemClassNames} />
            </Form.Item>
            <div className="mb-8px">
              {t('page.integration.form.labels.module_chat_start_page')}
            </div>
            <Form.Item className="mb-0px">
              <ChatStartPage assistants={assistants} isSub={true} startPageSettings={record?.enabled_module?.ai_chat?.start_page_config} logo={startPagelogos} setLogo={setStartPagelogos}/>
            </Form.Item>
          </>
        )
      }
    </Form.Item>
    {/* <Form.Item label=" ">
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
    </Form.Item> */}
    </>
  );
});
