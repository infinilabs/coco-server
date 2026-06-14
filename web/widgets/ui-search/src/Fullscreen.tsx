import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { formatESResult } from "./utils/es";
import { normalizeCoverIconUrl } from "./utils/utils";

import { debounce, isEmpty } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH_KEYWORD } from "./SearchBox/ActionBar/SearchActions";
import Chat from "./pages/Chat";

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
  getRawContent?: (...args: any[]) => any;
  apiConfig?: Record<string, any>;
  getFieldsMeta?: (...args: any[]) => any;
  onUpload?: (...args: any[]) => void;
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
    getRawContent,
    apiConfig,
    getFieldsMeta,
    onUpload,
    settings
  } = props;

  const containerRef = useRef<HTMLDivElement | null>(null);
  const getContainer = useCallback(() => containerRef.current, []);
  const [result, setResult] = useState(formatESResult());
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
    shouldAskRef.current = shouldAsk;
    shouldAggRef.current = shouldAgg;
    if (!isScroll) {
      resetScroll();
      isHomeSearchRef.current = true;
    }
    setQueryParams?.({
      ...queryParams,
      t: new Date().valueOf()
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
  }, []);

  useEffect(() => {
    if (queryParams.mode === 'chat' || !queryParams?.query && isEmpty(queryParams?.filter) && isEmpty(queryParams?.aggfilter)) return;

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    loadLock.current = true;
    setLoading(true);

    const { t, filter = {}, aggfilter = {}, ...rest } = queryParams;
    const filterWithoutAgg = {
      ...filter,
      'metadata.content_category': queryParams['metadata.content_category'] && queryParams['metadata.content_category'] !== 'all' ? [queryParams['metadata.content_category']] : undefined,
    }
    const newFilter = { ...filterWithoutAgg };
    Object.keys(aggfilter).forEach(key => {
      if (newFilter[key] !== undefined && aggfilter[key] !== undefined) {
        const filterVal = Array.isArray(newFilter[key]) ? newFilter[key] : [newFilter[key]];
        const aggVal = Array.isArray(aggfilter[key]) ? aggfilter[key] : [aggfilter[key]];
        newFilter[key] = [...new Set([...filterVal, ...aggVal])];
      } else if (aggfilter[key] !== undefined) {
        newFilter[key] = aggfilter[key];
      }
    });
    onSearch?.(
      {
        ...rest,
        filter: newFilter,
        search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
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
          setResult(os => ({
            ...rs,
            aggregations: res?.aggregations ? rs.aggregations : os.aggregations
          }));

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

        if (onAggregation && shouldAggRef.current) {
          setLoading(true);
          onAggregation({
            query: queryParams.query,
            search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
            filter: filterWithoutAgg
          }, (res: any) => {
            shouldAskRef.current = false
            if (res && !res.error) {
              rs = formatESResult(res);
              setResult(os => ({
                ...os,
                aggregations: res?.aggregations ? rs.aggregations : os.aggregations
              }));
            }
            setLoading(false);
          })
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

  const commonProps = { isMobile, theme, apiConfig, language };
  const { hits, aggregations } = result;

  const handleLogoClick = () => {
    setQueryParams?.({
      from: 0,
      size: 10,
      query: '',
      filter: {},
      aggfilter: {},
      sort: ''
    });
    setData([]);
    setHasMore(false);
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
      aggregations={aggregations}
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
      getRawContent={getRawContent}
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