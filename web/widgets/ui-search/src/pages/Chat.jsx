import {
  History,
  Chat as AIChat,
  AssistantList,
  ChatInput,
} from "@infinilabs/ai-chat";
import { useEffect, useRef, useState } from "react";

import ChatHeader from "../ChatHeader";
import ChatLayout from "../Layout/ChatLayout";

export default function Chat({
  commonProps,
  language,
  apiConfig,
  onBackToSearch,
  queryParams,
}) {
  const { BaseUrl, Token, endpoint } = apiConfig || {};

  const chatRef = useRef(null);

  const [isHistoryOpen, setIsHistoryOpen] = useState(true);
  const [inputValue, setInputValue] = useState("");

  // continue chat
  const processedQuery = useRef(null);
  useEffect(() => {
    const query = queryParams?.query;
    if (query && query !== processedQuery.current && chatRef.current) {
      processedQuery.current = query;
      // chatRef.current.clearChat();
      // setTimeout(() => {
      //   chatRef.current.init({
      //     message: query,
      //   });
      // }, 300);
      setInputValue(query);
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
          formatUrl={(data) => `${endpoint}${BaseUrl}${data.url}`}
          Token={Token}
          locale={language === "zh-CN" ? "zh" : "en"}
        />
      }
      input={
        <ChatInput
          locale={language === "zh-CN" ? "zh" : "en"}
          onSend={onSendMessage}
          inputValue={inputValue}
          changeInput={setInputValue}
          chatPlaceholder={
            language === "zh-CN" ? "请输入问题..." : "Type a message..."
          }
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
          locale={language === "zh-CN" ? "zh" : "en"}
        />
      }
    />
  );
}
