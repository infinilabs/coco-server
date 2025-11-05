import { Button, Card, Flex, Tabs } from 'antd';

import Steps from './components/Steps';
import Summary from './components/Summary';

import './index.scss';

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [
    {
      component: Summary,
      key: 'summary',
      label: 'Summary'
    },
    {
      component: Steps,
      key: 'steps',
      label: 'Steps'
    }
  ];

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items[0].key;
  }, [searchParams]);

  const activeItem = useMemo(() => {
    return items.find(item => item.key === activeKey);
  }, [activeKey]);

  return (
    <Flex
      vertical
      gap={8}
    >
      <Card>
        <Flex
          align="center"
          justify="space-between"
        >
          <Flex
            vertical
            gap={12}
          >
            <b>FAQ 提取 #1234</b>
            <span className="text-color-3 text-xs">Triggered via schedule 18 hours ago</span>
          </Flex>

          <Flex
            vertical
            gap={12}
          >
            <b>Success</b>
            <span className="text-color-3 text-xs">Status</span>
          </Flex>

          <Flex
            vertical
            gap={12}
          >
            <b>22m 40s</b>
            <span className="text-color-3 text-xs">Total duration</span>
          </Flex>

          <Button type="primary">重新运行</Button>
        </Flex>
      </Card>

      <Card styles={{ body: { padding: 0 } }}>
        <Tabs
          activeKey={activeKey}
          className="settings-tabs"
          items={items}
          onChange={onChange}
        />
        <div className="settings-tabs-content">{activeItem?.component ? <activeItem.component /> : null}</div>
      </Card>
    </Flex>
  );
}
