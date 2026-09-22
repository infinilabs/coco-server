import type { ElegantConstRoute } from '@elegant-router/types';
import type { MenuProps } from 'antd';

import { $t } from '@/locales';

/**
 * Route keys rendered in the primary "workspace" section of the sider —
 * the user-facing applications. Everything else falls into the trailing
 * administration group.
 */
const WORKSPACE_MENU_KEYS: readonly string[] = ['search', 'wiki', 'ai-assistant'];

/**
 * Get global menus by auth routes
 *
 * @param routes Auth routes
 */
export function getGlobalMenusByAuthRoutes(routes: ElegantConstRoute[]) {
  const menus: App.Global.Menu[] = [];

  routes.forEach(route => {
    if (!route.meta?.hideInMenu) {
      const menu = getGlobalMenuByBaseRoute(route);

      if (route.children?.some(child => !child.meta?.hideInMenu)) {
        menu.children = getGlobalMenusByAuthRoutes(route.children) || [];
      }

      menus.push(menu);
    }
  });

  return menus;
}

/**
 * Split top-level menus into the workspace section and the administration
 * group, so the sider reads "apps first, console settings after" instead of
 * one flat admin list.
 *
 * @param menus Top-level global menus
 */
export function getSectionedMenuItems(menus: App.Global.Menu[]): MenuProps['items'] {
  const workspace = menus.filter(menu => WORKSPACE_MENU_KEYS.includes(menu.key));
  const administration = menus.filter(menu => !WORKSPACE_MENU_KEYS.includes(menu.key));

  if (!administration.length) {
    return workspace as unknown as MenuProps['items'];
  }

  return [
    ...workspace,
    { type: 'divider', key: 'menu-section-divider' },
    {
      type: 'group',
      key: 'menu-section-administration',
      label: <BeyondHiding title={$t('menu.administration')} />,
      children: administration
    }
  ] as unknown as MenuProps['items'];
}

/**
 * Get global menu by route
 *
 * @param route
 */
export function getGlobalMenuByBaseRoute(route: ElegantConstRoute): App.Global.Menu {
  const { name } = route;

  const { i18nKey, icon = import.meta.env.VITE_MENU_ICON, localIcon, title } = route.meta ?? {};

  const label = i18nKey ? $t(i18nKey) : title;

  const menu: App.Global.Menu = {
    icon: (
      <SvgIcon
        icon={icon}
        localIcon={localIcon}
        style={{ fontSize: '14px' }}
      />
    ),
    key: name,
    label: <BeyondHiding title={label} />,
    title: label
  };

  return menu;
}
