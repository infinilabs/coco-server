import {
  forwardRef,
  memo,
  useCallback,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import { I18nextProvider, useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import i18n from "../i18n";
import { useChatStore, type IChatStore } from "./stores/chatStore";
import { ChatContent } from "./ChatContent";
import type { ChatItem, IChunk, Session } from "./types/chat";
import { postJSON } from "./api/streamFetch";
import { Get, Post } from "./api/axiosRequest";
import { useIconfontScript } from "./hooks/useScript";

const POLL_INTERVAL_MS = 500;

const DEEP_RESEARCH_CHUNK_TYPES = [
  "research_planner_start",
  "research_planner_end",
  "research_researcher_start",
  "research_researcher_step_start",
  "research_researcher_step_end",
  "research_researcher_end",
  "research_reporter_start",
  "research_reporter_end",
];

interface ChatAIProps {
  BaseUrl: string;
  Token?: string;
  headers?: Record<string, string>;
  formatUrl?: (data: any) => string;
  locale?: string;
  t?: TFunction;
}

export interface SendMessageParams {
  message?: string;
  attachments?: string[];
  search?: boolean;
  deep_thinking?: boolean;
  mcp?: boolean;
  datasource?: string;
  mcp_servers?: string;
  assistant_id?: string;
}

export interface ChatAIRef {
  init: (params: SendMessageParams) => void;
  cancelChat: () => void;
  clearChat: () => void;
  onSelectChat: (chat: Session) => void;
}

/**
 * Replay a single chunk into the items array (batch mode, for history).
 * Mutates the array in place for efficiency.
 */
function replayChunk(c: IChunk, items: ChatItem[]) {
  switch (c.type) {
    case "user_message":
      items.push({ type: "user", text: c.text ?? "" });
      break;
    case "message_start":
      break;
    case "text_delta": {
      const last = items[items.length - 1];
      if (last?.type === "assistant") last.text += c.text ?? "";
      else items.push({ type: "assistant", text: c.text ?? "" });
      break;
    }
    case "reasoning_delta":
      break;
    case "query_intent": {
      const last = items[items.length - 1];
      if (last?.type === "query_intent") last.text += c.text ?? "";
      else items.push({ type: "query_intent", text: c.text ?? "" });
      break;
    }
    case "fetch_source": {
      const last = items[items.length - 1];
      if (last?.type === "fetch_source") last.text += c.text ?? "";
      else items.push({ type: "fetch_source", text: c.text ?? "" });
      break;
    }
    case "pick_source": {
      const last = items[items.length - 1];
      if (last?.type === "pick_source") last.text += c.text ?? "";
      else items.push({ type: "pick_source", text: c.text ?? "" });
      break;
    }
    case "deep_read": {
      const last = items[items.length - 1];
      if (last?.type === "deep_read") last.text += "&" + (c.text ?? "");
      else items.push({ type: "deep_read", text: c.text ?? "" });
      break;
    }
    case "tool_call_start":
      items.push({
        type: "tool_call",
        toolName: c.tool_name ?? "",
        toolId: c.tool_id ?? "",
        args: "",
        result: "",
      });
      break;
    case "tool_call_args": {
      for (let i = items.length - 1; i >= 0; i--) {
        if (items[i].type === "tool_call") {
          (items[i] as ChatItem & { type: "tool_call" }).args += c.text ?? "";
          break;
        }
      }
      break;
    }
    case "tool_result": {
      const tid = c.tool_id;
      for (let i = items.length - 1; i >= 0; i--) {
        const it = items[i];
        if (it.type === "tool_call" && (!tid || it.toolId === tid)) {
          it.result = c.text ?? "";
          break;
        }
      }
      break;
    }
    case "research_reporter_end": {
      if (c.text) {
        try {
          items.push({ type: "payload", data: JSON.parse(c.text) });
        } catch {}
      }
      break;
    }
    default:
      if (DEEP_RESEARCH_CHUNK_TYPES.includes(c.type)) {
        const last = items[items.length - 1];
        if (last?.type === "deep_research") {
          last.chunks.push(c);
        } else {
          items.push({ type: "deep_research", chunks: [c] });
        }
      }
      break;
  }
}

/**
 * Apply a single live chunk to React state (incremental update for polling).
 */
function applyChunk(
  c: IChunk,
  setItems: IChatStore["setItems"],
  pushItem: IChatStore["pushItem"],
  updateLastItem: IChatStore["updateLastItem"],
) {
  switch (c.type) {
    case "user_message":
      pushItem({ type: "user", text: c.text ?? "" });
      break;
    case "message_start":
      break;
    case "text_delta": {
      const items = useChatStore.getState().items;
      const last = items[items.length - 1];
      if (last?.type === "assistant") {
        updateLastItem((item) =>
          item.type === "assistant" ? { ...item, text: item.text + (c.text ?? "") } : item
        );
      } else {
        pushItem({ type: "assistant", text: c.text ?? "" });
      }
      break;
    }
    case "reasoning_delta":
      break;
    case "query_intent": {
      const items = useChatStore.getState().items;
      const last = items[items.length - 1];
      if (last?.type === "query_intent") {
        updateLastItem((item) =>
          item.type === "query_intent" ? { ...item, text: item.text + (c.text ?? "") } : item
        );
      } else {
        pushItem({ type: "query_intent", text: c.text ?? "" });
      }
      break;
    }
    case "fetch_source": {
      const items = useChatStore.getState().items;
      const last = items[items.length - 1];
      if (last?.type === "fetch_source") {
        updateLastItem((item) =>
          item.type === "fetch_source" ? { ...item, text: item.text + (c.text ?? "") } : item
        );
      } else {
        pushItem({ type: "fetch_source", text: c.text ?? "" });
      }
      break;
    }
    case "pick_source": {
      const items = useChatStore.getState().items;
      const last = items[items.length - 1];
      if (last?.type === "pick_source") {
        updateLastItem((item) =>
          item.type === "pick_source" ? { ...item, text: item.text + (c.text ?? "") } : item
        );
      } else {
        pushItem({ type: "pick_source", text: c.text ?? "" });
      }
      break;
    }
    case "deep_read": {
      const items = useChatStore.getState().items;
      const last = items[items.length - 1];
      if (last?.type === "deep_read") {
        updateLastItem((item) =>
          item.type === "deep_read" ? { ...item, text: item.text + "&" + (c.text ?? "") } : item
        );
      } else {
        pushItem({ type: "deep_read", text: c.text ?? "" });
      }
      break;
    }
    case "tool_call_start":
      pushItem({
        type: "tool_call",
        toolName: c.tool_name ?? "",
        toolId: c.tool_id ?? "",
        args: "",
        result: "",
      });
      break;
    case "tool_call_args": {
      // Find last tool_call and append args.
      const items = useChatStore.getState().items;
      for (let i = items.length - 1; i >= 0; i--) {
        if (items[i].type === "tool_call") {
          const idx = i;
          setItems(items.map((it, j) =>
            j === idx && it.type === "tool_call"
              ? { ...it, args: it.args + (c.text ?? "") }
              : it
          ));
          break;
        }
      }
      break;
    }
    case "tool_result": {
      const items = useChatStore.getState().items;
      const tid = c.tool_id;
      for (let i = items.length - 1; i >= 0; i--) {
        const it = items[i];
        if (it.type === "tool_call" && (!tid || it.toolId === tid)) {
          const idx = i;
          setItems(items.map((x, j) =>
            j === idx && x.type === "tool_call"
              ? { ...x, result: c.text ?? "" }
              : x
          ));
          break;
        }
      }
      break;
    }
    case "research_reporter_end": {
      if (c.text) {
        try { pushItem({ type: "payload", data: JSON.parse(c.text) }); } catch {}
      }
      break;
    }
    default:
      if (DEEP_RESEARCH_CHUNK_TYPES.includes(c.type)) {
        const items = useChatStore.getState().items;
        const last = items[items.length - 1];
        if (last?.type === "deep_research") {
          updateLastItem((item) =>
            item.type === "deep_research"
              ? { ...item, chunks: [...item.chunks, c] }
              : item
          );
        } else {
          pushItem({ type: "deep_research", chunks: [c] });
        }
      }
      break;
  }
}

const InnerChatAI = memo(
  forwardRef<ChatAIRef, ChatAIProps>(
    ({ BaseUrl: _BaseUrl, formatUrl, headers: headersProp = {}, locale: _locale, t: tProp }, ref) => {
      useIconfontScript();
      const { t: tOriginal } = useTranslation();
      const t = tProp || tOriginal;

      const isStreaming = useChatStore((s) => s.isStreaming);
      const setIsStreaming = useChatStore((s) => s.setIsStreaming);
      const activeSessionId = useChatStore((s) => s.activeSessionId);
      const setActiveSessionId = useChatStore((s) => s.setActiveSessionId);
      const setActiveSessionSource = useChatStore((s) => s.setActiveSessionSource);
      const setItems = useChatStore((s) => s.setItems);
      const pushItem = useChatStore((s) => s.pushItem);
      const updateLastItem = useChatStore((s) => s.updateLastItem);
      const currentAssistant = useChatStore((s) => s.currentAssistant);
      const setCurrentAssistant = useChatStore((s) => s.setCurrentAssistant);
      const incrementHistoryVersion = useChatStore((s) => s.incrementHistoryVersion);

      const [timedoutShow, setTimedoutShow] = useState(false);

      const seqRef = useRef(0);
      const lastSessionIdRef = useRef<string | undefined>(undefined);

      const headersRef = useRef(headersProp);
      headersRef.current = headersProp;
      const setIsStreamingRef = useRef(setIsStreaming);
      setIsStreamingRef.current = setIsStreaming;
      const incrementHistoryVersionRef = useRef(incrementHistoryVersion);
      incrementHistoryVersionRef.current = incrementHistoryVersion;

      // --- Poll state ---
      const pollControllerRef = useRef<AbortController | null>(null);
      const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

      function closePoll() {
        if (pollControllerRef.current) {
          pollControllerRef.current.abort();
          pollControllerRef.current = null;
        }
      }

      function stopPolling() {
        if (pollTimerRef.current) {
          clearInterval(pollTimerRef.current);
          pollTimerRef.current = null;
        }
        closePoll();
      }

      function startPolling() {
        if (!pollTimerRef.current) {
          pollTimerRef.current = setInterval(pollLoop, POLL_INTERVAL_MS);
        }
      }

      async function pollLoop() {
        const sid = useChatStore.getState().activeSessionId;
        if (!sid || pollControllerRef.current) return;

        const controller = new AbortController();
        pollControllerRef.current = controller;

        const appStore = JSON.parse(localStorage.getItem("app-store") || "{}");
        let baseURL: string = appStore.state?.endpoint_http;
        if (!baseURL || baseURL === "undefined") baseURL = "";
        const fullUrl = `${baseURL}/chat/${sid}/_poll?since=${seqRef.current}`;

        try {
          const headersStorage = JSON.parse(localStorage.getItem("headers") || "{}") as Record<string, string>;
          const res = await fetch(fullUrl, {
            method: "GET",
            headers: { ...headersStorage, ...headersRef.current },
            credentials: "include",
            signal: controller.signal,
          });
          if (!res.ok || !res.body) return;

          const reader = res.body.getReader();
          const decoder = new TextDecoder("utf-8");
          let buffer = "";

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const lines = buffer.split("\n");
            for (let i = 0; i < lines.length - 1; i++) {
              const line = lines[i].trim();
              if (!line) continue;
              try {
                const chunk = JSON.parse(line) as IChunk;
                console.log("[poll chunk]", chunk.type, chunk.text?.slice(0, 50) ?? "", "items:", useChatStore.getState().items.length);
                if (chunk.type === "seq_sync") {
                  if (chunk.seq) seqRef.current = chunk.seq;
                  continue;
                }
                if (chunk.type === "agent_loop_end") {
                  if (useChatStore.getState().isStreaming) {
                    incrementHistoryVersionRef.current();
                  }
                  setIsStreamingRef.current(false);
                  closePoll();
                  continue;
                }
                if (!useChatStore.getState().isStreaming) {
                  setIsStreamingRef.current(true);
                }
                if (chunk.type === "message_start") continue;
                console.log("[applyChunk]", chunk.type, "items before:", useChatStore.getState().items.length);
                applyChunk(chunk, setItems, pushItem, updateLastItem);
              } catch {}
            }
            buffer = lines[lines.length - 1];
          }
        } catch {
          setIsStreamingRef.current(false);
        } finally {
          pollControllerRef.current = null;
        }
      }

      useEffect(() => {
        const sid = activeSessionId;
        if (!sid) { stopPolling(); return; }
        startPolling();
        return () => { closePoll(); stopPolling(); };
      }, [activeSessionId]);

      const fetchHistory = useCallback(
        async (sessionId: string) => {
          try {
            const [err, res] = await Get<{ chunks: IChunk[]; max_seq: number }>(
              `/chat/${sessionId}/_history`, {}, undefined, headersProp
            );
            if (err || !res || !res.chunks) return;
            if (res.max_seq) seqRef.current = res.max_seq;
            const currentSid = useChatStore.getState().activeSessionId;
            if (currentSid !== sessionId) return;
            const items: ChatItem[] = [];
            console.log("[fetchHistory] chunks:", res.chunks.length, "sessionId:", sessionId);
            for (const c of res.chunks) replayChunk(c, items);
            console.log("[fetchHistory] replayed items:", items.length, items.map(i => i.type));
            setItems(items);
          } catch (e) { console.error(e); }
        },
        [setItems, headersProp],
      );

      const onSelectChat = useCallback(
        async (session?: Session) => {
          console.log("[onSelectChat]", session?._id);
          setIsStreaming(false);
          setTimedoutShow(false);
          setActiveSessionId(session?._id);
          setActiveSessionSource(session?._source);
          seqRef.current = 0;
          setItems([]);
          if (session?._id) {
            await fetchHistory(session._id);
          }
        },
        [setActiveSessionId, setActiveSessionSource, setIsStreaming, setItems, fetchHistory],
      );

      useEffect(() => {
        const id = activeSessionId;
        if (!id) return;
        if (id === lastSessionIdRef.current) return;
        lastSessionIdRef.current = id;
        onSelectChat({ _id: id, _source: useChatStore.getState().activeSessionSource });
      }, [activeSessionId, onSelectChat]);

      const createNewChat = useCallback(
        async (params: SendMessageParams) => {
          const { message = "", attachments, datasource = [], mcp_servers = [], ...rest } = params;
          if (!message && (!attachments || attachments.length === 0)) return;

          setTimedoutShow(false);

          try {
            const res = await postJSON<{ _id: string; _source: Record<string, unknown> }>({
              url: "/chat/_create",
              body: { message, attachments },
              queryParams: { 
                assistant_id: params.assistant_id || currentAssistant?._id || "", 
                ...(rest || {}),
                datasource: datasource instanceof Array ? datasource.join(",") : undefined,
                mcp_servers: mcp_servers instanceof Array ? mcp_servers.join(",") : undefined,  
              },
              headers: headersProp,
            });
            setActiveSessionId(res._id);
            setActiveSessionSource(res._source as Session["_source"]);
            incrementHistoryVersion();
          } catch (err) {
            console.error("createNewChat error:", err);
          }
        },
        [currentAssistant?._id, headersProp, setActiveSessionId, setActiveSessionSource, setItems, incrementHistoryVersion],
      );

      const sendMessage = useCallback(
        async (sessionId: string, params?: SendMessageParams) => {
          if (!sessionId || !params) return;
          const { message = "", attachments, datasource = [], mcp_servers = [], ...rest } = params;
          if (!message && (!attachments || attachments.length === 0)) return;

          setTimedoutShow(false);

          try {
            await postJSON({ 
              url: `/chat/${sessionId}/_send`, 
              body: { 
                message, 
                attachments,
              }, 
              queryParams: { 
                ...(rest || {}),
                datasource: datasource instanceof Array ? datasource.join(",") : undefined,
                mcp_servers: mcp_servers instanceof Array ? mcp_servers.join(",") : undefined,  
              },
              headers: headersProp 
            });
          } catch (err) {
            console.error("sendMessage error:", err);
          }
        },
        [headersProp],
      );

      const handleSendMessage = useCallback(
        async (params?: SendMessageParams) => {
          if (isStreaming) return;
          if (!activeSessionId) await createNewChat(params || {});
          else await sendMessage(activeSessionId, params);
        },
        [createNewChat, isStreaming, sendMessage, activeSessionId],
      );

      const cancelChat = useCallback(async () => {
        closePoll();
        const sid = useChatStore.getState().activeSessionId;
        if (sid) {
          try { await Post(`/chat/${sid}/_cancel`, undefined, {}, headersProp); } catch {}
        }
        setIsStreaming(false);
      }, [setIsStreaming, headersProp]);

      const clearChat = useCallback(() => { onSelectChat(undefined); }, [onSelectChat]);

      const waitForAssistantList = useCallback(
        (callback: (list: NonNullable<IChatStore["assistantList"]>) => void) => {
          const list = useChatStore.getState().assistantList;
          if (list && list.length > 0) { callback(list); return; }
          const unsub = useChatStore.subscribe((state) => {
            if (state.assistantList && state.assistantList.length > 0) { unsub(); callback(state.assistantList); }
          });
        }, [],
      );

      useImperativeHandle(ref, () => ({
        init: (params: SendMessageParams) => {
          const proceed = () => {
            handleSendMessage(params);
          };
          if (params.assistant_id) {
            waitForAssistantList((list) => {
              const current = useChatStore.getState().currentAssistant;
              const target = list.find((a) => a._id === params.assistant_id);
              if (params.assistant_id !== current?._id) setCurrentAssistant(target ?? { _id: params.assistant_id! });
              const targetType = (target?._source?.type as string) || "simple";
              if (targetType === "deep_think") params.deep_thinking = true;
              if (targetType === "simple" || targetType === "deep_think") {
                const ds = target?._source?.datasource as { enabled?: boolean; enabled_by_default?: boolean } | undefined;
                if ((ds?.enabled ?? true) && ds?.enabled_by_default) params.search = true;
                const mcp = target?._source?.mcp_servers as { enabled?: boolean; enabled_by_default?: boolean } | undefined;
                if ((mcp?.enabled ?? true) && mcp?.enabled_by_default) params.mcp = true;
              }
              proceed();
            });
          } else proceed();
        },
        cancelChat: () => cancelChat(),
        clearChat,
        onSelectChat,
      }));

      return (
        <div className="flex flex-col rounded-md h-full overflow-hidden relative">
          <ChatContent
            timedoutShow={timedoutShow}
            handleSendMessage={(message) =>
              handleSendMessage({
                message: Array.isArray(message) ? message.join("") : String(message ?? ""),
              })
            }
            formatUrl={formatUrl}
            t={t}
          />
        </div>
      );
    },
  ),
);

const ChatAI = memo(
  forwardRef<ChatAIRef, ChatAIProps>((props, ref) => (
    <I18nextProvider i18n={i18n}><InnerChatAI {...props} ref={ref} /></I18nextProvider>
  )),
);

export default ChatAI;
