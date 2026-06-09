import { create } from "zustand";

import type { ChatItem, Session } from "../types/chat";

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
  // Streaming state
  isStreaming: boolean;
  setIsStreaming: (value: boolean) => void;

  // Session
  activeSessionId: string | undefined;
  setActiveSessionId: (id: string | undefined) => void;
  activeSessionSource: Session["_source"];
  setActiveSessionSource: (source: Session["_source"]) => void;

  // Session list (for history panel)
  sessions: Session[];
  setSessions: (sessions: Session[]) => void;

  // Chat items (the render state, driven directly by chunks)
  items: ChatItem[];
  setItems: (items: ChatItem[]) => void;
  pushItem: (item: ChatItem) => void;
  updateLastItem: (updater: (item: ChatItem) => ChatItem) => void;

  // Assistant
  currentAssistant?: Assistant;
  setCurrentAssistant: (assistant?: Assistant) => void;
  assistantList?: Assistant[];
  setAssistantList: (assistantList: Assistant[]) => void;

  // History panel version counter (triggers re-fetch of session list)
  historyVersion: number;
  incrementHistoryVersion: () => void;
};

export const useChatStore = create<IChatStore>()((set) => ({
  isStreaming: false,
  setIsStreaming: (value: boolean) => set({ isStreaming: value }),

  activeSessionId: undefined,
  setActiveSessionId: (id) => set({ activeSessionId: id }),
  activeSessionSource: undefined,
  setActiveSessionSource: (source) => set({ activeSessionSource: source }),

  sessions: [],
  setSessions: (sessions) => set({ sessions }),

  items: [],
  setItems: (items) => set({ items }),
  pushItem: (item) => set((state) => ({ items: [...state.items, item] })),
  updateLastItem: (updater) =>
    set((state) => {
      const items = [...state.items];
      const last = items[items.length - 1];
      if (!last) return {};
      items[items.length - 1] = updater(last);
      return { items };
    }),

  currentAssistant: undefined,
  setCurrentAssistant: (assistant) => set({ currentAssistant: assistant }),
  assistantList: [],
  setAssistantList: (assistantList) => set({ assistantList }),

  historyVersion: 0,
  incrementHistoryVersion: () =>
    set((state) => ({ historyVersion: state.historyVersion + 1 })),
}));
