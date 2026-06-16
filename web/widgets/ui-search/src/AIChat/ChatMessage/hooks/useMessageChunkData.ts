import { useState, useCallback, useRef } from "react";

import type { IChunkData } from "../types/chat";

export default function useMessageChunkData() {
  const [query_intent, setQuery_intent] = useState<IChunkData>();
  const [tools, setTools] = useState<IChunkData>();
  const [fetch_source, setFetch_source] = useState<IChunkData>();
  const [pick_source, setPick_source] = useState<IChunkData>();
  const [deep_read, setDeep_read] = useState<IChunkData>();
  const [think, setThink] = useState<IChunkData>();
  const [response, setResponse] = useState<IChunkData>();
  const [deepResearch, setDeepResearch] = useState<IChunkData[]>([]);
  const [replyEnd, setReplyEnd] = useState<IChunkData[]>([]);

  // Refs mirror state for synchronous access (e.g. getDetails/getResponseContent)
  const queryIntentRef = useRef<IChunkData>();
  const toolsRef = useRef<IChunkData>();
  const fetchSourceRef = useRef<IChunkData>();
  const pickSourceRef = useRef<IChunkData>();
  const deepReadRef = useRef<IChunkData>();
  const thinkRef = useRef<IChunkData>();
  const responseRef = useRef<IChunkData>();
  const deepResearchRef = useRef<IChunkData[]>([]);
  const replyEndRef = useRef<IChunkData[]>([]);

  const handlers = {
    deal_query_intent: useCallback((data: IChunkData) => {
      const prev = queryIntentRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + (data.message_chunk || "") };
      queryIntentRef.current = next;
      setQuery_intent(next);
    }, []),
    deal_tools: useCallback((data: IChunkData) => {
      const prev = toolsRef.current;
      const prevItems = (prev as any)?.tool_call_items || [];
      let newItems = prevItems;
      if (data.tool_call_message_chunk) {
        try {
          newItems = [...prevItems, JSON.parse(data.tool_call_message_chunk)];
        } catch {}
      }
      const next = !prev
        ? { ...data, tool_call_items: newItems }
        : {
            ...prev,
            message_chunk: (prev.message_chunk || "") + (data.message_chunk || ""),
            tool_call_message_chunk: data.tool_call_message_chunk || prev.tool_call_message_chunk,
            tool_call_items: newItems,
          };
      toolsRef.current = next;
      setTools(next);
    }, []),
    deal_fetch_source: useCallback((data: IChunkData) => {
      const prev = fetchSourceRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + (data.message_chunk || "") };
      fetchSourceRef.current = next;
      setFetch_source(next);
    }, []),
    deal_pick_source: useCallback((data: IChunkData) => {
      const prev = pickSourceRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + (data.message_chunk || "") };
      pickSourceRef.current = next;
      setPick_source(next);
    }, []),
    deal_deep_read: useCallback((data: IChunkData) => {
      const prev = deepReadRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + "&" + (data.message_chunk || "") };
      deepReadRef.current = next;
      setDeep_read(next);
    }, []),
    deal_think: useCallback((data: IChunkData) => {
      const prev = thinkRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + (data.message_chunk || "") };
      thinkRef.current = next;
      setThink(next);
    }, []),
    deal_response: useCallback((data: IChunkData) => {
      const prev = responseRef.current;
      const next = !prev ? data : { ...prev, message_chunk: (prev.message_chunk || "") + (data.message_chunk || "") };
      responseRef.current = next;
      setResponse(next);
    }, []),
    deal_deep_research: useCallback((data: IChunkData) => {
      const next = [...deepResearchRef.current, data];
      deepResearchRef.current = next;
      setDeepResearch(next);
    }, []),
    deal_reply_end: useCallback((data: IChunkData) => {
      const next = [...replyEndRef.current, data];
      replyEndRef.current = next;
      setReplyEnd(next);
    }, []),
  };

  const clearAllChunkData = () => {
    return new Promise<void>((resolve) => {
      setQuery_intent(undefined);
      setTools(undefined);
      setFetch_source(undefined);
      setPick_source(undefined);
      setDeep_read(undefined);
      setThink(undefined);
      setResponse(undefined);
      setDeepResearch([]);
      setReplyEnd([]);
      queryIntentRef.current = undefined;
      toolsRef.current = undefined;
      fetchSourceRef.current = undefined;
      pickSourceRef.current = undefined;
      deepReadRef.current = undefined;
      thinkRef.current = undefined;
      responseRef.current = undefined;
      deepResearchRef.current = [];
      replyEndRef.current = [];
      setTimeout(resolve, 0);
    });
  };

  return {
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
    refs: {
      queryIntentRef,
      toolsRef,
      fetchSourceRef,
      pickSourceRef,
      deepReadRef,
      thinkRef,
      responseRef,
      deepResearchRef,
      replyEndRef,
    },
  };
}
