import { Tabs } from 'antd';

import './index.scss';
import ConnectorSettings from './modules/Connector';
import ChartStartPage from './modules/ChartStartPage';

export function Component() {
  const [searchParams] = useSearchParams();
  const routerPush = useRouterPush();
  const { t } = useTranslation();

  const onChange = (key: string) => {
    routerPush.routerPushByKey('settings', { query: { tab: key } });
  };

  const items = [
    {
      children: <ConnectorSettings />,
      key: 'connector',
      label: 'Connector'
    },
    {
      children: <ChartStartPage />,
      key: 'chart_start_page',
      label: t(`page.chart_start_page.title`)
    }
  ];

  return (
    <ACard styles={{ body: { padding: 0 } }}>
      <Tabs
        className="settings-tabs"
        defaultActiveKey={searchParams.get('tab') || items[0].key}
        items={items}
        onChange={onChange}
      />
    </ACard>
  );
}
