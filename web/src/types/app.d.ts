/** The global namespace for the app */
declare namespace App {
  /** Theme namespace */
  namespace Theme {
    type ColorPaletteNumber = import('@sa/color').ColorPaletteNumber;

    /** Theme setting */
    interface ThemeSetting {
      /** colour weakness mode */
      colourWeakness: boolean;
      /** Fixed header and tab */
      fixedHeaderAndTab: boolean;
      /** Footer */
      footer: {
        /** Whether fixed the footer */
        fixed: boolean;
        /** Footer height */
        height: number;
        /** Whether float the footer to the right when the layout is 'horizontal-mix' */
        right: boolean;
        /** Whether to show the footer */
        visible: boolean;
      };
      /** grayscale mode */
      grayscale: boolean;
      /** Header */
      header: {
        /** Header breadcrumb */
        breadcrumb: {
          /** Whether to show the breadcrumb icon */
          showIcon: boolean;
          /** Whether to show the breadcrumb */
          visible: boolean;
        };
        /** Header height */
        height: number;
      };
      /** Whether info color is followed by the primary color */
      isInfoFollowPrimary: boolean;
      /** Whether only expand the current parent menu when the layout is 'vertical-mix' or 'horizontal-mix' */
      isOnlyExpandCurrentParentMenu: boolean;
      /** Layout */
      layout: {
        /** Layout mode */
        mode: UnionKey.ThemeLayoutMode;
        /**
         * Whether to reverse the horizontal mix
         *
         * if true, the vertical child level menus in left and horizontal first level menus in top
         */
        reverseHorizontalMix: boolean;
        /** Scroll mode */
        scrollMode: UnionKey.ThemeScrollMode;
      };
      /** Other color */
      otherColor: OtherColor;
      /** Page */
      page: {
        /** Whether to show the page transition */
        animate: boolean;
        /** Page animate mode */
        animateMode: UnionKey.ThemePageAnimateMode;
      };
      /** Whether to recommend color */
      recommendColor: boolean;
      /** Sider */
      sider: {
        /** Collapsed sider width */
        collapsedWidth: number;
        /** Inverted sider */
        inverted: boolean;
        /** Child menu width when the layout is 'vertical-mix' or 'horizontal-mix' */
        mixChildMenuWidth: number;
        /** Collapsed sider width when the layout is 'vertical-mix' or 'horizontal-mix' */
        mixCollapsedWidth: number;
        /** Sider width when the layout is 'vertical-mix' or 'horizontal-mix' */
        mixWidth: number;
        /** Sider width */
        width: number;
      };
      /** Tab */
      tab: {
        /**
         * Whether to cache the tab
         *
         * If cache, the tabs will get from the local storage when the page is refreshed
         */
        cache: boolean;
        /** Tab height */
        height: number;
        /** Tab mode */
        mode: UnionKey.ThemeTabMode;
        /** Whether to show the tab */
        visible: boolean;
      };
      /** Theme color */
      themeColor: string;
      /** Theme scheme */
      themeScheme: UnionKey.ThemeScheme;
      /** define some theme settings tokens, will transform to css variables */
      tokens: {
        dark?: {
          [K in keyof ThemeSettingToken]?: Partial<ThemeSettingToken[K]>;
        };
        light: ThemeSettingToken;
      };
      /** Watermark */
      watermark: {
        /** Watermark text */
        text: string;
        /** Whether to show the watermark */
        visible: boolean;
      };
    }

    interface OtherColor {
      error: string;
      info: string;
      success: string;
      warning: string;
    }

    interface ThemeColor extends OtherColor {
      primary: string;
    }

    type ThemeColorKey = keyof ThemeColor;

    type ThemePaletteColor = {
      [key in ThemeColorKey | `${ThemeColorKey}-${ColorPaletteNumber}`]: string;
    };

    type BaseToken = Record<string, Record<string, string>>;

    interface ThemeSettingTokenColor {
      'base-text': string;
      container: string;
      inverted: string;
      layout: string;
      /** the progress bar color, if not set, will use the primary color */
      nprogress?: string;
    }

