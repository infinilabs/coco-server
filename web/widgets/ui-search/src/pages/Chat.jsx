import {
  History,
  Chat as AIChat,
  AssistantList,
  ChatInput,
} from "@infinilabs/ai-chat";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';

import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";

export default function Chat({
  commonProps,
  language,
  apiConfig,
  onBackToSearch,
  queryParams,
  setQueryParams,
}) {
  const { BaseUrl, Token, endpoint, headers } = apiConfig || {};

  const chatRef = useRef(null);
  const { t } = useTranslation();

  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const [inputValue, setInputValue] = useState("");

  // continue chat
  const processedQuery = useRef(null);
  useEffect(() => {
    const query = queryParams?.query;
    if (query && query !== processedQuery.current && chatRef.current) {
      processedQuery.current = query;
      chatRef.current?.init({ message: query });
    }
  }, [queryParams?.query]);

  const onSendMessage = async (params) => {
    if (chatRef.current) {
      chatRef.current.init(params);
    }
  };

  const handleNewChat = () => {
    if (chatRef.current) {
      chatRef.current.clearChat();
    }
  };

  return (
    <ChatLayout
      {...commonProps}
      content={
        <AIChat
          ref={chatRef}
          BaseUrl={BaseUrl}
          formatUrl={(data) => {
            if (!data.url) return "";
            if (data.url.startsWith("http")) {
              return data.url;
            }
            return `${BaseUrl}${endpoint}${data.url}`;
          }}
          Token={Token}
          headers={headers}
          locale={language === "zh-CN" ? "zh" : "en"}
        />
      }
      input={
        <ChatInput
          locale={language === "zh-CN" ? "zh" : "en"}
          inputValue={inputValue}
          onSend={onSendMessage}
          changeInput={setInputValue}
          chatPlaceholder={t('labels.inputPlaceholder')}
        />
      }
      sidebarCollapsed={!isHistoryOpen}
      header={
        <ChatHeader
          onNewChat={handleNewChat}
          onToggleHistory={() => setIsHistoryOpen((open) => !open)}
          AssistantList={
            <AssistantList
              BaseUrl={BaseUrl}
              Token={Token}
              headers={headers}
              locale={language === "zh-CN" ? "zh" : "en"}
            />
          }
          onBackToSearch={onBackToSearch}
        />
      }
      sidebar={
        <History
          BaseUrl={BaseUrl}
          Token={Token}
          headers={headers}
          locale={language === "zh-CN" ? "zh" : "en"}
        />
      }
    />
  );
}
