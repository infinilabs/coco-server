import { useRef, useEffect, useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { ChatMessage, type ChatMessageRef } from "./ChatMessage/components";
import type { IChatMessage } from "./ChatMessage/components";
import { Post } from "./api/axiosRequest";

import { Greetings } from "./Greetings";
import { VMsgList, useScrollManager, ScrollToBottomBtn } from "./VMsgList";
import type { Chat, IChunkData } from "./types/chat";
import { useConnectStore } from "./stores/connectStore";
import { useChatStore, type Assistant } from "./stores/chatStore";
import { SendMessageParams } from "./Chat";
import { DeepResearchDrawerProvider } from "./ChatMessage/components/DeepResearch/DeepResearchDrawerContext";

export interface ActiveChatMessageProps {
  activeMessageRef?: React.RefObject<ChatMessageRef>;
  activeChat?: Chat;
  curChatEnd: boolean;
  Question: string;
  handleSendMessage: (params: SendMessageParams) => void;
  formatUrl?: (data: IChunkData) => string;
  requestHeaders?: Record<string, string>;
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
  requestHeaders,
  assistantList,
  currentAssistant,
  theme,
  t,
  onCancel,
}: ActiveChatMessageProps) => {
  const allMessages = activeChat?.messages || [];
  const replyMessage = [...allMessages]
    .reverse()
    .find((item) => item?._source?.type === "user");
  
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
      replyMessage={replyMessage}
      onResend={handleSendMessage}
      onCancel={onCancel}
      isTyping={!curChatEnd}
      formatUrl={formatUrl}
      requestHeaders={requestHeaders}
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
  handleSendMessage: (params: SendMessageParams) => void;
  getFileUrl: (path: string) => string;
  formatUrl?: (data: IChunkData) => string;
  requestHeaders?: Record<string, string>;
  curIdRef: React.MutableRefObject<string>;
  t?: TFunction;
  currentAssistant?: Assistant;
  theme?: string;
  isMobile?: boolean;
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
  requestHeaders,
  t: tProp,
  theme,
  isMobile,
  onCancel,
}: ChatContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const attachmentCacheRef = useRef<Map<string, { _id: string; _source: Record<string, unknown> }>>(new Map());
  const ATTACHMENT_CACHE_MAX = 1000;
  const storeAttachmentCache = useChatStore((state) => state.attachmentCache);

  const fetchAttachments = useCallback(async (ids: string[]) => {
    const cache = attachmentCacheRef.current;

    if (cache.size >= ATTACHMENT_CACHE_MAX) {
      cache.clear();
    }

    const cached: { _id: string; _source: Record<string, unknown> }[] = [];
    const uncachedIds: string[] = [];

    for (const id of ids) {
      const hit = cache.get(id) || storeAttachmentCache.get(id);
      if (hit) {
        cached.push(hit);
        cache.set(id, hit);
      } else {
        uncachedIds.push(id);
      }
    }

    if (uncachedIds.length === 0) return cached;

    const [, res] = await Post<{ hits?: { hits?: unknown[] } }>(
      "/attachment/_search",
      { attachments: uncachedIds },
    );
    const freshHits = (res?.hits?.hits ?? []) as { _id: string; _source: Record<string, unknown> }[];

    for (const hit of freshHits) {
      cache.set(hit._id, hit);
    }

    // Return results in the same order as the input ids
    return ids.map((id) => cache.get(id)).filter(Boolean) as { _id: string; _source: Record<string, unknown> }[];
  }, [storeAttachmentCache]);

  const setCurrentSessionId = useConnectStore(
    (state) => state.setCurrentSessionId
  );

  const curChatEnd = useChatStore((state) => state.curChatEnd);
  const assistantList = useChatStore((state) => state.assistantList);
  const currentAssistant = useChatStore((state) => state.currentAssistant);

  const scrollRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Strategy 3 & 4: Scroll manager with anchoring and bidirectional lock
  const {
    isAtBottom,
    scrollToBottom,
    resetScrollState,
  } = useScrollManager(scrollRef);

  const [prevChatId, setPrevChatId] = useState(activeChat?._id);

  if (activeChat?._id !== prevChatId) {
    setPrevChatId(activeChat?._id);
    resetScrollState();
  }

  useEffect(() => {
    setCurrentSessionId(activeChat?._id);
  }, [activeChat?._id, setCurrentSessionId]);

  useEffect(() => {
    scrollToBottom(true);
  }, [activeChat?._id, scrollToBottom]);

  // When new messages arrive, only scroll if not manually scrolled away
  useEffect(() => {
    scrollToBottom(false);
  }, [activeChat?.messages?.length, scrollToBottom]);

  // When a new generation starts (curChatEnd: true → false), force scroll to bottom
  // This covers both ChatInput send and resend/retry paths
  const prevCurChatEndRef = useRef(curChatEnd);
  useEffect(() => {
    if (prevCurChatEndRef.current && !curChatEnd) {
      // Generation just started
      resetScrollState();
      scrollToBottom(true);
    }
    prevCurChatEndRef.current = curChatEnd;
  }, [curChatEnd, resetScrollState, scrollToBottom]);

  return (
    <DeepResearchDrawerProvider isMobile={isMobile} chatId={activeChat?._id}>
    <div className="flex-1 overflow-hidden flex flex-col justify-between relative">
      <div
        ref={scrollRef}
        className="flex-1 w-full overflow-x-hidden overflow-y-auto relative px-4"
        style={{ overflowAnchor: "auto" }}
      >
        <div className="max-w-4xl mx-auto">
          {(!activeChat || activeChat?.messages?.length === 0) && (
            <Greetings t={t} />
          )}

          {/* Node Consolidation: groups of off-screen messages
              are collapsed into single spacer divs */}
          <VMsgList
            messages={activeChat?.messages || []}
            scrollRoot={scrollRef.current}
            renderMessage={(message) => {
              const msg = message as unknown as IChatMessage;
              const userMessage = msg._source?.type !== "user"
                ? activeChat?.messages?.find(
                    (m) => m._id === msg._source?.reply_to_message
                  )
                : undefined;

              return (
                <ChatMessage
                  message={msg}
                  replyMessage={userMessage}
                  isTyping={false}
                  onResend={handleSendMessage}
                  onCancel={onCancel}
                  formatUrl={formatUrl}
                  requestHeaders={requestHeaders}
                  assistantList={assistantList}
                  fetchAttachments={fetchAttachments}
                  theme={theme as any}
                  t={t}
                />
              );
            }}
          />

          {/* Strategy 1: Active streaming message - isolated from history, 
              rendered as plain DOM to avoid virtual list computation overhead */}
          {!curChatEnd && (
            <div data-message-id="current" style={{ overflowAnchor: "none" }}>
              <ActiveChatMessage
                key={activeMessageGen}
                activeMessageRef={activeMessageRef}
                activeChat={activeChat}
                curChatEnd={curChatEnd}
                Question={Question}
                handleSendMessage={handleSendMessage}
                formatUrl={formatUrl}
                requestHeaders={requestHeaders}
                assistantList={assistantList}
                currentAssistant={currentAssistant}
                theme={theme}
                t={t}
                onCancel={onCancel}
              />
              {/* Bottom spacer: absorbs height jumps from streaming text reflow */}
              <div style={{ height: 80, flexShrink: 0 }} aria-hidden />
            </div>
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

      <ScrollToBottomBtn
        scrollRef={scrollRef}
        isAtBottom={isAtBottom}
        onScrollToBottom={() => scrollToBottom(true)}
      />
    </div>
    </DeepResearchDrawerProvider>
  );
};
