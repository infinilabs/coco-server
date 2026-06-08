import { StyleProvider } from "@ant-design/cssinjs";
import React, { useEffect } from "react";

import { default as Page } from "./FullscreenPage";
import { default as Modal } from "./FullscreenModal";

export { DocDetail, ActionButton } from "./ResultDetail/DocDetail";

import "./index.css";
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

  return (
    <React.StrictMode>
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
    </React.StrictMode>
  )
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
