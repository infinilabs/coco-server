import { Spin } from 'antd';

import UserAvatar from '@/layouts/modules/global-header/components/UserAvatar';
import { localStg } from '@/utils/storage';
import { getDarkMode } from '@/store/slice/theme';
import { configResponsive } from 'ahooks';
import { fetchIntegration } from '@/service/api/integration';
import { useRequest } from '@sa/hooks';
import useQueryParams from '@/hooks/common/queryParams';
import { FullscreenPage } from 'ui-search';
import { assistantAsk, querySearch, fetchSuggestions, fetchRecommends } from '@/service/api/ai-search';
import { getApiBaseUrl } from '@/service/request';
import queryString from 'query-string';

configResponsive({ sm: 640 });

const AGGS_DEFAULT = {
  "aggs": {
    "category": { "terms": { "field": "category" } },
    "source.id": {
      "terms": {
        "field": "source.id"
      },
      "aggs": {
        "top": {
          "top_hits": {
            "size": 1,
            "_source": ["source.name"]
          }
        }
      }
    },
    "type": { "terms": { "field": "type" } },
    "tag": { "terms": { "field": "tags" } },
  }
}

const AGGS_IMAGE = {
  "aggs": {
    "category": { "terms": { "field": "category" } },
    "source.id": {
      "terms": {
        "field": "source.id"
      },
      "aggs": {
        "top": {
          "top_hits": {
            "size": 1,
            "_source": ["source.name"]
          }
        }
      }
    },
    "type": { "terms": { "field": "type" } },
    "tag": { "terms": { "field": "tags" } },
    "color": { "terms": { "field": "metadata.colors" } },
  }
}

const AGGS: any = {
  'all': AGGS_DEFAULT,
  'image': AGGS_IMAGE
}

export function Component() {
  const containerRef = useRef(null)

  const responsive = useResponsive();

  const [queryParams, setQueryParams] = useQueryParams();

  const darkMode = useAppSelector(getDarkMode);

  const providerInfo = localStg.get('providerInfo') || {}

  const { search_settings } = providerInfo;

  const isMobile = !responsive.sm;

  const { data, loading, run } = useRequest(fetchIntegration, {
    manual: true
  });

  const onSearch = async (query: { [key: string]: any }, callback: (data: any) => void, setLoading: (loading: boolean) => void, shouldAgg: boolean) => {
    if (setLoading) setLoading(true)
    const { filter = {}, ...rest } = query
    const filterStr = Object.keys(filter).map((key) => `filter=${key}:any(${filter[key].join(',')})`).join('&')
    const searchStr = `${filterStr ? filterStr + '&' : ''}v2=true&${queryString.stringify(rest)}`
    const body = shouldAgg ? JSON.stringify(AGGS[query['metadata.content_category']] || AGGS['all']) : undefined
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await querySearch(body, searchStr, { headers })
    if (callback) callback(res.data)
    if (setLoading) setLoading(false)
  }

  async function onAsk(assistantID: string, message: any, callback: (data: any) => void, setLoading: (loading: boolean) => void) {
    setLoading(true)
    const baseUrl = getApiBaseUrl();
    const body = JSON.stringify({
      message: JSON.stringify(message),
    })
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration, 'content-type': 'text/plain' }
    if (import.meta.env.VITE_SERVICE_TOKEN) {
      headers['X-API-TOKEN'] = import.meta.env.VITE_SERVICE_TOKEN
    }
    try {
      const response = await fetch(`${baseUrl}/assistant/${assistantID}/_ask`, {
        headers: headers,
        method: 'POST',
        credentials: 'include',
        body
      });

      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      if (!response.body) {
        throw new Error(`response body is null!`);
      }
      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let lineBuffer = '';

      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          setLoading(false)
          break;
        }

        const chunk = decoder.decode(value, { stream: true });

        lineBuffer += chunk;

        const lines = lineBuffer.split('\n');
        for (let i = 0; i < lines.length - 1; i++) {
          try {
            const json = JSON.parse(lines[i]);
            if (json && !(json._id && json._source && json.result)) {
              callback(json)
            }
          } catch (error) {
            console.log("error:", lines[i])
          }
        }

        lineBuffer = lines[lines.length - 1];
      }
    } catch (error) {
      setLoading(false)
      console.error('error:', error);
    }
  }

  async function onSuggestion(tag: string | undefined, params: { [key: string]: any }, callback: (data: any) => void) {
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await fetchSuggestions(tag, params, { headers })
    if (callback) callback(res.data)
  }

  async function onRecommend(tag: string | undefined, callback: (data: any) => void) {
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await fetchRecommends(tag, { headers })
    if (callback) callback(res.data)
  }

  useEffect(() => {
    if (search_settings?.integration) {
      run(search_settings?.integration);
    }
  }, [search_settings?.integration]);

  const { payload = {}, enabled_module = {} } = data?._source || {}

  const componentProps = {
    settings: data?._source,
    id: search_settings?.integration,
    theme: darkMode ? 'dark' : 'light',
    language: data?._source?.appearance?.language || 'zh-CN',
    "logo": {
      "light": payload?.logo?.light,
      "light_mobile": payload?.logo?.light_mobile,
      "dark": payload?.logo?.dark,
      "dark_mobile": payload?.logo?.dark_mobile,
    },
    "placeholder": enabled_module?.search?.placeholder,
    "welcome": payload?.welcome || "",
    "aiOverview": {
      ...(payload?.ai_overview || {}),
      "showActions": true,
    },
    "widgets": payload.ai_widgets?.enabled && payload.ai_widgets?.widgets ? payload.ai_widgets?.widgets.map((item) => ({
      ...item,
      "showActions": false,
    })) : [],
    "onSearch": onSearch,
    "onAsk": onAsk,
    "onSuggestion": onSuggestion,
    "onRecommend": onRecommend,
    "config": {
      "aggregations": {
        "source.id": {
          "displayName": "source",
        },
        "lang": {
          "displayName": "language"
        },
        "color": {
          'type': 'color'
        },
        "tag": {
          'type': 'tag'
        }
      }
    },
    getRawContent: (item: any) => {
      if (item.id && item.title) {
        return `${getEndpoint()}${getApiBaseUrl()}/document/${item.id}/raw_content/${item.title}`
      }
      return ''
    }
  }

  return (
    <Spin spinning={loading}>
      <div ref={containerRef}>
        <FullscreenPage
          {...componentProps}
          enableQueryParams={true}
          queryParams={queryParams}
          setQueryParams={setQueryParams}
          onLogoClick={() => {
            const hashWithoutParams = window.location.hash.split('?')[0] || '';
            const newUrl = window.location.origin + window.location.pathname + hashWithoutParams;
            history.replaceState(null, '', newUrl);
          }}
        />
      </div>
      <div className="absolute right-12px top-0px h-72px z-1 flex-y-center justify-end">
        {
          isMobile ? (
            <>
              <ThemeSchemaSwitch className="px-12px" />
              <UserAvatar className="px-8px" showHome showName={!isMobile} />
            </>
          ) : (
            <>
              <ThemeSchemaSwitch className="px-12px" />
              <UserAvatar className="px-8px" showHome showName={!isMobile} />
            </>
          )
        }
      </div>
    </Spin>
  );
}
