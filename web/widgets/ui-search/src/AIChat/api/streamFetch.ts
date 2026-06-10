import { useChatStore } from "../stores/chatStore";

export async function streamPost({
  url,
  body,
  queryParams,
  headers,
  onMessage,
  onError,
  signal,
}: {
  url: string;
  body: unknown;
  queryParams?: Record<
    string,
    string | number | boolean | null | undefined
  >;
  headers?: Record<string, string>;
  onMessage: (chunk: string) => void;
  onError?: (err: unknown) => void;
  signal?: AbortSignal;
}) {
  const { baseUrl, authHeaders } = useChatStore.getState();

  let baseURL = baseUrl;
  if (!baseURL || baseURL === "undefined") {
    baseURL = "";
  }

  const queryInit: Record<string, string> = {};
  if (queryParams) {
    Object.entries(queryParams).forEach(([key, value]) => {
      if (value === undefined || value === null) return;
      queryInit[key] = String(value);
    });
  }
  const query = new URLSearchParams(queryInit).toString();
  const fullUrl = query ? `${baseURL}${url}?${query}` : `${baseURL}${url}`;

  try {
    const res = await fetch(fullUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...authHeaders,
        ...(headers || {}),
      },
      credentials: "include",
      body: JSON.stringify(body),
      signal,
    });

    if (!res.ok || !res.body) throw new Error("Stream failed");

    const reader = res.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      const lines = buffer.split("\n");
      for (let i = 0; i < lines.length - 1; i++) {
        const line = lines[i].trim();
        if (line) onMessage(line);
      }
      buffer = lines[lines.length - 1];
    }

    // Process any remaining data in the buffer after stream ends
    if (buffer.trim()) {
      onMessage(buffer.trim());
    }
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      // 流被主动中断，不视为错误
      return;
    }
    console.error("streamPost error:", err);
    onError?.(err);
  }
}
