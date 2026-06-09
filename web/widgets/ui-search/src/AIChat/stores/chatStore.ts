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

export type IChatStore = {
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
};

export const useChatStore = create<IChatStore>()(
  (set) => ({
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
