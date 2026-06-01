import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { formatESResult } from "./utils/es";
import { normalizeCoverIconUrl } from "./utils/utils";
import PropTypes from 'prop-types';

import { debounce, isEmpty } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH, ACTION_TYPE_SEARCH_HYBRID, ACTION_TYPE_SEARCH_KEYWORD } from "./SearchBox/ActionBar/SearchActions";
import Chat from "./pages/Chat";

const Fullscreen = props => {
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
    onUpload
  } = props;

  const containerRef = useRef(null);
  const [result, setResult] = useState(formatESResult());
  const [askBody, setAskBody] = useState();
  const [loading, setLoading] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const shouldAskRef = useRef(true);
  const shouldAggRef = useRef(true);
  const [data, setData] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)
  const [showToolbar, setShowToolbar] = useState(false);
  const [checkViewport, setCheckViewport] = useState(false);
  const isViewportLoadingRef = useRef(false);
  const [chatParams, setChatParams] = useState({});
  const [attachments, setAttachments] = useState([]);

  const onChat = (params) => {
    setChatParams({
      query: params.query,
      attachments: params.attachments
    });
    setQueryParams({
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

  const handleSearch = (queryParams, shouldAsk, shouldAgg, isScroll = false) => {
    shouldAskRef.current = shouldAsk;
    shouldAggRef.current = shouldAgg;
    if (!isScroll) {
      resetScroll();
      isHomeSearchRef.current = true;
    }
    setQueryParams({
      ...queryParams,
      t: new Date().valueOf()
    });
  };

  const handleScroll = useCallback(() => {
    if (!containerRef.current || loading || !hasMore || loadLock.current) return;

    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const distanceToBottom = scrollHeight - scrollTop - clientHeight;
    if (distanceToBottom < 200) {
      loadLock.current = true;
      const { from, size } = queryParams;
      scrollRef.current = (scrollRef.current || from) + size;
      isViewportLoadingRef.current = false;
      handleSearch(queryParams, false, false, true);
    }
  }, [queryParams, loading, hasMore, handleSearch]);

  const checkViewportAndLoad = useCallback(() => {
    if (
      !containerRef.current ||
      loading ||
      !hasMore ||
      loadLock.current ||
      isHome
    ) {
      setCheckViewport(false);
      return;
    }

    const checkAfterRender = () => {
      if (loading || !hasMore || loadLock.current) {
        setCheckViewport(false);
        return;
      }

      const container = containerRef.current;
      const { scrollHeight, clientHeight, scrollTop } = container;

      const heightDiff = scrollHeight - clientHeight;
      const hasRealScrollbar = heightDiff > 20;
      const isViewportReallyFilled = hasRealScrollbar || scrollTop > 0;

      if (!isViewportReallyFilled && hasMore) {
        loadLock.current = true;
        isViewportLoadingRef.current = true;
        const { from, size } = queryParams;
        scrollRef.current = (scrollRef.current || from) + size;
        handleSearch(queryParams, false, false, true);
        setCheckViewport(true);
      } else {
        setCheckViewport(false);
      }
    };

    requestAnimationFrame(() => {
      setTimeout(checkAfterRender, 200);
    });
  }, [containerRef, loading, hasMore, loadLock, isHome, queryParams, handleSearch]);

  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkScreenSize();
    window.addEventListener('resize', checkScreenSize);
    return () => window.removeEventListener('resize', checkScreenSize);
  }, []);

  useEffect(() => {
    const contentContainer = containerRef.current;
    if (!contentContainer || isHome) return () => { };

    contentContainer.addEventListener('scroll', handleScroll);
    return () => {
      contentContainer.removeEventListener('scroll', handleScroll);
    };
  }, [isHome, handleScroll]);

  useEffect(() => {
    if (queryParams.mode === 'chat' || !queryParams?.query && isEmpty(queryParams?.filter) && isEmpty(queryParams?.aggfilter)) return;

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    loadLock.current = true;
    if (!isViewportLoadingRef.current) {
      setLoading(true);
    }

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
    onSearch(
      {
        ...rest,
        filter: newFilter,
        search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
        from: isScroll ? scrollRef.current : queryParams.from,
        'metadata.content_category': undefined
      },
      res => {
        loadLock.current = false;
        if (!isViewportLoadingRef.current) {
          setLoading(false);
        }
        isViewportLoadingRef.current = false;

        let rs;
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

          setCheckViewport(true);
        } else {
          if (!isScroll) {
            setResult(formatESResult());
            setData([]);
          }
          setHasMore(false);
          isHomeSearchRef.current = false;
          setCheckViewport(false);
        }

        if (onAggregation && shouldAggRef.current) {
          setLoading(true);
          onAggregation({
            query: queryParams.query,
            search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
            filter: filterWithoutAgg
          }, (res) => {
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
      (loadingState) => {
        if (!isViewportLoadingRef.current) {
          setLoading(loadingState);
        }
      }
    );
  }, [JSON.stringify(queryParams)]);

  useEffect(() => {
    if (!checkViewport) return;

    const timer = setTimeout(() => {
      checkViewportAndLoad();
    }, 300);

    return () => clearTimeout(timer);
  }, [checkViewport, checkViewportAndLoad]);

  useEffect(() => {
    if (data.length > 0 && !Number.isInteger(scrollRef.current) || scrollRef.current === 0) {
      const timer = setTimeout(() => {
        if (hasMore && !loading && !loadLock.current) {
          setCheckViewport(true);
        }
      }, 500);

      return () => clearTimeout(timer);
    }
  }, [data, hasMore, loading]);

  useEffect(() => {
    const handleResize = () => {
      if (containerRef.current && hasMore && !loading && !loadLock.current) {
        setCheckViewport(true);
      }
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [hasMore, loading]);

  useEffect(() => {
    window.onsearch = query => handleSearch({ ...queryParams, from: 0, query }, true, true);
    return () => {
      window.onsearch = undefined;
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
    setQueryParams({
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
        apiConfig={apiConfig}
        queryParams={queryParams}
        onBackToSearch={() => {
          setQueryParams({
            ...queryParams,
            action_type: ACTION_TYPE_SEARCH,
            mode: 'search'
          });
        }}
        setQueryParams={setQueryParams}
        defaultParams={chatParams}
        setDefaultParams={setChatParams}
        setAttachments={setAttachments}
      />
    )
  }

  if (isHome) {
    return (
      <Home
        commonProps={commonProps}
        loading={showFullScreenSpin}
        logo={logo}
        onSearch={(params, shouldAsk, shouldAgg) => {
          if (params.mode === 'chat') {
            onChat({
              query: params.query || '',
              attachments: params.attachments || attachments || [],
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
      config={config}
      data={data}
      filter={filter}
      getContainer={() => containerRef.current}
      handleLogoClick={handleLogoClick}
      hasMore={hasMore}
      hits={hits}
      initContainer={ref => {
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
      onSearchFilter={(aggfilter) => {
        handleSearch({ ...queryParams, aggfilter }, false, false)
      }}
      onSearch={(params, shouldAsk, shouldAgg) => {
        if (params.mode === 'chat') {
          onChat({
            query: params.query || '',
            attachments: params.attachments || attachments || [],
          });
          return;
        };
        handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)
      }}
      onAsk={onAsk}
      onSuggestion={debouncedSuggestion}
      onRecommend={onRecommend}
      getRawContent={getRawContent}
      onChatContinue={() => {
        onChat({
          query: queryParams.query || '',
          attachments: attachments || [],
        });
      }}
      getFieldsMeta={getFieldsMeta}
      onUpload={onUpload}
      attachments={attachments}
      setAttachments={setAttachments}
    />
  )
};

Fullscreen.propTypes = {
  logo: PropTypes.object,
  placeholder: PropTypes.string,
  welcome: PropTypes.string,
  aiOverview: PropTypes.shape({
    enabled: PropTypes.bool
  }),
  onSearch: PropTypes.func.isRequired,
  onAsk: PropTypes.func,
  config: PropTypes.object,
  isHome: PropTypes.bool,
  rightMenuWidth: PropTypes.number,
  queryParams: PropTypes.object.isRequired,
  setQueryParams: PropTypes.func.isRequired,
  onLogoClick: PropTypes.func,
  theme: PropTypes.oneOf(['light', 'dark']),
  language: PropTypes.string,
  onNewChat: PropTypes.func,
  onSuggestion: PropTypes.func,
  onRecommend: PropTypes.func,
  getRawContent: PropTypes.func,
  onUpload: PropTypes.func,
  apiConfig: PropTypes.object
};

export default Fullscreen;