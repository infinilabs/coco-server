export function getRootRouteIfSearch(applicationSetting: any) {
  let root = import.meta.env.VITE_ROUTE_HOME
  if (applicationSetting?.search_settings?.enabled && applicationSetting?.search_settings?.integration) {
    root = 'search'
  } else {
    root
  }
  return root
}
