import { BookOutlined, MessageOutlined, SearchOutlined, SettingOutlined } from '@ant-design/icons';
import { Button, Segmented, Tooltip } from 'antd';
import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import cocoIcon from '@/assets/svg-icon/coco.svg';
import UserAvatar from '../modules/global-header/components/UserAvatar';
import GlobalContent from '../modules/global-content';

/**
 * The user-facing application shell: one persistent header that carries the three first-class
 * apps (AI search / AI chat / AI knowledge base) as peers, so switching between them is a single
 * click and never drops the user back into the console. Console settings live behind the
 * explicit "Console" entry — user surface and administration stay cleanly separated.
 */
export function Component() {
  const { t } = useTranslation();
  const location = useLocation();

  const activeApp = useMemo(() => {
    const { pathname } = location;
    if (pathname.startsWith('/wiki')) return 'wiki';
    if (pathname.startsWith('/chat')) return 'chat';
    return 'search';
  }, [location.pathname]);

  const goApp = (key: string) => {
    const target = { search: '/search', chat: '/chat?mode=chat', wiki: '/wiki/list' }[key];
    if (target && `#${location.pathname}${location.search}` !== `#${target}`) {
      window.location.hash = `#${target}`;
    }
  };

  return (
    <div className="h-screen flex flex-col">
      <header className="h-48px flex-y-center shrink-0 justify-between border-b border-gray-200 bg-white px-12px dark:border-gray-700 dark:bg-[#141414]">
        <div className="flex items-center gap-4">
          <div
            className="flex cursor-pointer items-center gap-8px"
            onClick={() => goApp('search')}
          >
            <img alt="Coco AI" className="h-30px w-30px" src={cocoIcon} />
            <span className="text-16px font-bold text-primary">{t('system.title') || 'Coco AI'}</span>
          </div>
          <Segmented
            value={activeApp}
            onChange={value => goApp(value as string)}
            options={[
              { value: 'search', label: t('route.search'), icon: <SearchOutlined /> },
              { value: 'chat', label: t('route.chat'), icon: <MessageOutlined /> },
              { value: 'wiki', label: t('route.wiki'), icon: <BookOutlined /> }
            ]}
          />
        </div>
        <div className="flex-y-center justify-end">
          <Tooltip title={t('common.console')}>
            <Button
              size="small"
              icon={<SettingOutlined />}
              onClick={() => {
                window.location.hash = '#/home';
              }}
            >
              {t('common.console')}
            </Button>
          </Tooltip>
          <LangSwitch className="px-12px" />
          <ThemeSchemaSwitch className="px-12px" />
          <UserAvatar className="px-8px" showName={false} />
        </div>
      </header>
      <div className="relative min-h-0 flex-1">
        <GlobalContent closePadding={true} />
      </div>
    </div>
  );
}
