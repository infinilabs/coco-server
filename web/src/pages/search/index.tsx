import { Spin } from 'antd';
import { fetchIntegration } from '@/service/api/integration';
import useQueryParams from '@/hooks/common/queryParams';
import { FullscreenPage } from 'ui-search/source';
import { querySearch, fetchSuggestions, fetchRecommends, fetchFieldsMeta, uploadAttachment } from '@/service/api/ai-search';
import { getApiBaseUrl } from '@/service/request';
import queryString from 'query-string';
import { getDarkMode } from '@/store/slice/theme';
import { getLocale } from '@/store/slice/app';
import { getApplicationSetting } from '@/store/slice/server';
import { searchAssistant } from '@/service/api/assistant';
import { fetchBatchEntityLabels } from '@/service/api/entity';
import { SaveToWikiModal, type SaveToWikiPayload } from './components/SaveToWikiModal';

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
    "tags": { "terms": { "field": "tags" } },
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

  const [queryParams, setQueryParams] = useQueryParams({ mode: 'search' });

  const darkMode = useAppSelector(getDarkMode);

  const locale = useAppSelector(getLocale);

  // the app shell header owns the top-right controls (lang/theme/avatar/console)
  const rightMenuWidth = 0;

  // knowledge execution: answer -> knowledge-base draft (D1 server-side)
  const [saveToWikiPayload, setSaveToWikiPayload] = useState<SaveToWikiPayload | null>(null);

  const applicationSetting = useAppSelector(getApplicationSetting);

  const { search_settings } = applicationSetting || {};

  const [integration, setIntegration] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const getIntegrationSettings = async (integrationID: string) => {
    setLoading(true)
    const res = await fetchIntegration(integrationID);
    if (res?.data) {
      const { deep_research_assistant, deep_think_assistant } = res.data?._source || {}
      const integrationData = {
        ...(res.data._source || {}),
      }
      if (deep_research_assistant || deep_think_assistant) {
        const assistantRes = await searchAssistant({
          from: 0,
          size: 10000,
          filter: {
            id: [deep_research_assistant, deep_think_assistant].filter((id) => !!id)
          }
        }, {
          headers: { 'APP-INTEGRATION-ID': search_settings?.integration }
        });
        if (assistantRes?.data?.hits?.hits?.length) {
          assistantRes.data.hits.hits.forEach((item: any) => {
            if (item._id === deep_research_assistant) {
              integrationData.deep_research_assistant_entity = item._source
            }
            if (item._id === deep_think_assistant) {
              integrationData.deep_think_assistant_entity = item._source
            }
          })
        }
      }
      setIntegration(integrationData)
    }
    setLoading(false)
  }

  const onSearch = async (queryParams: { [key: string]: any }, body: any = {}, callback: (data: any) => void, setLoading: (loading: boolean) => void) => {
    if (setLoading) setLoading(true)
    const { filter = {}, start, end, ...rest } = queryParams
    const filterStr = Object.keys(filter).filter((key) => !!filter[key]).map((key) => `filter=${key}:any(${Array.isArray(filter[key]) ? filter[key].join(',') : filter[key]})`).join('&')
    const dateFilterStr = [
      start ? `filter=updated>=${start}` : '',
      end ? `filter=updated<=${end}` : '',
    ].filter(Boolean).join('&')
    const searchStr = [filterStr, dateFilterStr, queryString.stringify(rest)].filter(Boolean).join('&')
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await querySearch(body, searchStr, { headers, ignoreError: true })
    if (callback) callback(res.data)
    if (setLoading) setLoading(false)
  }

  const onAggregation = async (queryParams: { [key: string]: any }, callback: (data: any) => void, setLoading: (loading: boolean) => void) => {
    if (setLoading) setLoading(true)
    const { query, filter, search_type, fuzziness, start, end } = queryParams
    const filterStr = Object.keys(filter).filter((key) => !!filter[key]).map((key) => `filter=${key}:any(${Array.isArray(filter[key]) ? filter[key].join(',') : filter[key]})`).join('&')
    const dateFilterStr = [
      start ? `filter=updated>=${start}` : '',
      end ? `filter=updated<=${end}` : '',
    ].filter(Boolean).join('&')
    const searchStr = [filterStr, dateFilterStr, queryString.stringify({ query, search_type, fuzziness })].filter(Boolean).join('&')
    const body = JSON.stringify(AGGS[queryParams['metadata.content_category']] || AGGS['all'])
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await querySearch(body, searchStr, { headers, ignoreError: true })
    if (callback) callback(res.data)
    if (setLoading) setLoading(false)
  }

  async function onAsk(assistantID: string, message: any, callback: (data: any) => void, setLoading: (loading: boolean) => void) {
    setLoading(true)
    const baseUrl = getApiBaseUrl();
    const body = JSON.stringify({
      message: JSON.stringify(message),
    })
    const headers: Record<string, any> = { 'APP-INTEGRATION-ID': search_settings?.integration, 'content-type': 'text/plain' }
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
        setLoading(false)
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      if (!response.body) {
        setLoading(false)
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
    const res = await fetchSuggestions(tag, params, { headers, ignoreError: true })
    if (callback) callback(res.data)
  }

  async function getFieldsMeta(fields: string[], callback?: (data: any) => void) {
    if (!Array.isArray(fields) || fields.length === 0) {
      callback?.({})
      return;
    }
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await fetchFieldsMeta(fields, { headers, ignoreError: true })
    if (res && !res.error) {
      callback?.(res.data)
    } else {
      callback?.({})
    }
  }

  async function onRecommend(tag: string | undefined, callback: (data: any) => void) {
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await fetchRecommends(tag, { headers, ignoreError: true })
    if (callback) callback(res.data)
  }

  async function onUpload(files: any[], callback?: (data: any) => void) {
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const res = await uploadAttachment(files, { headers, ignoreError: true })
    if (res && !res.error) {
      callback?.(res.data)
    } else {
      callback?.({})
    }
  }

  async function getUserEntities(ids: string[], callback?: (data: any) => void) {
    const headers = { 'APP-INTEGRATION-ID': search_settings?.integration }
    const body = [{
      type: 'user',
      id: ids
    }]
    const res = await fetchBatchEntityLabels(body, { headers, ignoreError: true });
    if (res && !res.error) {
      callback?.(res.data)
    } else {
      callback?.({})
    }
  }

  useEffect(() => {
    if (search_settings?.integration) {
      getIntegrationSettings(search_settings?.integration);
    }
  }, [search_settings?.integration]);

  const { payload = {}, enabled_module = {} } = integration || {}

  const componentProps = {
    settings: integration,
    id: search_settings?.integration,
    theme: darkMode ? 'dark' : 'light',
    language: locale,
    // the app shell header carries the brand; null tells the widget to hide
    // its own banner logo instead of falling back to the bundled one
    "logo": null,
    "placeholder": enabled_module?.search?.placeholder,
    "welcome": payload?.welcome || "",
    rightMenuWidth,
    "aiOverview": {
      ...(payload?.ai_overview || {}),
      "showActions": true,
    },
    "onSearch": onSearch,
    "onSaveToWiki": (payload: SaveToWikiPayload) => setSaveToWikiPayload(payload),
    "onAggregation": onAggregation,
    "onAsk": onAsk,
    "onSuggestion": onSuggestion,
    "onRecommend": onRecommend,
    "config": {
      // labels are resolved by the widget's i18n (labels.source / labels.type / …);
      // only the widget-specific render type is configured here
      "aggregations": {
        "source.id": {
          "payload": { field_name: 'source.id', field_data_type: 'keyword', support_multi_select: true }
        },
        "lang": {
          "payload": { field_name: 'lang', field_data_type: 'keyword', support_multi_select: true }
        },
        "color": {
          'type': 'color',
          "payload": { field_name: 'color', field_data_type: 'keyword', support_multi_select: true }
        },
        "tags": {
          'type': 'tag',
          "payload": { field_name: 'tags', field_data_type: 'keyword', support_multi_select: true }
        },
        "category": {
          "payload": { field_name: 'category', field_data_type: 'keyword', support_multi_select: true }
        },
        "type": {
          "payload": { field_name: 'type', field_data_type: 'keyword', support_multi_select: true }
        },
      }
    },
    apiConfig: {
      BaseUrl: getApiBaseUrl(),
      Token: import.meta.env.VITE_SERVICE_TOKEN,
      endpoint: getEndpoint(),
      headers: {
        'APP-INTEGRATION-ID': search_settings?.integration,
      }
    },
    onLogoClick: () => {
      const hashWithoutParams = window.location.hash.split('?')[0] || '';
      const newUrl = window.location.origin + window.location.pathname + hashWithoutParams;
      history.replaceState(null, '', newUrl);
    },
    getFieldsMeta,
    onUpload,
    getUserEntities
  }

  if (loading) {
    return (
      <GlobalLoading spinning={loading} />
    )
  }

  if (!integration) return null;

  return (
    <>
      <FullscreenPage
        {...componentProps}
        enableQueryParams={true}
        queryParams={queryParams}
        setQueryParams={setQueryParams}
      />
      <SaveToWikiModal
        payload={saveToWikiPayload}
        onClose={() => setSaveToWikiPayload(null)}
        onSaved={(articleId, kbId) => {
          // jump straight into the draft for a human review pass (D1)
          setQueryParams({ ...queryParams, mode: 'search' });
          window.location.hash = `#/wiki/article/${articleId}?kb=${kbId}`;
        }}
      />
    </>
  );
}
