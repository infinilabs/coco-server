interface Window {
  /** Ant-design-vue message instance */
  $message?: import('antd/es/message/interface').MessageInstance;
  /** Ant-design-vue modal instance */
  $modal?: Omit<import('antd/es/modal/confirm').ModalStaticFunctions, 'warn'>;
  /** Ant-design-vue notification instance */
  $notification?: import('antd/es/notification/interface').NotificationInstance;
  /** NProgress instance */
  NProgress?: import('nprogress').NProgress;

  /** WuJie global props */
  // 是否存在无界
  __POWERED_BY_WUJIE__?: boolean;
  // 子应用公共加载路径
  __WUJIE_PUBLIC_PATH__: string;
  // 原生的querySelector
  __WUJIE_RAW_DOCUMENT_QUERY_SELECTOR__: typeof Document.prototype.querySelector;
  // 原生的querySelectorAll
  __WUJIE_RAW_DOCUMENT_QUERY_SELECTOR_ALL__: typeof Document.prototype.querySelectorAll;
  // 原生的window对象
  __WUJIE_RAW_WINDOW__: Window;
  // 子应用沙盒实例
  __WUJIE: any;
  // 子应用mount函数
  __WUJIE_MOUNT: () => void;
  // 子应用unmount函数
  __WUJIE_UNMOUNT: () => void | Promise<void>;
  // 注入对象
  $wujie: {
    bus: any;
    shadowRoot?: ShadowRoot;
    props?: { [key: string]: any };
    location?: Object;
  };
}

interface ImportMeta {
  readonly env: Env.ImportMeta;
}

/** Build time of the project */
declare const BUILD_TIME: string;
