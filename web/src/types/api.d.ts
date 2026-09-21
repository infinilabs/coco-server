/**
 * Namespace Api
 *
 * All backend api type
 */
declare namespace Api {
  namespace Common {
    /** common params of paginating */
    interface PaginatingCommonParams {
      /** current page number */
      current: number;
      /** page size */
      size: number;
      /** total count */
      total: number;
    }

    /** common params of paginating query list data */
    interface PaginatingQueryRecord<T = any> extends PaginatingCommonParams {
      records: T[];
    }

    type CommonSearchParams = Pick<Common.PaginatingCommonParams, 'current' | 'size'>;

    /**
     * enable status
     *
     * - "1": enabled
     * - "2": disabled
     */
    type EnableStatus = '1' | '2';

    /** common record */
    type CommonRecord<T = any> = {
      /** record creator */
      createBy: string;
      /** record create time */
      createTime: string;
      /** record id */
      id: number;
      /** record status */
      status: EnableStatus | null;
      /** record updater */
      updateBy: string;
      /** record update time */
      updateTime: string;
    } & T;
  }

  /**
   * namespace Auth
   *
   * backend api module: "auth"
   */
  namespace Auth {
    interface LoginToken {
      access_token: string;
      expire_in: number;
    }

    interface UserInfo {
      avatar: string;
      created: string;
      email: string;
      id: string;
      name: string;
      permissions?: string[];
      preferences: {
        language: string;
        theme: string;
      };
      roles: string[];
      updated: string;
      [key: string]: any;
    }
  }

  /**
   * namespace Route
   *
   * backend api module: "route"
   */
  namespace Route {
    type ElegantConstRoute = import('@elegant-router/types').ElegantConstRoute;

    interface MenuRoute extends ElegantConstRoute {
      id: string;
    }

    interface UserRoute {
      home: import('@elegant-router/types').LastLevelRouteKey;
      routes: MenuRoute[];
    }
  }

  /**
   * namespace SystemManage
   *
   * backend api module: "systemManage"
   */
  namespace SystemManage {
    type CommonSearchParams = Pick<Common.PaginatingCommonParams, 'current' | 'size'>;

    /** role */
    type Role = Common.CommonRecord<{
      /** role code */
      roleCode: string;
      /** role description */
      roleDesc: string;
      /** role name */
      roleName: string;
    }>;

    /** role search params */
    type RoleSearchParams = CommonType.RecordNullable<
      Pick<Api.SystemManage.Role, 'roleCode' | 'roleName' | 'status'> & CommonSearchParams
    >;

    /** role list */
    type RoleList = Common.PaginatingQueryRecord<Role>;

    /** all role */
    type AllRole = Pick<Role, 'id' | 'roleCode' | 'roleName'>;

    /**
     * user gender
     *
     * - "1": "male"
     * - "2": "female"
     */
    type UserGender = '1' | '2';

    /** user */
    type User = Common.CommonRecord<{
      /** user nick name */
      nickName: string;
      /** user email */
      userEmail: string;
      /** user gender */
      userGender: UserGender | null;
      /** user name */
      userName: string;
      /** user phone */
      userPhone: string;
      /** user role code collection */
      userRoles: string[];
    }>;

    /** user search params */
    type UserSearchParams = CommonType.RecordNullable<
      Pick<Api.SystemManage.User, 'nickName' | 'status' | 'userEmail' | 'userGender' | 'userName' | 'userPhone'> &
        CommonSearchParams
    >;

    /** user list */
    type UserList = Common.PaginatingQueryRecord<User>;

    /**
     * menu type
     *
     * - "1": directory
     * - "2": menu
     */
    type MenuType = '1' | '2';

    type MenuButton = {
      /**
       * button code
       *
       * it can be used to control the button permission
       */
      code: string;
      /** button description */
      desc: string;
    };

    /**
     * icon type
     *
     * - "1": iconify icon
     * - "2": local icon
     */
    type IconType = '1' | '2';

    type MenuPropsOfRoute = Pick<
      import('@ohh-889/react-auto-route').RouteMeta,
      | 'activeMenu'
      | 'constant'
      | 'fixedIndexInTab'
      | 'hideInMenu'
      | 'href'
      | 'i18nKey'
      | 'keepAlive'
      | 'multiTab'
      | 'order'
      | 'query'
    >;

    type Menu = Common.CommonRecord<{
      /** buttons */
      buttons?: MenuButton[] | null;
      /** children menu */
      children?: Menu[] | null;
      /** component */
      component?: string;
      /** iconify icon name or local icon name */
      icon: string;
      /** icon type */
      iconType: IconType;
      /** menu name */
      menuName: string;
      /** menu type */
      menuType: MenuType;
      /** parent menu id */
      parentId: number;
      /** route name */
      routeName: string;
      /** route path */
      routePath: string;
    }> &
      MenuPropsOfRoute;

