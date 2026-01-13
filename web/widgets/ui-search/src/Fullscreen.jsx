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
import HomeLayout from "./Layout/HomeLayout";
import BasicLayout from "./Layout/BasicLayout";
import Toolbar from "./Toolbar";
import PropTypes from 'prop-types';
import { ChartColumn, ListFilter } from 'lucide-react';
import ChatLayout from './Layout/ChatLayout';
import ChatContent from './ChatContent';
import ChatInput from './ChatContent/ChatInput';
import HistoryList from './History';
import ChatHeader from './ChatHeader';

function renderChatMode({
  activeChat,
  assistants,
  chats,
  commonProps,
  currentAssistant,
  isHistoryOpen,
  messages,
  onAssistantRefresh,
  onAssistantSelect,
  onToggleHistory,
  onHistoryRefresh,
  onHistoryRemove,
  onHistoryRename,
  onHistorySearch,
  onHistorySelect,
  onSendMessage
}) {
  return (
    <ChatLayout
      {...commonProps}
      content={<ChatContent messages={messages} />}
      input={<ChatInput onSendMessage={onSendMessage} />}
      sidebarCollapsed={!isHistoryOpen}
      header={
        <ChatHeader
          activeChat={activeChat}
          assistants={assistants}
          currentAssistant={currentAssistant}
          showChatHistory={true}
          onAssistantRefresh={onAssistantRefresh}
          onAssistantSelect={onAssistantSelect}
          onToggleHistory={onToggleHistory}
        />
      }
      sidebar={
        <HistoryList
          active={activeChat}
          chats={chats}
          onRefresh={onHistoryRefresh}
          onRemove={onHistoryRemove}
          onRename={onHistoryRename}
          onSearch={onHistorySearch}
          onSelect={onHistorySelect}
        />
      }
    />
  );
}

function renderHomeMode({ commonProps, isHome, loading, logo, onSearch: onSearchSubmit, placeholder, query, welcome }) {
  return (
    <HomeLayout
      {...commonProps}
      loading={loading}
      logo={
        <Logo
          isHome={isHome}
          {...commonProps}
          {...logo}
        />
      }
      searchbox={
        <SearchBox
          {...commonProps}
          placeholder={placeholder}
          query={query}
          onSearch={onSearchSubmit}
        />
      }
      welcome={
        welcome ? (
          <Welcome
            {...commonProps}
            text={welcome}
          />
        ) : null
      }
    />
  );
}

function renderSearchMode({
  aggregations,
  aiOverview,
  askBody,
  commonProps,
  config,
  data,
  filter,
  getContainer,
  handleLogoClick,
  hits,
  hasMore,
  initContainer,
  isHome,
  listType,
  loading,
  logo,
  onAsk,
  onSearchFilter,
  onSearchQuery,
  placeholder,
  query,
  queryParams,
  rightMenuWidth,
  showFullScreenSpin,
  setQueryParams,
  theme,
  welcome,
  widgets
}) {
  return (
    <BasicLayout
      {...commonProps}
      getContainer={getContainer}
      initContainer={initContainer}
      loading={showFullScreenSpin}
      rightMenuWidth={rightMenuWidth}
      aggregations={
        <Aggregations
          {...commonProps}
          aggregations={aggregations}
          config={config.aggregations}
          filter={filter}
          onSearch={onSearchFilter}
        />
      }
      aiOverview={
        listType?.showAIOverview && aiOverview?.enabled ? (
          <AIOverviewWrapper
            askBody={askBody}
            config={aiOverview}
            theme={theme}
            onAsk={onAsk}
          />
        ) : null
      }
      logo={
        <Logo
          isHome={isHome}
          onLogoClick={handleLogoClick}
          {...commonProps}
          {...logo}
        />
      }
      resultHeader={
        <ResultHeader
          hits={hits}
          {...commonProps}
        />
      }
      resultList={
        listType ? (
          <listType.component
            {...commonProps}
            data={data}
            getDetailContainer={getContainer}
            hasMore={hasMore}
            loading={loading}
            query={query}
            total={hits?.total || 0}
          />
        ) : null
      }
      searchbox={
        <SearchBox
          {...commonProps}
          minimize={true}
          placeholder={placeholder}
          query={query}
          onSearch={onSearchQuery}
        />
      }
      tabs={
        <Categories
          type={queryParams?.type}
          onChange={type => {
            setQueryParams({
              ...queryParams,
              type,
              t: new Date().valueOf()
            });
          }}
        />
      }
      tools={
        <div className='h-46px flex items-center gap-8px'>
          <ListFilter className='h-16px w-16px' />
          <ChartColumn className='h-16px w-16px' />
        </div>
      }
      welcome={
        welcome ? (
          <Welcome
            {...commonProps}
            text={welcome}
          />
        ) : null
      }
      widgets={
        <>
          {widgets.map((item, index) => (
            <AIOverviewWrapper
              askBody={askBody}
              config={item}
              key={index}
              onAsk={onAsk}
            />
          ))}
        </>
      }
    />
  );
}

