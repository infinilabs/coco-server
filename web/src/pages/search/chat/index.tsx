import { useEffect, useReducer, useRef, useState } from 'react';

import { request } from '@/service/request';
import { FullscreenPage } from 'ui-search';
import { searchAssistant } from '@/service/api/assistant';

function useSimpleQueryParams(defaultParams = {}) {
  const [params, setParams] = useState({
    from: 0,
    size: 10,
    sort: [],
    filter: {},
    ...defaultParams
  });

  return [params, setParams];
}

export function Component() {
  const [queryParams, setQueryParams] = useSimpleQueryParams();
  const [queryParamsState, setQueryParamsState] = useState({
    from: 0,
    size: 10
  });

  const [chats, setChats] = useState<any[]>([]);
  const [activeChat, setActiveChat] = useState<any>();
  const [messages, setMessages] = useState<any[]>([]);
  const [assistants, setAssistants] = useState<any[]>([]);
  const [currentAssistant, setCurrentAssistant] = useState<any>();
  const [assistantKeyword, setAssistantKeyword] = useState('');
  const [assistantPagination, setAssistantPagination] = useState({
    current: 1,
    pageSize: 5,
    total: 0
  });

  const [Question, setQuestion] = useReducer(
    (_state: string, value: string) => value,
    ''
  );
  const activeChatRef = useRef<any>(null);

  useEffect(() => {
    activeChatRef.current = activeChat;
  }, [activeChat]);

  const enableQueryParams = true;

  const fetchChatHistory = async (): Promise<void> => {
    try {
      const res: any = await request({
        method: 'get',
        params: {
          from: 0,
          size: 100
        },
        url: '/chat/_history'
      });

      const esResp = res?.data || res;
      const hits = esResp?.hits?.hits || [];
      setChats(hits);
      if (!activeChat && hits.length > 0) {
        const first = hits[0];
        setActiveChat(first);
        const firstId = first?._id || first?._source?.id;
        if (firstId) {
          try {
            const historyRes: any = await request({
              method: 'get',
              params: { from: 0, size: 1000 },
              url: `/chat/${firstId}/_history`
            });
            const historyEs = historyRes?.data || historyRes;
            const historyHits = historyEs?.hits?.hits || [];
            setActiveChat((prev: any) => ({
              ...(prev || {}),
              _id: firstId,
              messages: historyHits
            }));
            setMessages(historyHits);
          } catch (e) {
            console.error('session chat history error:', e);
          }
        }
      }
    } catch (error) {
      console.error('getChatHistory error:', error);
    }
  };

  const handleHistorySearch = async (keyword: string): Promise<void> => {
    try {
      const res: any = await request({
        method: 'get',
        params: {
          from: 0,
          size: 100,
          query: keyword
        },
        url: '/chat/_history'
      });
      const esResp = res?.data || res;
      const hits = esResp?.hits?.hits || [];
      setChats(hits);
    } catch (error) {
      console.error('getChatHistory search error:', error);
    }
  };

  const handleHistoryRemove = async (chatId: string): Promise<void> => {
    try {
      await request({
        method: 'delete',
        url: `/chat/${chatId}`
      });
      setChats((prev) => prev.filter((chat) => chat._id !== chatId));
      if (activeChat?._id === chatId) {
        setActiveChat(undefined);
        setMessages([]);
      }
    } catch (error) {
      console.error('delete chat error:', error);
    }
  };

  const handleHistoryRename = async (
    chatId: string,
    title: string
  ): Promise<void> => {
    try {
      await request({
        method: 'put',
        data: {
          title
        },
        url: `/chat/${chatId}`
      });

      setChats((prev) =>
        prev.map((chat) =>
          chat._id === chatId
            ? { ...chat, _source: { ...chat._source, title } }
            : chat
        )
      );

      if (activeChat?._id === chatId) {
        setActiveChat((prev: any) => {
          if (!prev) return prev;
          return { ...prev, _source: { ...prev._source, title } };
        });
      }
    } catch (error) {
      console.error('rename chat error:', error);
    }
  };

  const handleHistorySelect = async (chat: any): Promise<void> => {
    const chatId = chat?._id || chat;
    if (!chatId) return;
    try {
      const res: any = await request({
        method: 'get',
        params: {
          from: 0,
          size: 1000
        },
        url: `/chat/${chatId}/_history`
      });

      const esResp = res?.data || res;
      const hits = esResp?.hits?.hits || [];
      const updatedChat = {
        ...(typeof chat === 'object'
          ? chat
          : chats.find((item) => item._id === chatId)),
        messages: hits
      };

      setActiveChat(updatedChat);
      setMessages(hits);
    } catch (error) {
      console.error('session chat history error:', error);
    }
  };

  const handleSendMessage = async (content: string): Promise<void> => {
    const message = content?.trim();
    if (!message) return;

    const now = new Date().toISOString();
    const userMessage = {
      _id: `user-${Date.now()}`,
      _source: {
        type: 'user',
        message,
        question: message,
        created: now,
        user: { username: 'User' }
      }
    };

    setActiveChat((prev: any) => {
      if (prev && (prev._id || prev?._source?.id)) {
        const prevMessages = prev.messages || [];
        return {
          ...prev,
          messages: [...prevMessages, userMessage]
        };
      }

      const baseId = prev?._source?.id || prev?._id || 'chat';
      const prevMessages = prev?.messages || [];

      return {
        _id: prev?._id || '',
        _source: {
          ...(prev?._source || {}),
          id: baseId
        },
        messages: [...prevMessages, userMessage]
      };
    });

    setMessages((prev) => [...prev, userMessage]);

    setQuestion(message);
  };

  const handleNewChat = (): void => {
    setMessages([]);
    setActiveChat(undefined);
    setQuestion('');
    activeChatRef.current = null;
  };

  const getAssistantQueryParams = (options?: {
    keyword?: string;
    page?: number;
    pageSize?: number;
  }) => {
    const keyword = options?.keyword ?? assistantKeyword;
    const page = options?.page ?? assistantPagination.current;
    const size = options?.pageSize ?? assistantPagination.pageSize;
    const from = (page - 1) * size;
    return { keyword, page, size, from };
  };

  const parseAssistantResponse = (res: any) => {
    const esResp = res?.data || res;
    const hits = esResp?.hits?.hits || [];
    const totalRaw = esResp?.hits?.total;

    let total = 0;
    if (typeof totalRaw === 'object' && totalRaw !== null) {
      total = totalRaw.value ?? 0;
    } else if (typeof totalRaw === 'number') {
      total = totalRaw;
    }

    return { hits, total };
  };

  const fetchAssistants = async (options?: {
    keyword?: string;
    page?: number;
    pageSize?: number;
  }): Promise<void> => {
    try {
      const { keyword, page, size, from } = getAssistantQueryParams(options);
      const res: any = await searchAssistant({
        from,
        size,
        query: keyword,
        filter: {
          enabled: [true]
        }
      });

      const { hits, total } = parseAssistantResponse(res);

      setAssistants(hits);
      setAssistantPagination({
        current: page,
        pageSize: size,
        total
      });
      if (!currentAssistant && hits.length > 0) {
        setCurrentAssistant(hits[0]);
      }
    } catch (error) {
      console.error('assistant search error:', error);
    }
  };

  const handleAssistantSelect = (assistant: any): void => {
    setCurrentAssistant(assistant);
  };

  const handleAssistantRefresh = async (): Promise<void> => {
    await fetchAssistants({
      keyword: assistantKeyword,
      page: 1
    });
  };

  const handleAssistantSearch = async (keyword: string): Promise<void> => {
    setAssistantKeyword(keyword);
    await fetchAssistants({
      keyword,
      page: 1
    });
  };

  const handleAssistantPrevPage = async (): Promise<void> => {
    const nextPage = Math.max(1, assistantPagination.current - 1);
    if (nextPage === assistantPagination.current) return;
    await fetchAssistants({
      keyword: assistantKeyword,
      page: nextPage
    });
  };

  const handleAssistantNextPage = async (): Promise<void> => {
    const totalPages =
      assistantPagination.total > 0
        ? Math.ceil(assistantPagination.total / assistantPagination.pageSize)
        : 1;
    const nextPage = Math.min(totalPages, assistantPagination.current + 1);
    if (nextPage === assistantPagination.current) return;
    await fetchAssistants({
      keyword: assistantKeyword,
      page: nextPage
    });
  };

  useEffect(() => {
    fetchChatHistory();
    fetchAssistants();
  }, []);

  // 构建 componentProps，参考 Fullscreen.jsx 的结构
  const componentProps = {
    id: 'dev-ui-search',
    shadow: null,
    theme: 'light',
    language: 'zh-CN',
    logo: {
      // light: "/favicon.ico",
      // "light_mobile": "/favicon.ico",
    },
    placeholder: '搜索任何内容...',
    welcome:
      '欢迎使用 UI Search 开发环境！您可以在这里测试搜索功能和 AI 助手。',
    aiOverview: {
      enabled: true,
      showActions: true,
      assistant: 'dev-assistant',
      title: 'AI 概览',
      height: 400
    },
    widgets: [],
    messages,
    Question,
    onSendMessage: handleSendMessage,
    onNewChat: handleNewChat,
    // History props
    chats,
    activeChat,
    onHistorySelect: handleHistorySelect,
    onHistorySearch: handleHistorySearch,
    onHistoryRefresh: fetchChatHistory,
    onHistoryRename: handleHistoryRename,
    onHistoryRemove: handleHistoryRemove,
    assistants,
    currentAssistant,
    assistantPage: assistantPagination.current,
    assistantTotal: assistantPagination.total,
    onAssistantRefresh: handleAssistantRefresh,
    onAssistantSelect: handleAssistantSelect,
    onAssistantSearch: handleAssistantSearch,
    onAssistantPrevPage: handleAssistantPrevPage,
    onAssistantNextPage: handleAssistantNextPage,
    config: {
      aggregations: {
        'source.id': {
          displayName: 'source'
        },
        lang: {
          displayName: 'language'
        },
        category: {
          displayName: 'category'
        },
        type: {
          displayName: 'type'
        }
      }
    },
    apiConfig: {
      createChatUrl: '/chat/_create',
      continueChatUrl: '/chat/{id}/_chat',
      assistantId: currentAssistant?._id || ''
    }
  };

  const queryParamsProps = enableQueryParams
    ? {
        queryParams,
        setQueryParams
      }
    : {
        queryParams: queryParamsState,
        setQueryParams: setQueryParamsState
      };

  return (
    <FullscreenPage
      {...componentProps}
      {...queryParamsProps}
      enableQueryParams={enableQueryParams}
    />
  );
}
