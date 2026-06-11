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
import { type ChatMessageRef } from "./ChatMessage/components";

import i18n from "../i18n";
import { useChatStore, type IChatStore } from "./stores/chatStore";
import { ChatContent } from "./ChatContent";
import type { Chat, ChatMessageItem, IChunkData } from "./types/chat";
import { streamPost } from "./api/streamFetch";
import { Get, Post } from "./api/axiosRequest";
import { useIconfontScript } from "./hooks/useScript";

/**
 * ChatAI 组件接口定义
 * @property BaseUrl - API 基础地址
 * @property Token - 认证 Token (可选)
 * @property formatUrl - 自定义 URL 格式化函数 (可选)
 * @property locale - 语言环境 (可选)
 * @property t - 国际化翻译函数 (可选)
 */
interface ChatAIProps {
  BaseUrl: string;
  Token?: string;
  headers?: Record<string, string>;
  formatUrl?: (data: IChunkData) => string;
  locale?: string;
  t?: TFunction;
  theme?: string;
}

/**
 * 发送消息参数接口
 * @property message - 消息内容
 * @property attachments - 附件列表 (ID 数组)
 * @property search - 是否启用搜索
 * @property deep_thinking - 是否启用深度思考
 * @property mcp - 是否启用 MCP
 * @property datasource - 数据源
 * @property mcp_servers - MCP 服务器配置
 * @property assistant_id - 助手 ID
 */
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

/**
 * ChatAI 组件对外暴露的引用接口
 * @property init - 初始化并发送消息
 * @property cancelChat - 取消当前对话生成
 * @property clearChat - 清除当前对话状态
 * @property onSelectChat - 切换当前选中的对话
 */
export interface ChatAIRef {
  init: (params: SendMessageParams) => void;
  cancelChat: () => void;
  clearChat: () => void;
  onSelectChat: (chat: Chat) => void;
}

/**
 * 内部 ChatAI 组件实现
 * 处理核心聊天逻辑、状态管理和消息流式传输
 */
