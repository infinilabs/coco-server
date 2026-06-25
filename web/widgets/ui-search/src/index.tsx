import { StyleProvider } from "@ant-design/cssinjs";
import React, { useEffect } from "react";
import 'virtual:uno.css';

import { default as Page } from "./FullscreenPage";
import { default as Modal } from "./FullscreenModal";

export { DocDetail, ActionButton } from "./ResultDetail/DocDetail";

import "./index.css";
import "./styles/css/global.css";
import { setNProgressRoot } from "./utils/nprogress";
import { ConfigProvider } from "antd";

import enUS from 'antd/es/locale/en_US';
import zhCN from 'antd/es/locale/zh_CN';
import { getAntdTheme, initThemeSettings, setupThemeVarsToHtml } from './theme/shared';

export const antdLocales: Record<string, any> = {
  'en-US': enUS,
  'zh-CN': zhCN
};

interface WrapperProps {
  shadow?: ShadowRoot | HTMLElement;
  theme?: 'light' | 'dark';
  language?: string;
  children?: React.ReactNode;
  [key: string]: any;
}

const Wrapper = (props: WrapperProps) => {
  const { shadow, theme = 'light', language = 'en-US', children } = props;

  const themeSettings = initThemeSettings();
  const { isInfoFollowPrimary, otherColor, themeColor } = themeSettings
  const colors = {
    primary: themeColor,
    ...otherColor,
    info: isInfoFollowPrimary ? themeColor : otherColor.info
  };
  const darkMode = theme === 'dark';
  const antdTheme = getAntdTheme(colors, darkMode, themeSettings.tokens);

  useEffect(() => {
    setupThemeVarsToHtml(colors, themeSettings.tokens, themeSettings.recommendColor, shadow);
  }, [colors, themeSettings]);

  useEffect(() => {
    const html = document.documentElement;
    const originalFontSize = html.style.fontSize;
    html.style.fontSize = '16px';
    return () => {
      html.style.fontSize = originalFontSize;
    };
  }, []);

  useEffect(() => {
    // Point the widget's scoped progress bar at this container (the shadow
    // root when used in a Shadow DOM, otherwise the document). The widget's
    // progress bar uses a *unique* DOM id (`nprogress-ui-search`) and its own
    // CSS, so it is fully independent of any host's `nprogress` instance and
    // will no longer override / break the host's `#nprogress` styles.
    setNProgressRoot(() => shadow || document);
  }, [shadow]);

  return (
    <StyleProvider container={shadow}>
      <ConfigProvider
        button={{ classNames: { icon: 'flex items-center' } }}
        card={{ styles: { body: { flex: 1, overflow: 'hidden', padding: '12px 16px ' } } }}
        locale={antdLocales[language]}
        theme={antdTheme}
      >
        {children}
      </ConfigProvider>
    </StyleProvider>
  );
}


interface PageProps {
  shadow?: ShadowRoot | HTMLElement;
  [key: string]: any;
}

export const FullscreenPage = (props: PageProps) => {
  const { shadow } = props;

  return (
    <Wrapper {...props}>
      <Page {...props} shadowRoot={shadow || document}/>
    </Wrapper>
  );
};

export const FullscreenModal = (props: PageProps) => {
  const { shadow } = props;

  return (
    <Wrapper {...props}>
      <Modal {...props} shadowRoot={shadow || document}/>
    </Wrapper>
  );
};
