import {
  History,
  Chat as AIChat,
  AssistantList,
  ChatInput,
} from "../AIChat";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';

import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";

type BlockerState = "idle" | "blocked";

interface NavigationGuard {
  state: BlockerState;
  proceed: () => void;
  reset: () => void;
}

function useNavigationGuard(shouldBlock: () => boolean): NavigationGuard {
  const [state, setState] = useState<BlockerState>("idle");
  const shouldBlockRef = useRef(shouldBlock);
  shouldBlockRef.current = shouldBlock;
  const allowNextRef = useRef(false);

  const proceed = useCallback(() => {
    setState("idle");
    allowNextRef.current = true;
    history.back();
  }, []);

  const reset = useCallback(() => {
    setState("idle");
  }, []);

  useEffect(() => {
    const handlePopState = () => {
      if (allowNextRef.current) {
        allowNextRef.current = false;
        return;
      }
      if (shouldBlockRef.current()) {
        history.pushState(null, "", location.href);
        setState("blocked");
      }
    };

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (shouldBlockRef.current()) {
        e.preventDefault();
      }
    };

    window.addEventListener("popstate", handlePopState);
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("popstate", handlePopState);
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, []);

  return { state, proceed, reset };
}

interface ChatProps {
  commonProps?: Record<string, any>;
  logo?: Record<string, any>;
  handleLogoClick?: () => void;
  apiConfig?: Record<string, any>;
  onBackToSearch?: () => void;
  defaultParams?: Record<string, any>;
  setDefaultParams?: (params: any) => void;
  setAttachments?: (attachments: any[]) => void;
  [key: string]: any;
}

export default function Chat({
  commonProps,
  logo,
  handleLogoClick,
  apiConfig,
  onBackToSearch,
  defaultParams,
  setDefaultParams,
  setAttachments,
}: ChatProps) {
  const { BaseUrl, Token, endpoint, headers } = apiConfig || {};
  const { language, theme } = commonProps || {};

  const chatRef = useRef<any>(null);
  const { t } = useTranslation();

  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const [inputValue, setInputValue] = useState("");

  const clearChatRef = useRef<((cb?: () => void, force?: boolean) => void) | null>(null);
  useEffect(() => {
    clearChatRef.current = chatRef.current?.clearChat ?? null;
  });

  useEffect(() => {
    const cleanup = () => {
      (clearChatRef.current ?? chatRef.current?.clearChat)?.(undefined, true);
    };
    window.addEventListener("beforeunload", cleanup);
    window.addEventListener("popstate", cleanup);
    return () => {
      window.removeEventListener("beforeunload", cleanup);
      window.removeEventListener("popstate", cleanup);
      cleanup();
    }
  }, [])

  // continue chat
  const processedParams = useRef<Record<string, any> | null>(null);
  useEffect(() => {
    if (JSON.stringify(defaultParams) !== JSON.stringify(processedParams.current)) {
      processedParams.current = defaultParams || null;
      if ((!defaultParams?.session_id || !defaultParams?.session_id.trim()) 
        && (!defaultParams?.query || !defaultParams?.query.trim()) 
        && defaultParams?.attachments?.length === 0
        && !defaultParams?.assistant_id
      ) {
        return;
      }
      if (defaultParams?.session_id) {
        chatRef.current?.openChat({
          session_id: defaultParams?.session_id || '',
          assistant_id: defaultParams?.assistant_id
        });
      } else {
        chatRef.current?.init({ 
          message: defaultParams?.query || '',
          attachments: (defaultParams?.attachments || [])
          .filter((a: any) => a.status === "uploaded" && a.id)
          .map((a: any) => a.id), 
          assistant_id: defaultParams?.assistant_id
        });
      }
      setDefaultParams?.({})
      setAttachments?.([]);
    }
  }, [JSON.stringify(defaultParams)]);

  const onSendMessage = async (params: any) => {
    if (chatRef.current) {
      chatRef.current.init(params);
    }
  };

  const handleNewChat = () => {
    if (chatRef.current) {
      chatRef.current.clearChat();
    }
  };

  const locale = useMemo(() => {
    return language === "zh-CN" ? "zh" : "en"
  }, [language]);

  const isDeepResearchRunning = useCallback(() => {
    const chatInstance = chatRef.current;
    if (!chatInstance) return false;
    return !chatInstance._isChatEnd?.();
  }, []);

  const blocker = useNavigationGuard(() => isDeepResearchRunning());

  useEffect(() => {
    if (blocker.state === "blocked") {
      (clearChatRef.current ?? chatRef.current?.clearChat)?.(
        () => blocker.proceed?.(),
        false,
        () => blocker.reset?.(),
      );
    }
  }, [blocker.state]);

  return (
    <ChatLayout
      {...commonProps}
      logo={logo}
      handleLogoClick={handleLogoClick}
      content={
        <AIChat
          ref={chatRef}
          theme={theme}
          BaseUrl={BaseUrl}
          formatUrl={(data: any) => {
            if (!data.url) return "";
            if (data.url.startsWith("http")) {
              return data.url;
            }
            return `${BaseUrl}${data.url}`;
          }}
          Token={Token}
          headers={headers}
          locale={locale}
          t={t}
        />
      }
      input={
        <ChatInput
          t={t}
          locale={locale}
          inputValue={inputValue}
          onSend={onSendMessage}
          changeInput={setInputValue}
          chatPlaceholder={t('labels.inputPlaceholder')}
          onCancel={() => {
            chatRef.current?.cancelChat();
          }}
          disabled={false}
        />
      }
      sidebarCollapsed={!isHistoryOpen}
      header={
        <ChatHeader
          onNewChat={handleNewChat}
          isHistoryOpen={isHistoryOpen}
          onToggleHistory={() => setIsHistoryOpen((open) => !open)}
          AssistantList={
            <AssistantList
              BaseUrl={BaseUrl}
              Token={Token}
              headers={headers}
              locale={locale}
              t={t}
            />
          }
          onBackToSearch={() => {
            chatRef.current?.clearChat(() => onBackToSearch?.());
          }}
        />
      }
      sidebar={
        <History
          BaseUrl={BaseUrl}
          Token={Token}
          headers={headers}
          locale={locale}
          t={t}
        />
      }
    />
  );
}
