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
import CancelDeepResearchDialog from "./CancelDeepResearchDialog";
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
  isMobile?: boolean;
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
  clearChat: (cb?: () => void, force?: boolean, onReject?: () => void, keepAssistant?: boolean) => void;
  onSelectChat: (chat: Chat) => void;
  _isChatEnd: () => boolean;
}

/**
 * 内部 ChatAI 组件实现
 * 处理核心聊天逻辑、状态管理和消息流式传输
 */
const InnerChatAI = memo(
  forwardRef<ChatAIRef, ChatAIProps>(
    ({ BaseUrl, formatUrl, headers: headersProp = {}, locale, t: tProp, theme, isMobile }, ref) => {
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
      const setOnSelectChatHandler = useChatStore((state) => state.setOnSelectChatHandler);
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
      const [showCancelDialog, setShowCancelDialog] = useState(false); // 深度研究取消确认弹框

      // Refs 用于在闭包和异步操作中保持最新值
      const curIdRef = useRef(""); // 当前生成的消息 ID
      const curSessionIdRef = useRef(""); // 当前会话 ID
      const activeMessageRef = useRef<ChatMessageRef>(null); // 活跃消息组件的引用
      const streamGenRef = useRef(0); // 流式请求的代次，用于切换后忽略旧流的渲染
      const generatingSessionRef = useRef<string | undefined>(undefined); // 当前正在生成回复的会话 ID
      const replyToMessageRef = useRef(""); // 当前 AI 回复对应的用户消息 ID（chunk 中的 reply_to_message）
      const fetchHistoryRef = useRef<(chatId: string) => Promise<void>>(); // 避免 handleStreamMessage 的循环依赖
      const pendingCancelActionRef = useRef<(() => void) | null>(null); // 深度研究取消确认后要执行的操作

      /**
       * 将 reply_end chunk 直接写入已持久化的消息 details 中
       * 用于 ActiveChatMessage 已卸载后收到 reply_end 的场景（如 cancel/error/timeout）
       */
      const patchReplyEndToMessage = useCallback((parsed: { message_id?: string; message_chunk?: string; chunk_type?: string }) => {
        const latestChat = useChatStore.getState().activeChat;
        if (!latestChat) return;
        const messageId = parsed.message_id || curIdRef.current;
        const messages = latestChat.messages || [];
        let targetIdx = messages.findIndex(
          (m) => m._id === messageId && m._source?.type === "assistant"
        );
        // 如果精确匹配失败，回退到最后一条 assistant 消息
        if (targetIdx === -1) {
          for (let i = messages.length - 1; i >= 0; i--) {
            if (messages[i]._source?.type === "assistant") {
              targetIdx = i;
              break;
            }
          }
        }
        if (targetIdx === -1) return;
        let payload;
        try {
          payload = parsed.message_chunk ? JSON.parse(parsed.message_chunk) : undefined;
        } catch {}
        if (!payload) return;
        const target = messages[targetIdx];
        const existingDetails: any[] = target._source?.details || [];
        // 如果已有 reply_end，用新的覆盖（user_cancelled/error/timeout 优先级高于 completed）
        const filteredDetails = existingDetails.filter((d) => d.type !== "reply_end");
        const updatedMessage = {
          ...target,
          _source: {
            ...target._source,
            details: [...filteredDetails, { type: "reply_end", payload }],
          },
        };
        const updatedMessages = [...messages];
        updatedMessages[targetIdx] = updatedMessage;
        setActiveChat({ ...latestChat, messages: updatedMessages });
      }, [setActiveChat]);

      /**
       * 将活跃消息组件中的 chunk 数据持久化到 activeChat.messages
       * 注意：不在此处 reset，避免在组件卸载前出现空白状态
       */
      const persistActiveMessage = useCallback(() => {
        const responseContent = activeMessageRef.current?.getResponseContent();
        const details = activeMessageRef.current?.getDetails() || [];
        const latestChat = useChatStore.getState().activeChat;
        if (latestChat && (responseContent || details.length > 0)) {
          const messageId = curIdRef.current || `ai-${Date.now()}`;
          const existingMessages = latestChat.messages || [];
          // 避免重复：只检查 assistant 类型消息
          if (!existingMessages.some((m) => m._id === messageId && m._source?.type === "assistant")) {
            const currentAst = useChatStore.getState().currentAssistant;
            const lastUserMsg = [...existingMessages].reverse().find((m) => m._source?.type === "user");
            const aiMessage: ChatMessageItem = {
              _id: messageId,
              _source: {
                type: "assistant",
                message: responseContent || "",
                assistant_id: currentAst?._id || "",
                reply_to_message: lastUserMsg?._id,
                details,
              },
            };
            setActiveChat({
              ...latestChat,
              messages: [...existingMessages, aiMessage],
            });
          }
        }
      }, [setActiveChat]);

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

                // 只将用户消息加入列表；AI 消息由 ActiveChatMessage 通过 chunks 渲染，
                // reply_end 时由 persistActiveMessage 持久化
                const isUserMessage = messageItem._source?.type === "user" || withPayload._source?.type === "user";
                nextChat = {
                  ...baseChat,
                  // 如果 baseChat._id 为空（新建会话的乐观状态），用后端返回的 session_id 覆盖
                  _id: baseChat._id || (typeof sessionId === "string" ? sessionId : "") || messageItem._id,
                  messages: isUserMessage
                    ? [...existingMessages, messageItem]
                    : existingMessages,
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

              // 从任何 chunk 更新消息 ID，确保 cancel 时 persistActiveMessage 使用正确的 AI message ID
              if (chunkData.message_id) {
                curIdRef.current = chunkData.message_id;
              }
              if (chunkData.reply_to_message) {
                replyToMessageRef.current = chunkData.reply_to_message;
              }

              // 将 chunk 数据传递给活跃的消息组件进行展示
              activeMessageRef.current?.addChunk(chunkData);

              // 标记回复结束，将完整回复（含 details）持久化到 activeChat.messages
              if (chunkData.chunk_type === "reply_end") {
                if (activeMessageRef.current) {
                  // 组件仍在，正常持久化
                  persistActiveMessage();
                } else {
                  // 组件已卸载（如 cancel 后），将 reply_end 补丁到已持久化的消息
                  patchReplyEndToMessage(chunkData);
                }
                setCurChatEnd(true);
              }
            }
          } catch (error) {
            // JSON 解析失败或其他错误处理
            console.error("Failed to parse chat message:", error);
          }
        },
        [setActiveChat, setCurChatEnd, incrementHistoryVersion, persistActiveMessage, patchReplyEndToMessage],
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
              attachments: attachments || [],
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
                  // Only handle reply_end if no new generation is actively running
                  if (!generatingSessionRef.current) {
                    try {
                      const parsed = JSON.parse(msg);
                      if (parsed.chunk_type === "reply_end") {
                        if (parsed.message_id) {
                          curIdRef.current = parsed.message_id;
                        }
                        // 将 reply_end 直接写入已持久化的消息（内部已去重）
                        patchReplyEndToMessage(parsed);
                        // 同时通知组件更新 UI（如果组件尚未卸载）
                        activeMessageRef.current?.addChunk(parsed);
                      }
                    } catch {}
                  }
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
        [handleStreamMessage, prepareChatSession, currentAssistant?._id, headersProp, setCurChatEnd, setActiveChat, persistActiveMessage, patchReplyEndToMessage],
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
              streamGenRef.current++;
              generatingSessionRef.current = undefined;
              activeMessageRef.current?.reset();
              setActiveMessageGen((v) => v + 1);
              setCurChatEnd(true);
              setTimedoutShow(false);
              setQuestion("");
              lastActiveChatIdRef.current = params.session_id;
              setActiveChat({ _id: params.session_id });
              await fetchHistory(params.session_id);
              incrementHistoryVersion();
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

          // 持久化上一条活跃消息（如果有数据），然后重置
          persistActiveMessage();
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
                attachments: attachments || [],
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
                  // Only handle reply_end if no new generation is actively running
                  if (!generatingSessionRef.current) {
                    try {
                      const parsed = JSON.parse(msg);
                      if (parsed.chunk_type === "reply_end") {
                        if (parsed.message_id) {
                          curIdRef.current = parsed.message_id;
                        }
                        // 将 reply_end 直接写入已持久化的消息（内部已去重）
                        patchReplyEndToMessage(parsed);
                        // 同时通知组件更新 UI（如果组件尚未卸载）
                        activeMessageRef.current?.addChunk(parsed);
                      }
                    } catch {}
                  }
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
          currentAssistant?._id,
          handleStreamMessage,
          headersProp,
          setCurChatEnd,
          persistActiveMessage,
          patchReplyEndToMessage,
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
        const sessionToCancel = generatingSessionRef.current || useChatStore.getState().activeChat?._id;

        if (sessionToCancel) {
          try {
            await Post(
              `/chat/${sessionToCancel}/_cancel?message_id=${replyToMessageRef.current || curIdRef.current}&lang=${locale || i18n.language}`,
              undefined,
              {},
              headersProp,
            );
          } catch (e) {
            console.error(e);
          }
        }
        // _cancel 正常返回后，持久化当前已渲染内容并恢复按钮状态
        persistActiveMessage();
        // 确保消息列表中有 assistant 消息，以便后续 reply_end 能通过 patchReplyEndToMessage 写入
        const latestChat = useChatStore.getState().activeChat;
        if (latestChat) {
          const messages = latestChat.messages || [];
          const messageId = curIdRef.current || `ai-${Date.now()}`;
          if (!messages.some((m) => m._source?.type === "assistant" && (m._id === messageId || !messageId))) {
            const currentAst = useChatStore.getState().currentAssistant;
            const lastUserMsg = [...messages].reverse().find((m) => m._source?.type === "user");
            const aiMessage: ChatMessageItem = {
              _id: messageId,
              _source: {
                type: "assistant",
                message: activeMessageRef.current?.getResponseContent() || "",
                assistant_id: currentAst?._id || "",
                reply_to_message: lastUserMsg?._id,
                details: activeMessageRef.current?.getDetails() || [],
              },
            };
            setActiveChat({
              ...latestChat,
              messages: [...messages, aiMessage],
            });
          }
        }
        setCurChatEnd(true);
      }, [headersProp, persistActiveMessage, setCurChatEnd, setActiveChat]);

      /**
       * 切换当前选中的对话
       * 负责重置状态并加载新对话的历史记录
       */
      const onSelectChat = useCallback(
        async (chat?: Chat) => {
          const generatingSession = generatingSessionRef.current;
          const curChatEndNow = useChatStore.getState().curChatEnd;

          // If a response is in progress, cancel it before switching.
          if (generatingSession && !curChatEndNow) {
            const assistantType = (useChatStore.getState().currentAssistant?._source?.type as string) || "simple";
            if (assistantType === "deep_research") {
              pendingCancelActionRef.current = () => {
                streamGenRef.current++;
                generatingSessionRef.current = undefined;
                cancelChat();

                activeMessageRef.current?.reset();
                setActiveMessageGen((v) => v + 1);
                setTimedoutShow(false);
                setQuestion("");

                setActiveChat(chat);
                if (chat?._id) {
                  lastActiveChatIdRef.current = chat._id;
                  fetchHistory(chat._id);
                } else {
                  lastActiveChatIdRef.current = undefined;
                }
              };
              setShowCancelDialog(true);
              return;
            }

            // 非深度研究：切换时递增 generation 使旧流不再渲染，同时发送取消请求
            streamGenRef.current++;
            generatingSessionRef.current = undefined;
            cancelChat();
          } else {
            // 递增 generation，使旧流的回调不再渲染
            streamGenRef.current++;
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
        [setActiveChat, setCurChatEnd, fetchHistory, cancelChat],
      );

      /**
       * 清除当前选中的对话（返回初始状态）
       * 如果深度研究正在进行，弹出确认框，确认后执行 cancel + 清空 + cb
       * 否则直接执行清空 + cb
       */
      const pendingRejectRef = useRef<(() => void) | null>(null);

      const clearChat = useCallback(async (cb?: () => void, force?: boolean, onReject?: () => void, keepAssistant?: boolean) => {
        const generatingSession = generatingSessionRef.current;
        const curChatEndNow = useChatStore.getState().curChatEnd;
        const assistantType = (useChatStore.getState().currentAssistant?._source?.type as string) || "simple";

        if (!force && generatingSession && !curChatEndNow && assistantType === "deep_research") {
          pendingCancelActionRef.current = () => {
            streamGenRef.current++;
            generatingSessionRef.current = undefined;
            cancelChat();

            activeMessageRef.current?.reset();
            setActiveMessageGen((v) => v + 1);
            setTimedoutShow(false);
            setQuestion("");

            setActiveChat(undefined);
            if (!keepAssistant) setCurrentAssistant(undefined);
            lastActiveChatIdRef.current = undefined;
            cb?.();
          };
          pendingRejectRef.current = onReject || null;
          setShowCancelDialog(true);
          return;
        }

        if (force) {
          // force 模式：直接取消并清空，跳过 onSelectChat 的弹框逻辑
          if (generatingSession && !curChatEndNow) {
            streamGenRef.current++;
            generatingSessionRef.current = undefined;
            cancelChat();
          }
          activeMessageRef.current?.reset();
          setActiveMessageGen((v) => v + 1);
          setCurChatEnd(true);
          setTimedoutShow(false);
          setQuestion("");
          setActiveChat(undefined);
          if (!keepAssistant) setCurrentAssistant(undefined);
          lastActiveChatIdRef.current = undefined;
          cb?.();
          return;
        }

        await onSelectChat(undefined);
        if (!keepAssistant) setCurrentAssistant(undefined);
        cb?.();
      }, [onSelectChat, cancelChat, setActiveChat, setCurChatEnd, setCurrentAssistant]);

      // 注册 onSelectChat 到 store，供 History 组件直接调用（避免先 setActiveChat 再撤回导致滚动抖动）
      useEffect(() => {
        setOnSelectChatHandler(onSelectChat);
      }, [onSelectChat, setOnSelectChatHandler]);

      // Use a ref to track the last processed active chat ID
      const lastActiveChatIdRef = useRef<string | undefined>(undefined);
      const lastActiveChatRef = useRef<Chat | undefined>(undefined);

      // 在每次渲染时同步 lastActiveChatRef，确保包含最新的流式消息
      // 只在 ID 仍匹配时更新，这样 ID 变化时 ref 保留的是旧会话的最新完整状态
      if (activeChat?._id && activeChat._id === lastActiveChatIdRef.current) {
        lastActiveChatRef.current = activeChat;
      }

      useEffect(() => {
        if (activeChat?._id && activeChat._id !== lastActiveChatIdRef.current) {
          // Skip only if the generating session matches the new activeChat
          // (i.e. the stream handler updated the session ID, not a user click).
          // If the user clicked a different history item while generation is in
          // progress, we still need to call onSelectChat to cancel the stream.
          if (!generatingSessionRef.current || generatingSessionRef.current !== activeChat._id) {
            // 深度研究进行中：回退 activeChat，弹出确认框，确认后再切换
            const curChatEndNow = useChatStore.getState().curChatEnd;
            const assistantType = (useChatStore.getState().currentAssistant?._source?.type as string) || "simple";
            if (generatingSessionRef.current && !curChatEndNow && assistantType === "deep_research") {
              const targetChat = activeChat;
              // 回退到当前正在生成的会话，防止 UI 提前切换（保留完整的消息列表）
              setActiveChat(lastActiveChatRef.current || { _id: lastActiveChatIdRef.current! });
              pendingCancelActionRef.current = () => {
                streamGenRef.current++;
                generatingSessionRef.current = undefined;
                cancelChat();

                activeMessageRef.current?.reset();
                setActiveMessageGen((v) => v + 1);
                setTimedoutShow(false);
                setQuestion("");

                lastActiveChatIdRef.current = targetChat._id;
                setActiveChat(targetChat);
                fetchHistory(targetChat._id);
              };
              setShowCancelDialog(true);
              return;
            }

            lastActiveChatIdRef.current = activeChat._id;
            lastActiveChatRef.current = activeChat;
            setTimeout(() => {
              onSelectChat(activeChat);
            }, 0);
          } else {
            lastActiveChatIdRef.current = activeChat._id;
            lastActiveChatRef.current = activeChat;
          }
        } else if (!activeChat?._id && lastActiveChatIdRef.current) {
          // Active chat was cleared externally (e.g. deleted from history)
          // Skip cancel dialog — force cancel directly since the chat is already removed
          const generatingSession = generatingSessionRef.current;
          const curChatEndNow = useChatStore.getState().curChatEnd;
          if (generatingSession && !curChatEndNow) {
            streamGenRef.current++;
            generatingSessionRef.current = undefined;
            cancelChat();
          }
          streamGenRef.current++;
          activeMessageRef.current?.reset();
          setActiveMessageGen((v) => v + 1);
          setCurChatEnd(true);
          setTimedoutShow(false);
          setQuestion("");
          lastActiveChatIdRef.current = undefined;
          setActiveChat(undefined);
        }
      }, [activeChat?._id, onSelectChat, cancelChat, setActiveChat, fetchHistory]);

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
          const assistantType = (useChatStore.getState().currentAssistant?._source?.type as string) || "simple";
          if (assistantType === "deep_research" && !useChatStore.getState().curChatEnd) {
            pendingCancelActionRef.current = () => cancelChat();
            setShowCancelDialog(true);
          } else {
            cancelChat();
          }
        },
        clearChat,
        onSelectChat,
        _isChatEnd: () => {
          const state = useChatStore.getState();
          if (state.curChatEnd) return true;
          const assistantType = (state.currentAssistant?._source?.type as string) || "simple";
          // 只有深度研究进行中才视为"未结束"，其他类型不阻拦
          return assistantType !== "deep_research";
        },
      }));

      const handleCancelWithConfirm = useCallback(() => {
        const assistantType = (currentAssistant?._source?.type as string) || "simple";
        if (assistantType === "deep_research" && !curChatEnd) {
          pendingCancelActionRef.current = () => cancelChat();
          setShowCancelDialog(true);
        } else {
          cancelChat();
        }
      }, [currentAssistant, curChatEnd, cancelChat]);

      return (
        <div className="flex flex-col rounded-md h-full overflow-hidden relative">
          <ChatContent
            activeChat={activeChat}
            activeMessageRef={activeMessageRef}
            activeMessageGen={activeMessageGen}
            timedoutShow={timedoutShow}
            Question={Question}
            handleSendMessage={(params) =>{
              const appendFeatureParams = useChatStore.getState().appendFeatureParams;
              if (appendFeatureParams) {
                appendFeatureParams(params);
              }
              handleSendMessage(activeChat, {
                ...params,
                message: Array.isArray(params.message) ? params.message.join("") : String(params.message ?? "")
              });
            }}
            getFileUrl={getFileUrl}
            formatUrl={formatUrl}
            requestHeaders={headersProp}
            curIdRef={curIdRef}
            t={t}
            currentAssistant={currentAssistant}
            theme={theme}
            isMobile={isMobile}
            onCancel={handleCancelWithConfirm}
          />

          <CancelDeepResearchDialog
            isOpen={showCancelDialog}
            active={activeChat}
            setIsOpen={(open) => {
              setShowCancelDialog(open);
              if (!open) {
                pendingCancelActionRef.current = null;
                pendingRejectRef.current?.();
                pendingRejectRef.current = null;
              }
            }}
            handleRemove={() => {
              setShowCancelDialog(false);
              pendingRejectRef.current = null;
              pendingCancelActionRef.current?.();
              pendingCancelActionRef.current = null;
            }}
            t={t}
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
