import { CopyOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';
import Dedup from '../settings/modules/Dedup';

/**
 * 内容查重 — promoted from 系统管理-设置 to a first-class AI-search entry
 * (grouped under AI 搜索 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.settings.dedup.desc')}
      icon={<CopyOutlined />}
      title={t('page.settings.dedup.title')}
    >
      <Dedup />
    </ModuleShell>
  );
}
