import {
  memo,
  useState,
  useEffect,
  forwardRef,
  useImperativeHandle,
  useRef,
  useMemo,
} from "react";
import { useTranslation, I18nextProvider } from "react-i18next";
import clsx from "clsx";
import i18nInstance from "../../../i18n";
import Markdown from "@infinilabs/markdown";

import logoImg from "../assets/icon.svg";
import type { IChatMessage, IChunkData } from "../types/chat";
export type { IChatMessage, IChunkData };
import { QueryIntent } from "./QueryIntent";
import { CallTools } from "./CallTools";
import { FetchSource } from "./FetchSource";
import { PickSource } from "./PickSource";
import { DeepRead } from "./DeepRead";
import { Think } from "./Think";
import { MessageActions } from "./MessageActions";
import { SuggestionList } from "./SuggestionList";
import { UserMessage, type AttachmentHit } from "./UserMessage";
import FontIcon from "./Common/Icons/FontIcon";
import useMessageChunkData from "../hooks/useMessageChunkData";
import { DeepResearch } from "./DeepResearch";
import { PayloadCard } from "./PayloadCard";

import { type TFunction } from "i18next";

import "./index.css";

const DEEP_RESEARCH_CHUNK_TYPES = [
  "research_planner_start",
  "research_planner_progress",
  "research_planner_end",
  "research_researcher_start",
  "research_researcher_step_start",
  "research_researcher_step_end",
  "research_researcher_end",
  "research_reporter_start",
  "research_reporter_end",
];

export interface ChatMessageProps {
  message: IChatMessage;
  isTyping?: boolean;
  onResend?: (value: string) => void;
  onCancel?: () => void;
  hide_assistant?: boolean;
  rootClassName?: string;
  actionClassName?: string;
  actionIconSize?: number;
  copyButtonId?: string;
  formatUrl?: (data: IChunkData) => string;
  theme?: "light" | "dark" | "system";
  locale?: string;
  report_content?: string;
  assistantList?: any[];
  currentAssistant?: any;
  /** Fetch attachment metadata by IDs for rendering in user messages. */
  fetchAttachments?: (ids: string[]) => Promise<AttachmentHit[]>;
  t?: TFunction;
}

export interface ChatMessageRef {
  addChunk: (chunk: IChunkData) => void;
  reset: () => void;
}

function resolveTheme(
  theme: "light" | "dark" | "system" | undefined,
): "light" | "dark" | undefined {
  if (!theme) return undefined;
  if (theme === "light") return "light";
  if (theme === "dark") return "dark";
  if (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  ) {
    return "dark";
  }
  return "light";
}