    interface ThemeSettingTokenBoxShadow {
      header: string;
      sider: string;
      tab: string;
    }

    interface ThemeSettingToken {
      boxShadow: ThemeSettingTokenBoxShadow;
      colors: ThemeSettingTokenColor;
    }

    type ThemeTokenColor = ThemePaletteColor & ThemeSettingTokenColor;

    /** Theme token CSS variables */
    type ThemeTokenCSSVars = {
      boxShadow: ThemeSettingTokenBoxShadow & { [key: string]: string };
      colors: ThemeTokenColor & { [key: string]: string };
    };
  }

  /** Global namespace */
  namespace Global {
    type RouteLocationNormalizedLoaded = import('@sa/simple-router').Route;
    type RouteKey = import('@elegant-router/types').RouteKey;
    type RouteMap = import('@elegant-router/types').RouteMap;
    type RoutePath = import('@elegant-router/types').RoutePath;
    type LastLevelRouteKey = import('@elegant-router/types').LastLevelRouteKey;

    /** The global header props */
    interface HeaderProps {
      /** Whether to show the logo */
      showLogo?: boolean;
      /** Whether to show the menu */
      showMenu?: boolean;
      /** Whether to show the menu toggler */
      showMenuToggler?: boolean;
    }

    interface IconProps {
      className?: string;
      /** Iconify icon name */
      icon?: string;
      /** Local svg icon name */
      localIcon?: string;
      style?: React.CSSProperties;
    }

    /** The global menu */
    interface Menu {
      /** The menu children */
      children?: Menu[];
      /** The menu i18n key */
      i18nKey?: I18n.I18nKey | null;
      /** The menu icon */
      icon?: React.FunctionComponentElement<IconProps>;
      /**
       * The menu key
       *
       * Equal to the route key
       */
      key: string;
      /** The menu label */
      label: React.ReactNode;
      /** The tooltip title */
      title?: string;
    }

    type Breadcrumb = Omit<Menu, 'children'> & {
      options?: Breadcrumb[];
    };

    /** Tab route */
    type TabRoute = Pick<RouteLocationNormalizedLoaded, 'meta' | 'name' | 'path'> &
      Partial<Pick<RouteLocationNormalizedLoaded, 'fullPath' | 'matched' | 'query'>>;

    /** The global tab */
    type Tab = {
      /** The tab fixed index */
      fixedIndex?: number | null;
      /** The tab route full path */
      fullPath: string;
      /** I18n key */
      i18nKey?: I18n.I18nKey | null | string;
      /**
       * Tab icon
       *
       * Iconify icon
       */
      icon?: string;
      /** The tab id */
      id: string;
      /** The tab label */
      label: string;
      /**
       * Tab local icon
       *
       * Local icon
       */
      localIcon?: string;
      /**
       * The new tab label
       *
       * If set, the tab label will be replaced by this value
       */
      newLabel?: string;
      /**
       * The old tab label
       *
       * when reset the tab label, the tab label will be replaced by this value
       */
      oldLabel?: string | null;
      /** The tab route key */
      routeKey: LastLevelRouteKey;
      /** The tab route path */
      routePath: RouteMap[LastLevelRouteKey];
    };

    /** Form rule */
    type FormRule = import('antd').FormRule;

    /** The global dropdown key */
    type DropdownKey = 'closeAll' | 'closeCurrent' | 'closeLeft' | 'closeOther' | 'closeRight';
  }

  /**
   * I18n namespace
   *
   * Locales type
   */
  namespace I18n {
    type RouteKey = import('@elegant-router/types').RouteKey;

    type LangType = 'en-US' | 'zh-CN';

    type LangOption = {
      key: LangType;
      label: string;
    };

    type I18nRouteKey = Exclude<RouteKey, 'not-found' | 'root'>;

    type FormMsg = {
      invalid: string;
      required: string;
    };

