import { handleUpdateRootRouteRedirect } from "../route";

export function updateRootRoute(providerInfo: any) {
  if (providerInfo?.search_settings?.enabled && providerInfo?.search_settings?.integration) {
    handleUpdateRootRouteRedirect('search')
  } else {
    handleUpdateRootRouteRedirect('home')
  }
}
