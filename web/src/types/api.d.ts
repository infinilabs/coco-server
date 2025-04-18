import { b } from 'vite/dist/node/types.d-aGj9QkWt';

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
      preferences: {
        language: string;
        theme: string;
      };
      roles: string[];
      updated: string;
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
      sync_config: any;
      sync_enabled: boolean;
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
}
