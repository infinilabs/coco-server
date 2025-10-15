import { Tabs } from 'antd';

import './index.scss';
import Role from './modules/Role';

export function Component() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [
    {
      component: Role,
      key: 'role',
      label: t(`page.role.title`),
    },
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
