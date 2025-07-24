import { useState, useCallback } from "react";

export default function useMessage() {
  const [data, setData] = useState({});

  const handleMessage = useCallback((data) => {
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
