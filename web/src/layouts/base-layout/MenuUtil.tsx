import type { ElegantConstRoute } from '@elegant-router/types';
import type { MenuProps } from 'antd';

import { $t } from '@/locales';

/**
 * Route keys rendered in the primary "workspace" section of the sider — the three first-class applications
 * (AI search / AI chat / AI knowledge base) plus the quick-start entry. Everything else is settings and data
 * support for those apps and falls into the classified administration groups below.
 */
const WORKSPACE_MENU_KEYS: readonly string[] = ['home', 'search', 'chat', 'wiki'];

/**
 * Administration menu groups, in display order, organized by the application
 * each feature serves (AI search / AI chat / knowledge base), then the shared
 * data-ingestion base and platform administration. Route keys not listed here
 * land in a trailing "others" group, so a new page never disappears.
 */
const ADMIN_MENU_GROUPS: { key: string; labelKey: string; routeKeys: readonly string[] }[] = [
  { key: 'menu-group-app-search', labelKey: 'menu.group.appSearch', routeKeys: ['search-studio', 'search-ops', 'search-settings', 'engine-ai'] },
  { key: 'menu-group-app-chat', labelKey: 'menu.group.appChat', routeKeys: ['ai-assistant', 'skill', 'memory-settings'] },
  { key: 'menu-group-app-kb', labelKey: 'menu.group.appKb', routeKeys: ['ontology'] },
  { key: 'menu-group-data', labelKey: 'menu.group.data', routeKeys: ['data-source', 'pipeline', 'gateway-instances', 'connector-settings', 'data-security', 'content-dedup', 'webhook'] },
  {
    key: 'menu-group-system',
    labelKey: 'menu.group.system',
    routeKeys: ['model-provider', 'mcp-server', 'api-token', 'user', 'role', 'security', 'integration', 'settings', 'guide']
  }
];

const OTHERS_GROUP_KEY = 'menu-group-others';

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
 * Split top-level menus into the workspace section and classified administration
 * groups, so the sider reads "apps first, then console features by domain"
 * instead of one flat admin list. The group label alone carries the section
 * boundary (no divider); pass `sectioned: false` for icon-only contexts such as
 * the collapsed sider, where a truncated group title would be noise.
 *
 * @param menus Top-level global menus
 * @param sectioned Whether to render the administration group labels
 */
export function getSectionedMenuItems(menus: App.Global.Menu[], sectioned = true): MenuProps['items'] {
  const workspace = WORKSPACE_MENU_KEYS.map(key => menus.find(menu => menu.key === key)).filter(Boolean) as App.Global.Menu[];
  const administration = menus.filter(menu => !WORKSPACE_MENU_KEYS.includes(menu.key));

  if (!sectioned || !administration.length) {
    return [...workspace, ...administration] as unknown as MenuProps['items'];
  }

  const groupedKeys = new Set<string>();
  const groups = ADMIN_MENU_GROUPS.map(({ key, labelKey, routeKeys }) => {
    const children = routeKeys.map(routeKey => administration.find(menu => menu.key === routeKey)).filter(Boolean) as App.Global.Menu[];
    children.forEach(child => groupedKeys.add(child.key));
    return {
      type: 'group',
      key,
      label: <BeyondHiding title={$t(labelKey)} />,
      children
    };
  }).filter(group => group.children.length > 0);

  const others = administration.filter(menu => !groupedKeys.has(menu.key));
  if (others.length > 0) {
    groups.push({
      type: 'group',
      key: OTHERS_GROUP_KEY,
      label: <BeyondHiding title={$t('menu.group.others')} />,
      children: others
    });
  }

  return [...workspace, ...groups] as unknown as MenuProps['items'];
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
