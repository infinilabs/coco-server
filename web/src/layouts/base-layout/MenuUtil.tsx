import type { ElegantConstRoute } from '@elegant-router/types';
import type { MenuProps } from 'antd';

import { $t } from '@/locales';

/**
 * Route keys rendered in the primary "workspace" section of the sider — the three first-class applications
 * (AI search / AI chat / AI knowledge base). Everything else is settings and data support for those apps and
 * falls into the trailing administration group.
 */
const WORKSPACE_MENU_KEYS: readonly string[] = ['search', 'chat', 'wiki'];

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
 * Split top-level menus into the workspace section and the administration group, so the sider reads "apps first,
 * console settings after" instead of one flat admin list. The group label alone carries the section boundary (no
 * divider); pass `sectioned: false` for icon-only contexts such as the collapsed sider, where a truncated group title
 * would be noise.
 *
 * @param menus Top-level global menus
 * @param sectioned Whether to render the administration group label
 */
export function getSectionedMenuItems(menus: App.Global.Menu[], sectioned = true): MenuProps['items'] {
  const workspace = WORKSPACE_MENU_KEYS.map(key => menus.find(menu => menu.key === key)).filter(Boolean) as App.Global.Menu[];
  const administration = menus.filter(menu => !WORKSPACE_MENU_KEYS.includes(menu.key));

  if (!sectioned || !administration.length) {
    return [...workspace, ...administration] as unknown as MenuProps['items'];
  }

  return [
    ...workspace,
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
