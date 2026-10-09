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

  const items: { component: any; key: string; label: string }[] = [];

  if (permissions.viewConnector) {
    items.push({
      component: ConnectorSettings,
      key: 'connector',
      label: t(`page.settings.connector.title`),
    })
  }

  if (permissions.viewSystemSettings) {
    items.push({
      component: AppearanceSettings,
      key: 'appearance',
      label: t(`page.settings.appearance.title`),
    })
    items.push({
      component: AppSettings,
      key: 'app_settings',
      label: t(`page.settings.app_settings.title`),
    })
    items.push({
      component: SearchSettings,
      key: 'search_settings',
      label: t(`page.settings.search_settings.title`),
    })
    items.push({
      component: WikiSettings,
      key: 'wiki_settings',
      label: t(`page.settings.wiki_settings.title`),
    })
    // long-term memory (W6): the owner confirms what the distiller learned
    items.push({
      component: MemorySettings,
      key: 'memory_settings',
      label: t(`page.settings.memory_settings.title`),
    })
    items.push({
      component: DefaultModel,
      key: 'default_model',
      label: t(`page.settings.default_model.title`),
    })
    items.push({
      component: DocProcessing,
      key: 'document_processing',
      label: t(`page.settings.document_processing.title`),
    })
    items.push({
      component: DataSecurity,
      key: 'data_security',
      label: t(`page.settings.data_security.title`),
    })
    items.push({
      component: EngineAI,
      key: 'engine_ai',
      label: t(`page.settings.engine_ai.title`),
    })
    items.push({
      component: Dedup,
      key: 'dedup',
      label: t(`page.settings.dedup.title`),
    })
    items.push({
      component: Environment,
      key: 'environment',
      label: t(`page.settings.environment.title`),
    })
  }

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items?.[0]?.key
  }, [])

  const activeItem = useMemo(() => {
    return items.find(item => item.key === activeKey);
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
