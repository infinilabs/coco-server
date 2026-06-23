import { useState, useCallback } from "react";

interface ChunkData {
  chunk_type: string;
  message_chunk: string;
  [key: string]: any;
}

interface MessageItem {
  message_chunk: string;
  [key: string]: any;
}

type MessageData = Record<string, MessageItem> | undefined;

export default function useMessage() {
  const [data, setData] = useState<MessageData>({});

  const handleMessage = useCallback((data: ChunkData) => {
    setData((prev) => {
      const type = data.chunk_type
      const item = prev?.[type]
      let newMessage = item ? item.message_chunk : ''
      if (type === 'deep_read') {
        newMessage += "&" + data.message_chunk
      } else {
        newMessage += data.message_chunk
      }
      return {
        ...(prev || {}),
        [type]: {
          ...(item || {}),
          message_chunk: newMessage
        }
      }
    })
  }, [])

  const clearData = () => {
    setData(undefined);
  };

  return {
    data,
    handleMessage,
    clearData,
  };
}
