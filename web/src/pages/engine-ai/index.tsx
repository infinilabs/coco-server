import { ThunderboltOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';
import EngineAI from '../settings/modules/EngineAI';

/**
 * 引擎 AI (语义检索) — promoted from 系统管理-设置 to a first-class entry,
 * sider-grouped under 应用-检索 (see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.settings.engine_ai.labels.engine_ai_desc')}
      icon={<ThunderboltOutlined />}
      title={t('page.settings.engine_ai.title')}
    >
      <EngineAI />
    </ModuleShell>
  );
}
