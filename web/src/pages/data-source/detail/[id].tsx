import { Tabs } from 'antd';

import './index.scss';
import FileManagement from './modules/FileManagement';
import FaqManagement from './modules/FaqManagement';
import StatsOverview from './modules/StatsOverview';
import StatsTab from './modules/StatsTab';
import { useRoute } from '@sa/simple-router';
import MappingManagement from './modules/MappingManagement';

export function Component() {
  const route = useRoute();
  // hash 路由跳转时 route.params 会先于本组件卸载更新为下个路由的参数，
  // 子模块的 effect 会拿错误的 id 重发请求（/datasource/<docId> 404、
  // /document/_search source.id:any(<docId>) 500）——只在自家路由激活时更新 id
  const [datasourceID, setDatasourceID] = useState(route.params.id as string);
  useEffect(() => {
    if (route.name === 'data-source_detail' && route.params.id) {
      setDatasourceID(route.params.id);
    }
  }, [route.name, route.params.id]);
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    viewFile: hasAuth(['coco#datasource/search', 'coco#datasource/read']),
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
    // FAQ entries are documents of this datasource (W5) — same view gate
    items.push({
      component: FaqManagement,
      key: 'faq',
      label: t(`page.faq.tabTitle`),
    })
    // 深度统计 tab：全部走 /document/_search 聚合透传，document:search 权限即可
    items.push({
      component: StatsTab,
      key: 'stats',
      label: t(`page.datasource.stats.tabTitle`),
    })
    // items.push({
    //   component: MappingManagement,
    //   key: 'mapping',
    //   label: t(`page.datasource.mapping.title`),
    // })
  }

  // activeKey 必须跟随 searchParams 变化：useMemo 空依赖会冻结在首渲染值，
  // 点击 tab 只改了 URL、内容区不切换（多 tab 下的静默 bug）
  const activeKey = searchParams.get('tab') || items?.[0]?.key
  const activeItem = items.find((item) => item.key === activeKey)

  return (
    <div className="flex flex-col gap-16px pb-16px">
      {/* 卡片一：数据源基本信息 + 解析统计（ops 接口 403 时内部降级为仅基本信息+capabilities） */}
      {permissions.viewFile ? <StatsOverview id={datasourceID} /> : null}
      {/* 卡片二：tab 内容区 */}
      <ACard styles={{ body: { padding: 0 } }}>
        <Tabs
          className="settings-tabs"
          activeKey={activeKey}
          items={items}
          onChange={onChange}
        />
        <div className="settings-tabs-content">
          { activeItem?.component ? <activeItem.component id={datasourceID}/> : null}
        </div>
      </ACard>
    </div>
  );
}
