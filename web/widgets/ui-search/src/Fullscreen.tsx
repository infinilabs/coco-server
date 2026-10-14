import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import dayjs from "dayjs";
import { formatESResult } from "./utils/es";
import { normalizeCoverIconUrl } from "./utils/utils";

import { debounce, isEmpty } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH_KEYWORD, DEFAULT_SEARCH_SORT, normalizeSearchFuzziness, normalizeSearchSort } from "./SearchBox/ActionBar/SearchActions";
import Chat from "./pages/Chat";
import { calcFixedBucketCount } from "./utils/date";
import { pushRecentSearch } from "./utils/recentSearches";

const formatDateRangeParam = (value: number | string, endOfDay = false) => {
  const timestamp = typeof value === 'number' ? value : Number(value);
  const date = Number.isFinite(timestamp) ? dayjs(timestamp) : dayjs(value);

  if (!date.isValid()) return value;
  // Values carrying a time part (the histogram brush passes minute-precision
  // bucket keys) must keep their exact instant — snapping to day boundaries
  // silently widens every within-day selection to the whole day, making the
  // brush appear to do nothing. Day-boundary semantics stay for bare dates.
  const hasTime = typeof value === 'number' || /\d:\d/.test(String(value));
  if (hasTime) return date.valueOf();
  return endOfDay ? date.endOf('day').valueOf() : date.startOf('day').valueOf();
};

const getDateRangeParams = (dateRange?: string) => {
  const now = dayjs();

  if (dateRange === '7d') {
    return {
      start: now.subtract(7, 'day').startOf('day').valueOf(),
      end: now.endOf('day').valueOf(),
    };
  }

  if (dateRange === '90d') {
    return {
      start: now.subtract(90, 'day').startOf('day').valueOf(),
      end: now.endOf('day').valueOf(),
    };
  }

  if (dateRange === '1y') {
    return {
      start: now.subtract(1, 'year').startOf('day').valueOf(),
      end: now.endOf('day').valueOf(),
    };
  }

  return {};
};

interface FullscreenProps {
  /** pass null to hide the widget's own logo (host app already shows its brand) */
  logo?: Record<string, any> | null;
  /** overrides `logo` for the chat sidebar brand strip — hosts whose shell
   * already brands the chat page pass null to remove it entirely */
  chatLogo?: Record<string, any> | null;
  /** overrides `logo` for the search results header — hosts whose shell
   * already brands the results page pass null */
  searchLogo?: Record<string, any> | null;
  /** search-home background image per theme: { light, dark } */
  background?: Record<string, any>;
  /** banner slot max height in px (width stays adaptive) */
  logoMaxHeight?: number;
  /** welcome text font size in px (default 30) */
  welcomeFontSize?: number;
  /** brand-gradient welcome text (default on) */
  welcomeGradient?: boolean;
  placeholder?: string;
  welcome?: string;
  aiOverview?: { enabled?: boolean };
  /** seed the chat mode with a pending conversation (query / attachments /
   * assistant) — how a host's global search box hands off an AI-mode search
   * across the route boundary into this page */
  initialChatParams?: Record<string, any>;
  onSearch?: (...args: any[]) => void;
  onAggregation?: (...args: any[]) => void;
  onAsk?: (...args: any[]) => void;
  /** host hook: leave chat mode and go back to search — hosts whose shell
   * already offers search/chat navigation (e.g. a persistent header) omit it,
   * which hides the chat header's back-to-search button entirely */
  onBackToSearch?: () => void;
  /** host hook: report an assistant answer as wrong/outdated (correction loop) */
  onCorrectAnswer?: (payload: { content: string; question: string; id: string }) => void;
  config?: Record<string, any>;
  isHome?: boolean;
  rightMenuWidth?: number;
  queryParams?: Record<string, any>;
  setQueryParams?: (params: any) => void;
  onLogoClick?: () => void;
  theme?: 'light' | 'dark';
  language?: string;
  onSuggestion?: (...args: any[]) => void;
  onRecommend?: (...args: any[]) => void;
  apiConfig?: Record<string, any>;
  getFieldsMeta?: (...args: any[]) => any;
  onUpload?: (...args: any[]) => void;
  getUserEntities?: (...args: any[]) => void;
  settings?: Record<string, any>;
  [key: string]: any;
}

