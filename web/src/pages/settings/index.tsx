import { Tabs } from 'antd';

import './index.scss';
import ConnectorSettings from './modules/Connector';
import Server from './modules/Server';

export function Component() {
  const [searchParams] = useSearchParams();
  const routerPush = useRouterPush();

  const onChange = (key: string) => {
    routerPush.routerPushByKey('settings', { query: { tab: key } });
  };

  const items = [
    // {
    //   key: 'server',
    //   label: 'Server',
    //   children: <Server />
    // },
    {
      children: <ConnectorSettings />,
      key: 'connector',
      label: 'Connector'
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
