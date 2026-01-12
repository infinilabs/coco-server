import { useRef, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChatMessage } from "@infinilabs/chat-message";

// Demo data for simulation
const DEMO_DATA = {
  hits: {
    hits: [
      {
        _id: "demo-1",
        _source: {
          type: "user",
          message: "What is Coco AI?",
          created: new Date().toISOString()
        }
      },
      {
        _id: "demo-2",
        _source: {
          type: "assistant",
          message: "Coco AI is a powerful, open-source, and cross-platform unified AI search and productivity tool.",
          details: [
            {
              type: "query_intent",
              payload: {
                category: "Info Retrieval",
                intent: "Request for Information",
                query: ["What is Coco AI?", "Can you explain what Coco AI is?"],
                keyword: ["Coco AI", "definition"],
                suggestion: ["What is the purpose of Coco AI?", "What are the features of Coco AI?"]
              }
            },
            {
              type: "fetch_source",
              payload: [
                {
                  id: "doc1",
                  title: "Coco AI v0.2 Unleashed",
                  summary: "Coco AI v0.2 is now available! A powerful, open-source AI-powered search.",
                  url: "http://blog.infinilabs.com/posts/2025/03-16-product-released-coco-ai-v0.2/"
                },
                {
                  id: "doc2",
                  title: "Introducing Coco AI",
                  summary: "Discover how Coco AI revolutionizes enterprise search and collaboration.",
                  url: "http://blog.infinilabs.com/posts/2024/introducing-coco-ai/"
                }
              ]
            },
            {
              type: "pick_source",
              payload: [
                {
                  id: "doc1",
                  title: "Coco AI v0.2 Unleashed",
                  explain: "This document discusses the release of Coco AI v0.2."
                }
              ]
            },
            {
              type: "deep_read",
              description: "Obtaining and analyzing documents in depth: Coco AI v0.2 Unleashed\nObtaining and analyzing documents in depth: Introducing Coco AI\n"
            },
            {
              type: "think",
              description: "Analyzing the user's request about Coco AI. Based on the retrieved documents, I should explain its core features like unified search and AI assistant capabilities."
            }
          ]
        }
      }
    ]
  }
};

