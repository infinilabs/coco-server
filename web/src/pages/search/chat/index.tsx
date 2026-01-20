import { useState } from 'react';

import { getApiBaseUrl } from '@/service/request';
import { FullscreenPage } from 'ui-search';

function useSimpleQueryParams(defaultParams = {}) {
  const [params, setParams] = useState({
    from: 0,
    size: 10,
    sort: [],
    filter: {},
    ...defaultParams
  });

  return [params, setParams];
}

export function Component() {
  const [queryParams, setQueryParams] = useSimpleQueryParams();
  const [queryParamsState, setQueryParamsState] = useState({
    from: 0,
    size: 10
  });

  const enableQueryParams = true;

  // 构建 componentProps，参考 Fullscreen.jsx 的结构
  const componentProps = {
    id: 'dev-ui-search',
    shadow: null,
    theme: 'light',
    language: 'zh-CN',
    logo: {
      // light: "/favicon.ico",
      // "light_mobile": "/favicon.ico",
    },
    placeholder: '搜索任何内容...',
    welcome:
      '欢迎使用 UI Search 开发环境！您可以在这里测试搜索功能和 AI 助手。',
    aiOverview: {
      enabled: true,
      showActions: true,
      assistant: 'dev-assistant',
      title: 'AI 概览',
      height: 400
    },
    widgets: [],
    config: {
      aggregations: {
        'source.id': {
          displayName: 'source'
        },
        lang: {
          displayName: 'language'
        },
        category: {
          displayName: 'category'
        },
        type: {
          displayName: 'type'
        }
      }
    },
    apiConfig: {
      BaseUrl: getApiBaseUrl(),
      Token: import.meta.env.VITE_SERVICE_TOKEN
    }
  };

  const queryParamsProps = enableQueryParams
    ? {
        queryParams,
        setQueryParams
      }
    : {
        queryParams: queryParamsState,
        setQueryParams: setQueryParamsState
      };

  return (
    <FullscreenPage
      {...componentProps}
      {...queryParamsProps}
      enableQueryParams={enableQueryParams}
    />
  );
}
