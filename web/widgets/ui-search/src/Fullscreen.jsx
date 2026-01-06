import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import SearchBox from "./SearchBox";
import Logo from "./Logo";
import Aggregations from "./Aggregations";
import ResultHeader from "./ResultHeader";
import { LIST_TYPES } from "./ResultList";
import { formatESResult } from "./utils/es";
import Welcome from "./Welcome";
import AIOverviewWrapper from "./AIOverview/AIOverviewWrapper";
import Categories from "./Categories";
import { ChartColumn } from "lucide-react";
import { ListFilter } from "lucide-react";
import HomeLayout from "./Layout/HomeLayout";
import BasicLayout from "./Layout/BasicLayout";

const Fullscreen = (props) => {
  const {
    logo = {},
    placeholder,
    welcome,
    aiOverview,
    widgets = [],
    onSearch,
    onAsk,
    config = {},
    isHome = false,
    rightMenuWidth,
    queryParams = {},
    setQueryParams,
    onLogoClick,
    theme = 'light', 
    language = 'en-US',
  } = props;

  const containerRef = useRef(null);
  const [result, setResult] = useState(formatESResult());
  const [askBody, setAskBody] = useState();
  const [loading, setLoading] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const shouldAskRef = useRef(true);
  const [data, setData] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)

  const resetScroll = () => {
    scrollRef.current = 0;
    if (containerRef.current) {
      try {
        containerRef.current.scrollTo({
          top: 0,
          behavior: 'instant'
        });
      } catch (e) {
        containerRef.current.scrollTop = 0;
      }
    }
  }

  const handleSearch = (queryParams, shouldAsk, isScroll = false) => {
    shouldAskRef.current = shouldAsk;
    if (!isScroll) {
      resetScroll();
      isHomeSearchRef.current = true
    }
    setQueryParams({
      ...queryParams,
      t: new Date().valueOf(),
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
      handleSearch(queryParams, false, true);
    }
  }, [queryParams, loading, hasMore, handleSearch]);

  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };
    checkScreenSize();
    window.addEventListener("resize", checkScreenSize);
    return () => window.removeEventListener("resize", checkScreenSize);
  }, []);

  useEffect(() => {
    const contentContainer = containerRef.current;
    if (!contentContainer || isHome) return;

    contentContainer.addEventListener("scroll", handleScroll);
    return () => {
      contentContainer.removeEventListener("scroll", handleScroll);
    };
  }, [isHome, handleScroll]);

  useEffect(() => {
    if (!queryParams.query) return;

    const shouldAgg = queryParams.filter && Object.keys(queryParams.filter).length === 0;
    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;
    
    loadLock.current = true;
    setLoading(true);
    onSearch(
      {
        ...queryParams,
        from: isScroll ? scrollRef.current : queryParams.from,
      },
      (res) => {
        loadLock.current = false;
        setLoading(false);
        
        let rs;
        if (res && !res.error) {
          rs = formatESResult(res);
          setResult((os) => ({
            ...rs,
            aggregations: res?.aggregations ? rs.aggregations : os.aggregations,
          }));
          
          const newData = isScroll ? [...data, ...(rs.hits?.hits || [])] : (rs.hits?.hits || []);
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
              result: rs?.hits,
            }),
            t: new Date().valueOf(),
          });
        }
      },
      setLoading,
      shouldAgg,
    );
  }, [JSON.stringify(queryParams)]);

  useEffect(() => {
    window.onsearch = (query) =>
      handleSearch({ ...queryParams, from: 0, query }, true);
    return () => {
      window.onsearch = undefined;
    };
  }, [queryParams]);

  const { query, filter, type = 'all' } = queryParams;

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find((item) => item.type === type) || LIST_TYPES[0];
  }, [type]);

  const commonProps = { isMobile, theme };
  const { hits, aggregations } = result;

  const handleLogoClick = () => {
    setQueryParams({
      from: 0,
      size: 10,
      query: '',
      filter: {},
      sort: '',
    });
    setData([]);
    setHasMore(false);
    resetScroll();
    isHomeSearchRef.current = true;
    if (onLogoClick) onLogoClick();
  };

  const showFullScreenSpin = loading && isHomeSearchRef.current;

  return isHome ? (
    <HomeLayout 
      {...commonProps}
      loading={showFullScreenSpin}
      logo={<Logo isHome={isHome} {...commonProps} {...logo} />}
      welcome={welcome ? <Welcome {...commonProps} text={welcome} /> : null}
      searchbox={
        <SearchBox
          {...commonProps}
          placeholder={placeholder}
          query={query}
          onSearch={(query) => {
            handleSearch({ ...queryParams, from: 0, query }, true)
          }}
        />
      }
    />
  ) : (
    <BasicLayout
      {...commonProps}
      initContainer={(ref) => {
        containerRef.current = ref;
      }}
      getContainer={() => containerRef.current}
      loading={showFullScreenSpin}
      logo={<Logo isHome={isHome} onLogoClick={handleLogoClick} {...commonProps} {...logo} />}
      welcome={welcome ? <Welcome {...commonProps} text={welcome} /> : null}
      searchbox={
        <SearchBox
          {...commonProps}
          placeholder={placeholder}
          query={query}
          onSearch={(query) => {
            handleSearch({ ...queryParams, from: 0, query }, true)
          }}
          minimize={true}
        />
      }
      tabs={(
        <Categories 
          type={queryParams?.type}
          onChange={(type) => {
            setQueryParams({
              ...queryParams,
              type,
              t: new Date().valueOf(),
            });
          }}
        />
      )}
      tools={(
        <div className="h-46px flex items-center gap-8px">
          <ListFilter className="w-16px h-16px"/>
          <ChartColumn className="w-16px h-16px"/>
        </div>
      )}
      rightMenuWidth={rightMenuWidth}
      aggregations={
        <Aggregations
          {...commonProps}
          config={config.aggregations}
          aggregations={aggregations}
          filter={filter}
          onSearch={(filter) => {
            handleSearch({ ...queryParams, filter }, true)
          }}
        />
      }
      resultHeader={<ResultHeader hits={hits} {...commonProps} />}
      aiOverview={
        listType?.showAIOverview && aiOverview?.enabled ? (
          <AIOverviewWrapper
            askBody={askBody}
            config={aiOverview}
            onAsk={onAsk}
          />
        ) : null
      }
      resultList={
        listType ? (
          <listType.component
            {...commonProps}
            getDetailContainer={() => containerRef.current}
            data={data}
            query={query}
            total={hits?.total || 0}
            loading={loading}
            hasMore={hasMore}
          />
        ) : null
      }
      widgets={
        <>
          {widgets.map((item, index) => (
            <AIOverviewWrapper
              key={index}
              askBody={askBody}
              config={item}
              onAsk={onAsk}
            />
          ))}
        </>
      }
    />
  );
};

export default Fullscreen;