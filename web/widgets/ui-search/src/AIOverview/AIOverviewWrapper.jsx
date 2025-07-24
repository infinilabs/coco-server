import { useEffect, useState } from "react";
import AIOverview from ".";

const AIOverviewWrapper = (props) => {
  const { askBody, config, onAsk } = props;

  const [data, setData] = useState();
  const [loading, setLoading] = useState(false)
  const [visible, setVisible] = useState(true);

  const handleAsk = (message, config) => {
    if (message && config.assistant) {
      setData()
      onAsk(config.assistant, message, (res) => {
        setData((prev) => handleMessage(res, prev))
      }, setLoading)
    }
  }

  const handleMessage = (data, prevData) => {
    const type = data.chunk_type
    const item = prevData?.[type]
    let newMessage = item ? item.message_chunk : ''
    if (type === 'deep_read') {
      newMessage += "&" + data.message_chunk
    } else {
      newMessage += data.message_chunk
    }
    return {
      ...(prevData || {}),
      [type]: {
        ...(item || {}),
        message_chunk: newMessage
      }
    }
  }

  useEffect(() => {
    handleAsk(askBody?.message, config)
  }, [askBody?.message, askBody?._t, config])

  useEffect(() => {
    if (askBody?._t) {
      setVisible(true)
    }
  }, [askBody?._t])

  return <AIOverview config={config} data={data} loading={loading} visible={visible} setVisible={setVisible}/>
}

export default AIOverviewWrapper