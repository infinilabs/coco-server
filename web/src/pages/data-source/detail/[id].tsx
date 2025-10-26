import { Tabs } from 'antd';

import './index.scss';
import FileManagement from './modules/FileManagement';
import { useRoute } from '@sa/simple-router';
import MappingManagement from './modules/MappingManagement';

export function Component() {
  const route = useRoute();
  const datasourceID = route.params.id
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewFile: hasAuth('coco:datasource/view'),
  }

  const onChange = (key: string) => {
    setSearchParams({ tab: key });
  };

  const items = [];

  if (permissions.viewFile) {
    items.push({
      component: FileManagement,
      key: 'file',
      label: t(`page.datasource.file.title`),
    })
    items.push({
      component: MappingManagement,
      key: 'mapping',
      label: t(`page.datasource.mapping.title`),
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
        { activeItem?.component ? <activeItem.component id={datasourceID} isMapping={true} /> : null}
      </div>
    </ACard>
  );
}
