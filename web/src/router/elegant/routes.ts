/* eslint-disable */
/* prettier-ignore */
// Generated by elegant-router

import type { GeneratedRoute } from '@elegant-router/types';

export const generatedRoutes: GeneratedRoute[] = [
  {
    name: '403',
    path: '/403',
    component: 'layout.base$view.403',
    meta: {
      title: '403',
      i18nKey: 'route.403',
      constant: true,
      hideInMenu: true
    }
  },
  {
    name: '404',
    path: '/404',
    component: 'layout.base$view.404',
    meta: {
      title: '404',
      i18nKey: 'route.404',
      constant: true,
      hideInMenu: true
    }
  },
  {
    name: '500',
    path: '/500',
    component: 'layout.base$view.500',
    meta: {
      title: '500',
      i18nKey: 'route.500',
      constant: true,
      hideInMenu: true
    }
  },
  {
    name: 'ai-assistant',
    path: '/ai-assistant',
    component: 'layout.base',
    redirect: 'list',
    meta: {
      i18nKey: 'route.ai-assistant',
      title: 'ai-assistant',
      icon: 'mdi:robot-outline',
      order: 2
    },
    children: [
      {
        name: 'ai-assistant_edit',
        path: 'edit/:id',
        component: 'view.ai-assistant_edit',
        meta: {
          i18nKey: 'route.ai-assistant_edit',
          title: 'ai-assistant_edit',
          hideInMenu: true,
          activeMenu: 'ai-assistant'
        }
      },
      {
        name: 'ai-assistant_list',
        path: 'list',
        component: 'view.ai-assistant_list',
        meta: {
          i18nKey: 'route.ai-assistant_list',
          title: 'ai-assistant_list',
          hideInMenu: true,
          activeMenu: 'ai-assistant'
        }
      },
      {
        name: 'ai-assistant_new',
        path: 'new',
        component: 'view.ai-assistant_new',
        meta: {
          i18nKey: 'route.ai-assistant_new',
          title: 'ai-assistant_new',
          hideInMenu: true,
          activeMenu: 'ai-assistant'
        }
      }
    ]
  },
  {
    name: 'api-token',
    path: '/api-token',
    component: 'layout.base',
    meta: {
      i18nKey: 'route.api-token',
      title: 'api-token',
      order: 4,
      localIcon: 'security'
    },
    children: [
      {
        name: 'api-token_list',
        path: 'list',
        component: 'view.api-token_list',
        meta: {
          i18nKey: 'route.api-token_list',
          title: 'api-token_list',
          hideInMenu: true,
          activeMenu: 'api-token'
        }
      }
    ]
  },
  {
    name: 'connector',
    path: '/connector',
    component: 'layout.base',
    meta: {
      i18nKey: 'route.connector',
      title: 'connector',
      hideInMenu: true,
      activeMenu: 'settings'
    },
    children: [
      {
        name: 'connector_edit',
        path: 'edit/:id',
        component: 'view.connector_edit',
        meta: {
          i18nKey: 'route.connector_edit',
          title: 'connector_edit'
        }
      },
      {
        name: 'connector_new',
        path: 'new',
        component: 'view.connector_new',
        meta: {
          i18nKey: 'route.connector_new',
          title: 'connector_new',
          hideInMenu: true,
          activeMenu: 'settings'
        }
      }
    ]
  },
  {
    name: 'data-source',
    path: '/data-source',
    component: 'layout.base',
    redirect: 'list',
    meta: {
      i18nKey: 'route.data-source',
      title: 'data-source',
      icon: 'mdi:folder-open-outline',
      order: 3
    },
    children: [
      {
        name: 'data-source_detail',
        path: 'detail/:id',
        component: 'view.data-source_detail',
        meta: {
          i18nKey: 'route.data-source_detail',
          title: 'data-source_detail',
          hideInMenu: true,
          activeMenu: 'data-source'
        }
      },
      {
        name: 'data-source_edit',
        path: 'edit/:id',
        component: 'view.data-source_edit',
        meta: {
          i18nKey: 'route.data-source_edit',
          title: 'data-source_edit',
          hideInMenu: true,
          activeMenu: 'data-source'
        }
      },
      {
        name: 'data-source_list',
        path: 'list',
        component: 'view.data-source_list',
        meta: {
          i18nKey: 'route.data-source_list',
          title: 'data-source_list',
          hideInMenu: true,
          activeMenu: 'data-source'
        }
      },
      {
        name: 'data-source_new',
        path: 'new',
        component: 'view.data-source_new',
        meta: {
          i18nKey: 'route.data-source_new',
          title: 'data-source_new',
          hideInMenu: true,
          activeMenu: 'data-source'
        }
      },
      {
        name: 'data-source_new-first',
        path: 'new-first',
        component: 'view.data-source_new-first',
        meta: {
          i18nKey: 'route.data-source_new-first',
          title: 'data-source_new-first',
          hideInMenu: true,
          activeMenu: 'data-source'
        }
      }
    ]
  },
  {
    name: 'guide',
    path: '/guide',
    component: 'layout.blank$view.guide',
    meta: {
      title: 'guide',
      i18nKey: 'route.guide',
      constant: true,
      hideInMenu: true
    }
  },
  {
    name: 'home',
    path: '/home',
    component: 'layout.base$view.home',
    meta: {
      i18nKey: 'route.home',
      title: 'home',
      icon: 'mdi:home',
      order: 1
    }
  },
  {
    name: 'integration',
    path: '/integration',
    component: 'layout.base',
    redirect: 'list',
    meta: {
      i18nKey: 'route.integration',
      title: 'integration',
      icon: 'mdi:puzzle-outline',
      order: 5
    },
    children: [
      {
        name: 'integration_edit',
        path: 'edit/:id',
        component: 'view.integration_edit',
        meta: {
          i18nKey: 'route.integration_edit',
          title: 'integration_edit',
          hideInMenu: true,
          activeMenu: 'integration'
        }
      },
      {
        name: 'integration_list',
        path: 'list',
        component: 'view.integration_list',
        meta: {
          i18nKey: 'route.integration_list',
          title: 'integration_list',
          hideInMenu: true,
          activeMenu: 'integration'
        }
      },
      {
        name: 'integration_new',
        path: 'new',
        component: 'view.integration_new',
        meta: {
          i18nKey: 'route.integration_new',
          title: 'integration_new',
          hideInMenu: true,
          activeMenu: 'integration'
        }
      }
    ]
  },
  {
    name: 'login',
    path: '/login',
    component: 'layout.blank$view.login',
    meta: {
      title: 'login',
      i18nKey: 'route.login',
      constant: true,
      hideInMenu: true
    }
  },
  {
    name: 'mcp-server',
    path: '/mcp-server',
    component: 'layout.base',
    meta: {
      i18nKey: 'route.mcp-server',
      title: 'mcp-server',
      order: 2,
      localIcon: 'mcp'
    },
    redirect: 'list',
    children: [
      {
        name: 'mcp-server_edit',
        path: 'edit/:id',
        component: 'view.mcp-server_edit',
        meta: {
          i18nKey: 'route.mcp-server_edit',
          title: 'mcp-server_edit',
          hideInMenu: true,
          activeMenu: 'mcp-server'
        }
      },
      {
        name: 'mcp-server_list',
        path: 'list',
        component: 'view.mcp-server_list',
        meta: {
          i18nKey: 'route.mcp-server_list',
          title: 'mcp-server_list',
          hideInMenu: true,
          activeMenu: 'mcp-server'
        }
      },
      {
        name: 'mcp-server_new',
        path: 'new',
        component: 'view.mcp-server_new',
        meta: {
          i18nKey: 'route.mcp-server_new',
          title: 'mcp-server_new',
          hideInMenu: true,
          activeMenu: 'mcp-server'
        }
      }
    ]
  },
  {
    name: 'model-provider',
    path: '/model-provider',
    component: 'layout.base',
    redirect: 'list',
    meta: {
      i18nKey: 'route.model-provider',
      title: 'model-provider',
      order: 2,
      localIcon: 'model-provider'
    },
    children: [
      {
        name: 'model-provider_edit',
        path: 'edit/:id',
        component: 'view.model-provider_edit',
        meta: {
          i18nKey: 'route.model-provider_edit',
          title: 'model-provider_edit',
          hideInMenu: true,
          activeMenu: 'model-provider'
        }
      },
      {
        name: 'model-provider_list',
        path: 'list',
        component: 'view.model-provider_list',
        meta: {
          i18nKey: 'route.model-provider_list',
          title: 'model-provider_list',
          hideInMenu: true,
          activeMenu: 'model-provider'
        }
      },
      {
        name: 'model-provider_new',
        path: 'new',
        component: 'view.model-provider_new',
        meta: {
          i18nKey: 'route.model-provider_new',
          title: 'model-provider_new',
          hideInMenu: true,
          activeMenu: 'model-provider'
        }
      }
    ]
  },
  {
    name: 'settings',
    path: '/settings',
    component: 'layout.base$view.settings',
    meta: {
      i18nKey: 'route.settings',
      title: 'settings',
      icon: 'mdi:settings-outline',
      order: 10
    }
  }
];
