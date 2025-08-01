import { Tabs } from 'antd';

import './index.scss';
import ConnectorSettings from './modules/Connector';
import AppSettings from './modules/AppSettings';

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [
    {
      component: ConnectorSettings,
      key: 'connector',
      label: t(`page.settings.connector.title`),
    },
    {
      component: AppSettings,
      key: 'chart_start_page',
      label: t(`page.settings.app_settings.title`),
    }
  ];

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items[0].key
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