const InnerChatAI = memo(
  forwardRef<ChatAIRef, ChatAIProps>(
    ({ BaseUrl, formatUrl, headers: headersProp = {}, locale, t: tProp, theme }, ref) => {
      // 动态加载 iconfont 脚本
      useIconfontScript();

      const { t: tOriginal } = useTranslation();
      const t = tProp || tOriginal;
      const baseUrl = BaseUrl;

      // 从全局 Store 获取聊天状态
      const curChatEnd = useChatStore((state) => state.curChatEnd); // 当前对话是否结束
      const setCurChatEnd = useChatStore((state) => state.setCurChatEnd);
      const activeChat = useChatStore((state) => state.activeChat); // 当前选中的对话
      const setActiveChat = useChatStore((state) => state.setActiveChat);
      const currentAssistant = useChatStore((state) => state.currentAssistant); // 当前助手信息
      const setCurrentAssistant = useChatStore((state) => state.setCurrentAssistant);
      const incrementHistoryVersion = useChatStore((state) => state.incrementHistoryVersion);
      const setBaseUrl = useChatStore((state) => state.setBaseUrl);
      const setAuthHeaders = useChatStore((state) => state.setAuthHeaders);

      // 初始化 store 中的 baseUrl 和 authHeaders
      useEffect(() => {
        if (BaseUrl) {
          setBaseUrl(BaseUrl);
        }
        const mergedHeaders: Record<string, string> = { ...headersProp };
        if (Object.keys(mergedHeaders).length > 0) {
          setAuthHeaders(mergedHeaders);
        }
      }, [BaseUrl, headersProp, setBaseUrl, setAuthHeaders]);

      // 本地状态
      const [timedoutShow, setTimedoutShow] = useState(false); // 超时提示显示状态
      const [Question, setQuestion] = useState<string>(""); // 当前正在处理的问题文本
      const [activeMessageGen, setActiveMessageGen] = useState(0); // 用于强制 ActiveChatMessage 重新挂载

      // Refs 用于在闭包和异步操作中保持最新值
      const curIdRef = useRef(""); // 当前生成的消息 ID
      const curSessionIdRef = useRef(""); // 当前会话 ID
      const activeMessageRef = useRef<ChatMessageRef>(null); // 活跃消息组件的引用
      const streamGenRef = useRef(0); // 流式请求的代次，用于切换后忽略旧流的渲染
      const generatingSessionRef = useRef<string | undefined>(undefined); // 当前正在生成回复的会话 ID
      const fetchHistoryRef = useRef<(chatId: string) => Promise<void>>(); // 避免 handleStreamMessage 的循环依赖

      type ChatStreamSingle = {
        _id?: string;
        _source?: {
          [key: string]: unknown;
        };
        payload?: {
          id?: string;
          session_id?: string;
          [key: string]: unknown;
        };
      };

      /**
       * 处理流式消息回调
       * 负责解析服务端返回的数据流，更新消息列表或处理流式 chunks
       */
      const handleStreamMessage = useCallback(
        (msg: string) => {
          try {
            // Attempt to parse the message, as it might be a JSON string
            if (msg.startsWith("{") && msg.endsWith("}")) {
              //
            }

            // 逻辑分支 1: 处理历史记录或完整消息更新
            // 通过检查消息中是否包含特定关键字来判断是否为历史记录或用户消息回执
            if (
              msg.includes('"user"') &&
              msg.includes("_source") &&
              msg.includes("result")
            ) {
              // ... (现有的解析逻辑)
              const parsed = JSON.parse(msg) as
                | ChatMessageItem[]
                | ChatStreamSingle;
              // ... (其余的现有逻辑)
              let nextChat: Chat;

              // 使用最新的 store 状态，避免闭包捕获的旧值
              const latestActiveChat = useChatStore.getState().activeChat;

              if (Array.isArray(parsed)) {
                // 情况 A: 收到消息数组（通常是加载历史记录）
                const hits = parsed as ChatMessageItem[];
                const first = hits[0];
                let resolvedSessionId = "";
                if (first) {
                  // 更新当前消息 ID 和会话 ID
                  curIdRef.current = first._id;
                  const source = first._source as { [key: string]: unknown };
                  const sessionId = source.session_id as string | undefined;
                  if (sessionId) {
                    curSessionIdRef.current = sessionId;
                    resolvedSessionId = sessionId;
                  }
                }
                // 获取当前活动聊天对象或创建一个新的基础对象
                const baseChat: Chat = latestActiveChat || {
                  _id: first?._id ?? "",
                };
                // 合并新消息到消息列表中（先移除乐观插入的临时消息）
                const existingMessages = (baseChat.messages || []).filter(
                  (m) => !m._id.startsWith("optimistic-")
                );
                nextChat = {
                  ...baseChat,
                  // 如果 baseChat._id 为空（新建会话的乐观状态），用后端返回的 session_id 覆盖
                  _id: baseChat._id || resolvedSessionId || first?._id || "",
                  messages: [...existingMessages, ...hits],
                };
              } else {
                // 情况 B: 收到单个消息对象（通常是新发送的用户消息回执）
                const withPayload = parsed as ChatStreamSingle;
                const payload = withPayload.payload ?? {};
                const id = payload.id;
                const sessionId = payload.session_id;

                // 更新当前消息 ID 和会话 ID
                if (typeof id === "string") {
                  curIdRef.current = id;
                }
                if (typeof sessionId === "string") {
                  curSessionIdRef.current = sessionId;
                  // Keep generatingSessionRef in sync when backend creates a new session
                  generatingSessionRef.current = sessionId;
                }

                // 构造标准消息项对象
                const messageItem: ChatMessageItem = {
                  _id:
                    withPayload._id ??
                    (typeof id === "string" ? id : "") ??
                    latestActiveChat?._id ??
                    "",
                  _source: {
                    ...(withPayload._source || {}),
                    ...payload,
                  } as ChatMessageItem["_source"],
                };

                // 获取当前活动聊天对象或创建一个新的基础对象
                const baseChat: Chat = latestActiveChat || {
                  _id: messageItem._id,
                };

                // 将新消息追加到消息列表中（先移除乐观插入的临时消息）
                const existingMessages = (baseChat.messages || []).filter(
                  (m) => !m._id.startsWith("optimistic-")
                );
                nextChat = {
                  ...baseChat,
                  // 如果 baseChat._id 为空（新建会话的乐观状态），用后端返回的 session_id 覆盖
                  _id: baseChat._id || (typeof sessionId === "string" ? sessionId : "") || messageItem._id,
                  messages: [...existingMessages, messageItem],
                };
              }

              // 更新全局活动聊天状态，触发 UI 重绘
              // 同步更新 lastActiveChatIdRef，防止 useEffect 在 finally 清除
              // generatingSessionRef 后误触发 onSelectChat
              if (nextChat._id) {
                // 当 _id 从空变为真实值时（新建会话首次获取后端 ID），立即刷新历史列表
                if (!lastActiveChatIdRef.current || lastActiveChatIdRef.current !== nextChat._id) {
                  const prevId = latestActiveChat?._id;
                  if (!prevId && nextChat._id) {
                    incrementHistoryVersion();
                  }
                }
                lastActiveChatIdRef.current = nextChat._id;
              }
              setActiveChat(nextChat);
            }

            // 逻辑分支 2: 处理流式 Chunks (打字机效果、思考过程等)
            const chunkData = JSON.parse(msg);

            if (chunkData.chunk_type) {
              // 标记回复开始
              if (chunkData.chunk_type === "reply_start") {
                activeMessageRef.current?.reset(); // 收到新回复后再清空上一条，避免发送期间屏幕空白
                setCurChatEnd(false);
              }

              // 将 chunk 数据传递给活跃的消息组件进行展示
              activeMessageRef.current?.addChunk(chunkData);

              // 标记回复结束
              if (chunkData.chunk_type === "reply_end") {
                setCurChatEnd(true);
                // 流式结束后，不立即 fetchHistory 加载到 messages，
                // 让 AI 回答留在 ActiveChatMessage 的 chunk 数据中继续展示，
                // 避免 messages 列表与 ActiveChatMessage 同时渲染导致重复。
                // 下次 sendMessage 时 fetchHistory + reset 会正确完成内容切换。
                incrementHistoryVersion();
              }
            }
          } catch (error) {
            // JSON 解析失败或其他错误处理
            console.error("Failed to parse chat message:", error);
          }
        },
        [setActiveChat, setCurChatEnd, incrementHistoryVersion],
      );

      /**
       * 准备新的聊天会话
       * 重置当前消息状态，为新一轮问答做准备
       */
      const prepareChatSession = useCallback(async (value: string) => {
        activeMessageRef.current?.reset(); // 重置活跃消息组件状态
        setTimedoutShow(false);
        setQuestion(value); // 设置当前问题文本
      }, []);

      /**
       * 拉取指定会话的历史记录
       * @param chatId - 会话 ID
       */
      const fetchHistory = useCallback(
        async (chatId: string) => {
          try {
            const [err, res] = await Get<{
              hits: { hits: ChatMessageItem[] };
            }>(`/chat/${chatId}/_history`, {
              from: 0,
              size: 1000,
            }, undefined, headersProp);
            if (err || !res) return;
            const hits = (res?.hits?.hits ?? []) as ChatMessageItem[];

            // 获取最新状态以确保我们在更新正确的聊天
            const currentActive = useChatStore.getState().activeChat;
            if (currentActive?._id === chatId) {
              setActiveChat({
                ...currentActive,
                messages: hits,
              });
            }
          } catch (e) {
            console.error(e);
          }
        },
        [setActiveChat, headersProp],
      );

      // 保持 fetchHistoryRef 始终指向最新的 fetchHistory
      fetchHistoryRef.current = fetchHistory;

      /**
       * 创建新会话并发送第一条消息
       */
      const createNewChat = useCallback(
        async (params: SendMessageParams) => {
          const text = params.message ?? "";
          const attachments = params.attachments;
          // 如果没有文本且没有附件，则不发送
          if (!text && (!attachments || attachments.length === 0)) {
            return;
          }
          await prepareChatSession(text);
          setCurChatEnd(false); // 立即进入生成状态，按钮开始转圈
          generatingSessionRef.current = curSessionIdRef.current || activeChat?._id;

          // 乐观插入用户消息，立即隐藏欢迎语并显示用户消息
          const optimisticUserMessage: ChatMessageItem = {
            _id: `optimistic-${Date.now()}`,
            _source: {
              type: "user",
              message: text,
            },
          };
          setActiveChat({
            _id: "",
            messages: [optimisticUserMessage],
          });

          // 构建查询参数，包含助手配置
          const queryParams = {
            search: params.search,
            deep_thinking: params.deep_thinking,
            mcp: params.mcp,
            datasource: params.datasource,
            mcp_servers: params.mcp_servers,
            assistant_id: params.assistant_id || currentAssistant?._id || "",
          };

          // 发送创建会话请求
          const gen = ++streamGenRef.current;
          try {
            await streamPost({
              url: "/chat/_create",
              body: {
                message: text,
                attachments,
              },
              queryParams,
              headers: headersProp,
              onMessage: (msg) => {
                if (streamGenRef.current !== gen) {
                  // Allow reply_end through even after cancel so the component can handle the reason
                  try {
                    const parsed = JSON.parse(msg);
                    if (parsed.chunk_type === "reply_end") {
                      activeMessageRef.current?.addChunk(parsed);
                    }
                  } catch {}
                  return;
                }
                handleStreamMessage(msg);
              },
            });
          } finally {
            if (streamGenRef.current === gen) {
              setCurChatEnd(true);
              generatingSessionRef.current = undefined;
            }
          }

          // 历史列表已在 handleStreamMessage 中首次获取到真实 session_id 时刷新
        },
        [handleStreamMessage, prepareChatSession, currentAssistant?._id, headersProp, setCurChatEnd, setActiveChat],
      );

      const openChat = useCallback(
        async (params: { session_id: string, assistant_id: string }) => {
          try {
            const res = await Post<{ found: boolean }>(
              `/chat/${params.session_id}/_open`,
              undefined,
              {},
              headersProp,
            );
            if (res?.[1]?.found) {
              incrementHistoryVersion();
              streamGenRef.current++;
              activeMessageRef.current?.reset();
              setActiveMessageGen((v) => v + 1);
              setCurChatEnd(true);
              setTimedoutShow(false);
              setQuestion("");
              lastActiveChatIdRef.current = params.session_id;
              setActiveChat({ _id: params.session_id });
              await fetchHistory(params.session_id);
            }
          } catch (e) {
            console.error(e);
          }
          if (params.assistant_id) {
            waitForAssistantList((list) => {
              const latestCurrentAssistant = useChatStore.getState().currentAssistant;
              const target = list.find((a) => a._id === params.assistant_id);
              if (params.assistant_id !== latestCurrentAssistant?._id) {
                setCurrentAssistant(target ?? { _id: params.assistant_id! });
              }
            });
          }
        },
        [headersProp, incrementHistoryVersion, setCurChatEnd, setActiveChat, fetchHistory, setCurrentAssistant],
      );

      /**
       * 在现有会话中发送消息
       */
      const sendMessage = useCallback(
        async (chat: Chat, params?: SendMessageParams) => {
          if (!chat?._id || !params) return;
          const text = params.message ?? "";
          const attachments = params.attachments;
          if (!text && (!attachments || attachments.length === 0)) {
            return;
          }
          setTimedoutShow(false);
          setQuestion(text);
          generatingSessionRef.current = chat._id;

          // 拉取历史，将之前的 AI 回答从服务器加载到 activeChat.messages
          await fetchHistory(chat._id);
          // fetchHistory 完成后，之前的 AI 回答已持久化到 messages 列表中，
          // 此时可以安全 reset 活跃消息组件，避免旧的 chunk 数据（含 suggestion）残留显示
          activeMessageRef.current?.reset();
          setCurChatEnd(false); // 立即进入生成状态，按钮开始转圈

          // 乐观追加用户消息到本地状态，使其立即可见
          const currentChat = useChatStore.getState().activeChat;
          if (currentChat) {
            const userMessage: ChatMessageItem = {
              _id: `optimistic-${Date.now()}`,
              _source: {
                type: "user",
                message: text,
              },
            };
            setActiveChat({
              ...currentChat,
              messages: [...(currentChat.messages || []), userMessage],
            });
          }

          const queryParams = {
            search: params.search,
            deep_thinking: params.deep_thinking,
            mcp: params.mcp,
            datasource: params.datasource,
            mcp_servers: params.mcp_servers,
            assistant_id: params.assistant_id || currentAssistant?._id || "",
          };

          // 发送聊天消息请求
          const gen = ++streamGenRef.current;
          try {
            await streamPost({
              url: `/chat/${chat._id}/_chat`,
              body: { message: text, attachments },
              queryParams,
              headers: headersProp,
              onMessage: (msg) => {
                if (streamGenRef.current !== gen) {
                  // Allow reply_end through even after cancel so the component can handle the reason
                  try {
                    const parsed = JSON.parse(msg);
                    if (parsed.chunk_type === "reply_end") {
                      activeMessageRef.current?.addChunk(parsed);
                    }
                  } catch {}
                  return;
                }
                handleStreamMessage(msg);
              },
            });
          } finally {
            if (streamGenRef.current === gen) {
              setCurChatEnd(true);
              generatingSessionRef.current = undefined;
            }
          }
        },
        [
          fetchHistory,
          currentAssistant?._id,
          handleStreamMessage,
          headersProp,
          setCurChatEnd,
        ],
      );

      /**
       * 处理发送消息的统一入口
       * 根据是否存在 activeChat 决定是创建新会话还是追加消息
       */
      const handleSendMessage = useCallback(
        async (chat?: Chat, params?: SendMessageParams) => {
          if (!curChatEnd) return; // 如果当前正在生成中，阻止发送

          if (!chat?._id) {
            await createNewChat(params || {});
          } else {
            await sendMessage(chat, params);
          }
        },
        [createNewChat, curChatEnd, sendMessage],
      );

      /**
       * 取消当前对话生成
       */
      const cancelChat = useCallback(async () => {
        // 递增 generation，使进行中的流回调不再处理
        streamGenRef.current++;
        const sessionToCancel = generatingSessionRef.current || useChatStore.getState().activeChat?._id;
        generatingSessionRef.current = undefined;

        if (sessionToCancel) {
          try {
            await Post(
              `/chat/${sessionToCancel}/_cancel?message_id=${curIdRef.current}&lang=${locale || i18n.language}`,
              undefined,
              {},
              headersProp,
            );
          } catch (e) {
            console.error(e);
          }
        }
        setCurChatEnd(true); // 强制标记为结束
      }, [setCurChatEnd, headersProp]);

      /**
       * 切换当前选中的对话
       * 负责重置状态并加载新对话的历史记录
       */
      const onSelectChat = useCallback(
        async (chat?: Chat) => {
          const generatingSession = generatingSessionRef.current;
          const curChatEndNow = useChatStore.getState().curChatEnd;
          // 递增 generation，使旧流的回调不再渲染
          streamGenRef.current++;

          // If a response is in progress, cancel it before switching.
          // Use generatingSessionRef (not activeChat from closure) to get the
          // correct session ID, because activeChat may have already been
          // updated by the stream handler (e.g. backend created a new session).
          if (generatingSession && !curChatEndNow) {
            generatingSessionRef.current = undefined;
            Post(
              `/chat/${generatingSession}/_cancel?message_id=${curIdRef.current}&lang=${locale || i18n.language}`,
              undefined,
              {},
              headersProp,
            ).catch(console.error);
          }

          activeMessageRef.current?.reset(); // 重置上一条消息的 UI 状态
          setActiveMessageGen((v) => v + 1); // 强制 ActiveChatMessage 重新挂载，彻底清除旧 chunk 数据
          setCurChatEnd(true);
          setTimedoutShow(false);
          setQuestion(""); // 重置问题文本，避免切换聊天后残留旧问题

          setActiveChat(chat);
          if (chat?._id) {
            lastActiveChatIdRef.current = chat._id;
            await fetchHistory(chat?._id); // 加载历史记录
          } else {
            lastActiveChatIdRef.current = undefined;
          }
        },
        [setActiveChat, setCurChatEnd, fetchHistory, headersProp],
      );

      /**
       * 清除当前选中的对话（返回初始状态）
       */
      const clearChat = useCallback(async () => {
        if (!useChatStore.getState().curChatEnd) {
          cancelChat();
        }
        onSelectChat(undefined);
      }, [onSelectChat, cancelChat]);

      // Use a ref to track the last processed active chat ID
      const lastActiveChatIdRef = useRef<string | undefined>(undefined);

      useEffect(() => {
        if (activeChat?._id && activeChat._id !== lastActiveChatIdRef.current) {
          lastActiveChatIdRef.current = activeChat._id;

          // Skip only if the generating session matches the new activeChat
          // (i.e. the stream handler updated the session ID, not a user click).
          // If the user clicked a different history item while generation is in
          // progress, we still need to call onSelectChat to cancel the stream.
          if (!generatingSessionRef.current || generatingSessionRef.current !== activeChat._id) {
            setTimeout(() => {
              onSelectChat(activeChat);
            }, 0);
          }
        } else if (!activeChat?._id && lastActiveChatIdRef.current) {
          // Active chat was cleared externally (e.g. deleted from history)
          lastActiveChatIdRef.current = undefined;
          onSelectChat(undefined);
        }
      }, [activeChat?._id, onSelectChat]);

      // 生成文件预览 URL 的辅助函数
      const getFileUrl = useCallback(
        (path: string) =>
          `${baseUrl?.replace(/\/$/, "")}/files/${encodeURIComponent(path)}`,
        [baseUrl],
      );

      /**
       * 等待 assistantList 就绪后执行回调
       * 如果列表已有数据则立即执行，否则订阅 store 变化等待填充
       */
      const waitForAssistantList = useCallback(
        (callback: (list: NonNullable<IChatStore["assistantList"]>) => void) => {
          const list = useChatStore.getState().assistantList;
          if (list && list.length > 0) {
            callback(list);
            return;
          }
          const unsubscribe = useChatStore.subscribe((state) => {
            if (state.assistantList && state.assistantList.length > 0) {
              unsubscribe();
              callback(state.assistantList);
            }
          });
        },
        [],
      );

      // 暴露给父组件的方法
      useImperativeHandle(ref, () => ({
        init: (params: SendMessageParams) => {
          const proceed = () => {
            if (!activeChat?._id) {
              createNewChat(params);
            } else {
              handleSendMessage(activeChat, params);
            }
          };

          // 如果传入了 assistant_id，等待 assistantList 就绪后再切换助手并发送
          if (params.assistant_id) {
            waitForAssistantList((list) => {
              const latestCurrentAssistant = useChatStore.getState().currentAssistant;
              const target = list.find((a) => a._id === params.assistant_id);
              if (params.assistant_id !== latestCurrentAssistant?._id) {
                setCurrentAssistant(target ?? { _id: params.assistant_id! });
              }
              const targetType = (target?._source?.type as string) || "simple";
              if (targetType === "deep_think") {
                params.deep_thinking = true;
              }
              const showSearchMCP = targetType === "simple" || targetType === "deep_think";
              if (showSearchMCP) {
                const ds = target?._source?.datasource as
                  | { enabled?: boolean; enabled_by_default?: boolean; ids?: string[] }
                  | undefined;
                if ((ds?.enabled ?? true) && ds?.enabled_by_default) {
                  params.search = true;
                }
                const mcp = target?._source?.mcp_servers as
                  | { enabled?: boolean; enabled_by_default?: boolean; ids?: string[] }
                  | undefined;
                if ((mcp?.enabled ?? true) && mcp?.enabled_by_default) {
                  params.mcp = true;
                }
              }
              proceed();
            });
          } else {
            proceed();
          }
        },
        openChat,
        cancelChat: () => {
          cancelChat();
        },
        clearChat,
        onSelectChat,
      }));

      return (
        <div className="flex flex-col rounded-md h-full overflow-hidden relative">
          <ChatContent
            activeChat={activeChat}
            activeMessageRef={activeMessageRef}
            activeMessageGen={activeMessageGen}
            timedoutShow={timedoutShow}
            Question={Question}
            handleSendMessage={(message) =>
              handleSendMessage(activeChat, {
                message: Array.isArray(message) ? message.join("") : String(message ?? ""),
              })
            }
            getFileUrl={getFileUrl}
            formatUrl={formatUrl}
            curIdRef={curIdRef}
            t={t}
            currentAssistant={currentAssistant}
            theme={theme}
            onCancel={cancelChat}
          />
        </div>
      );
    },
  ),
);

const ChatAI = memo(
  forwardRef<ChatAIRef, ChatAIProps>((props, ref) => {
    return (
      <I18nextProvider i18n={i18n}>
        <InnerChatAI {...props} ref={ref} />
      </I18nextProvider>
    );
  }),
);

export default ChatAI;
