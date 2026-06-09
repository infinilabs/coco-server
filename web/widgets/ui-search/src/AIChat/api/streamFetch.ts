/**
 * Sends a non-streaming POST request. Used for /_create and /_send.
 */
export async function postJSON<T = unknown>({
  url,
  body,
  queryParams,
  headers,
}: {
  url: string;
  body: unknown;
  queryParams?: Record<string, string | number | boolean | null | undefined>;
  headers?: Record<string, string>;
}): Promise<T> {
  const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");
  let baseURL: string = appStore.state?.endpoint_http;
  if (!baseURL || baseURL === "undefined") baseURL = "";
  const headersStorage = JSON.parse(localStorage.getItem("headers") || "{}") as Record<string, string>;

  const qs: Record<string, string> = {};
  if (queryParams) {
    Object.entries(queryParams).forEach(([k, v]) => {
      if (v !== undefined && v !== null) qs[k] = String(v);
    });
  }
  const query = new URLSearchParams(qs).toString();
  const fullUrl = query ? `${baseURL}${url}?${query}` : `${baseURL}${url}`;

  const res = await fetch(fullUrl, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...headersStorage, ...(headers || {}) },
    credentials: "include",
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`);
  return res.json();
}
