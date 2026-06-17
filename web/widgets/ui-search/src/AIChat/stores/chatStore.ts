import { create } from "zustand";

interface SynthesizeItem {
  id: string;
  content: string;
}

import type { Chat, ChatMessageSource } from "../types/chat";

export interface AssistantSource {
  name?: string;
  icon?: string;
  [key: string]: unknown;
}

export interface Assistant {
  _id: string;
  _source?: AssistantSource;
}

export type AttachmentCacheItem = { _id: string; _source: Record<string, unknown> };

export type IChatStore = {
  baseUrl: string;
  setBaseUrl: (value: string) => void;
  authHeaders: Record<string, string>;
  setAuthHeaders: (value: Record<string, string>) => void;
  curChatEnd: boolean;
  setCurChatEnd: (value: boolean) => void;
  stopChat: boolean;
  setStopChat: (value: boolean) => void;
  connected: boolean;
  setConnected: (value: boolean) => void;
  messages: string;
  setMessages: (value: string | ((prev: string) => string)) => void;
  synthesizeItem?: SynthesizeItem;
  setSynthesizeItem: (synthesizeItem?: SynthesizeItem) => void;
  chats: Chat[];
  setChats: (chats: Chat[]) => void;
  activeChat?: Chat;
  setActiveChat: (chat?: Chat) => void;
  currentAssistant?: Assistant;
  setCurrentAssistant: (assistant?: Assistant) => void;
  assistantList?: Assistant[];
  setAssistantList: (assistantList: Assistant[]) => void;
  updateLastMessage: (updates: Partial<ChatMessageSource>) => void;
  historyVersion: number;
  incrementHistoryVersion: () => void;
  attachmentCache: Map<string, AttachmentCacheItem>;
  cacheAttachments: (items: AttachmentCacheItem[]) => void;
  onSelectChatHandler?: (chat?: Chat) => void;
  setOnSelectChatHandler: (handler: (chat?: Chat) => void) => void;
  appendFeatureParams?: (params: Record<string, any>) => void;
  setAppendFeatureParams: (fn: (params: Record<string, any>) => void) => void;
  deepResearchDrawerOpen: boolean;
  setDeepResearchDrawerOpen: (value: boolean) => void;
};

export const useChatStore = create<IChatStore>()(
  (set) => ({
      baseUrl: "",
      setBaseUrl: (value: string) => set(() => ({ baseUrl: value })),
      authHeaders: {},
      setAuthHeaders: (value: Record<string, string>) => set(() => ({ authHeaders: value })),
      curChatEnd: true,
      setCurChatEnd: (value: boolean) => set(() => ({ curChatEnd: value })),
      stopChat: false,
      setStopChat: (value: boolean) => set(() => ({ stopChat: value })),
      connected: false,
      setConnected: (value: boolean) => set(() => ({ connected: value })),
      messages: "",
      setMessages: (value: string | ((prev: string) => string)) =>
        set((state) => ({
          messages: typeof value === "function" ? value(state.messages) : value,
        })),
      setSynthesizeItem: (synthesizeItem?: SynthesizeItem) => {
        return set(() => ({ synthesizeItem }));
      },
      chats: [],
      setChats: (chats: Chat[]) => set(() => ({ chats })),
      activeChat: undefined,
      setActiveChat: (chat?: Chat) => set(() => ({ activeChat: chat })),
      currentAssistant: undefined,
      setCurrentAssistant: (assistant?: Assistant) =>
        set(() => ({ currentAssistant: assistant })),
      assistantList: [],
      setAssistantList: (assistantList: Assistant[]) =>
        set(() => ({ assistantList })),
      historyVersion: 0,
      incrementHistoryVersion: () =>
        set((state) => ({ historyVersion: state.historyVersion + 1 })),
      attachmentCache: new Map(),
      cacheAttachments: (items: AttachmentCacheItem[]) =>
        set((state) => {
          const cache = new Map(state.attachmentCache);
          for (const item of items) {
            cache.set(item._id, item);
          }
          return { attachmentCache: cache };
        }),
      onSelectChatHandler: undefined,
      setOnSelectChatHandler: (handler: (chat?: Chat) => void) =>
        set(() => ({ onSelectChatHandler: handler })),
      appendFeatureParams: undefined,
      setAppendFeatureParams: (fn: (params: Record<string, any>) => void) =>
        set(() => ({ appendFeatureParams: fn })),
      deepResearchDrawerOpen: false,
      setDeepResearchDrawerOpen: (value: boolean) =>
        set(() => ({ deepResearchDrawerOpen: value })),
      updateLastMessage: (updates: Partial<ChatMessageSource>) =>
        set((state) => {
          if (!state.activeChat || !state.activeChat.messages) return {};
          const messages = [...state.activeChat.messages];
          const lastIndex = messages.length - 1;
          if (lastIndex < 0) return {};

          const lastMessage = { ...messages[lastIndex] };
          lastMessage._source = { ...lastMessage._source, ...updates };
          messages[lastIndex] = lastMessage;

          return {
            activeChat: {
              ...state.activeChat,
              messages,
            },
          };
        }),
    }),
);
