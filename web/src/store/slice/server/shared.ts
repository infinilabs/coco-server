export function getRootRouteIfSearch(providerInfo: any) {
  let root = import.meta.env.VITE_ROUTE_HOME
  if (providerInfo?.search_settings?.enabled && providerInfo?.search_settings?.integration) {
    root = 'search'
  } else {
    root
  }
  return root
}
