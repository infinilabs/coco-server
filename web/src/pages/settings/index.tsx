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
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewConnector: hasAuth('coco#connector/search'),
    viewSystemSettings: hasAuth('coco#system/read'),
  }

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  type SettingsTab = { component: any; key: string; label: string };

  // W18: the flat 14-tab list grouped into five domains (通用/检索/处理/安全/知识/智能).
  // Hidden tabs drop out; groups collapse when empty; single-tab groups flatten
  // (a group header for one item is noise).
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
        { component: DefaultModel, key: 'default_model', label: t(`page.settings.default_model.title`) },
      ]
    : [];

  const byKey = (key: string): SettingsTab | undefined => systemTabs.find(tab => tab.key === key);
  const pick = (...keys: string[]): SettingsTab[] =>
    keys.map(k => byKey(k)).filter((tab): tab is SettingsTab => Boolean(tab));

  // connector tab keeps its own permission gate and first position
  const items: any[] = [];
  if (permissions.viewConnector) {
    items.push({
      component: ConnectorSettings,
      key: 'connector',
      label: t(`page.settings.connector.title`),
    });
  }

  const groups: Array<{ key: string; label: string; tabs: SettingsTab[] }> = [
    { key: 'general', label: t(`page.settings.groups.general`), tabs: pick('appearance', 'app_settings', 'environment') },
    { key: 'search', label: t(`page.settings.groups.search`), tabs: pick('search_settings', 'engine_ai') },
    { key: 'processing', label: t(`page.settings.groups.processing`), tabs: pick('document_processing') },
    { key: 'security', label: t(`page.settings.groups.security`), tabs: pick('data_security', 'dedup') },
    { key: 'knowledge', label: t(`page.settings.groups.knowledge`), tabs: pick('wiki_settings', 'memory_settings') },
    { key: 'ai', label: t(`page.settings.groups.ai`), tabs: pick('default_model') },
  ];

  for (const g of groups) {
    if (g.tabs.length === 0) continue;
    // single-tab groups flatten (a group header for one item is noise)
    if (g.tabs.length === 1) {
      items.push(g.tabs[0]);
      continue;
    }
    items.push({
      type: 'group',
      label: g.label,
      children: g.tabs,
    } as any);
  }

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items?.[0]?.key
  }, [])

  const activeItem = useMemo(() => {
    // group items nest their tabs under children — flatten one level
    for (const item of items) {
      if ('children' in item && Array.isArray(item.children)) {
        const match = (item.children as Array<{ key: string }>).find(
          (child: { key: string }) => child.key === activeKey
        );
        if (match) return match;
        continue;
      }
      if (item.key === activeKey) return item;
    }
    return undefined;
  }, [activeKey]);

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
