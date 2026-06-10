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

interface AIOverviewWrapperProps {
  askBody?: AskBody;
  config: AIOverviewConfig;
  onAsk: (
    assistant: string,
    message: string,
    onData: (res: ChunkData) => void,
    setLoading: (loading: boolean) => void
  ) => void;
  theme?: "light" | "dark" | "auto";
  onChatContinue?: (session_id: string) => void;
}

const AIOverviewWrapper = (props: AIOverviewWrapperProps) => {
  const { askBody, config, onAsk, theme, onChatContinue } = props;

  const [data, setData] = useState<DataState>();
  const [loading, setLoading] = useState(false);
  const [visible, setVisible] = useState(true);
  const sessionIdRef = useRef<string | undefined>(undefined);

  const handleMessage = (data: ChunkData, prevData: DataState): DataState => {
    const type = data.chunk_type;
    if (sessionIdRef.current !== data.session_id) {
      sessionIdRef.current = data.session_id;
    }
    const item = prevData?.[type];
    let newMessage = item ? item.message_chunk : '';
    if (type === 'deep_read') {
      newMessage += "&" + data.message_chunk;
    } else {
      newMessage += data.message_chunk;
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
    if (message && config.assistant) {
      setData(undefined);
      setLoading(true);
      onAsk(config.assistant, message, (res) => {
        setData((prev) => handleMessage(res, prev));
      }, setLoading);
    }
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
    />
  );
};

export default AIOverviewWrapper;
