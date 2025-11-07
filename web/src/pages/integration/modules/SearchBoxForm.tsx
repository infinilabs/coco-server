import { Form, Input, Radio, Switch } from 'antd';

import All from '@/assets/integration/all.png';
import Embedded from '@/assets/integration/embedded.png';
import Floating from '@/assets/integration/floating.png';

import { getDarkMode } from '@/store/slice/theme';

import { HotKeys } from './HotKeys';
import ChatStartPage from '@/pages/settings/modules/ChatStartPage';
import { SEARCHBOX_TYPES } from '../list';

export const SearchBoxForm = memo(props => {
  const { record, startPagelogos, setStartPagelogos, assistants, enabledList, setEnabledList } = props;
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();

  const darkMode = useAppSelector(getDarkMode);

  const [mode, setMode] = useState();

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
            <Input className={itemClassNames} placeholder={`${window.location.origin}/icon.svg`}/>
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
        <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, search: checked }))}/>
      </Form.Item>
    </Form.Item>
    {
      enabledList?.search && (
        <>
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
        </>
      )
    }
    <Form.Item label=" ">
      <Form.Item
        className="mb-0px"
        label={t('page.integration.form.labels.module_chat')}
        name={['enabled_module', 'ai_chat', 'enabled']}
      >
        <Switch size="small" onChange={(checked) => setEnabledList((state) => ({ ...state, ai_chat: checked }))}/>
      </Form.Item>
    </Form.Item>
    {
      enabledList?.ai_chat && (
        <>
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
        </>
      )
    }
    <Form.Item label=" ">
      <ChatStartPage assistants={assistants} isSub={true} startPageSettings={record?.enabled_module?.ai_chat?.start_page_config} logo={startPagelogos} setLogo={setStartPagelogos}/>
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