const Fullscreen = props => {
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
    messages = [],
    onSendMessage,
    assistants = [],
    currentAssistant,
    onAssistantRefresh,
    onAssistantSelect,
    // History props
    chats = [],
    activeChat,
    onHistorySelect,
    onHistorySearch,
    onHistoryRefresh,
    onHistoryRename,
    onHistoryRemove
  } = props;

  const containerRef = useRef(null);
  const [result, setResult] = useState(formatESResult());
  const [askBody, setAskBody] = useState();
  const [loading, setLoading] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const shouldAskRef = useRef(true);
  const [data, setData] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)
  const [showToolbar, setShowToolbar] = useState(false);

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

  const handleSearch = (queryParams, shouldAsk, isScroll = false) => {
    shouldAskRef.current = shouldAsk;
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
      handleSearch(queryParams, false, true);
    }
  }, [queryParams, loading, hasMore, handleSearch]);

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
    if (!contentContainer || isHome) return () => {};

    contentContainer.addEventListener('scroll', handleScroll);
    return () => {
      contentContainer.removeEventListener('scroll', handleScroll);
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
        from: isScroll ? scrollRef.current : queryParams.from
      },
      res => {
        loadLock.current = false;
        setLoading(false);

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
      setLoading,
      shouldAgg
    );
  }, [JSON.stringify(queryParams)]);

  useEffect(() => {
    window.onsearch = query => handleSearch({ ...queryParams, from: 0, query }, true);
    return () => {
      window.onsearch = undefined;
    };
  }, [queryParams]);

  const { query, filter, type = 'all' } = queryParams;

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find(item => item.type === type) || LIST_TYPES[0];
  }, [type]);

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

  const isChatMode = messages && messages.length > 0;
  if (isChatMode) {
    return renderChatMode({
      activeChat,
      assistants,
      chats,
      commonProps,
      currentAssistant,
      isHistoryOpen,
      messages,
      onAssistantRefresh,
      onAssistantSelect,
      onToggleHistory: () => setIsHistoryOpen(open => !open),
      onHistoryRefresh,
      onHistoryRemove,
      onHistoryRename,
      onHistorySearch,
      onHistorySelect,
      onSendMessage
    });
  }

  if (isHome) {
    return renderHomeMode({
      commonProps,
      isHome,
      loading: showFullScreenSpin,
      logo,
      onSearch: query => handleSearch({ ...queryParams, from: 0, query }, true),
      placeholder,
      query,
      welcome
    });
  }

  return renderSearchMode({
    aggregations,
    aiOverview,
    askBody,
    commonProps,
    config,
    data,
    filter,
    getContainer: () => containerRef.current,
    handleLogoClick,
    hasMore,
    hits,
    initContainer: ref => {
      containerRef.current = ref;
    },
    isHome,
    listType,
    loading,
    logo,
    onAsk,
    onSearchFilter: filter => handleSearch({ ...queryParams, filter }, true),
    onSearchQuery: query => handleSearch({ ...queryParams, from: 0, query }, true),
    placeholder,
    query,
    queryParams,
    rightMenuWidth,
    showFullScreenSpin,
    setQueryParams,
    theme,
    welcome,
    widgets
  });
};

Fullscreen.propTypes = {
  logo: PropTypes.object,
  placeholder: PropTypes.string,
  welcome: PropTypes.string,
  aiOverview: PropTypes.shape({
    enabled: PropTypes.bool
  }),
  widgets: PropTypes.array,
  onSearch: PropTypes.func,
  onAsk: PropTypes.func,
  config: PropTypes.object,
  isHome: PropTypes.bool,
  rightMenuWidth: PropTypes.number,
  queryParams: PropTypes.object,
  setQueryParams: PropTypes.func,
  onLogoClick: PropTypes.func,
  theme: PropTypes.oneOf(['light', 'dark']),
  language: PropTypes.string,
  messages: PropTypes.array,
  onSendMessage: PropTypes.func,
  assistants: PropTypes.array,
  currentAssistant: PropTypes.any,
  onAssistantRefresh: PropTypes.func,
  onAssistantSelect: PropTypes.func,
  chats: PropTypes.array,
  activeChat: PropTypes.any,
  onHistorySelect: PropTypes.func,
  onHistorySearch: PropTypes.func,
  onHistoryRefresh: PropTypes.func,
  onHistoryRename: PropTypes.func,
  onHistoryRemove: PropTypes.func
};

export default Fullscreen;
