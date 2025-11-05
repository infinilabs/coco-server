import { Tabs } from 'antd';

import Pipeline from './components/Pipeline';
import Runs from './components/Runs';

import './index.scss';

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [
    {
      component: Runs,
      key: 'runs',
      label: 'Runs'
    },
    {
      component: Pipeline,
      key: 'pipeline',
      label: 'Pipeline'
    }
  ];

  const activeKey = useMemo(() => {
    return searchParams.get('tab') || items[0].key;
  }, [searchParams]);

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
