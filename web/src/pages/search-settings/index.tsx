import { SearchOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';
import SearchSettings from '../settings/modules/SearchSettings';

/**
 * 搜索设置 — promoted from 系统管理-设置 to a first-class AI-search entry
 * (grouped under AI 搜索 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.settings.search_settings.subtitle')}
      icon={<SearchOutlined />}
      title={t('page.settings.search_settings.title')}
    >
      <SearchSettings />
    </ModuleShell>
  );
}
