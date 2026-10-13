import { Tabs } from 'antd';

import './index.scss';
import AppearanceSettings from './modules/Appearance';
import AppSettings from './modules/AppSettings';
import ConnectorSettings from './modules/Connector';
import SearchSettings from './modules/SearchSettings';
import WikiSettings from './modules/WikiSettings';
import MemorySettings from './modules/MemorySettings';
import DefaultModel from './modules/DefaultModel';
import DocProcessing from './modules/DocProcessing';
import DataSecurity from './modules/DataSecurity';
import EngineAI from './modules/EngineAI';
import Dedup from './modules/Dedup';
import Environment from './modules/Environment';

export function Component() {
  const [searchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewConnector: hasAuth('coco#connector/search'),
    viewSystemSettings: hasAuth('coco#system/read'),
  }

  type SettingsTab = { component: any; key: string; label: string };

  // All tabs live behind `coco#system/read`; the connector tab keeps its own
  // permission gate and first position.
  //
  // NOTE: W18 close-out originally grouped these into antd Tabs `type:'group'`
  // items, but antd 6's Tabs (@rc-component/tabs `Tab` = {key, label}) has no
  // group support — those items were silently dropped and nine tabs vanished
  // from the page. Tabs render flat until antd ships grouped items again.
  const systemTabs: SettingsTab[] = permissions.viewSystemSettings
    ? [
        { component: AppearanceSettings, key: 'appearance', label: t(`page.settings.appearance.title`) },
        { component: AppSettings, key: 'app_settings', label: t(`page.settings.app_settings.title`) },
        { component: SearchSettings, key: 'search_settings', label: t(`page.settings.search_settings.title`) },
        { component: EngineAI, key: 'engine_ai', label: t(`page.settings.engine_ai.title`) },
        { component: DocProcessing, key: 'document_processing', label: t(`page.settings.document_processing.title`) },
        { component: DataSecurity, key: 'data_security', label: t(`page.settings.data_security.title`) },
        { component: Dedup, key: 'dedup', label: t(`page.settings.dedup.title`) },
        { component: Environment, key: 'environment', label: t(`page.settings.environment.title`) },
        { component: WikiSettings, key: 'wiki_settings', label: t(`page.settings.wiki_settings.title`) },
        { component: MemorySettings, key: 'memory_settings', label: t(`page.settings.memory_settings.title`) },
        { component: DefaultModel, key: 'default_model', label: t(`page.settings.default_model.title`) }
      ]
    : [];

  const items: any[] = [];
  if (permissions.viewConnector) {
    items.push({
      component: ConnectorSettings,
      key: 'connector',
      label: t(`page.settings.connector.title`)
    });
  }
  items.push(...systemTabs);

  // The `?tab=` query param seeds the initial tab only. Switching is driven by
  // local state on purpose: routing tab clicks through setSearchParams trips
  // the hash router's navigation blocker (POP navigations "fail silently") and
  // bounces the user to login, so the URL is deliberately left untouched.
  const [activeKey, setActiveKey] = useState<string>(() => searchParams.get('tab') || items?.[0]?.key);

  const onChange = (key: string) => {
    setActiveKey(key);
  };

  const activeItem = items.find(item => item.key === activeKey);

  return (
    <ACard styles={{ body: { padding: 0 } }}>
      <Tabs
        activeKey={activeKey}
        className="settings-tabs"
        items={items}
        onChange={onChange}
      />
      <div className="settings-tabs-content">{activeItem?.component ? <activeItem.component /> : null}</div>
    </ACard>
  );
}
