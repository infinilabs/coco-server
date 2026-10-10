import type { ProxyOptions } from 'vite';

import { createServiceConfig } from '../../src/utils/service';

/**
 * Set http proxy
 *
 * @param env - The current env
 * @param enable - If enable http proxy
 */
export function createViteProxy(env: Env.ImportMeta, enable: boolean) {
  const isEnableHttpProxy = enable && env.VITE_HTTP_PROXY === 'Y';

  if (!isEnableHttpProxy) return undefined;

  const { baseURL, other, proxyPattern } = createServiceConfig(env);

  const proxy: Record<string, ProxyOptions> = createProxyItem({ baseURL, proxyPattern });

  other.forEach(item => {
    Object.assign(proxy, createProxyItem(item));
  });

  // the backend serves attachments at `/attachment/:file_id` on the same origin
  // as the API, and stored attachment URLs are relative — keep the prefix
  // intact (no rewrite) so they resolve against the dev server too
  proxy['/attachment'] = {
    secure: false,
    changeOrigin: true,
    target: baseURL
  };

  // same for a document's raw content at `/document/:doc_id/raw_content/:hint`:
  // search results embed it as an absolute URL built from the server endpoint,
  // which in dev points at this origin — without the rule Vite answers with the
  // SPA's index.html and every preview (image/pdf/video) silently breaks
  proxy['/document'] = {
    secure: false,
    changeOrigin: true,
    target: baseURL
  };

  return proxy;
}

function createProxyItem(item: App.Service.ServiceConfigItem) {
  const proxy: Record<string, ProxyOptions> = {};

  proxy[item.proxyPattern] = {
    secure: false,
    changeOrigin: true,
    rewrite: path =>
      path?.replace(new RegExp(`^${item.proxyPattern}`), item.key?.startsWith('/') ? item.proxyPattern : ''),
    target: item.baseURL
  };

  return proxy;
}
