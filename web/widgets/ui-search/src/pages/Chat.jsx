import {
  History,
  Chat as AIChat,
  AssistantList,
  ChatInput,
} from "@infinilabs/ai-chat";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';

import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";

export default function Chat({
  commonProps,
  apiConfig,
  onBackToSearch,
  defaultParams,
  setDefaultParams,
  setAttachments
}) {
  const { BaseUrl, Token, endpoint, headers } = apiConfig || {};
  const { language } = commonProps || {};

  const chatRef = useRef(null);
  const { t } = useTranslation();

  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const [inputValue, setInputValue] = useState("");

  // continue chat
  const processedParams = useRef(null);
  useEffect(() => {
    if (JSON.stringify(defaultParams) !== JSON.stringify(processedParams.current)) {
      processedParams.current = defaultParams;
      if ((!defaultParams?.query || !defaultParams?.query.trim()) 
        && defaultParams?.attachments?.length === 0
        && !defaultParams?.assistant_id
      ) {
        return;
      }
      chatRef.current?.init({ 
        message: defaultParams.query || '',
        attachments: (defaultParams.attachments || [])
        .filter((a) => a.status === "uploaded" && a.id)
        .map((a) => a.id), 
        assistant_id: defaultParams.assistant_id
      });
      setDefaultParams({})
      setAttachments([]);
    }
  }, [JSON.stringify(defaultParams)]);

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

  const locale = useMemo(() => {
    return language === "zh-CN" ? "zh" : "en"
  }, [language]);

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
            chatRef.current.cancelChat();
          }}
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
            chatRef.current.clearChat();
            onBackToSearch();
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