const Fullscreen = (props: FullscreenProps) => {
  const {
    logo = {},
    chatLogo,
    searchLogo,
    background,
    logoMaxHeight,
    welcomeFontSize,
    welcomeGradient,
    onSaveToWiki,
    onCorrectAnswer,
    placeholder,
    welcome,
    aiOverview,
    initialChatParams,
    onSearch,
    onAggregation,
    onAsk,
    onBackToSearch,
    config = {},
    isHome = false,
    rightMenuWidth,
    queryParams = {},
    setQueryParams,
    onLogoClick,
    theme = 'light',
    language = 'en-US',
    onSuggestion,
    onRecommend,
    apiConfig,
    getFieldsMeta,
    onUpload,
    getUserEntities,
    settings
  } = props;

  const containerRef = useRef<HTMLDivElement | null>(null);
  const getContainer = useCallback(() => containerRef.current, []);
  const [result, setResult] = useState(formatESResult());
  const [aggregationResult, setAggregationResult] = useState<ReturnType<typeof formatESResult>['aggregations']>([]);
  // per-category availability for the result tabs: image count comes from the
  // content_category aggregation, non-image from the aggregation query's total
  // minus that — null when the host doesn't request the aggregation (legacy
  // host: all tabs stay visible)
  const [categoryCounts, setCategoryCounts] = useState<{ image: number; nonImage: number } | null>(null);
  const [askBody, setAskBody] = useState<any>();
  // a mount with search params runs a search in the first effect — start in
  // loading so the very first paint already carries the list/facet skeletons
  // (otherwise one frame renders the empty layout at its no-sider width and
  // the columns settle 100ms+ later); attachments count too: an image-only
  // multimodal search has no typed query
  const willSearchOnMount =
    queryParams.mode !== 'chat' &&
    (Boolean(queryParams?.query) || !isEmpty(queryParams?.filter) || !isEmpty(queryParams?.aggfilter) || Boolean(queryParams?.attachments));
  const [loading, setLoading] = useState(willSearchOnMount);
  const [isMobile, setIsMobile] = useState(false);
  const shouldAskRef = useRef(true);
  const shouldAggRef = useRef(true);
  const [data, setData] = useState<any[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)

  const [chatParams, setChatParams] = useState<Record<string, any>>(initialChatParams || {});
  const [attachments, setAttachments] = useState<any[]>(initialChatParams?.attachments || []);

  const onChat = (params: Record<string, any>) => {
    setChatParams(params);
    setQueryParams?.({
      mode: 'chat',
    })
  }

  // Multimodal search handed in from another surface (e.g. the app-shell
  // header) arrives as `attachments=<ids>` in the URL. The chips live in
  // component state, so fetch their metadata once to keep the search box
  // honest about what is being searched; failures leave the search itself
  // untouched (the server enriches from the IDs regardless of chip state).
  useEffect(() => {
    const idsParam = queryParams.attachments;
    if (queryParams.mode === 'chat' || attachments.length > 0 || typeof idsParam !== 'string' || !idsParam) return;

    let cancelled = false;
    const baseUrl = String(apiConfig?.BaseUrl || '').replace(/\/+$/, '');
    const headers: Record<string, string> = { ...(apiConfig?.headers || {}) };
    if (apiConfig?.Token) headers['X-API-TOKEN'] = String(apiConfig.Token);

    fetch(`${baseUrl}/attachment/_search?from=0&size=50`, {
      method: 'POST',
      headers: { ...headers, 'content-type': 'application/json' },
      body: JSON.stringify({ attachments: idsParam.split(',').map((id: string) => id.trim()).filter(Boolean) }),
      credentials: 'include'
    })
      .then(res => (res.ok ? res.json() : null))
      .then(res => {
        if (cancelled || !res) return;
        const hits = res?.hits?.hits || [];
        setAttachments(hits.map((hit: any) => {
          const src = hit?._source || {};
          const name = src.name || hit?._id || '';
          return {
            id: hit?._id,
            filename: name,
            extname: name.includes('.') ? name.split('.').pop() : '',
            type: src.mime_type || '',
            size: src.size || 0,
            status: 'uploaded'
          };
        }));
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [queryParams.attachments, queryParams.mode, attachments.length, apiConfig]);

  const resetScroll = () => {
    scrollRef.current = 0;
    if (containerRef.current) {
      try {
        containerRef.current.scrollTo({
          top: 0,
          behavior: 'instant'
        });
      } catch {
        containerRef.current.scrollTop = 0;
      }
    }
  };

  const handleSearch = (params: Record<string, any>, shouldAsk: boolean, shouldAgg: boolean, isScroll = false) => {
    const fuzziness = normalizeSearchFuzziness(params?.fuzziness);
    const sort = normalizeSearchSort(params?.sort);
    shouldAskRef.current = shouldAsk;
    shouldAggRef.current = shouldAgg;
    if (!isScroll) {
      resetScroll();
      isHomeSearchRef.current = true;
    }
    const nextQueryParams: Record<string, any> = {
      ...params,
      fuzziness,
      sort,
      ...(shouldAgg ? { aggfilter: {} } : {}),
      t: new Date().valueOf()
    };
    // empty attachments (chips removed) must not linger in the URL as
    // attachments= and re-trigger multimodal enrichment server-side
    if (!nextQueryParams.attachments) {
      delete nextQueryParams.attachments;
    }
    delete nextQueryParams.dateRange;
    if (!nextQueryParams.date_range || nextQueryParams.date_range === 'all-time') {
      delete nextQueryParams.date_range;
    }
    if (!nextQueryParams.start) {
      delete nextQueryParams.start;
    }
    if (!nextQueryParams.end) {
      delete nextQueryParams.end;
    }
    setQueryParams?.({
      ...nextQueryParams,
    });
  };

  const handleLoadMore = useCallback(() => {
    if (loading || !hasMore || loadLock.current) return;
    loadLock.current = true;
    const { from, size } = queryParams;
    scrollRef.current = (scrollRef.current || from) + size;
    handleSearch(queryParams, false, false, true);
  }, [queryParams, loading, hasMore, handleSearch]);

  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkScreenSize();
    window.addEventListener('resize', checkScreenSize);
    return () => window.removeEventListener('resize', checkScreenSize);
  }, []);

  const handleCategoryChange = useCallback(() => {
    setData([]);
    setHasMore(false);
    setResult(formatESResult());
    setAggregationResult([]);
    setCategoryCounts(null);
  }, []);

  useEffect(() => {
    if (queryParams.mode === 'chat' || !queryParams?.query && isEmpty(queryParams?.filter) && isEmpty(queryParams?.aggfilter) && !queryParams.attachments) {
      // nothing to search — release any mount-time loading so a skeleton
      // never lingers on a static view
      setLoading(false);
      return;
    }

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    // record the executed query for the recent-searches dropdown; scrolling
    // deeper re-enters with the same query, and re-recording is a no-op
    // anyway thanks to move-to-front dedupe, but skipping keeps intent clear
    if (!isScroll && (queryParams.query || '').trim()) {
      pushRecentSearch(queryParams.query);
    }

    loadLock.current = true;
    setLoading(true);

    const { t, date_range, start, end, filter = {}, aggfilter = {}, ...rest } = queryParams;
    const fuzziness = normalizeSearchFuzziness(queryParams?.fuzziness);
    const sort = normalizeSearchSort(queryParams?.sort);
    const dateRangeParams = start && end ? { start: formatDateRangeParam(start), end: formatDateRangeParam(end, true) } : getDateRangeParams(date_range);
    const filterWithoutAgg = {
      ...filter,
      'metadata.content_category': queryParams['metadata.content_category'] && queryParams['metadata.content_category'] !== 'all' ? [queryParams['metadata.content_category']] : undefined,
    }

    // highlight only marks something when there is a query term; pure-filter
    // browsing sends none, and the server's empty text clauses would light
    // whole fragments up instead
    const hasQuery = Boolean((queryParams.query || '').trim());

    const doSearch = (validatedAggfilter: Record<string, any>) => {
      const newFilter = { ...filterWithoutAgg };
      Object.keys(validatedAggfilter).forEach(key => {
        if (newFilter[key] !== undefined && validatedAggfilter[key] !== undefined) {
          const filterVal = Array.isArray(newFilter[key]) ? newFilter[key] : [newFilter[key]];
          const aggVal = Array.isArray(validatedAggfilter[key]) ? validatedAggfilter[key] : [validatedAggfilter[key]];
          newFilter[key] = [...new Set([...filterVal, ...aggVal])];
        } else if (validatedAggfilter[key] !== undefined) {
          newFilter[key] = validatedAggfilter[key];
        }
      });
      onSearch?.(
        {
          ...rest,
          ...dateRangeParams,
          filter: newFilter,
          search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
          fuzziness,
          sort,
          from: isScroll ? scrollRef.current : queryParams.from,
          'metadata.content_category': undefined
        },
        {
          "aggs": {
            "counts": {
              "auto_date_histogram": {
                "field": "updated",
                "buckets": calcFixedBucketCount(dateRangeParams.start as number, dateRangeParams.end as number),
                "time_zone": "Asia/Shanghai"
              }
            }
          },
          // mark matched terms so the result list can highlight the query —
          // fragments come back as plain text with literal <em> wrappers,
          // rendered by splitting the string (never as raw HTML)
          ...(hasQuery ? {
            "highlight": {
              "pre_tags": ["<em>"],
              "post_tags": ["</em>"],
              "require_field_match": false,
              "fields": {
                "title": { "number_of_fragments": 0 },
                "summary": { "fragment_size": 200, "number_of_fragments": 1 },
                "content": { "fragment_size": 200, "number_of_fragments": 1 }
              }
            }
          } : {})
        },
        (res: any) => {
          loadLock.current = false;
          setLoading(false);

          let rs: any;
          if (res && !res.error) {
            res = normalizeCoverIconUrl(res, apiConfig?.BaseUrl);
            rs = formatESResult(res);
            setResult(rs);

            const newData = isScroll ? [...data, ...(rs.hits?.hits || [])] : rs.hits?.hits || [];
            setData(newData);
            setHasMore(newData.length < (rs.hits.total || 0));
            if (!isScroll) isHomeSearchRef.current = false;
          } else {
            if (!isScroll) {
              setResult(formatESResult());
              setData([]);
            }
            setHasMore(false);
            isHomeSearchRef.current = false;
          }

          if (shouldAskRef.current) {
            shouldAskRef.current = false;
            setAskBody({
              message: rs?.hits?.hits?.length > 0 ? JSON.stringify({
                query: queryParams.query,
                result: rs?.hits
              }) : '',
              t: new Date().valueOf()
            });
          }
        },
        (loadingState: boolean) => {
          setLoading(loadingState);
        }
      );
    };

    if (onAggregation && shouldAggRef.current) {
      shouldAggRef.current = false;
      // Fetch aggregations first, validate aggfilter, then search
      onAggregation({
        query: queryParams.query,
        search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
        fuzziness,
        // multimodal attachments must shape the facet counts the same way
        // they shape the result list
        ...(queryParams.attachments ? { attachments: queryParams.attachments } : {}),
        ...dateRangeParams,
        filter: filterWithoutAgg
      }, (res: any) => {
        let validatedAggfilter: Record<string, any> = {};
        if (res && !res.error) {
          const rs = formatESResult(res);
          // the content_category aggregation feeds the tab visibility, not the
          // facet rail — strip it from the rail list
          const contentCategoryAgg = rs.aggregations.find((agg: any) => agg?.key === 'content_category');
          if (contentCategoryAgg) {
            const imageCount = Number(
              (contentCategoryAgg.list || []).find((item: any) => String(item?.key) === 'image')?.count || 0
            );
            setCategoryCounts({ image: imageCount, nonImage: Math.max(0, Number(rs.hits?.total || 0) - imageCount) });
          } else {
            setCategoryCounts(null);
          }
          setAggregationResult(rs.aggregations.filter((agg: any) => agg?.key !== 'content_category'));
          // Validate aggfilter values against actual aggregation results
          if (!isEmpty(aggfilter)) {
            const aggKeys = new Map<string, Set<string>>();
            (rs.aggregations || []).forEach((agg: any) => {
              const values = new Set<string>();
              (agg.list || []).forEach((item: any) => values.add(item.key));
              aggKeys.set(agg.key, values);
            });
            Object.keys(aggfilter).forEach(key => {
              const validValues = aggKeys.get(key);
              if (validValues) {
                const vals = Array.isArray(aggfilter[key]) ? aggfilter[key] : [aggfilter[key]];
                const filtered = vals.filter((v: string) => validValues.has(v));
                if (filtered.length > 0) {
                  validatedAggfilter[key] = filtered;
                }
              } else {
                // no aggregation behind this field (e.g. a filter carried in
                // from an external view) — nothing to validate against, keep as-is
                validatedAggfilter[key] = aggfilter[key];
              }
            });
            // If aggfilter changed after validation, update URL and re-trigger
            if (JSON.stringify(validatedAggfilter) !== JSON.stringify(aggfilter)) {
              setQueryParams?.({ ...queryParams, aggfilter: validatedAggfilter, t: new Date().valueOf() });
              setLoading(false);
              return;
            }
          }
        } else {
          setAggregationResult([]);
          setCategoryCounts(null);
        }
        doSearch(validatedAggfilter);
      });
    } else {
      // No agg needed, search directly with aggfilter as-is
      doSearch(aggfilter);
    }
  }, [JSON.stringify(queryParams)]);

  useEffect(() => {
    (window as any).onsearch = (query: string) => handleSearch({ ...queryParams, from: 0, query }, true, true);
    return () => {
      (window as any).onsearch = undefined;
    };
  }, [queryParams]);

  const debouncedSuggestion = useMemo(() => {
    if (typeof onSuggestion === 'function') {
      return debounce(onSuggestion, 500);
    }
    return () => { };
  }, [onSuggestion]);

  const { query, filter, aggfilter, filters = [] } = queryParams;

  const commonProps = { isMobile, theme, apiConfig, language, getUserEntities };
  const { hits } = result;

  const handleLogoClick = () => {
    setQueryParams?.({
      from: 0,
      size: 10,
      query: '',
      filter: {},
      aggfilter: {},
      sort: DEFAULT_SEARCH_SORT
    });
    setData([]);
    setHasMore(false);
    setAggregationResult([]);
    setCategoryCounts(null);
    resetScroll();
    isHomeSearchRef.current = true;
    if (onLogoClick) onLogoClick();
  };

  const showFullScreenSpin = loading && isHomeSearchRef.current;

  const histogramData = useMemo(() => {
    const countsAgg = result?.aggregations?.find((agg: any) => agg?.key === 'counts');
    if (!countsAgg) return undefined;

    const points = (countsAgg.list || []).map((item: any) => ({
      date: dayjs(item.key).format('YYYY-MM-DD HH:mm:ss'),
      count: Number.isInteger(item.count) ? item.count : 0,
    }));

    return points;
  }, [result?.aggregations]);

  const { mode = 'search' } = queryParams

  if (mode === 'chat') {
    return (
      <Chat
        commonProps={commonProps}
        logo={chatLogo === undefined ? logo : chatLogo}
        handleLogoClick={handleLogoClick}
        onSaveToWiki={onSaveToWiki}
        onCorrectAnswer={onCorrectAnswer}
        apiConfig={apiConfig}
        queryParams={queryParams}
        onBackToSearch={onBackToSearch}
        setQueryParams={setQueryParams}
        defaultParams={chatParams}
        setDefaultParams={setChatParams}
        setAttachments={setAttachments}
        initContainer={(ref: HTMLDivElement | null) => {
          containerRef.current = ref;
        }}
        getContainer={getContainer}
        rightMenuWidth={rightMenuWidth}
      />
    )
  }

  if (isHome) {
    return (
      <Home
        commonProps={commonProps}
        loading={showFullScreenSpin}
        logo={logo}
        background={background}
        logoMaxHeight={logoMaxHeight}
        welcomeFontSize={welcomeFontSize}
        welcomeGradient={welcomeGradient}
        settings={settings}
        onSearch={(params: Record<string, any>, shouldAsk: boolean, shouldAgg: boolean) => {
          if (params.mode === 'chat') {
            let assistant_id = params.assistant_id;
            if (!assistant_id) {
              if (params.action === 'deepthink') {
                assistant_id = settings?.deep_think_assistant;
              } else if (params.action === 'deepresearch') {
                assistant_id = settings?.deep_research_assistant;
              }
            }
            onChat({
              query: params.query || '',
              attachments: params.attachments || attachments || [],
              assistant_id,
            });
            return;
          };
          handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)
        }}
        placeholder={placeholder}
        welcome={welcome}
        queryParams={queryParams}
        setQueryParams={setQueryParams}
        onSuggestion={debouncedSuggestion}
        onRecommend={onRecommend}
        onUpload={onUpload}
        attachments={attachments}
        setAttachments={setAttachments}
      />
    )
  }

  return (
    <Search
      aggregations={aggregationResult}
      categoryCounts={categoryCounts}
      aiOverview={aiOverview}
      askBody={askBody}
      commonProps={commonProps}
      settings={settings}
      config={config}
      data={data}
      onCategoryChange={handleCategoryChange}
      filter={filter}
      getContainer={getContainer}
      handleLogoClick={handleLogoClick}
      hasMore={hasMore}
      hits={hits}
      initContainer={(ref: HTMLDivElement | null) => {
        containerRef.current = ref;
      }}
      loading={loading}
      logo={searchLogo === undefined ? logo : searchLogo}
      placeholder={placeholder}
      rightMenuWidth={rightMenuWidth}
      theme={theme}
      welcome={welcome}
      showFullScreenSpin={showFullScreenSpin}
      queryParams={queryParams}
      setQueryParams={setQueryParams}
      onLoadMore={handleLoadMore}
      onSearchFilter={(aggfilter: Record<string, any>) => {
        handleSearch({ ...queryParams, aggfilter }, false, false)
      }}
      onSearch={(params: Record<string, any>, shouldAsk: boolean, shouldAgg: boolean) => {
        if (params.mode === 'chat') {
          let assistant_id = params.assistant_id;
          if (!assistant_id) {
            if (params.action === 'deepthink') {
              assistant_id = settings?.deep_think_assistant;
            } else if (params.action === 'deepresearch') {
              assistant_id = settings?.deep_research_assistant;
            }
          }
          onChat({
            query: params.query || '',
            attachments: params.attachments || attachments || [],
            assistant_id,
          });
          return;
        };
        handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)
      }}
      onAsk={onAsk}
      onSuggestion={debouncedSuggestion}
      onRecommend={onRecommend}
      onChatContinue={(session_id) => {
        onChat({
          query: queryParams.query || '',
          attachments: attachments || [],
          assistant_id: settings?.payload?.ai_overview?.assistant,
          session_id,
        });
      }}
      getFieldsMeta={getFieldsMeta}
      onUpload={onUpload}
      attachments={attachments}
      setAttachments={setAttachments}
      histogramData={histogramData}
    />
  )
};

export default Fullscreen;