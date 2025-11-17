import { StyleProvider } from "@ant-design/cssinjs";
import React, { useEffect } from "react";

import { default as Page } from "./FullscreenPage";
import { default as Modal } from "./FullscreenModal";

import "./index.css";
import { ConfigProvider } from "antd";

import enUS from 'antd/es/locale/en_US';
import zhCN from 'antd/es/locale/zh_CN';
import { getAntdTheme, initThemeSettings, setupThemeVarsToHtml } from './theme/shared';

export const antdLocales = {
  'en-US': enUS,
  'zh-CN': zhCN
};

const Wrapper = (props) => {
  const { shadow, theme = 'light', language = 'en-US', children } = props;

  const themeSettings = initThemeSettings();
  const { isInfoFollowPrimary, otherColor, themeColor } = themeSettings
  const colors = {
    primary: themeColor,
    ...otherColor,
    info: isInfoFollowPrimary ? themeColor : otherColor.info
  };
  console.log('theme', theme)
  const darkMode = theme === 'dark';
  const antdTheme = getAntdTheme(colors, darkMode, themeSettings.tokens);

  useEffect(() => {
    setupThemeVarsToHtml(colors, themeSettings.tokens, themeSettings.recommendColor, shadow);
  }, [colors, themeSettings]);

  return (
    <React.StrictMode>
      <StyleProvider container={shadow}>
        <ConfigProvider
          button={{ classNames: { icon: 'align-1px  text-icon' } }}
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


export const FullscreenPage = (props) => {
  const { shadow } = props;

  return (
    <Wrapper {...props}>
      <Page {...props} root={shadow || document} />
    </Wrapper>
  );
};

export const FullscreenModal = (props) => {
  const { shadow } = props;

  return (
    <Wrapper {...props}>
      <Modal {...props} root={shadow || document} />
    </Wrapper>
  );
};
