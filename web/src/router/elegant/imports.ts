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
  "data-source": () => import("@/pages/data-source/index.tsx"),
  guide: () => import("@/pages/guide/index.tsx"),
  home: () => import("@/pages/home/index.tsx"),
  login: () => import("@/pages/login/index.tsx"),
  settings: () => import("@/pages/settings/index.tsx"),
};