    type Schema = {
      translation: {
        common: {
          action: string;
          add: string;
          addSuccess: string;
          advanced: string;
          backToHome: string;
          batchDelete: string;
          cancel: string;
          check: string;
          close: string;
          columnSetting: string;
          comingSoon: string;
          config: string;
          confirm: string;
          confirmDelete: string;
          copy: string;
          copySuccess: string;
          create: string;
          delete: string;
          deleteSuccess: string;
          edit: string;
          error: string;
          errorHint: string;
          expandColumn: string;
          index: string;
          keywordSearch: string;
          loginAgain: string;
          logout: string;
          logoutConfirm: string;
          lookForward: string;
          modify: string;
          modifyPassword: string;
          modifySuccess: string;
          newPassword: string;
          noData: string;
          oldPassword: string;
          operate: string;
          operation: string;
          password: string;
          pleaseCheckValue: string;
          refresh: string;
          rename: string;
          reset: string;
          save: string;
          search: string;
          switch: string;
          testConnection: string;
          tip: string;
          trigger: string;
          tryAgain: string;
          update: string;
          updateSuccess: string;
          userCenter: string;
          warning: string;
          yesOrNo: {
            no: string;
            yes: string;
          };

          enableOrDisable: {
            enable: string;
            disable: string;
          };
          preview: string;
          language: any;
        };
        datatable: {
          itemCount: string;
        };
        dropdown: Record<Global.DropdownKey, string>;
        form: {
          code: FormMsg;
          confirmPwd: FormMsg;
          email: FormMsg;
          endpoint: FormMsg;
          phone: FormMsg;
          pwd: FormMsg;
          required: string;
          userName: FormMsg;
        };
        icon: {
          collapse: string;
          expand: string;
          fullscreen: string;
          fullscreenExit: string;
          lang: string;
          pin: string;
          reload: string;
          themeConfig: string;
          themeSchema: string;
          unpin: string;
        };
        page: {
          datasource: {
            columns: {
              enabled: string;
              latest_sync_time: string;
              name: string;
              sync_policy: string;
              sync_status: string;
              type: string;
            };
            new: {
              labels: {
                data_sync: string;
                immediate_sync: string;
                indexing_scope: string;
                manual_sync: string;
                manual_sync_desc: string;
                name: string;
                realtime_sync: string;
                realtime_sync_desc: string;
                scheduled_sync: string;
                scheduled_sync_desc: string;
                type: string;
              };
              title: string;
            };
          };
          guide: {
            llm: {
              desc: string;
              title: string;
            };
            setupLater: string;
            user: {
              desc: string;
              email: string;
              name: string;
              password: string;
              title: string;
              language: string;
            };
          };
          home: {
            server: {
              address: string;
              addressDesc: string;
              downloadCocoAI: string;
              title: string;
            };
            settings: {
              aiAssistant: string;
              aiAssistantDesc: string;
              dataSource: string;
              dataSourceDesc: string;
              llm: string;
              llmDesc: string;
            };
          };
          login: {
            cocoAI: {
              autoDesc: string;
              copyDesc: string;
              enterCocoServer: string;
              enterCocoServerDesc: string;
              launchCocoAI: string;
              title: string;
            };
            common: {
              back: string;
              codeLogin: string;
              codePlaceholder: string;
              confirm: string;
              confirmPasswordPlaceholder: string;
              loginOrRegister: string;
              loginSuccess: string;
              passwordPlaceholder: string;
              phonePlaceholder: string;
              userNamePlaceholder: string;
              validateSuccess: string;
              welcomeBack: string;
            };
            desc: string;
            password: string;
            title: string;
          };
          settings: {
            llm: {
              defaultModel: string;
              endpoint: string;
              enhanced_inference: string;
              frequency_penalty: string;
              frequency_penalty_desc: string;
              keepalive: string;
              max_tokens: string;
              max_tokens_desc: string;
              presence_penalty: string;
              presence_penalty_desc: string;
              requestParams: string;
              temperature: string;
              temperature_desc: string;
              top_p: string;
              top_p_desc: string;
              type: string;
            };
          };
        };
        request: {
          logout: string;
          logoutMsg: string;
          logoutWithModal: string;
          logoutWithModalMsg: string;
          refreshToken: string;
          tokenExpired: string;
        };
        route: Record<I18nRouteKey, string>;
        system: {
          errorReason: string;
          reload: string;
          title: string;
          updateCancel: string;
          updateConfirm: string;
          updateContent: string;
          updateTitle: string;
        };
        theme: {
          colourWeakness: string;
          configOperation: {
            copyConfig: string;
            copySuccessMsg: string;
            resetConfig: string;
            resetSuccessMsg: string;
          };
          fixedHeaderAndTab: string;
          footer: {
            fixed: string;
            height: string;
            right: string;
            visible: string;
          };
          grayscale: string;
          header: {
            breadcrumb: {
              showIcon: string;
              visible: string;
            };
            height: string;
          };
          isOnlyExpandCurrentParentMenu: string;
          layoutMode: { reverseHorizontalMix: string; title: string } & Record<UnionKey.ThemeLayoutMode, string>;
          page: {
            animate: string;
            mode: { title: string } & Record<UnionKey.ThemePageAnimateMode, string>;
          };
          pageFunTitle: string;
          recommendColor: string;
          recommendColorDesc: string;
          scrollMode: { title: string } & Record<UnionKey.ThemeScrollMode, string>;
          sider: {
            collapsedWidth: string;
            inverted: string;
            mixChildMenuWidth: string;
            mixCollapsedWidth: string;
            mixWidth: string;
            width: string;
          };
          tab: {
            cache: string;
            height: string;
            mode: { title: string } & Record<UnionKey.ThemeTabMode, string>;
            visible: string;
          };
          themeColor: {
            followPrimary: string;
            title: string;
          } & Theme.ThemeColor;
          themeDrawerTitle: string;
          themeSchema: { title: string } & Record<UnionKey.ThemeScheme, string>;
          watermark: {
            text: string;
            visible: string;
          };
        };
      };
    };

