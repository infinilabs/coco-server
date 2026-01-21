export function getEndpoint() {
  return window.__POWERED_BY_WUJIE__ && window.$wujie?.props?.endpoint ? window.$wujie?.props?.endpoint : `${window.location.origin}${window.location.pathname}`;
}

export function getProxyEndpoint() {
  const proxy = window.__POWERED_BY_WUJIE__ ? window.$wujie?.props?.proxy_endpoint : ''
  return proxy
}

export function getName(name: string) {
  return window.__POWERED_BY_WUJIE__ && window.$wujie?.props?.name ? window.$wujie?.props?.name : name;
}