const InnerChatMessage = memo(
  forwardRef<ChatMessageRef, ChatMessageProps>(function InnerChatMessage(
    {
      message,
      isTyping,
      onResend,
      onCancel,
      hide_assistant = false,
      rootClassName,
      actionClassName,
      actionIconSize,
      copyButtonId,
      formatUrl,
      theme,
      assistantList,
      currentAssistant,
      fetchAttachments,
      t: tProp,
    },
    ref,
  ) {
    const { t: tOriginal } = useTranslation();
    const t = tProp || tOriginal;
    const resolvedTheme = resolveTheme(theme);

    const [assistant, setAssistant] = useState<any>({});

    const {
      data: {
        query_intent,
        tools,
        fetch_source,
        pick_source,
        deep_read,
        think,
        response,
        deepResearch,
        replyEnd,
      },
      handlers,
      clearAllChunkData,
    } = useMessageChunkData();

    const [loadingStep, setLoadingStep] = useState<Record<string, boolean>>({
      query_intent: false,
      tools: false,
      fetch_source: false,
      pick_source: false,
      deep_read: false,
      think: false,
      response: false,
      deepResearch: false,
      replyEnd: false,
    });

    const inThinkRef = useRef<boolean>(false);

    useImperativeHandle(ref, () => ({
      addChunk: (chunkData: IChunkData) => {
        setLoadingStep(() => ({
          query_intent: false,
          tools: false,
          fetch_source: false,
          pick_source: false,
          deep_read: false,
          think: false,
          response: false,
          deepResearch: false,
          [chunkData.chunk_type || ""]: true,
        }));

        if (chunkData.chunk_type === "reply_start") {
          //
        } else if (chunkData.chunk_type === "query_intent") {
          handlers.deal_query_intent(chunkData);
        } else if (chunkData.chunk_type === "tools") {
          handlers.deal_tools(chunkData);
        } else if (chunkData.chunk_type === "fetch_source") {
          handlers.deal_fetch_source(chunkData);
        } else if (chunkData.chunk_type === "pick_source") {
          handlers.deal_pick_source(chunkData);
        } else if (chunkData.chunk_type === "deep_read") {
          handlers.deal_deep_read(chunkData);
        } else if (chunkData.chunk_type === "think") {
          handlers.deal_think(chunkData);
        } else if (chunkData.chunk_type === "response") {
          const message_chunk = chunkData.message_chunk;
          if (typeof message_chunk === "string") {
            if (
              message_chunk.includes("\u003cthink\u003e") ||
              message_chunk.includes("<think>")
            ) {
              inThinkRef.current = true;
              return;
            } else if (
              message_chunk.includes("\u003c/think\u003e") ||
              message_chunk.includes("</think>")
            ) {
              inThinkRef.current = false;
              return;
            }

            if (inThinkRef.current) {
              handlers.deal_think({ ...chunkData, chunk_type: "think" });
            } else {
              handlers.deal_response(chunkData);
            }
          }
        } else if (
          DEEP_RESEARCH_CHUNK_TYPES.includes(chunkData.chunk_type || "")
        ) {
          handlers.deal_deep_research(chunkData);
        } else if (chunkData.chunk_type === "reply_end") {
          handlers.deal_reply_end(chunkData);
        }
      },
      reset: () => {
        clearAllChunkData();
        setSuggestion([]);
        setLoadingStep({
          query_intent: false,
          tools: false,
          fetch_source: false,
          pick_source: false,
          deep_read: false,
          think: false,
          response: false,
          deepResearch: false,
          replyEnd: false,
        });
        inThinkRef.current = false;
      },
    }));

    const isAssistant = message?._source?.type === "assistant";
    const assistant_id = message?._source?.assistant_id;
    const assistant_item = message?._source?.assistant_item;

    useEffect(() => {
      if (assistant_item) {
        setAssistant(assistant_item);
        return;
      }

      if (isAssistant && assistant_id && Array.isArray(assistantList)) {
        setAssistant(
          assistantList.find((item) => item._id === assistant_id) ?? {},
        );
        return;
      }

      setAssistant(currentAssistant);
    }, [
      isAssistant,
      assistant_item,
      assistant_id,
      assistantList,
      currentAssistant,
    ]);

    const source = message?._source;
    const messageContent = source?.message || "";

    
    const payload = source?.payload;

    const attachments = source?.attachments ?? [];
    const details = source?.details || [];
    const deepResearchDetail = details.find((item) => item.type === "deep_research")
    const question = source?.question || "";

    const endChunk = useMemo(() => {
      const endDetail = details.find((item) => item.type === "reply_end");
      if (endDetail) {
        return endDetail;
      }
      const last = replyEnd?.length > 0 ? replyEnd[replyEnd.length - 1] : undefined;
      let payload;
      try {
        payload = last && last.message_chunk ? JSON.parse(last.message_chunk) : undefined;
      } catch (e) {

      }
      return last ? {
        type: last.chunk_type,
        payload
      } : undefined;
    }, [details, replyEnd])

    const isCancelled = useMemo(() => {
      return endChunk?.payload?.reason === "user_cancelled";
    }, [endChunk]);

    const isError = useMemo(() => {
      return endChunk?.payload?.reason === "error";
    }, [endChunk]);

    const showActions =
      isTyping === false && (messageContent || response?.message_chunk);

    const [suggestion, setSuggestion] = useState<string[]>([]);

    const getSuggestion = (suggestion: string[]) => {
      setSuggestion(suggestion);
    };

    const hasContent =
      !isAssistant ||
      (details && details.length > 0) ||
      !!query_intent ||
      !!tools ||
      !!fetch_source ||
      !!pick_source ||
      !!deep_read ||
      !!think ||
      (deepResearch && deepResearch.length > 0) ||
      !!messageContent ||
      !!response?.message_chunk ||
      !!payload ||
      isTyping ||
      (suggestion && suggestion.length > 0);

    const renderContent = () => {
      if (!isAssistant) {
        return (
          <UserMessage message={messageContent} attachments={attachments} fetchAttachments={fetchAttachments} />
        );
      }

      return (
        <>
          <QueryIntent
            Detail={details.find((item) => item.type === "query_intent")}
            ChunkData={query_intent}
            getSuggestion={getSuggestion}
            loading={loadingStep?.query_intent}
            t={t}
          />

          <CallTools
            Detail={details.find((item) => item.type === "tools")}
            ChunkData={tools}
            loading={loadingStep?.tools}
            t={t}
          />

          <FetchSource
            Detail={details.find((item) => item.type === "fetch_source")}
            ChunkData={fetch_source}
            loading={loadingStep?.fetch_source}
            formatUrl={formatUrl}
            t={t}
          />

          <PickSource
            Detail={details.find((item) => item.type === "pick_source")}
            ChunkData={pick_source}
            loading={loadingStep?.pick_source}
            t={t}
          />

          <DeepRead
            Detail={details.find((item) => item.type === "deep_read")}
            ChunkData={deep_read}
            loading={loadingStep?.deep_read}
            t={t}
          />

          <Think
            Detail={details.find((item) => item.type === "think")}
            ChunkData={think}
            loading={loadingStep?.think}
            t={t}
          />

          <div className="cm-markdown">
            <Markdown
              content={messageContent || response?.message_chunk || ""}
              dark={resolvedTheme === "dark"}
            />
          </div>

          {
            !deepResearchDetail && (
              <PayloadCard payload={payload as any} formatUrl={formatUrl} />
            )
          }

          <DeepResearch
            detail={deepResearchDetail}
            endChunk={endChunk}
            ChunkData={deepResearch}
            question={question}
            formatUrl={formatUrl}
            theme={resolvedTheme}
            t={t}
            payload={payload as any}
            onCancel={onCancel}
          />

          {
            isCancelled && (
              <div className="mt-16px text-14px leading-20px text-[#999]">
                {t("deepResearch.status.cancelled")}
              </div>
            )
          }
          {
            isError && (
              <div className="mt-16px text-14px leading-20px text-[#F04444]">
                {t("deepResearch.status.error")}
              </div>
            )
          }

          {deepResearch.length === 0 && isTyping && (
            <div className="inline-block w-1.5 h-5 ml-0.5 -mb-0.5 bg-[#666666] dark:bg-[#A3A3A3] rounded-sm animate-typing" />
          )}

          {(showActions || endChunk) && (
            <MessageActions
              id={message._id ?? ""}
              content={messageContent || response?.message_chunk || ""}
              question={question}
              actionClassName={actionClassName}
              actionIconSize={actionIconSize}
              copyButtonId={copyButtonId}
              onResend={() => {
                if (onResend) {
                  onResend(question);
                }
              }}
            />
          )}

          {!isTyping && (
            <SuggestionList
              suggestions={suggestion}
              onSelect={(text) => onResend && onResend(text)}
            />
          )}
        </>
      );
    };

    return (
      <div
        className={clsx(
          "w-full py-8 flex",
          [isAssistant ? "justify-start" : "justify-end"],
          resolvedTheme === "dark" && "dark",
          rootClassName,
        )}
      >
        <div
          className={`w-full flex gap-4 ${
            isAssistant ? "w-full" : "flex-row-reverse"
          }`}
        >
          <div
            className={`w-full space-y-2 ${
              isAssistant ? "text-left" : "text-right"
            }`}
          >
            {!hide_assistant && hasContent && (
              <div className="w-full flex items-center gap-8px font-semibold text-sm text-[#333] dark:text-white">
                {isAssistant ? (
                  <div className="w-32px h-32px flex justify-center items-center rounded-full bg-white dark:bg-[#2A2A2A] border border-[#F0F0F0] dark:border-[#303030]">
                    {assistant?._source?.icon?.startsWith("font_") ? (
                      <FontIcon
                        name={assistant._source.icon}
                        className="w-32px h-32px"
                      />
                    ) : (
                      <img
                        src={logoImg}
                        className="w-32px h-32px"
                        alt={t("assistant.message.logo")}
                      />
                    )}
                  </div>
                ) : null}
                {isAssistant ? assistant?._source?.name || "Coco AI" : ""}
              </div>
            )}
            <div className="w-full max-w-none">
              <div
                className={clsx(
                  "w-full text-[#333] dark:text-white leading-relaxed",
                  isAssistant && "pl-40px",
                )}
              >
                {renderContent()}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }),
);

export const ChatMessage = memo(
  forwardRef<ChatMessageRef, ChatMessageProps>((props, ref) => {
    return (
      <I18nextProvider i18n={i18nInstance}>
        <InnerChatMessage {...props} ref={ref} />
      </I18nextProvider>
    );
  }),
);
