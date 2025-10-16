import { Tabs } from 'antd';

import './index.scss';
import ConnectorSettings from './modules/Connector';
import AppSettings from './modules/AppSettings';
import SearchSettings from './modules/SearchSettings';

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewConnector: hasAuth('coco:connector/view'),
    viewAppSettings: hasAuth('coco:app_settings/view'),
    viewSearchSettings: hasAuth('coco:search_settings/view'),
  }

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [];

  if (permissions.viewConnector) {
    items.push({
      component: ConnectorSettings,
      key: 'connector',
      label: t(`page.settings.connector.title`),
    })
  }

  if (permissions.viewAppSettings) {
    items.push({
      component: AppSettings,
      key: 'app_settings',
      label: t(`page.settings.app_settings.title`),
    })
  }

  if (permissions.viewSearchSettings) {
    items.push({
      component: SearchSettings,
      key: 'search_settings',
      label: t(`page.settings.search_settings.title`),
    })
  }

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items?.[0]?.key
  }, [])

  const activeItem = useMemo(() => {
    return items.find((item) => item.key === activeKey);
  }, [activeKey])

  return (
    <ACard styles={{ body: { padding: 0 } }}>
      <Tabs
        className="settings-tabs"
        activeKey={activeKey}
        items={items}
        onChange={onChange}
      />
      <div className="settings-tabs-content">
        { activeItem?.component ? <activeItem.component /> : null}
      </div>
    </ACard>
  );
}
