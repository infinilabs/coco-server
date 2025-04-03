import { Tabs } from "antd";
import "./index.scss";
import Server from "./modules/Server";
import LLM from "./modules/LLM";
import ConnectorSettings from "./modules/Connector";

export function Component() {

  const [searchParams] = useSearchParams();
  const routerPush = useRouterPush();

  const onChange = (key: string) => {
    routerPush.routerPushByKey('settings', { query: { tab: key }})
  };

  const items = [
    // {
    //   key: 'server',
    //   label: 'Server',
    //   children: <Server />
    // },
    {
      key: 'llm',
      label: 'LLMs',
      children: <LLM />,
    },
    {
      key: 'connector',
      label: 'Connector',
      children: <ConnectorSettings />,
    },
  ];

  return (
    <ACard styles={{ body: { padding: 0 }}}>
      <Tabs className="settings-tabs" defaultActiveKey={searchParams.get('tab') || items[0].key} items={items} onChange={onChange} />
    </ACard>
  )
}
