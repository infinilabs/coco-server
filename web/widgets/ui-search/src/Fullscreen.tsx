import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import dayjs from "dayjs";
import { formatESResult } from "./utils/es";
import { normalizeCoverIconUrl } from "./utils/utils";

import { debounce, isEmpty } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH_KEYWORD, DEFAULT_SEARCH_SORT, normalizeSearchFuzziness, normalizeSearchSort } from "./SearchBox/ActionBar/SearchActions";
import Chat from "./pages/Chat";

const formatDateRangeParam = (value: number | string) => {
  const timestamp = typeof value === 'number' ? value : Number(value);
  const date = Number.isFinite(timestamp) ? dayjs(timestamp) : dayjs(value);

  return date.isValid() ? date.valueOf() : value;
};

const getDateRangeParams = (dateRange?: string) => {
  const now = dayjs();

  if (dateRange === '7d') {
    return {
      start: now.subtract(7, 'day').valueOf(),
      end: now.valueOf(),
    };
  }

  if (dateRange === '90d') {
    return {
      start: now.subtract(90, 'day').valueOf(),
      end: now.valueOf(),
    };
  }

  if (dateRange === '1y') {
    return {
      start: now.subtract(1, 'year').valueOf(),
      end: now.valueOf(),
    };
  }

  return {};
};

interface FullscreenProps {
  logo?: Record<string, any>;
  placeholder?: string;
  welcome?: string;
  aiOverview?: { enabled?: boolean };
  onSearch?: (...args: any[]) => void;
  onAggregation?: (...args: any[]) => void;
  onAsk?: (...args: any[]) => void;
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
    placeholder,
    welcome,
    aiOverview,
    onSearch,
    onAggregation,
    onAsk,
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
  const [askBody, setAskBody] = useState<any>();
  const [loading, setLoading] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const shouldAskRef = useRef(true);
  const shouldAggRef = useRef(true);
  const [data, setData] = useState<any[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)

  const [chatParams, setChatParams] = useState<Record<string, any>>({});
  const [attachments, setAttachments] = useState<any[]>([]);

  const onChat = (params: Record<string, any>) => {
    setChatParams(params);
    setQueryParams?.({
      mode: 'chat',
    })
  }

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

  const handleSearch = (queryParams: Record<string, any>, shouldAsk: boolean, shouldAgg: boolean, isScroll = false) => {
    const fuzziness = normalizeSearchFuzziness(queryParams?.fuzziness);
    const sort = normalizeSearchSort(queryParams?.sort);
    shouldAskRef.current = shouldAsk;
    shouldAggRef.current = shouldAgg;
    if (!isScroll) {
      resetScroll();
      isHomeSearchRef.current = true;
    }
    const nextQueryParams: Record<string, any> = {
      ...queryParams,
      fuzziness,
      sort,
      ...(shouldAgg ? { aggfilter: {} } : {}),
      t: new Date().valueOf()
    };
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
  }, []);

  useEffect(() => {
    if (queryParams.mode === 'chat' || !queryParams?.query && isEmpty(queryParams?.filter) && isEmpty(queryParams?.aggfilter)) return;

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    loadLock.current = true;
    setLoading(true);

    const { t, date_range, start, end, filter = {}, aggfilter = {}, ...rest } = queryParams;
    const fuzziness = normalizeSearchFuzziness(queryParams?.fuzziness);
    const sort = normalizeSearchSort(queryParams?.sort);
    const dateRangeParams = start && end ? { start: formatDateRangeParam(start), end: formatDateRangeParam(end) } : getDateRangeParams(date_range);
    const filterWithoutAgg = {
      ...filter,
      'metadata.content_category': queryParams['metadata.content_category'] && queryParams['metadata.content_category'] !== 'all' ? [queryParams['metadata.content_category']] : undefined,
    }

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
              message: JSON.stringify({
                query: queryParams.query,
                result: rs?.hits
              }),
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
        ...dateRangeParams,
        filter: filterWithoutAgg
      }, (res: any) => {
        let validatedAggfilter: Record<string, any> = {};
        if (res && !res.error) {
          const rs = formatESResult(res);
          setAggregationResult(rs.aggregations || []);
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
    resetScroll();
    isHomeSearchRef.current = true;
    if (onLogoClick) onLogoClick();
  };

  const showFullScreenSpin = loading && isHomeSearchRef.current;

  const { mode = 'search' } = queryParams

  if (mode === 'chat') {
    return (
      <Chat
        commonProps={commonProps}
        logo={logo}
        handleLogoClick={handleLogoClick}
        apiConfig={apiConfig}
        queryParams={queryParams}
        onBackToSearch={() => {
          handleLogoClick();
        }}
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
      logo={logo}
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
    />
  )
};

export default Fullscreen;