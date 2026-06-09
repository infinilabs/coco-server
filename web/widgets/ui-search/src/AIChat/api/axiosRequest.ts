import axios, { type AxiosRequestConfig } from "axios";

axios.defaults.withCredentials = true;

import {
  handleChangeRequestHeader,
  handleConfigureAuth,
  // handleAuthError,
  // handleGeneralError,
  handleNetworkError,
} from "./tools";

type Fn = (data: unknown) => unknown;

type RequestParams = Record<string, unknown>;
type RequestHeaders = Record<string, string>;

axios.interceptors.request.use((config) => {
  config = handleChangeRequestHeader(config);
  config = handleConfigureAuth(config);
  // console.log("config", config);
  return config;
});

axios.interceptors.response.use(
  (response) => {
    if (response.status !== 200) return Promise.reject(response.data);
    return response;
  },
  (err: { response?: { status?: number } } | undefined) => {
    handleNetworkError(err?.response?.status);
    return Promise.reject(err);
  }
);

type AxiosErrorLike = {
  response?: {
    status?: number;
    data?: { message?: string };
    config?: { url?: string };
  };
  request?: { config?: { url?: string } };
  config?: { url?: string };
  message?: string;
};

export const handleApiError = (error: unknown) => {
  let message = "Request failed";

  const err = error as AxiosErrorLike;

  if (err.response) {
    message =
      err.response.data?.message || `Error (${err.response.status})`;
  } else if (err.request) {
    message = "Network connection failed";
  } else {
    message = err.message || message;
  }

  const url =
    err.config?.url ||
    err.response?.config?.url ||
    err.request?.config?.url;

  const suppressProfileError =
    typeof url === "string" && url.includes("/account/profile");

  console.error(error);
  if (!suppressProfileError) {
    console.error(message);
  }

  return error as AxiosErrorLike;
};

export const Get = <T>(
  url: string,
  params: RequestParams = {},
  clearFn?: Fn,
  headers: RequestHeaders = {}
): Promise<[unknown, T | undefined]> =>
  new Promise((resolve) => {
    const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");

    const meta = import.meta as unknown as { env?: { DEV?: boolean } };
    const isDev = meta.env?.DEV === true;
    const PROXY_PREFIXES: readonly string[] = [
      "account",
      "chat",
      "query",
      "connector",
      "integration",
      "assistant",
      "datasource",
      "settings",
      "mcp_server",
    ];
    const shouldProxy =
      isDev &&
      url.startsWith("/") &&
      PROXY_PREFIXES.some((p) => url.startsWith(`/${p}`));

    let baseURL: string = appStore.state?.endpoint_http as string;
    if (!baseURL || baseURL === "undefined" || shouldProxy) {
      baseURL = "";
    }

    axios
      .get<T>(baseURL + url, { params, headers, withCredentials: true })
      .then((result) => {
        let res: T;
        if (clearFn !== undefined) {
          res = clearFn(result?.data) as unknown as T;
        } else {
          res = result?.data as T;
        }

        resolve([null, res]);
      })
      .catch((err: unknown) => {
        handleApiError(err);
        resolve([err, undefined]);
      });
  });

export const Post = <T>(
  url: string,
  data: RequestParams | undefined,
  params: RequestParams = {},
  headers: RequestHeaders = {}
): Promise<[unknown, T | undefined]> => {
  return new Promise((resolve) => {
    const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");

    const meta = import.meta as unknown as { env?: { DEV?: boolean } };
    const isDev = meta.env?.DEV === true;
    const PROXY_PREFIXES: readonly string[] = [
      "account",
      "chat",
      "query",
      "connector",
      "integration",
      "assistant",
      "datasource",
      "settings",
      "mcp_server",
    ];
    const shouldProxy =
      isDev &&
      url.startsWith("/") &&
      PROXY_PREFIXES.some((p) => url.startsWith(`/${p}`));

    let baseURL: string = appStore.state?.endpoint_http as string;
    if (!baseURL || baseURL === "undefined" || shouldProxy) {
      baseURL = "";
    }

    const config: AxiosRequestConfig = {
      params,
      headers,
      withCredentials: true,
    };

    axios
      .post<T>(baseURL + url, data, config)
      .then((result) => {
        resolve([null, result.data as T]);
      })
      .catch((err: unknown) => {
        handleApiError(err);
        resolve([err, undefined]);
      });
  });
};