    type GetI18nKey<T extends Record<string, unknown>, K extends keyof T = keyof T> = K extends string
      ? T[K] extends Record<string, unknown>
        ? `${K}.${GetI18nKey<T[K]>}`
        : K
      : never;

    type I18nKey = GetI18nKey<Schema['translation']>;

    type TranslateOptions<Locales extends string> = import('react-i18next').TranslationProps<Locales>;

    interface $T {
      (key: I18nKey): string;
      (key: I18nKey, plural: number, options?: TranslateOptions<LangType>): string;
      (key: I18nKey, defaultMsg: string, options?: TranslateOptions<I18nKey>): string;
      (key: I18nKey, list: unknown[], options?: TranslateOptions<I18nKey>): string;
      (key: I18nKey, list: unknown[], plural: number): string;
      (key: I18nKey, list: unknown[], defaultMsg: string): string;
      (key: I18nKey, named: Record<string, unknown>, options?: TranslateOptions<LangType>): string;
      (key: I18nKey, named: Record<string, unknown>, plural: number): string;
      (key: I18nKey, named: Record<string, unknown>, defaultMsg: string): string;
    }
  }

  /** Service namespace */
  namespace Service {
    /** Other baseURL key */
    type OtherBaseURLKey = 'demo';

    interface ServiceConfigItem {
      /** The backend service base url */
      baseURL: string;
      /** The proxy pattern of the backend service base url */
      proxyPattern: string;
    }

    interface OtherServiceConfigItem extends ServiceConfigItem {
      key: OtherBaseURLKey;
    }

    /** The backend service config */
    interface ServiceConfig extends ServiceConfigItem {
      /** Other backend service config */
      other: OtherServiceConfigItem[];
    }

    interface SimpleServiceConfig extends Pick<ServiceConfigItem, 'baseURL'> {
      other: Record<OtherBaseURLKey, string>;
    }

    /** The backend service response data */
    type Response<T = unknown> = {
      /** The backend service response data */
      data: T;
      error: T;
      /** The backend service response message */
      msg: string;
      /** The backend service response code */
      status: number;
    };

    /** The demo backend service response data */
    type DemoResponse<T = unknown> = {
      /** The backend service response message */
      message: string;
      /** The backend service response data */
      result: T;
      /** The backend service response code */
      status: string;
    };
  }
}
