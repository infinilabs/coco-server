import { Spin } from 'antd';
import dayjs from 'dayjs';
import { useRef, useState } from 'react';
import { fetchIntegration } from '@/service/api/integration';
import useQueryParams from '@/hooks/common/queryParams';
import { FullscreenPage, OWNER_FILTER_FIELD, TIME_FILTER_FIELD } from 'ui-search/source';
import { querySearch, fetchSuggestions, fetchRecommends, fetchFieldsMeta, uploadAttachment } from '@/service/api/ai-search';
import { getApiBaseUrl } from '@/service/request';
import { consumePendingSearch } from '@/utils/search-handoff';
import queryString from 'query-string';
import { getDarkMode } from '@/store/slice/theme';
import { getLocale } from '@/store/slice/app';
import { getApplicationSetting } from '@/store/slice/server';
import { searchAssistant } from '@/service/api/assistant';
import { fetchBatchEntityLabels } from '@/service/api/entity';
import { CorrectionModal, type CorrectionPayload } from './components/CorrectionModal';
import { SaveToWikiModal, type SaveToWikiPayload } from './components/SaveToWikiModal';

// creator facet: buckets carry user ids — display names are resolved client-side
// via getUserEntities (same as the result list's owner names). The agg is named
// after the filter field so facet clicks produce `filter=_system.owner_id:any()`;
// it aggregates on the .keyword subfield because the engine's dynamic mapping
// types the bare field as text, which rejects terms aggregations outright.
const OWNER_AGG = {
  "terms": { "field": "_system.owner_id.keyword", "size": 10 }
};

// relative time buckets on `updated`, keyed by the URL date_range param values
// the widget already understands ('7d'/'90d'/'1y') — selecting a bucket applies
// that param, so boundaries must match the widget's getDateRangeParams
const timeRangeBuckets = () => {
  const now = dayjs();
  return [
    { key: '7d', from: now.subtract(7, 'day').startOf('day').valueOf() },
    { key: '90d', from: now.subtract(90, 'day').startOf('day').valueOf() },
    { key: '1y', from: now.subtract(1, 'year').startOf('day').valueOf() }
  ];
};

const TIME_AGG = () => ({
  "date_range": {
    "field": "updated",
    // explicit epoch millis (not date math) keep the DSL engine-portable;
    // ranges nest (7d ⊂ 90d ⊂ 1y), so counts read like the toolbar presets
    "ranges": timeRangeBuckets()
  }
});

// built per aggregation request so the time buckets always anchor to now
const buildAggsDefault = () => ({
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
    [OWNER_FILTER_FIELD]: OWNER_AGG,
    [TIME_FILTER_FIELD]: TIME_AGG(),
    // drives the result tabs' visibility (image/doc only when such results
    // exist for the query) — the widget consumes it and hides empty tabs
    "content_category": { "terms": { "field": "metadata.content_category" } },
  }
});

const buildAggsImage = () => ({
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
    [OWNER_FILTER_FIELD]: OWNER_AGG,
    [TIME_FILTER_FIELD]: TIME_AGG(),
    "content_category": { "terms": { "field": "metadata.content_category" } },
  }
});

export function Component() {

  const [queryParams, setQueryParams] = useQueryParams({ mode: 'search' });

  // a search started in the app shell header (chat mode, with attachments)
  // parks its pending conversation in the one-shot handoff — consume it at
  // first render and seed the widget's chat mode with it
  const handoffRef = useRef<any>(undefined);
  if (handoffRef.current === undefined) {
    handoffRef.current = consumePendingSearch();
  }
  const initialChatParams = handoffRef.current || undefined;

  const darkMode = useAppSelector(getDarkMode);

  const locale = useAppSelector(getLocale);

  // the app shell header owns the top-right controls (lang/theme/avatar/console)
  const rightMenuWidth = 0;

  // knowledge execution: answer -> knowledge-base draft (D1 server-side)
  const [saveToWikiPayload, setSaveToWikiPayload] = useState<SaveToWikiPayload | null>(null);

  // correction loop (D7): answer -> governance proposal, human gate stays
  const [correctionPayload, setCorrectionPayload] = useState<CorrectionPayload | null>(null);

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
    const { query, filter = {}, search_type, fuzziness, start, end, attachments } = queryParams
    // the content_category aggregation must see the full category distribution
    // for the current query — strip the active tab's category filter, or the
    // image/doc counts would collapse to the open tab and empty the other tabs
    const { 'metadata.content_category': _activeCategory, ...categoryAgnosticFilter } = filter
    const filterStr = Object.keys(categoryAgnosticFilter).filter((key) => Boolean(categoryAgnosticFilter[key])).map((key) => `filter=${key}:any(${Array.isArray(categoryAgnosticFilter[key]) ? categoryAgnosticFilter[key].join(',') : categoryAgnosticFilter[key]})`).join('&')
    const dateFilterStr = [
      start ? `filter=updated>=${start}` : '',
      end ? `filter=updated<=${end}` : '',
    ].filter(Boolean).join('&')
    // multimodal attachments shape the facet counts like they shape the results
    const searchStr = [filterStr, dateFilterStr, queryString.stringify({ query, search_type, fuzziness, ...(attachments ? { attachments } : {}) })].filter(Boolean).join('&')
    const aggs = queryParams['metadata.content_category'] === 'image' ? buildAggsImage() : buildAggsDefault();
    const body = JSON.stringify(aggs)
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

  // search-home branding comes from the built-in integration (edited in
  // search settings): the banner logo above the search box and the page
  // background. When no banner logo is configured, pass null so the widget
  // hides its own logo instead of falling back to the bundled one
  const integrationLogo = payload?.logo
  const hasIntegrationLogo = Boolean(integrationLogo && Object.values(integrationLogo).some(Boolean))
  const searchLogo = hasIntegrationLogo
    ? {
        dark: integrationLogo?.dark || integrationLogo?.light,
        dark_mobile: integrationLogo?.dark || integrationLogo?.light,
        light: integrationLogo?.light || integrationLogo?.dark,
        light_mobile: integrationLogo?.light || integrationLogo?.dark
      }
    : null
  const searchBackground = payload?.background

  const componentProps = {
    settings: integration,
    id: search_settings?.integration,
    theme: darkMode ? 'dark' : 'light',
    language: locale,
    "logo": searchLogo,
    // the app shell already brands the chat page (sider "站点标题") — no
    // widget branding in the chat sidebar
    "chatLogo": null,
    // ditto for the search results header — the banner only belongs on the
    // search home
    "searchLogo": null,
    "background": searchBackground,
    "logoMaxHeight": payload?.banner_height,
    "welcomeFontSize": payload?.welcome_font_size,
    "welcomeGradient": payload?.welcome_gradient !== false,
    "placeholder": enabled_module?.search?.placeholder,
    "welcome": payload?.welcome || "",
    rightMenuWidth,
    "aiOverview": {
      ...(payload?.ai_overview || {}),
      "showActions": true,
    },
    "onSearch": onSearch,
    "onSaveToWiki": (payload: SaveToWikiPayload) => setSaveToWikiPayload(payload),
    "onCorrectAnswer": (payload: CorrectionPayload) => setCorrectionPayload(payload),
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
        initialChatParams={initialChatParams}
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
      <CorrectionModal payload={correctionPayload} onClose={() => setCorrectionPayload(null)} />
    </>
  );
}