export const Put = <T>(
  url: string,
  data: RequestParams | undefined,
  params: RequestParams = {},
  headers: RequestHeaders = {}
): Promise<[unknown, T | undefined]> => {
  return new Promise((resolve) => {
    const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");

    const meta = import.meta as unknown as { env?: { DEV?: boolean } };
    const isDev = meta.env?.DEV === true;
    const PROXY_PREFIXES: readonly string[] = [
      "account",
      "chat",
      "query",
      "connector",
      "integration",
      "assistant",
      "datasource",
      "settings",
      "mcp_server",
    ];
    const shouldProxy =
      isDev &&
      url.startsWith("/") &&
      PROXY_PREFIXES.some((p) => url.startsWith(`/${p}`));

    let baseURL: string = appStore.state?.endpoint_http as string;
    if (!baseURL || baseURL === "undefined" || shouldProxy) {
      baseURL = "";
    }

    const config: AxiosRequestConfig = {
      params,
      headers,
      withCredentials: true,
    };

    axios
      .put<T>(baseURL + url, data, config)
      .then((result) => {
        resolve([null, result.data as T]);
      })
      .catch((err: unknown) => {
        handleApiError(err);
        resolve([err, undefined]);
      });
  });
};

/**
 * Upload one or more files via multipart/form-data.
 * Files are appended under the form field `files` (the backend expects this name).
 *
 * @param url     - target endpoint, e.g. "/attachment/_upload"
 * @param files   - one or more File objects to upload
 * @param extra   - optional extra form fields to send alongside the files
 * @param headers - optional request headers
 * @param onProgress - optional upload progress callback (0..1)
 */
export const Upload = <T>(
  url: string,
  files: File[],
  extra: Record<string, string> = {},
  headers: RequestHeaders = {},
  onProgress?: (percent: number) => void
): Promise<[unknown, T | undefined]> => {
  return new Promise((resolve) => {
    const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");

    const meta = import.meta as unknown as { env?: { DEV?: boolean } };
    const isDev = meta.env?.DEV === true;
    const PROXY_PREFIXES: readonly string[] = [
      "account",
      "chat",
      "query",
      "connector",
      "integration",
      "assistant",
      "datasource",
      "settings",
      "mcp_server",
      "attachment",
    ];
    const shouldProxy =
      isDev &&
      url.startsWith("/") &&
      PROXY_PREFIXES.some((p) => url.startsWith(`/${p}`));

    let baseURL: string = appStore.state?.endpoint_http as string;
    if (!baseURL || baseURL === "undefined" || shouldProxy) {
      baseURL = "";
    }

    const formData = new FormData();
    for (const f of files) {
      formData.append("files", f, f.name);
    }
    for (const [k, v] of Object.entries(extra)) {
      formData.append(k, v);
    }

    const config: AxiosRequestConfig = {
      headers: {
        // Let axios/browser set the multipart boundary automatically.
        ...headers,
      },
      withCredentials: true,
      onUploadProgress: onProgress
        ? (evt) => {
            if (evt.total) {
              onProgress(evt.loaded / evt.total);
            }
          }
        : undefined,
    };

    axios
      .post<T>(baseURL + url, formData, config)
      .then((result) => {
        resolve([null, result.data as T]);
      })
      .catch((err: unknown) => {
        handleApiError(err);
        resolve([err, undefined]);
      });
  });
};

export const Delete = <T>(
  url: string,
  params: RequestParams = {},
  headers: RequestHeaders = {}
): Promise<[unknown, T | undefined]> => {
  return new Promise((resolve) => {
    const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");

    const meta = import.meta as unknown as { env?: { DEV?: boolean } };
    const isDev = meta.env?.DEV === true;
    const PROXY_PREFIXES: readonly string[] = [
      "account",
      "chat",
      "query",
      "connector",
      "integration",
      "assistant",
      "datasource",
      "settings",
      "mcp_server",
    ];
    const shouldProxy =
      isDev &&
      url.startsWith("/") &&
      PROXY_PREFIXES.some((p) => url.startsWith(`/${p}`));

    let baseURL: string = appStore.state?.endpoint_http as string;
    if (!baseURL || baseURL === "undefined" || shouldProxy) {
      baseURL = "";
    }

    const config: AxiosRequestConfig = {
      params,
      headers,
      withCredentials: true,
    };

    axios
      .delete<T>(baseURL + url, config)
      .then((result) => {
        resolve([null, result.data as T]);
      })
      .catch((err: unknown) => {
        handleApiError(err);
        resolve([err, undefined]);
      });
  });
};
