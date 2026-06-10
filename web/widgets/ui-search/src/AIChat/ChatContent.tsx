import { useRef, useEffect, useState, useCallback } from "react";
import type { UIEvent } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { ChatMessage, type ChatMessageRef } from "./ChatMessage/components";
import { Post } from "./api/axiosRequest";

import { Greetings } from "./Greetings";
import { useChatScroll } from "./hooks/useChatScroll";
import type { Chat, IChunkData } from "./types/chat";
import { useConnectStore } from "./stores/connectStore";
import ScrollToBottom from "./Common/ScrollToBottom";
import { useChatStore, type Assistant } from "./stores/chatStore";

export interface ActiveChatMessageProps {
  activeMessageRef?: React.RefObject<ChatMessageRef>;
  activeChat?: Chat;
  curChatEnd: boolean;
  Question: string;
  handleSendMessage: (content: string, newChat?: Chat) => void;
  formatUrl?: (data: IChunkData) => string;
  assistantList?: Assistant[];
  currentAssistant?: Assistant;
  theme?: string;
  t?: TFunction;
  onCancel?: () => void;
}

export const ActiveChatMessage = ({
  activeMessageRef,
  activeChat,
  curChatEnd,
  Question,
  handleSendMessage,
  formatUrl,
  assistantList,
  currentAssistant,
  theme,
  t,
  onCancel,
}: ActiveChatMessageProps) => {
  const allMessages = activeChat?.messages || [];

  return (
    <ChatMessage
      key={"current"}
      ref={activeMessageRef}
      message={{
        _id: "current",
        _source: {
          type: "assistant",
          assistant_id:
            allMessages[allMessages.length - 1]?._source?.assistant_id,
          message: "",
          question: Question,
        },
      }}
      onResend={handleSendMessage}
      onCancel={onCancel}
      isTyping={!curChatEnd}
      formatUrl={formatUrl}
      assistantList={assistantList}
      currentAssistant={currentAssistant}
      theme={theme as any}
      t={t}
    />
  );
};

interface ChatContentProps {
  activeChat?: Chat;
  activeMessageRef?: React.RefObject<ChatMessageRef>;
  activeMessageGen?: number;
  timedoutShow: boolean;
  Question: string;
  handleSendMessage: (content: string, newChat?: Chat) => void;
  getFileUrl: (path: string) => string;
  formatUrl?: (data: IChunkData) => string;
  curIdRef: React.MutableRefObject<string>;
  t?: TFunction;
  currentAssistant?: Assistant;
  theme?: string;
  onCancel?: () => void;
}

export const ChatContent = ({
  activeChat,
  activeMessageRef,
  activeMessageGen = 0,
  timedoutShow,
  Question,
  handleSendMessage,
  formatUrl,
  t: tProp,
  theme,
  onCancel,
}: ChatContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const fetchAttachments = useCallback(async (ids: string[]) => {
    const [, res] = await Post<{ hits?: { hits?: unknown[] } }>(
      "/attachment/_search",
      { attachments: ids },
    );
    return (res?.hits?.hits ?? []) as { _id: string; _source: Record<string, unknown> }[];
  }, []);

  const setCurrentSessionId = useConnectStore(
    (state) => state.setCurrentSessionId
  );

  const curChatEnd = useChatStore((state) => state.curChatEnd);
  const assistantList = useChatStore((state) => state.assistantList);
  const currentAssistant = useChatStore((state) => state.currentAssistant);

  const scrollRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const { scrollToBottom, resetUserScrolling } = useChatScroll(scrollRef);

  const [isAtBottom, setIsAtBottom] = useState(true);
  const [prevChatId, setPrevChatId] = useState(activeChat?._id);

  if (activeChat?._id !== prevChatId) {
    setPrevChatId(activeChat?._id);
    setIsAtBottom(true);
    resetUserScrolling();
  }

  useEffect(() => {
    setCurrentSessionId(activeChat?._id);
  }, [activeChat?._id, setCurrentSessionId]);

  useEffect(() => {
    scrollToBottom(true);
  }, [activeChat?._id, activeChat?.messages?.length, scrollToBottom]);

  useEffect(() => {
    return () => {
      scrollToBottom.cancel();
    };
  }, [scrollToBottom]);

  const handleScroll = (event: UIEvent<HTMLDivElement>) => {
    const { scrollHeight, scrollTop, clientHeight } =
      event.currentTarget as HTMLDivElement;

    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;

    setIsAtBottom(isAtBottom);
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col justify-between relative">
      <div
        ref={scrollRef}
        className="flex-1 w-full overflow-x-hidden overflow-y-auto relative px-4"
        onScroll={handleScroll}
      >
        <div className="max-w-4xl mx-auto">
          {(!activeChat || activeChat?.messages?.length === 0) && (
            <Greetings t={t} />
          )}

          {activeChat?.messages?.map((message) => (
            <ChatMessage
              key={message._id}
              message={message}
              isTyping={false}
              onResend={handleSendMessage}
              onCancel={onCancel}
              formatUrl={formatUrl}
              assistantList={assistantList}
              fetchAttachments={fetchAttachments}
              theme={theme as any}
              t={t}
            />
          ))}

          {(activeChat || !curChatEnd) && (
            <ActiveChatMessage
              key={activeMessageGen}
              activeMessageRef={activeMessageRef}
              activeChat={activeChat}
              curChatEnd={curChatEnd}
              Question={Question}
              handleSendMessage={handleSendMessage}
              formatUrl={formatUrl}
              assistantList={assistantList}
              currentAssistant={currentAssistant}
              theme={theme}
              t={t}
              onCancel={onCancel}
            />
          )}

          {timedoutShow ? (
            <ChatMessage
              key={"timedout"}
              message={{
                _id: "timedout",
                _source: {
                  type: "assistant",
                  message: t("assistant.chat.timedout"),
                  question: Question,
                },
              }}
              onResend={handleSendMessage}
              isTyping={false}
              theme={theme as any}
              t={t}
            />
          ) : null}
          <div ref={messagesEndRef} />
        </div>

      </div>

      <ScrollToBottom scrollRef={scrollRef} isAtBottom={isAtBottom} />
    </div>
  );
};
