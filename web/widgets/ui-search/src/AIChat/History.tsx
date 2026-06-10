import { useCallback, useEffect, useRef, useState } from "react";
import { I18nextProvider, useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { message } from "antd";

import type { Chat } from "./types/chat";
import { useChatStore } from "./stores/chatStore";
import { Get, Put, Delete } from "./api/axiosRequest";
import HistoryList from "./HistoryList";
import i18n from "../i18n";

interface HistoryProps {
  BaseUrl: string;
  Token?: string;
  headers?: Record<string, string>;
  locale?: string;
  t?: TFunction;
}

function InnerHistory({
  BaseUrl,
  Token,
  headers: headersProp = {},
  locale = "en",
  t: tProp,
}: HistoryProps) {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const chats = useChatStore((state) => state.chats);
  const setChats = useChatStore((state) => state.setChats);
  const activeChat = useChatStore((state) => state.activeChat);
  const setActiveChat = useChatStore((state) => state.setActiveChat);
  const historyVersion = useChatStore((state) => state.historyVersion);
  const setBaseUrl = useChatStore((state) => state.setBaseUrl);
  const setAuthHeaders = useChatStore((state) => state.setAuthHeaders);

  const [messageApi, contextHolder] = message.useMessage();

  const [keyword, setKeyword] = useState("");
  const [renamingId, setRenamingId] = useState<string>("");
  const [deletingId, setDeletingId] = useState<string>("");

  useEffect(() => {
    if (!BaseUrl) {
      return;
    }
    setBaseUrl(BaseUrl);

    const mergedHeaders: Record<string, string> = { ...headersProp };
    if (Token) {
      mergedHeaders["X-API-TOKEN"] = Token;
    }
    if (Object.keys(mergedHeaders).length > 0) {
      setAuthHeaders(mergedHeaders);
    }
  }, [BaseUrl, Token, headersProp, setBaseUrl, setAuthHeaders]);

  const lastFetchTimeRef = useRef(0);

  const fetchChatHistory = useCallback(async () => {
    const now = Date.now();
    if (now - lastFetchTimeRef.current < 500) return;
    lastFetchTimeRef.current = now;
    try {
      const [err, res] = await Get<{
        hits?: { hits?: Chat[] };
      }>("/chat/_history", {
        from: 0,
        size: 100,
        keyword,
      }, undefined, headersProp);
      if (err) {
        return;
      }
      const hits = (res?.hits?.hits as Chat[] | undefined) || [];
      setChats(hits);
    } catch (e) {
      console.error(e);
    }
  }, [keyword, setChats, headersProp]);

  useEffect(() => {
    fetchChatHistory();
  }, [fetchChatHistory, historyVersion]);

  const onSelect = useCallback(
    async (chat: Chat) => {
      setActiveChat(chat);
    },
    [setActiveChat],
  );

  const onRename = useCallback(
    async (chatId: string, title: string) => {
      const key = "rename_message";
      try {
        setRenamingId(chatId);
        const [err] = await Put(`/chat/${chatId}`, { title }, {}, headersProp);
        if (err) {
          messageApi.open({
            key,
            type: "error",
            content: t("history_list.operate.rename_error"),
          });
          return;
        }
        await fetchChatHistory();
      } catch (e) {
        console.error(e);
      } finally {
        setRenamingId("");
      }
    },
    [messageApi, t, fetchChatHistory, headersProp],
  );

  const onRemove = useCallback(
    async (chatId: string) => {
      const key = "delete_message";

      try {
        setDeletingId(chatId);

        const [err] = await Delete(`/chat/${chatId}`, {}, headersProp);
        if (err) {
          messageApi.open({
            key,
            type: "error",
            content: t("history_list.operate.delete_error"),
          });
          return;
        }

        // Clear active chat if the deleted item is the current one
        if (activeChat?._id === chatId) {
          setActiveChat(undefined);
        }

        // Fetch latest history after successful deletion
        await fetchChatHistory();
      } catch (e) {
        console.error(e);

        messageApi.open({
          key,
          type: "error",
          content: t("history_list.operate.delete_error"),
        });
      } finally {
        setDeletingId("");
      }
    },
    [messageApi, t, fetchChatHistory, headersProp, activeChat, setActiveChat],
  );

  return (
    <>
      {contextHolder}
      <HistoryList
        historyPanelId="history-panel"
        chats={chats}
        active={activeChat}
        onSearch={setKeyword}
        onRefresh={fetchChatHistory}
        onSelect={onSelect}
        onRename={onRename}
        onRemove={onRemove}
        renamingId={renamingId}
        deletingId={deletingId}
        t={t}
      />
    </>
  );
}

export function History(props: HistoryProps) {
  return (
    <I18nextProvider i18n={i18n}>
      <InnerHistory {...props} />
    </I18nextProvider>
  );
}
