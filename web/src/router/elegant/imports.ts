/* eslint-disable */
/* prettier-ignore */
// Generated by elegant-router
// Read more: https://github.com/mufeng889/elegant-router
// Vue auto route: https://github.com/soybeanjs/elegant-router


import type { LazyRouteFunction, RouteObject } from "react-router-dom";
import type { LastLevelRouteKey, RouteLayout } from "@elegant-router/types";
type CustomRouteObject = Omit<RouteObject, 'Component'|'index'> & {
  Component?: React.ComponentType<any>|null;
};

export const layouts: Record<RouteLayout, LazyRouteFunction<CustomRouteObject>> = {
  base: () => import("@/layouts/base-layout/index.tsx"),
  blank: () => import("@/layouts/blank-layout/index.tsx"),
};

export const pages: Record<LastLevelRouteKey, LazyRouteFunction<CustomRouteObject>> = {
  403: () => import("@/pages/_builtin/403/index.tsx"),
  404: () => import("@/pages/_builtin/404/index.tsx"),
  500: () => import("@/pages/_builtin/500/index.tsx"),
  "ai-assistant": () => import("@/pages/ai-assistant/index.tsx"),
  "data-source_detail": () => import("@/pages/data-source/detail/[id].tsx"),
  "data-source_list": () => import("@/pages/data-source/list/index.tsx"),
  "data-source_new": () => import("@/pages/data-source/new/index.tsx"),
  "login_code-login": () => import("@/pages/login/code-login/index.tsx"),
  login: () => import("@/pages/login/index.tsx"),
  "login_pwd-login": () => import("@/pages/login/pwd-login/index.tsx"),
  login_register: () => import("@/pages/login/register/index.tsx"),
  "login_reset-pwd": () => import("@/pages/login/reset-pwd/index.tsx"),
  server: () => import("@/pages/server/index.tsx"),
  settings: () => import("@/pages/settings/index.tsx"),
};
