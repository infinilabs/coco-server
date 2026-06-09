import { useRef, useEffect, useState } from "react";
import type { UIEvent } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from "@infinilabs/markdown";

import { Greetings } from "./Greetings";
import { useChatScroll } from "./hooks/useChatScroll";
import { useChatStore } from "./stores/chatStore";
import ScrollToBottom from "./Common/ScrollToBottom";
import type { ChatItem } from "./types/chat";

function RenderItem({ item, formatUrl: _formatUrl, t }: { item: ChatItem; formatUrl?: (data: any) => string; t: TFunction }) {
  switch (item.type) {
    case "user":
      return (
        <div className="w-full py-4 flex justify-end">
          <div className="max-w-[80%] px-4 py-2 rounded-lg bg-[#f0f0f0] dark:bg-[#2a2a2a] text-sm">
            {item.text}
          </div>
        </div>
      );
    case "assistant":
      return (
        <div className="w-full py-2">
          <div className="cm-markdown prose dark:prose-invert prose-sm max-w-none">
            <Markdown content={item.text} />
          </div>
        </div>
      );

    case "query_intent":
      return (
        <div className="w-full py-1 text-xs text-gray-400">
          <span>{t("chat.query_intent", "Query intent")}: </span>
          <span>{item.text}</span>
        </div>
      );
    case "fetch_source":
      return (
        <div className="w-full py-1 text-xs text-gray-400">
          <span>{t("chat.fetch_source", "Fetching sources...")}</span>
        </div>
      );
    case "pick_source":
      return (
        <div className="w-full py-1 text-xs text-gray-400">
          <span>{t("chat.pick_source", "Selecting sources...")}</span>
        </div>
      );
    case "deep_read":
      return (
        <div className="w-full py-1 text-xs text-gray-400">
          <span>{t("chat.deep_read", "Deep reading...")}</span>
        </div>
      );
    case "tool_call":
      return (
        <div className="w-full py-2 border border-gray-200 dark:border-gray-700 rounded-md text-xs">
          <div className="px-3 py-1 font-semibold border-b border-gray-200 dark:border-gray-700">
            ⚙ {item.toolName}
          </div>
          {item.args && <div className="px-3 py-1 font-mono whitespace-pre-wrap max-h-32 overflow-auto">{item.args}</div>}
          {item.result && <div className="px-3 py-1 border-t border-gray-200 dark:border-gray-700 whitespace-pre-wrap max-h-32 overflow-auto">{item.result}</div>}
        </div>
      );
    case "deep_research":
      return (
        <div className="w-full py-1 text-xs text-gray-400">
          <span>{t("chat.deep_research", "Researching...")} ({item.chunks.length} steps)</span>
        </div>
      );
    case "payload":
      return null;
    default:
      return null;
  }
}

interface ChatContentProps {
  timedoutShow: boolean;
  handleSendMessage: (content: string) => void;
  formatUrl?: (data: any) => string;
  t?: TFunction;
}

export const ChatContent = ({
  timedoutShow,
  handleSendMessage: _handleSendMessage,
  formatUrl,
  t: tProp,
}: ChatContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const items = useChatStore((s) => s.items);
  const isStreaming = useChatStore((s) => s.isStreaming);
  const activeSessionId = useChatStore((s) => s.activeSessionId);

  const scrollRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const { scrollToBottom, resetUserScrolling } = useChatScroll(scrollRef);
  const [isAtBottom, setIsAtBottom] = useState(true);
  const [prevSessionId, setPrevSessionId] = useState(activeSessionId);

  if (activeSessionId !== prevSessionId) {
    setPrevSessionId(activeSessionId);
    setIsAtBottom(true);
    resetUserScrolling();
  }

  useEffect(() => {
    scrollToBottom(true);
  }, [activeSessionId, items.length, scrollToBottom]);

  useEffect(() => {
    return () => { scrollToBottom.cancel(); };
  }, [scrollToBottom]);

  const handleScroll = (event: UIEvent<HTMLDivElement>) => {
    const { scrollHeight, scrollTop, clientHeight } = event.currentTarget;
    setIsAtBottom(scrollHeight - scrollTop - clientHeight < 50);
  };

  return (
    <div className="flex-1 overflow-hidden flex flex-col justify-between relative user-select-text">
      <div
        ref={scrollRef}
        className="flex-1 w-full overflow-x-hidden overflow-y-auto custom-scrollbar relative"
        onScroll={handleScroll}
      >
        <div className="max-w-4xl mx-auto px-4">
          {items.length === 0 && <Greetings t={t} />}

          {items.map((item, i) => (
            <RenderItem key={i} item={item} formatUrl={formatUrl} t={t} />
          ))}

          {isStreaming && (
            <div className="py-2">
              <div className="inline-block w-1.5 h-5 ml-0.5 -mb-0.5 bg-[#666] dark:bg-[#A3A3A3] rounded-sm animate-pulse" />
            </div>
          )}

          {timedoutShow && (
            <div className="py-4 text-sm text-red-500">
              {t("assistant.chat.timedout")}
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>
      </div>
      <ScrollToBottom scrollRef={scrollRef} isAtBottom={isAtBottom} />
    </div>
  );
};
