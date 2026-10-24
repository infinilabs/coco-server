import { SettingOutlined } from '@ant-design/icons';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { Tabs } from 'antd';

import './index.scss';
import { SaveBarSlotContext } from './modules/SaveBar';
import AppearanceSettings from './modules/Appearance';
import AppSettings from './modules/AppSettings';
import DefaultParams from './modules/DefaultParams';
import Environment from './modules/Environment';
import ProcessingSettings from './modules/ProcessingSettings';

export function Component() {
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewSystemSettings: hasAuth('coco#system/read'),
  }

  type SettingsTab = { component: any; key: string; label: string };

  // All tabs live behind `coco#system/read`.
  //
  // App-related tabs (search/chat/KB settings) are NOT here anymore — they are
  // first-class routes promoted under their application groups in the sider
  // (search-settings / content-dedup / memory-settings / wiki-settings).
  //
  // Data-ingestion-related tabs (connector / engine AI / data security) were
  // promoted the same way under the 数据接入 group
  // (connector-settings / engine-ai / data-security routes).
  //
  // NOTE: W18 close-out originally grouped these into antd Tabs `type:'group'`
  // items, but antd 6's Tabs (@rc-component/tabs `Tab` = {key, label}) has no
  // group support — those items were silently dropped and nine tabs vanished
  // from the page. Tabs render flat until antd ships grouped items again.
  const items: SettingsTab[] = permissions.viewSystemSettings
    ? [
        { component: AppearanceSettings, key: 'appearance', label: t(`page.settings.appearance.title`) },
        { component: AppSettings, key: 'app_settings', label: t(`page.settings.app_settings.title`) },
        // 默认模型 + 文档处理合并为「默认参数」（两者同为全局兜底参数）
        { component: DefaultParams, key: 'default_params', label: t(`page.settings.default_params.title`) },
        // 解析管道：两条固定管道（文档/附件增强）的处理器开关与参数
        { component: ProcessingSettings, key: 'processing', label: t(`page.settings.processing.title`) },
        { component: Environment, key: 'environment', label: t(`page.settings.environment.title`) }
      ]
    : [];

  // 旧链接兼容：合并前的 ?tab=default_model / ?tab=document_processing 落到合并后的 tab
  const TAB_ALIAS: Record<string, string> = { default_model: 'default_params', document_processing: 'default_params' };

  // The `?tab=` query param seeds the initial tab only. Switching is driven by
  // local state on purpose: routing tab clicks through setSearchParams trips
  // the hash router's navigation blocker (POP navigations "fail silently") and
  // bounces the user to login, so the URL is deliberately left untouched.
  const requestedTab = searchParams.get('tab');
  const [activeKey, setActiveKey] = useState<string>(
    () => (requestedTab && (TAB_ALIAS[requestedTab] || requestedTab)) || items?.[0]?.key
  );

  // The component stays mounted across hash-query navigations (same route),
  // so the useState initializer alone can't pick up later ?tab= changes —
  // sync explicit URL seeds here too. Local tab clicks still bypass the URL.
  useEffect(() => {
    if (!requestedTab) return;
    const resolved = TAB_ALIAS[requestedTab] || requestedTab;
    if (items.some(item => item.key === resolved)) {
      setActiveKey(resolved);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestedTab]);

  const onChange = (key: string) => {
    setActiveKey(key);
  };

  const activeItem = items.find(item => item.key === activeKey);

  // SaveBar portal target: modules' save bars render OUTSIDE the white card
  // on the page canvas, matching the ontology page's footer-style save bar.
  const [slot, setSlot] = useState<HTMLElement | null>(null);

  return (
    <SaveBarSlotContext.Provider value={slot}>
      <ACard
        styles={{ body: { padding: 0 } }}
        title={
          <PageHeaderTitle
            description={t('page.settings.subtitle')}
            icon={<SettingOutlined />}
            title={t('page.settings.title')}
          />
        }
      >
        <Tabs
          activeKey={activeKey}
          className="settings-tabs"
          items={items}
          onChange={onChange}
        />
        <div className="settings-tabs-content">{activeItem?.component ? <activeItem.component /> : null}</div>
      </ACard>
      <div ref={setSlot} style={{ display: 'contents' }} />
    </SaveBarSlotContext.Provider>
  );
}
