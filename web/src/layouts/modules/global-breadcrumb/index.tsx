import { Breadcrumb } from 'antd';
import type { BreadcrumbProps } from 'antd';
import type { MenuItemType } from 'antd/es/menu/interface';
import type { MenuInfo } from 'rc-menu/lib/interface';
import { cloneElement } from 'react';

import { useRouterPush } from '@/hooks/common/routerPush';
import { $t } from '@/locales';

import { getBreadcrumbsByRoute } from './shared';

function BreadcrumbContent({ icon, label }: { readonly icon: JSX.Element; readonly label: JSX.Element }) {
  return (
    <div className="i-flex-y-center align-middle">
      {cloneElement(icon, { className: 'mr-4px text-icon', ...icon.props })}
      {label}
    </div>
  );
}

const GlobalBreadcrumb: FC<Omit<BreadcrumbProps, 'items'>> = props => {
  const { allMenus: menus, route } = useMixMenuContext();

  const routerPush = useRouterPush();

  const breadcrumb = getBreadcrumbsByRoute(route, menus);

  function handleClickMenu(menuInfo: MenuInfo) {
    routerPush.routerPushByKeyWithMetaQuery(menuInfo.key);
  }

  const items: BreadcrumbProps['items'] = breadcrumb.map((item, index) => {
    const commonTitle = (
      <BreadcrumbContent
        icon={item.icon as JSX.Element}
        key={item.key}
        label={item.label as JSX.Element}
      />
    );

    return {
      title: (
        <span 
          className="cursor-pointer hover:text-blue-500" 
          onClick={() => routerPush.routerPushByKeyWithMetaQuery(item.key as string)}
        >
          {commonTitle}
        </span>
      ),
      ...('children' in item &&
        item.children && {
          menu: {
            items: item.children.filter(Boolean) as MenuItemType[],
            onClick: handleClickMenu,
            selectedKeys: [breadcrumb[index + 1]?.key] as string[]
          }
        })
    };
  });

  // Check if current route is home or if breadcrumb already contains home
  const isHomePage = route.matched[0]?.name === import.meta.env.VITE_ROUTE_HOME;
  const hasHomeBreadcrumb = breadcrumb.some(item => item.key === import.meta.env.VITE_ROUTE_HOME);

  let finalItems = items;
  
  if (!isHomePage && !hasHomeBreadcrumb) {
    // Add home breadcrumb only if we're not on home page and it's not already in breadcrumbs
    finalItems = [
      {
        path: '/',
        title: (
          <span 
            className="cursor-pointer hover:text-blue-500" 
            onClick={() => routerPush.routerPushByKeyWithMetaQuery(import.meta.env.VITE_ROUTE_HOME)}
          >
            <BreadcrumbContent
              key="home"
              label={<span>{$t('route.home')}</span>}
              icon={
                <SvgIcon
                  className="mr-4px text-14px"
                  icon="mdi:home"
                />
              }
            />
          </span>
        )
      }
    ].concat(items);
  }

  return (
    <Breadcrumb
      {...props}
      items={finalItems}
    />
  );
};

export default GlobalBreadcrumb;
