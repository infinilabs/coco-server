import { RouterProvider } from '@sa/simple-router';
import { useUpdateEffect } from 'ahooks';
import type { WatermarkProps } from 'antd';

import { info } from '@/constants/app';
import { router } from '@/router';
import { changeLocale, getLocale } from '@/store/slice/app';
import { getDarkMode, getThemeSettings, themeColors } from '@/store/slice/theme';
import { getAntdTheme, setupThemeVarsToHtml, toggleCssDarkMode } from '@/store/slice/theme/shared';
import { localStg } from '@/utils/storage';

import AppProvider from './components/stateful/AppProvider';
import { antdLocales } from './locales/antd';

const watermarkProps: WatermarkProps = {
  font: {
    fontSize: 16
  },
  height: 128,
  offset: [12, 60],
  rotate: -15,
  width: 240,
  zIndex: 9999
};

function useTheme() {
  const themeSettings = useAppSelector(getThemeSettings);
  const colors = useAppSelector(themeColors);

  const darkMode = useAppSelector(getDarkMode);
  const antdTheme = getAntdTheme(colors, darkMode, themeSettings.tokens);

  useEffect(() => {
    setupThemeVarsToHtml(colors, themeSettings.tokens, themeSettings.recommendColor);
    localStg.set('themeColor', colors.primary);
  }, [colors, themeSettings]);

  useEffect(() => {
    toggleCssDarkMode(darkMode);
    // theme flips (and the first paint) only rewrite CSS vars — composited
    // layers (watermark canvas, sticky rails, keepalive transforms) can keep a
    // stale raster and render a blank or half-blended frame. After the fresh
    // frame has painted, tear the body down once to force every layer to
    // re-rasterize with the current tokens.
    requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        document.body.style.display = 'none';
        void document.body.offsetHeight;
        document.body.style.display = '';
      })
    );
  }, [darkMode]);

  // console.info(`%c${info}`, `color: ${colors.primary}`);

  return { antdTheme, themeSettings };
}

const App = () => {
  const locale = useAppSelector(getLocale);

  const { antdTheme, themeSettings } = useTheme();
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (window.$wujie?.props?.locale) {
      dispatch(changeLocale(window.$wujie?.props?.locale));
    }
  }, [window.$wujie?.props?.locale]);

  return (
    <AConfigProvider
      button={{ classNames: { icon: 'flex items-center' } }}
      card={{ styles: { body: { flex: 1, overflow: 'hidden', padding: '12px 16px ' } } }}
      locale={antdLocales[locale]}
      theme={antdTheme}
    >
      <AppProvider>
        <AWatermark
          className="h-full"
          content={themeSettings.watermark.visible ? themeSettings.watermark?.text || 'InfiniLabs' : ''}
          {...watermarkProps}
        >
          <RouterProvider
            fallback={<GlobalLoading className={window.__POWERED_BY_WUJIE__ ? 'absolute' : ''} />}
            router={router}
          />
        </AWatermark>
      </AppProvider>
    </AConfigProvider>
  );
};

export default App;