const ChatContent = (props) => {
  const {
    activeChat,
    messages: propMessages,
    query_intent: prop_query_intent,
    tools: prop_tools,
    fetch_source: prop_fetch_source,
    pick_source: prop_pick_source,
    deep_read: prop_deep_read,
    think: prop_think,
    response: prop_response,
    timedoutShow,
    Question,
    handleSendMessage,
    formatUrl,
    curChatEnd: prop_curChatEnd, // Boolean: true if current chat stream ended
    locale = 'en',
    theme = 'light'
  } = props;

  const { t } = useTranslation();
  const messagesEndRef = useRef(null);
  const scrollRef = useRef(null);
  const abortControllerRef = useRef(null);

  // Demo State
  const [isDemo, setIsDemo] = useState(false);
  const [demoMessages, setDemoMessages] = useState([]);
  const [demoState, setDemoState] = useState({
    query_intent: null,
    tools: null,
    fetch_source: null,
    pick_source: null,
    deep_read: null,
    think: null,
    response: null,
    curChatEnd: true,
    question: ""
  });

  // Effective props (either from parent or demo)
  const messages = isDemo ? demoMessages : (activeChat?.messages || propMessages || []);
  const query_intent = isDemo ? demoState.query_intent : prop_query_intent;
  const tools = isDemo ? demoState.tools : prop_tools;
  const fetch_source = isDemo ? demoState.fetch_source : prop_fetch_source;
  const pick_source = isDemo ? demoState.pick_source : prop_pick_source;
  const deep_read = isDemo ? demoState.deep_read : prop_deep_read;
  const think = isDemo ? demoState.think : prop_think;
  const response = isDemo ? demoState.response : prop_response;
  const curChatEnd = isDemo ? demoState.curChatEnd : prop_curChatEnd;
  const currentQuestion = isDemo ? demoState.question : Question;

  // Determine if we should show the active/streaming message
  const showActiveMessage = (!curChatEnd ||
    query_intent ||
    tools ||
    fetch_source ||
    pick_source ||
    deep_read ||
    think ||
    response) && (isDemo || activeChat?._source?.id || currentQuestion);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  // Scroll to bottom when messages change or streaming data updates
  useEffect(() => {
    scrollToBottom();
  }, [
    messages.length,
    query_intent?.message_chunk,
    fetch_source?.message_chunk,
    pick_source?.message_chunk,
    deep_read?.message_chunk,
    think?.message_chunk,
    response?.message_chunk,
    curChatEnd
  ]);

  const handleScroll = (event) => {
    // Scroll logic can be extended here if needed (e.g., show "scroll to bottom" button)
  };

  // Demo Simulation Logic
  const runDemo = async () => {
    if (isDemo && !demoState.curChatEnd) return; // Already running

    setIsDemo(true);
    setDemoMessages([]);
    setDemoState({
      query_intent: null,
      tools: null,
      fetch_source: null,
      pick_source: null,
      deep_read: null,
      think: null,
      response: null,
      curChatEnd: false,
      question: "What is Coco AI?"
    });

    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    // 1. Add User Message
    const userMsg = DEMO_DATA.hits.hits[0];
    setDemoMessages([userMsg]);

    await new Promise(r => setTimeout(r, 500));

    // 2. Start Streaming Assistant Response
    const demoHit = DEMO_DATA.hits.hits[1];
    const demoDetails = demoHit._source.details || [];
    const demoMessage = demoHit._source.message;

    for (const detail of demoDetails) {
        if (abortController.signal.aborted) return;
        const { type, payload, description } = detail;

        if (type === "query_intent") {
            await new Promise(r => setTimeout(r, 500));
            setDemoState(prev => ({
                ...prev,
                query_intent: { chunk_type: "query_intent", message_chunk: "<JSON>" + JSON.stringify(payload) + "</JSON>" }
            }));
        } else if (type === "fetch_source") {
            const sources = payload;
            setDemoState(prev => ({
                ...prev,
                fetch_source: { chunk_type: "fetch_source", message_chunk: `<Payload total=${sources.length}>` }
            }));
            await new Promise(r => setTimeout(r, 300));
            setDemoState(prev => ({
                ...prev,
                fetch_source: { chunk_type: "fetch_source", message_chunk: JSON.stringify(sources) }
            }));
        } else if (type === "pick_source") {
            await new Promise(r => setTimeout(r, 800));
            setDemoState(prev => ({
                ...prev,
                pick_source: { chunk_type: "pick_source", message_chunk: "<JSON>" + JSON.stringify(payload) + "</JSON>" }
            }));
        } else if (type === "deep_read") {
            const lines = (description || "").split("\n").filter(l => l.trim());
            for (const line of lines) {
                 const title = line.split(":").pop()?.trim();
                 if (title) {
                     setDemoState(prev => ({
                         ...prev,
                         deep_read: { chunk_type: "deep_read", message_chunk: title }
                     }));
                     await new Promise(r => setTimeout(r, 500));
                 }
            }
        } else if (type === "think") {
             const text = description || "";
             for (let i = 0; i < text.length; i+=5) {
                if (abortController.signal.aborted) return;
                setDemoState(prev => ({
                    ...prev,
                    think: { chunk_type: "think", message_chunk: text.slice(i, i+5) }
                }));
                await new Promise(r => setTimeout(r, 20));
             }
        }
        await new Promise(r => setTimeout(r, 200));
    }

    // 3. Stream Text Response
    for (let i = 0; i < demoMessage.length; i+=2) { // optimize a bit
        if (abortController.signal.aborted) return;
        const char = demoMessage.slice(i, i+2);
        setDemoState(prev => ({
            ...prev,
            response: { chunk_type: "response", message_chunk: char }
        }));
        await new Promise(r => setTimeout(r, 10));
    }

    // 4. Finish
    setDemoState(prev => ({ ...prev, curChatEnd: true }));
    setDemoMessages(prev => [...prev, demoHit]);
    setIsDemo(false);
  };

  return (
    <div
      ref={scrollRef}
      className={`w-full h-full overflow-x-hidden overflow-y-auto custom-scrollbar relative flex flex-col gap-4 p-4 ${theme === 'dark' ? 'dark' : ''}`}
      onScroll={handleScroll}
    >
      {/* Demo Button */}
      {messages.length === 0 && (
        <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 text-center">
            <button
                onClick={runDemo}
                className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition"
            >
                Start Demo Chat
            </button>
        </div>
      )}

      {/* History Messages */}
      {messages.map((msg, index) => (
        <ChatMessage
          key={msg._id || index}
          message={msg}
          locale={locale}
          theme={theme}
          rootClassName="ui-search-chat-message"
          isTyping={false}
          onResend={handleSendMessage}
        />
      ))}

      {/* Active Streaming Message */}
      {showActiveMessage ? (
        <ChatMessage
          key="current"
          message={{
            _id: "current",
            _source: {
              type: "assistant",
              assistant_id: messages[messages.length - 1]?._source?.assistant_id || 'coco-bot',
              message: "",
              question: Question,
            },
          }}
          locale={locale}
          theme={theme}
          rootClassName="ui-search-chat-message"
          onResend={handleSendMessage}
          isTyping={!curChatEnd}
          query_intent={query_intent}
          tools={tools}
          fetch_source={fetch_source}
          pick_source={pick_source}
          deep_read={deep_read}
          think={think}
          response={response}
          formatUrl={formatUrl}
        />
      ) : null}

      {/* Timeout Message */}
      {timedoutShow && (
        <ChatMessage
          key="timedout"
          message={{
            _id: "timedout",
            _source: {
              type: "assistant",
              message: t("assistant.chat.timedout"),
              question: Question,
            },
          }}
          locale={locale}
          theme={theme}
          rootClassName="ui-search-chat-message"
          onResend={handleSendMessage}
          isTyping={false}
        />
      )}

      <div ref={messagesEndRef} />
    </div>
  );
};

export default ChatContent;
