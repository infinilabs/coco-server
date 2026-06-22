import { StyleProvider } from "@ant-design/cssinjs";
import React, { useEffect } from "react";
import 'virtual:uno.css';

import { default as Page } from "./FullscreenPage";
import { default as Modal } from "./FullscreenModal";

export { DocDetail, ActionButton } from "./ResultDetail/DocDetail";

import "./index.css";
import "./styles/css/global.css";
import nprogressCSS from "./styles/css/nprogress.css?inline";
import NProgress from "nprogress";
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
    const root = shadow || document;
    let style: HTMLStyleElement | null = null;

    if (shadow) {
      // Inject NProgress CSS into shadow container
      style = document.createElement("style");
      style.setAttribute("data-nprogress", "");
      style.textContent = nprogressCSS;
      shadow.prepend(style);
    }

    // Patch NProgress to render inside the .ui-search container
    const originalRender = NProgress.render;
    const originalRemove = NProgress.remove;
    const originalIsRendered = NProgress.isRendered;

    NProgress.isRendered = function () {
      return !!root.querySelector("#nprogress");
    };

    NProgress.render = function (fromStart?: boolean) {
      if (NProgress.isRendered()) return root.querySelector("#nprogress") as HTMLDivElement;

      const progress = document.createElement("div");
      progress.id = "nprogress";
      progress.innerHTML = NProgress.settings.template;

      const bar = progress.querySelector(
        NProgress.settings.barSelector
      ) as HTMLElement;
      const perc = fromStart
        ? "-100"
        : String(((NProgress.status ?? 0) - 1) * 100);
      if (bar) {
        bar.style.transition = "all 0 linear";
        bar.style.transform = "translate3d(" + perc + "%,0,0)";
      }

      if (!NProgress.settings.showSpinner) {
        const spinner = progress.querySelector(NProgress.settings.spinnerSelector);
        if (spinner) spinner.remove();
      }

      const container = root.querySelector('.ui-search') || (shadow ? shadow : document.body);
      container.appendChild(progress);
      return progress;
    };

    NProgress.remove = function () {
      const progress = root.querySelector("#nprogress");
      if (progress) progress.remove();
    };

    return () => {
      if (style) style.remove();
      NProgress.render = originalRender;
      NProgress.remove = originalRemove;
      NProgress.isRendered = originalIsRendered;
    };
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
