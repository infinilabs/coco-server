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

  // the integration preview's widget loader references its bundle with a
  // root-relative `/widgets/:name/index.js` (plus iconfonts) — in a srcdoc
  // iframe that resolves against THIS origin, and without the rule Vite's
  // SPA fallback returns index.html for the module import, which fails the
  // MIME check and leaves the preview blank.
  // Scope to the BUILT widget names only: a bare `/widgets` prefix would also
  // capture `/widgets/ui-search/src/*.tsx` — in dev the app imports the
  // workspace widget by source through that very path, and proxying those
  // module requests to the backend 404s the whole layout chunk
  for (const name of ['fullscreen', 'searchbox']) {
    proxy[`^/widgets/${name}/`] = {
      secure: false,
      changeOrigin: true,
      target: baseURL
    };
  }

  // the loader module itself is `/integration/:id/widget` — in dev the
  // server endpoint falls back to this origin, so the srcdoc iframe imports
  // it from here too; same SPA-fallback failure without the rule
  proxy['/integration'] = {
    secure: false,
    changeOrigin: true,
    target: baseURL
  };

  // the embedded widget fetches its settings and answers from the same
  // origin it is served from (`${server}/assistant/_search`,
  // `${server}/query/_search`) — in dev that is this origin
  proxy['/assistant'] = {
    secure: false,
    changeOrigin: true,
    target: baseURL
  };
  proxy['/query'] = {
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
