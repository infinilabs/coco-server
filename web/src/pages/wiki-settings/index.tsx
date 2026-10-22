import { ReadOutlined } from '@ant-design/icons';
import ModuleShell from '../settings/ModuleShell';
import WikiSettings from '../settings/modules/WikiSettings';

/**
 * AI 知识库设置 — promoted from 系统管理-设置 to a first-class knowledge-base
 * entry (grouped under 知识库 in the sider; see MenuUtil ADMIN_MENU_GROUPS).
 */
export function Component() {
  const { t } = useTranslation();

  return (
    <ModuleShell
      description={t('page.settings.wiki_settings.subtitle')}
      icon={<ReadOutlined />}
      title={t('page.settings.wiki_settings.title')}
    >
      <WikiSettings />
    </ModuleShell>
  );
}
