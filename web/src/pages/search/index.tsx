import { Spin } from 'antd';

import UserAvatar from '@/layouts/modules/global-header/components/UserAvatar';
import { localStg } from '@/utils/storage';
import { getDarkMode } from '@/store/slice/theme';
import { configResponsive } from 'ahooks';
import { fetchIntegration } from '@/service/api/integration';
import { useRequest } from '@sa/hooks';
import useQueryParams from '@/hooks/common/queryParams';
import { FullscreenPage } from 'ui-search';
import { querySearch, fetchSuggestions, fetchRecommends, fetchFieldsMeta } from '@/service/api/ai-search';
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

  const [queryParams, setQueryParams] = useQueryParams({ mode: 'search' });

  const darkMode = useAppSelector(getDarkMode);

  const providerInfo = localStg.get('providerInfo') || {}

  const { search_settings } = providerInfo;

  const isMobile = !responsive.sm;

  const { data, loading, run } = useRequest(fetchIntegration, {
    manual: true
  });

  const onSearch = async (queryParams: { [key: string]: any }, callback: (data: any) => void, setLoading: (loading: boolean) => void) => {
    if (setLoading) setLoading(true)
    const { filter = {}, ...rest } = queryParams
    const filterStr = Object.keys(filter).filter((key) => !!filter[key]).map((key) => `filter=${key}:any(${filter[key].join(',')})`).join('&')
    const searchStr = `${filterStr ? filterStr + '&' : ''}${queryString.stringify(rest)}`
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await querySearch({}, searchStr, { headers })
    if (callback) callback(res.data)
    if (setLoading) setLoading(false)
  }

  const onAggregation = async (queryParams: { [key: string]: any }, callback: (data: any) => void, setLoading: (loading: boolean) => void) => {
    if (setLoading) setLoading(true)
    const { query, filter } = queryParams
    const filterStr = Object.keys(filter).filter((key) => !!filter[key]).map((key) => `filter=${key}:any(${filter[key].join(',')})`).join('&')
    const searchStr = `${filterStr ? filterStr + '&' : ''}${queryString.stringify({ query })}`
    const body = JSON.stringify(AGGS[queryParams['metadata.content_category']] || AGGS['all'])
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
              setLoading(false)
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

  async function getFieldsMeta(fields: string[], callback: (data: any) => void) {
    if (!Array.isArray(fields) || fields.length === 0) {
      callback({})
      return;
    }
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await fetchFieldsMeta(fields, { headers })
    if (callback && res && !res.error) {
      callback(res.data)
    } else {
      callback({})
    }
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
    "onAggregation": onAggregation,
    "onAsk": onAsk,
    "onSuggestion": onSuggestion,
    "onRecommend": onRecommend,
    "config": {
      "aggregations": {
        "source.id": {
          "label": "source",
          "payload": { field_name: 'source.id', field_data_type: 'keyword', support_multi_select: true }
        },
        "lang": {
          "label": "language",
          "payload": { field_name: 'lang', field_data_type: 'keyword', support_multi_select: true }
        },
        "color": {
          'label': 'color',
          'type': 'color',
          "payload": { field_name: 'color', field_data_type: 'keyword', support_multi_select: true }
        },
        "tag": {
          'label': 'tag',
          'type': 'tag',
          "payload": { field_name: 'tag', field_data_type: 'keyword', support_multi_select: true }
        },
        "category": {
          'label': 'category',
          "payload": { field_name: 'category', field_data_type: 'keyword', support_multi_select: true }
        },
        "type": {
          'label': 'type',
          "payload": { field_name: 'type', field_data_type: 'keyword', support_multi_select: true }
        },
      }
    },
    getRawContent: (item: any) => {
      if (item.id && item.title) {
        return `${getEndpoint()}${getApiBaseUrl()}/document/${item.id}/raw_content/${item.title}`
      }
      return ''
    },
    apiConfig: {
      BaseUrl: getApiBaseUrl(),
      Token: import.meta.env.VITE_SERVICE_TOKEN,
      endpoint: getEndpoint()
    },
    onLogoClick: () => {
      const hashWithoutParams = window.location.hash.split('?')[0] || '';
      const newUrl = window.location.origin + window.location.pathname + hashWithoutParams;
      history.replaceState(null, '', newUrl);
    },
    getFieldsMeta
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
      <div className="absolute right-12px top-0px h-72px z-1002 flex-y-center justify-end">
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
