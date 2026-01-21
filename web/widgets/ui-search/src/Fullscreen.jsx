import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { formatESResult } from "./utils/es";
import PropTypes from 'prop-types';
import { useTranslation } from "react-i18next";

import { debounce, isEmpty } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH_KEYWORD } from "./SearchBox/SearchActions";
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
  const { t } = useTranslation();
  const checkViewportRef = useRef(false);
  const isViewportLoadingRef = useRef(false);

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
      checkViewportRef.current = false;
      return;
    }

    const { scrollHeight, clientHeight } = containerRef.current;
    const isViewportFilled = scrollHeight > clientHeight + 10;

    if (!isViewportFilled && hasMore) {
      loadLock.current = true;
      isViewportLoadingRef.current = true;
      const { from, size } = queryParams;
      scrollRef.current = (scrollRef.current || from) + size;
      handleSearch(queryParams, false, false, true);
      checkViewportRef.current = true;
    } else {
      checkViewportRef.current = false;
    }
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
    console.log(queryParams)
    if (!queryParams?.query && isEmpty(queryParams?.filter)) return;

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    loadLock.current = true;
    if (!isViewportLoadingRef.current) {
      setLoading(true);
    }
    
    const { t, ...rest } = queryParams
    onSearch(
      {
        ...rest,
        search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
        from: isScroll ? scrollRef.current : queryParams.from,
      },
      res => {
        loadLock.current = false;
        if (!isViewportLoadingRef.current) {
          setLoading(false);
        }
        isViewportLoadingRef.current = false;

        let rs;
        if (res && !res.error) {
          rs = formatESResult(res);
          setResult(os => ({
            ...rs,
            aggregations: res?.aggregations ? rs.aggregations : os.aggregations
          }));

          const newData = isScroll ? [...data, ...(rs.hits?.hits || [])] : rs.hits?.hits || [];
          setData(newData);
          setHasMore(newData.length < (rs.hits.total || 0));
          if (!isScroll) isHomeSearchRef.current = false;

          checkViewportRef.current = true;
        } else {
          if (!isScroll) {
            setResult(formatESResult());
            setData([]);
          }
          setHasMore(false);
          isHomeSearchRef.current = false;
          checkViewportRef.current = false;
        }

        if (onAggregation && shouldAggRef.current) {
          setLoading(true);
          onAggregation({ query: queryParams.query }, (res) => {
            shouldAskRef.current = false
            if (res && !res.error) {
              rs = formatESResult(res);
              setResult(os => ({
                ...rs,
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
    if (!checkViewportRef.current) return;
    
    const timer = setTimeout(() => {
      checkViewportAndLoad();
    }, 100);

    return () => clearTimeout(timer);
  }, [checkViewportRef.current, checkViewportAndLoad]);

  useEffect(() => {
    const handleResize = () => {
      if (containerRef.current && hasMore && !loading && !loadLock.current) {
        checkViewportRef.current = true;
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

  const { query, filter, filters = [] } = queryParams;

  const commonProps = { isMobile, theme };
  const { hits, aggregations } = result;

  const handleLogoClick = () => {
    setQueryParams({
      from: 0,
      size: 10,
      query: '',
      filter: {},
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
        language={language}
        apiConfig={apiConfig}
        queryParams={queryParams}
        onBackToSearch={() => {
          setQueryParams({
            ...queryParams,
            mode: 'search'
          });
        }}
      />
    )
  }

  if (isHome) {
    return (
      <Home
        commonProps={commonProps}
        loading={showFullScreenSpin}
        logo={logo}
        onSearch={(params, shouldAsk, shouldAgg) => handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)}
        placeholder={placeholder}
        welcome={welcome}
        queryParams={queryParams}
        setQueryParams={setQueryParams}
        onSuggestion={debouncedSuggestion}
        onRecommend={onRecommend}
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
      onSearchFilter={(filter, filter_payload) => {
        handleSearch({ ...queryParams, filter, filter_payload }, false, false)
      }}
      onSearch={(params, shouldAsk, shouldAgg) => handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)}
      onAsk={onAsk}
      onSuggestion={debouncedSuggestion}
      onRecommend={onRecommend}
      getRawContent={getRawContent}
      onChatContinue={() => {
        setQueryParams({
          ...queryParams,
          mode: 'chat'
        });
      }}
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
  apiConfig: PropTypes.object
};

export default Fullscreen;