    /** menu list */
    type MenuList = Common.PaginatingQueryRecord<Menu>;

    type MenuTree = {
      children?: MenuTree[];
      id: number;
      label: string;
      pId: number;
    };
  }

  namespace Server {
    type OAuthProvider = {
      name?: string;
      icon?: string;
      url?: string;
      description?: string;
      type?: string;
    };

    type Info = {
      auth_provider: {
        sso: {
          url: string;
        };
      };
      endpoint: string;
      name: string;
      provider: {
        banner: string;
        description: string;
        eula: string;
        icon: string;
        name: string;
        privacy_policy: string;
        website: string;
      };
      public: boolean;
      setup_required: boolean;
      updated: string;
      version: {
        number: string;
      };
      search_settings?: {
        enabled?: boolean;
        integration?: string;
      };
      security?: {
        managed?: boolean;
        auth?: {
          native?: boolean;
          oauth?: Record<string, OAuthProvider>;
        };
      };
    };
  }
  namespace Datasource {
    interface ConnectorConfig {
      urls: string[];
    }

    interface Connector {
      assets: {
        icons: ConnectorIcons;
      };
      category: string;
      config: ConnectorConfig;
      description: string;
      icon: string;
      id: string;
      name: string;
      tags: string[];
      url: string;
    }
    interface ConnectorIcons {
      database: string;
      default: string;
      page: string;
      web_page: string;
    }

    interface Datasource {
      connector: Connector;
      created: string;
      enabled: boolean;
      id: string;
      name: string;
      sync:{
        enabled: boolean;
        interval: string;
        strategy: string;
      }
      // ISO 8601 timestamp
      type: 'connector';
      // ISO 8601 timestamp
      updated: string;
    }
  }
  namespace APIToken {
    interface APIToken {
      access_token: string;
      expire_in: number;
      login: string;
      name: string;
      provider: string;
      userid: string;
    }
  }
  namespace LLM {
    interface ModelProvider {
      name: string;
      icon: string;
      api_endpoint: string;
      api_key: string;
      models: string[];
      enabled: boolean;
    }
    interface Assistant {
      name: string;
      icon: string;
      type: string;
      enabled: boolean;
      description: string;
    }
    interface MCPServer {
      name: string;
      type: string;
      enabled: boolean;
      config: any;
      description: string;
    }
  }

  namespace Wiki {
    type Visibility = 'public' | 'private' | 'team';
    type SyncStrategy = 'realtime' | 'scheduled' | 'manual';
    type AiStatus = 'ready' | 'processing' | 'queued' | 'updating';
    type PageType = 'entity' | 'concept' | 'source';
    type ArticleStatus = 'draft' | 'reviewed' | 'published' | 'archived';
    type Confidence = 'high' | 'medium' | 'low';
    type MemberRole = 'owner' | 'editor' | 'viewer' | 'agent';
    type ChangeType = 'ai-generated' | 'human-edited' | 'auto-updated';

    interface Member {
      id: string;
      name: string;
      avatar: string;
      role: MemberRole;
      email?: string;
    }

    interface DatasourceInfo {
      id: string;
      type: string;
      name: string;
      status: 'connected' | 'syncing' | 'error' | 'disconnected';
      last_synced: string;
      document_count: number;
    }

    interface Kb {
      id: string;
      name: string;
      description: string;
      icon: string;
      visibility: Visibility;
      workspace_id: string;
      datasource_ids: string[];
      assistant_id?: string;
      sync_strategy?: SyncStrategy;
      article_count: number;
      last_updated: string;
      members: Member[];
      datasources: DatasourceInfo[];
      ai_status?: AiStatus;
    }

    interface SourceRef {
      doc_id: string;
      source_type: string;
      source_name: string;
      title: string;
      url?: string;
      excerpt: string;
      locator?: string;
    }

    interface Article {
      id: string;
      kb_id: string;
      toc_node_id?: string;
      title: string;
      summary: string;
      /** structured markdown (Obsidian wiki format), see pages/wiki/shared/content.ts */
      content: string;
      page_type?: PageType;
      subtype?: string;
      aliases?: string[];
      tags: string[];
      status: ArticleStatus;
      ai_generated: boolean;
      confidence?: Confidence;
      sources: SourceRef[];
      entity_id?: string;
      created_by: Member;
      contributors: Member[];
      created_at: string;
      updated_at: string;
    }

    interface Version {
      id: string;
      article_id: string;
      version: number;
      change_type: ChangeType;
      change_summary?: string;
      content: string;
      created_by: string;
      created_at: string;
    }

    interface TocNode {
      id: string;
      title: string;
      type: 'folder' | 'article';
      article_id?: string;
      icon?: string;
      children?: TocNode[];
    }
  }
}

declare module 'ui-search/source' {
  export const FullscreenPage: any;
  export const FullscreenModal: any;
  export const DocDetail: any;
  export const ActionButton: any;
  export const Preview: any;
}
