import { SafetyOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';
import DataSecurity from '../settings/modules/DataSecurity';

/**
 * 数据安全 — promoted from 系统管理-设置 to a first-class data-ingestion entry
 * (grouped under 知识加工 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.settings.data_security.subtitle')}
      icon={<SafetyOutlined />}
      title={t('page.settings.data_security.title')}
    >
      <DataSecurity />
    </ModuleShell>
  );
}
