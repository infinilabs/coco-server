import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { LIST_TYPES } from "./ResultList";
import { formatESResult } from "./utils/es";
import PropTypes from 'prop-types';
import { ChartColumn, ListFilter } from 'lucide-react';
import { useTranslation } from "react-i18next";

import ChatLayout from './Layout/ChatLayout';
import ChatHeader from './ChatHeader';
import { History, Chat, AssistantList, ChatInput } from "@infinilabs/ai-chat";
import { debounce } from 'lodash';
import Home from "./pages/Home";
import Search from "./pages/Search";
import { ACTION_TYPE_SEARCH_KEYWORD } from "./SearchBox/SearchActions";

function renderChatMode({
  commonProps,
  isHistoryOpen,
  onNewChat,
  onToggleHistory,
  onSendMessage,
  language,
  apiConfig,
  chatRef,
  inputValue,
  changeInput,
  isDeepThinkActive,
  setIsDeepThinkActive
}) {
  const { BaseUrl, Token, endpoint } = apiConfig || {};

  return (
    <ChatLayout
      {...commonProps}
      content={
        <Chat
          ref={chatRef}
          BaseUrl={BaseUrl}
          formatUrl={(data) => `${endpoint}${BaseUrl}${data.url}`}
          Token={Token}
          locale={language === 'zh-CN' ? 'zh' : 'en'}
        />
      }
      input={
        <ChatInput
          onSend={onSendMessage}
          disabled={false}
          isChatMode={true}
          inputValue={inputValue}
          changeInput={changeInput}
          isDeepThinkActive={isDeepThinkActive}
          setIsDeepThinkActive={setIsDeepThinkActive}
          chatPlaceholder={language === 'zh-CN' ? '请输入问题...' : 'Type a message...'}
        />
      }
      sidebarCollapsed={!isHistoryOpen}
      header={
        <ChatHeader
          onNewChat={onNewChat}
          showChatHistory={true}
          onToggleHistory={onToggleHistory}
          AssistantList={
            <AssistantList
              BaseUrl={BaseUrl}
              Token={Token}
              locale={language === 'zh-CN' ? 'zh' : 'en'}
            />
          }
        />
      }
      sidebar={
        <History
          BaseUrl={BaseUrl}
          Token={Token}
          locale={language === 'zh-CN' ? 'zh' : 'en'}
        />
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
    onNewChat,
    assistantPage,
    assistantTotal,
    // History props
    chats = [],
    activeChat,
    onHistorySelect,
    onHistorySearch,
    onHistoryRefresh,
    onHistoryRename,
    onHistoryRemove,
    language = 'en-US',
    onSuggestion,
    onRecommend,
    getRawContent,
  } = props;

  const containerRef = useRef(null);
  const [result, setResult] = useState(formatESResult());
  const [askBody, setAskBody] = useState();
  const [loading, setLoading] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const shouldAskRef = useRef(true);
  const shouldAggRef = useRef(true);
  const [data, setData] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const loadLock = useRef(false);
  const isHomeSearchRef = useRef(true);
  const scrollRef = useRef(0)
  const [showToolbar, setShowToolbar] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const [isDeepThinkActive, setIsDeepThinkActive] = useState(false);
  const { t } = useTranslation();
  const queryFiltersRef = useRef([]);

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
    const { filters, ...rest } = queryParams
    queryFiltersRef.current = filters || []
    setQueryParams({
      ...rest,
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
      handleSearch(queryParams, false, false, true);
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
    if (!contentContainer || isHome) return () => { };

    contentContainer.addEventListener('scroll', handleScroll);
    return () => {
      contentContainer.removeEventListener('scroll', handleScroll);
    };
  }, [isHome, handleScroll]);

  useEffect(() => {
    if (!queryParams.query && (!Array.isArray(queryFiltersRef.current)|| queryFiltersRef.current.length === 0)) return;

    const isScroll = Number.isInteger(scrollRef.current) && scrollRef.current > 0;

    loadLock.current = true;
    setLoading(true);
    const { t, ...rest } = queryParams
    const filters = {}
    if (Array.isArray(queryFiltersRef.current)) {
      queryFiltersRef.current.map((item) => {
        const field = item.field?.payload?.field_name
        if (field && item.value) {
          filters[field] = Array.isArray(item.value) ? item.value : [item.value]
        }
      })
    }
    onSearch(
      {
        ...rest,
        search_type: queryParams?.search_type || ACTION_TYPE_SEARCH_KEYWORD,
        from: isScroll ? scrollRef.current : queryParams.from,
        filter: {
          ...(rest.filter || {}),
          ...filters
        }
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
      shouldAggRef.current
    );
  }, [JSON.stringify(queryParams)]);

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
    return () => {};
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

  const chatRef = useRef(null);
  const handleChatSendMessage = async (params) => {
    if (chatRef.current) {
      chatRef.current.init(params);
    }
  };

  const handleNewChat = () => {
    if (onNewChat) {
      onNewChat();
    } else if (chatRef.current) {
      chatRef.current.clearChat();
    }
  };

  const isChatMode = true;
  if (isChatMode) {
    return renderChatMode({
      activeChat,
      assistants,
      assistantPage,
      assistantTotal,
      chats,
      commonProps,
      currentAssistant,
      isHistoryOpen,
      messages,
      query_intent: props.query_intent,
      tools: props.tools,
      fetch_source: props.fetch_source,
      pick_source: props.pick_source,
      deep_read: props.deep_read,
      think: props.think,
      response: props.response,
      timedoutShow: props.timedoutShow,
      Question: props.Question,
      curChatEnd: props.curChatEnd,
      onNewChat: handleNewChat,
      registerStreamHandler: props.registerStreamHandler,
      onAssistantRefresh,
      onAssistantSelect,
      onAssistantPrevPage: props.onAssistantPrevPage,
      onAssistantNextPage: props.onAssistantNextPage,
      onAssistantSearch: props.onAssistantSearch,
      onToggleHistory: () => setIsHistoryOpen(open => !open),
      onHistoryRefresh,
      onHistoryRemove,
      onHistoryRename,
      onHistorySearch,
      onHistorySelect,
      onSendMessage: handleChatSendMessage,
      inputValue,
      changeInput: setInputValue,
      isDeepThinkActive,
      setIsDeepThinkActive,
      language,
      apiConfig: props.apiConfig,
      onStream: props.onStream,
      chatRef
    });
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
      onSearchFilter={filter => handleSearch({ ...queryParams, filter }, false, false)}
      onSearch={(params, shouldAsk, shouldAgg) => handleSearch({ ...queryParams, ...params, from: 0 }, shouldAsk, shouldAgg)}
      onAsk={onAsk}
      onSuggestion={debouncedSuggestion}
      onRecommend={onRecommend}
      getRawContent={getRawContent}
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
  onNewChat: PropTypes.func,
  registerStreamHandler: PropTypes.func,
  chats: PropTypes.array,
  activeChat: PropTypes.any,
  onHistorySelect: PropTypes.func,
  onHistorySearch: PropTypes.func,
  onHistoryRefresh: PropTypes.func,
  onHistoryRename: PropTypes.func,
  onHistoryRemove: PropTypes.func,
  query_intent: PropTypes.any,
  tools: PropTypes.any,
  fetch_source: PropTypes.any,
  pick_source: PropTypes.any,
  deep_read: PropTypes.any,
  think: PropTypes.any,
  response: PropTypes.any,
  timedoutShow: PropTypes.bool,
  Question: PropTypes.string,
  curChatEnd: PropTypes.bool
};

export default Fullscreen;
