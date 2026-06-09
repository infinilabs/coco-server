import { useCallback, useEffect, useState } from "react";
import { I18nextProvider, useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { message } from "antd";

import type { Session } from "./types/chat";
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

  const sessions = useChatStore((s) => s.sessions);
  const setSessions = useChatStore((s) => s.setSessions);
  const activeSessionId = useChatStore((s) => s.activeSessionId);
  const setActiveSessionId = useChatStore((s) => s.setActiveSessionId);
  const setActiveSessionSource = useChatStore((s) => s.setActiveSessionSource);
  const historyVersion = useChatStore((s) => s.historyVersion);

  const [messageApi, contextHolder] = message.useMessage();
  const [keyword, setKeyword] = useState("");
  const [renamingId, setRenamingId] = useState<string>("");
  const [deletingId, setDeletingId] = useState<string>("");

  useEffect(() => {
    if (locale && i18n.language !== locale) {
      i18n.changeLanguage(locale);
    }
  }, [locale]);

  useEffect(() => {
    if (!BaseUrl) return;
    const store = { state: { endpoint_http: BaseUrl } };
    localStorage.setItem("app-store", JSON.stringify(store));
    const mergedHeaders: Record<string, string> = { ...headersProp };
    if (Token) mergedHeaders["X-API-TOKEN"] = Token;
    if (Object.keys(mergedHeaders).length > 0) {
      localStorage.setItem("headers", JSON.stringify(mergedHeaders));
    }
  }, [BaseUrl, Token, headersProp]);

  const fetchChatHistory = useCallback(async () => {
    try {
      const [err, res] = await Get<{
        hits?: { hits?: Session[] };
      }>("/chat/_history", { from: 0, size: 100, keyword }, undefined, headersProp);
      if (err) return;
      const hits = (res?.hits?.hits as Session[] | undefined) || [];
      setSessions(hits);
    } catch (e) {
      console.error(e);
    }
  }, [keyword, setSessions, headersProp]);

  useEffect(() => { fetchChatHistory(); }, [fetchChatHistory]);
  useEffect(() => { if (historyVersion > 0) fetchChatHistory(); }, [historyVersion, fetchChatHistory]);

  const onSelect = useCallback(
    (session: Session) => {
      setActiveSessionId(session._id);
      setActiveSessionSource(session._source);
    },
    [setActiveSessionId, setActiveSessionSource],
  );

  const onRename = useCallback(
    async (chatId: string, title: string) => {
      try {
        setRenamingId(chatId);
        const [err] = await Put(`/chat/${chatId}`, { title }, {}, headersProp);
        if (err) {
          messageApi.open({ key: "rename", type: "error", content: t("history_list.operate.rename_error") });
          return;
        }
        await fetchChatHistory();
      } catch (e) { console.error(e); }
      finally { setRenamingId(""); }
    },
    [messageApi, t, fetchChatHistory, headersProp],
  );

  const onRemove = useCallback(
    async (chatId: string) => {
      try {
        setDeletingId(chatId);
        const [err] = await Delete(`/chat/${chatId}`, {}, headersProp);
        if (err) {
          messageApi.open({ key: "delete", type: "error", content: t("history_list.operate.delete_error") });
          return;
        }
        if (activeSessionId === chatId) {
          setActiveSessionId(undefined);
          setActiveSessionSource(undefined);
        }
        await fetchChatHistory();
      } catch (e) { console.error(e); }
      finally { setDeletingId(""); }
    },
    [messageApi, t, fetchChatHistory, headersProp, activeSessionId, setActiveSessionId, setActiveSessionSource],
  );

  // Adapt to HistoryList's expected interface
  const activeChat = activeSessionId ? { _id: activeSessionId } : undefined;

  return (
    <>
      {contextHolder}
      <HistoryList
        historyPanelId="history-panel"
        chats={sessions}
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
