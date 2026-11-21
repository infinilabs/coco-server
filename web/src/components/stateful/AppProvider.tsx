import { App } from 'antd';

import { DARK_MODE_MEDIA_QUERY } from '@/constants/common';
import { cacheTabs } from '@/store/slice/tab';
import { cacheThemeSettings, getDarkMode } from '@/store/slice/theme';
import { setDarkMode } from '@/store/slice/theme/index.ts';
import { applyAppearanceTheme } from '@/store/slice/theme/appearance';
import { getAppearance } from '@/store/slice/server';
import { createQuietMessage } from '@/utils/quiet-message';

function ContextHolder() {
  const { message, modal, notification } = App.useApp();
  // deduped + capped: identical toasts merge into one, error toasts are capped
  // (see quiet-message) so request-failure bursts don't flood the screen
  window.$message = createQuietMessage(message);
  window.$modal = modal;
  window.$notification = notification;
  return null;
}

const AppProvider = memo(({ children }: { children: React.ReactNode }) => {
  const dispatch = useAppDispatch();

  // admin-configured branding colors are enforced: re-applied on every boot
  // and on every light/dark switch so they always win over the locally
  // cached theme
  const appearance = useAppSelector(getAppearance);
  const themeColorsConfig = appearance?.theme_colors;
  const darkMode = useAppSelector(getDarkMode);
  useEffect(() => {
    dispatch(applyAppearanceTheme(appearance, darkMode));
  }, [JSON.stringify(themeColorsConfig), darkMode]);

  useEventListener(
    'beforeunload',
    () => {
      dispatch(cacheTabs());
      dispatch(cacheThemeSettings());
    },
    { target: window }
  );

  useMount(() => {
    window.matchMedia(DARK_MODE_MEDIA_QUERY).addEventListener('change', event => {
      dispatch(setDarkMode(event.matches));
    });
  });

  return (
    <App className='h-full'>
      <ContextHolder />
      {children}
    </App>
  );
});

export default AppProvider;
