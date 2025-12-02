import { useRoute } from '@sa/simple-router';
import type { FC, ReactNode } from 'react';
import { useMemo } from 'react';

import { getLocale } from '@/store/slice/app';
import { getSortRoutes } from '@/store/slice/route';
import { selectActiveFirstLevelMenuKey, setActiveFirstLevelMenuKey } from '@/store/slice/tab';
import { getActiveFirstLevelMenuKey } from '@/store/slice/tab/shared';

import { MixMenuContext } from '../context';

import { getGlobalMenusByAuthRoutes } from './MenuUtil';
import { renderToString } from 'react-dom/server';

interface Props {
  readonly children: ReactNode;
}

const MenuProvider: FC<Props> = ({ children }) => {
  const sortRoutes = useAppSelector(getSortRoutes);

  const menus = getGlobalMenusByAuthRoutes(sortRoutes);

  const locale = useAppSelector(getLocale);

  const { t } = useTranslation();

  const update = useUpdate();

  const activeFirstLevelMenuKey = useAppSelector(selectActiveFirstLevelMenuKey);

  const dispatch = useAppDispatch();

  const route = useRoute();

  const selectKey = useMemo(() => {
    const { activeMenu, hideInMenu } = route.meta;

    const name = route.name as string;

    const routeName = (hideInMenu ? activeMenu : name) || name;

    return [routeName];
  }, [route]);

  /** - 可以手动指定菜单或者是默认当前路由的一级菜单 */
  function changeActiveFirstLevelMenuKey(key?: string) {
    let routeKey = key;

    if (!routeKey) {
      routeKey = getActiveFirstLevelMenuKey(route);
    }

    dispatch(setActiveFirstLevelMenuKey(routeKey));
  }

  const firstLevelMenu = useMemo(
    () =>
      menus.map(menu => {
        const { children: _, ...rest } = menu;
        return rest;
      }) as App.Global.Menu[],
    [menus]
  );

  const childLevelMenus = useMemo(
    () => menus.find(menu => menu.key === activeFirstLevelMenuKey)?.children as App.Global.Menu[],
    [activeFirstLevelMenuKey, menus]
  );

  useUpdateEffect(() => {
    update();
  }, [locale]);

  useEffect(() => {
    if (window.$wujie?.props?.onRoutesUpdate) {
      const formatRoutes = (routes: any[] = [], newRoutes: any[] = [], pathPrefix?: string) => {
        routes.forEach((item) => {
          let activeRoute
          if (item.meta?.activeMenu) {
            activeRoute = routes.find((r) => r.name === item.meta?.activeMenu)
          }
          const route = {
            ...item,
            path: `${pathPrefix || ''}${item.path}`,
            localeName: item.meta?.i18nKey ? t(item.meta?.i18nKey) : undefined,
            icon: renderToString(<SvgIcon icon={item.meta?.icon} localIcon={item.meta?.localIcon} />),
            hideInMenu: item.meta?.hideInMenu,
            activeRoute: pathPrefix ? undefined : (
              activeRoute ? {
                ...activeRoute,
                path: `${activeRoute.path}`,
                localeName: activeRoute.meta?.i18nKey ? t(activeRoute.meta?.i18nKey) : undefined,
                icon: renderToString(<SvgIcon icon={activeRoute.meta?.icon} localIcon={activeRoute.meta?.localIcon} />),
                hideInMenu: activeRoute.meta?.hideInMenu,
              } : undefined
            )
          }
          newRoutes.push(route)
          if (item.children) {
            formatRoutes(item.children, newRoutes, `${route.path}/`)
          }
        })
      }
      const newRoutes: any[] = []
      formatRoutes(sortRoutes, newRoutes)
      window.$wujie?.props?.onRoutesUpdate(newRoutes)
    }
  }, [sortRoutes])

  const mixMenuContext = {
    activeFirstLevelMenuKey,
    allMenus: menus,
    childLevelMenus: childLevelMenus || [],
    firstLevelMenu,
    isActiveFirstLevelMenuHasChildren: activeFirstLevelMenuKey ? Boolean(childLevelMenus) : false,
    route,
    selectKey,
    setActiveFirstLevelMenuKey: changeActiveFirstLevelMenuKey
  };

  return <MixMenuContext.Provider value={mixMenuContext}>{children}</MixMenuContext.Provider>;
};

export default MenuProvider;
