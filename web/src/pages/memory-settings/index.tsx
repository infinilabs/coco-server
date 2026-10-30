import { HistoryOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';

import MemorySettings from '../settings/modules/MemorySettings';

/**
 * 长期记忆 — promoted from 系统管理-设置 to a first-class AI-chat entry
 * (grouped under AI 对话 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.memory_settings.hint')}
      icon={<HistoryOutlined />}
      title={t('page.memory_settings.title')}
    >
      <MemorySettings />
    </ModuleShell>
  );
}
