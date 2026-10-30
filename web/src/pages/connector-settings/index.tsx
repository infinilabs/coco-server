import ConnectorSettings from '../settings/modules/Connector';

/**
 * 连接器 — promoted from 系统管理-设置 to a first-class data-ingestion entry
 * (grouped under 数据接入 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  return <ConnectorSettings />;
}
