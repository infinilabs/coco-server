import { BookOutlined, MessageOutlined, SearchOutlined } from '@ant-design/icons';
import { Segmented } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { I18nextProvider, useTranslation } from 'react-i18next';
import { useResponsive } from 'ahooks';
import { debounce } from 'lodash';

import cocoIcon from '@/assets/svg-icon/coco.svg';
import { getApplicationSetting, getAppearance } from '@/store/slice/server';
import { getLocale } from '@/store/slice/app';
import { fetchIntegration } from '@/service/api/integration';
import { fetchSuggestions, uploadAttachment } from '@/service/api/ai-search';
import { setPendingSearch } from '@/utils/search-handoff';
import { SearchBox, searchWidgetI18n } from 'ui-search/source';
import UserAvatar from '../modules/global-header/components/UserAvatar';
import GlobalContent from '../modules/global-content';

/**
 * The user-facing application shell: one persistent header that carries the three first-class apps (AI search / AI chat
 * / AI knowledge base) as peers, so switching between them is a single click and never drops the user back into the
 * console. Console settings live behind the explicit "Console" entry — user surface and administration stay cleanly
 * separated.
 */
export function Component() {
  const { t } = useTranslation();
  const location = useLocation();
  const responsive = useResponsive();

  // one 48px row cannot hold a labeled segmented + console/lang/theme/login
  // below md: icon-only segmented, and the console entry moves into the avatar
  // dropdown — otherwise flexbox squeezes the logo to 0px and pushes the login
  // entry off-screen
  const compact = !responsive.md;

  // the shell-wide search entry embeds the ui-search SearchBox (suggestions,
  // filter chips, attachments — the same box the search page uses). It needs
  // the integration's settings and the upload/suggestion endpoints, same as
  // the search page; fetch once per shell mount.
  const applicationSetting = useAppSelector(getApplicationSetting);
  const appearance = useAppSelector(getAppearance);
  const locale = useAppSelector(getLocale);
  const { search_settings } = applicationSetting || {};
  const showSearchBox = Boolean(search_settings?.enabled && search_settings?.integration);
  const [integration, setIntegration] = useState<any>(null);
  const [headerAttachments, setHeaderAttachments] = useState<any[]>([]);

  // the embedded SearchBox resolves labels through the widget's i18n instance
  useEffect(() => {
    searchWidgetI18n.changeLanguage(locale);
  }, [locale]);

  useEffect(() => {
    if (!search_settings?.integration) return undefined;
    let cancelled = false;
    fetchIntegration(search_settings.integration)
      .then(res => {
        if (!cancelled && res?.data) setIntegration(res.data._source || {});
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [search_settings?.integration]);

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

  // the header box hands its submit to the search page: search mode crosses
  // the route as URL params; chat mode (attachments / AI action) parks its
  // in-memory state in the one-shot handoff before navigating
  const handleHeaderSearch = (params: any) => {
    if (params?.mode === 'chat') {
      setPendingSearch({
        query: params.query || '',
        attachments: params.attachments || headerAttachments || [],
        assistant_id: params.assistant_id
      });
      window.location.hash = '#/chat?mode=chat';
      return;
    }
    const usp = new URLSearchParams();
    if (params?.query) usp.set('query', params.query);
    if (params?.search_type) usp.set('search_type', params.search_type);
    if (typeof params?.fuzziness === 'number') usp.set('fuzziness', String(params.fuzziness));
    if (params?.sort) usp.set('sort', params.sort);
    // multimodal search: attachment IDs travel in the URL; the search page
    // hydrates its chips from them and the server joins their extracted text
    if (params?.attachments) usp.set('attachments', params.attachments);
    Object.entries(params?.filter || {}).forEach(([field, values]) => {
      (Array.isArray(values) ? values : [values]).forEach(value => {
        if (value !== undefined && value !== null && String(value) !== '') {
          usp.append('filter', `${field}:${String(value)}`);
        }
      });
    });
    window.location.hash = `#/search${usp.toString() ? `?${usp.toString()}` : ''}`;
  };

  const debouncedSuggestion = useMemo(
    () =>
      debounce((tag: string | undefined, reqParams: Record<string, any>, callback: (data: any) => void) => {
        if (!search_settings?.integration) return;
        fetchSuggestions(tag, reqParams, {
          headers: { 'APP-INTEGRATION-ID': search_settings.integration },
          ignoreError: true
        }).then(res => callback?.(res.data));
      }, 500),
    [search_settings?.integration]
  );

  const handleHeaderUpload = (files: File[], callback?: (data: any) => void) => {
    if (!search_settings?.integration) {
      callback?.({});
      return;
    }
    uploadAttachment(files, {
      headers: { 'APP-INTEGRATION-ID': search_settings.integration },
      ignoreError: true
    }).then(res => callback?.(res.data));
  };

  return (
    <div className='h-screen flex flex-col'>
      <header className='h-48px flex-y-center shrink-0 justify-between border-b border-gray-200 bg-white px-12px dark:border-gray-700 dark:bg-[#141414]'>
        <div className='flex flex-none items-center gap-4'>
          <div
            className='flex flex-none cursor-pointer items-center'
            onClick={() => goApp('search')}
          >
            <img
              alt='Coco AI'
              className='h-30px w-30px shrink-0 object-contain'
              src={appearance?.logo?.icon || cocoIcon}
            />
          </div>
          <Segmented
            value={activeApp}
            options={[
              { value: 'search', label: activeApp === 'search' ? t('route.search') : undefined, icon: <SearchOutlined /> },
              { value: 'chat', label: activeApp === 'chat' ? t('route.chat') : undefined, icon: <MessageOutlined /> },
              { value: 'wiki', label: activeApp === 'wiki' ? t('route.wiki') : undefined, icon: <BookOutlined /> }
            ]}
            onChange={value => goApp(value as string)}
          />
        </div>
        {/* the shell-wide search entry — the full AI-search box, present on
            every app page except the search app itself (which carries its own
            box); submit lands on the search results (chat-mode submits land on
            the chat app with attachments handed over) */}
        {showSearchBox && activeApp !== 'search' && !compact && (
          <div className='mx-12px w-340px max-w-340px flex-none'>
            {/* the widget resolves its labels against its own i18n instance —
                without this provider the keys render raw next to the host's */}
            <I18nextProvider i18n={searchWidgetI18n}>
              <SearchBox
                attachments={headerAttachments}
                className='!h-40px'
                minimize
                onSearch={handleHeaderSearch}
                onSuggestion={debouncedSuggestion}
                onUpload={handleHeaderUpload}
                placeholder={t('common.search')}
                queryParams={{}}
                setAttachments={setHeaderAttachments}
                settings={integration || undefined}
              />
            </I18nextProvider>
          </div>
        )}
        <div className='flex-y-center flex-none justify-end'>
          <LangSwitch className={compact ? 'px-4px' : 'px-12px'} />
          <ThemeSchemaSwitch className={compact ? 'px-4px' : 'px-12px'} />
          {/* the console entry lives in this dropdown (logged in + admin only);
              showConsole replaces the old standalone 管理后台 header button */}
          <UserAvatar
            className='px-8px'
            showConsole
            showName={false}
          />
        </div>
      </header>
      {/* contain:layout makes this wrapper the containing block for position:fixed
          descendants — the ui-search widget pins its own header with `fixed top-0`,
          which would otherwise cover this shell's header (fixed resolves against the
          viewport, not the wrapper) */}
      <div className='[contain:layout] relative min-h-0 flex-1'>
        <GlobalContent closePadding={true} />
      </div>
    </div>
  );
}
