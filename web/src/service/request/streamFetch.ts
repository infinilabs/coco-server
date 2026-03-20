import { getApiBaseUrl } from './index';
import { localStg } from '@/utils/storage';

export async function streamPost({
  url,
  body,
  queryParams,
  headers,
  onMessage,
  onError,
}: {
  url: string;
  body: any;
  queryParams?: Record<string, any>;
  headers?: Record<string, string>;
  onMessage: (chunk: string) => void;
  onError?: (err: any) => void;
}) {
  const baseURL = getApiBaseUrl();

  const qs = new URLSearchParams(
    Object.entries(queryParams || {}).reduce((acc: Record<string, string>, [k, v]) => {
      if (v === undefined || v === null || v === '') return acc;
      acc[k] = String(v);
      return acc;
    }, {})
  ).toString();

  const finalUrl = qs ? `${url}?${qs}` : url;
  const fullUrl = `${baseURL}${finalUrl}`;

  const defaultHeaders: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = localStg.get('token');
  if (token) {
    defaultHeaders['Authorization'] = `Bearer ${token}`;
  }

  if (import.meta.env.VITE_SERVICE_TOKEN) {
    defaultHeaders['X-API-TOKEN'] = import.meta.env.VITE_SERVICE_TOKEN;
  }

  try {
    const res = await fetch(fullUrl, {
      method: "POST",
      headers: {
        ...defaultHeaders,
        ...(headers || {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!res.ok || !res.body) {
        const text = await res.text();
        throw new Error(`Stream failed: ${res.status} ${text}`);
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      let index;
      // assume server send each json per line
      // split by newline
      while ((index = buffer.indexOf('\n')) >= 0) {
        const line = buffer.slice(0, index).trim();
        buffer = buffer.slice(index + 1);
        if (!line) continue;
        onMessage(line);
      }
    }
    if (buffer.trim()) {
      onMessage(buffer.trim());
    }
  } catch (err) {
    console.error("streamPost error:", err);
    onError?.(err);
  }
}
