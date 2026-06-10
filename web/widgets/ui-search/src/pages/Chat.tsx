import {
  History,
  Chat as AIChat,
  AssistantList,
  ChatInput,
} from "../AIChat";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';

import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";

interface ChatProps {
  commonProps?: Record<string, any>;
  apiConfig?: Record<string, any>;
  onBackToSearch?: () => void;
  defaultParams?: Record<string, any>;
  setDefaultParams?: (params: any) => void;
  setAttachments?: (attachments: any[]) => void;
  [key: string]: any;
}

export default function Chat({
  commonProps,
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

  useEffect(() => {
    return () => {
      chatRef.current?.clearChat();
    }
  }, [])

  // continue chat
  const processedParams = useRef<Record<string, any> | null>(null);
  useEffect(() => {
    if (JSON.stringify(defaultParams) !== JSON.stringify(processedParams.current)) {
      processedParams.current = defaultParams || null;
      if ((!defaultParams?.session_id || !defaultParams?.session_id.trim()) 
        && !defaultParams?.assistant_id
      ) {
        return;
      }
      chatRef.current?.openChat({ 
        session_id: defaultParams?.session_id || '',
        assistant_id: defaultParams?.assistant_id
      });
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

  return (
    <ChatLayout
      {...commonProps}
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
            return `${BaseUrl}${endpoint}${data.url}`;
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
            chatRef.current?.clearChat();
            onBackToSearch?.();
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
