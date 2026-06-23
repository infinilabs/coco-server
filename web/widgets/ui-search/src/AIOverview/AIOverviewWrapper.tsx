import { useEffect, useRef, useState } from "react";
import AIOverview from ".";

interface AskBody {
  message?: string;
  t?: number;
}

interface AIOverviewConfig {
  assistant?: string;
  height?: string | number;
  [key: string]: unknown;
}

interface ChunkData {
  chunk_type: string;
  message_chunk: string;
  session_id: string;
}

type DataState = Record<string, { message_chunk: string }> | undefined;

const getReplyEndErrorMessage = (messageChunk?: string) => {
  if (!messageChunk) return undefined;
  try {
    const payload = JSON.parse(messageChunk);
    return payload?.error || payload?.reason;
  } catch {
    return undefined;
  }
};

interface AIOverviewWrapperProps {
  readonly askBody?: AskBody;
  readonly config: AIOverviewConfig;
  readonly onAsk: (
    assistant: string,
    message: string,
    onData: (res: ChunkData) => void,
    setLoading: (loading: boolean) => void
  ) => void;
  readonly theme?: "auto" | "dark" | "light";
  readonly onChatContinue?: (session_id: string) => void;
  readonly requestHeaders?: Record<string, string>;
}

const AIOverviewWrapper = (props: AIOverviewWrapperProps) => {
  const { askBody, config, onAsk, theme, onChatContinue, requestHeaders } = props;

  const [data, setData] = useState<DataState>();
  const [loading, setLoading] = useState(false);
  const [isReplyEnd, setIsReplyEnd] = useState(false);
  const [visible, setVisible] = useState(true);
  const sessionIdRef = useRef<string | undefined>(undefined);
  const requestIdRef = useRef(0);

  const handleMessage = (data: ChunkData, prevData: DataState): DataState => {
    const type = data.chunk_type;
    if (sessionIdRef.current !== data.session_id) {
      sessionIdRef.current = data.session_id;
    }
    const item = prevData?.[type];
    let newMessage = item ? item.message_chunk : '';
    newMessage += data.message_chunk;
    if (data.chunk_type === "reply_end") {
      setIsReplyEnd(true);
      setLoading(false);
      const errorMessage = getReplyEndErrorMessage(data.message_chunk);
      if (errorMessage) {
        const response = prevData?.response;
        return {
          ...(prevData || {}),
          response: {
            ...(response || {}),
            message_chunk: response?.message_chunk || errorMessage
          },
          [type]: {
            ...(item || {}),
            message_chunk: newMessage
          }
        };
      }
    }
    return {
      ...(prevData || {}),
      [type]: {
        ...(item || {}),
        message_chunk: newMessage
      }
    };
  };

  const handleAsk = (message: string | undefined, config: AIOverviewConfig) => {
    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;
    sessionIdRef.current = undefined;
    setData(undefined);
    setIsReplyEnd(false);

    if (!message || !config.assistant) {
      setLoading(false);
      return;
    }

    const setCurrentLoading = (loading: boolean) => {
      if (requestIdRef.current === requestId) {
        setLoading(loading);
      }
    };

    setCurrentLoading(true);
    onAsk(config.assistant, message, (res) => {
      if (requestIdRef.current !== requestId) return;
      setData((prev) => requestIdRef.current === requestId ? handleMessage(res, prev) : prev);
    }, setCurrentLoading);
  };

  useEffect(() => {
    handleAsk(askBody?.message, config);
  }, [askBody?.message, askBody?.t, JSON.stringify(config)]);

  useEffect(() => {
    if (askBody?.t) {
      setVisible(true);
    }
  }, [askBody?.t]);

  return (
    <AIOverview
      config={config}
      data={data}
      loading={loading}
      visible={visible}
      setVisible={setVisible}
      theme={theme}
      onChatContinue={() => sessionIdRef.current && onChatContinue?.(sessionIdRef.current)}
      isReplyEnd={isReplyEnd}
      requestHeaders={requestHeaders}
    />
  );
};

export default AIOverviewWrapper;
