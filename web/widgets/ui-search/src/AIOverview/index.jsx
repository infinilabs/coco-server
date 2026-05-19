import { AIAnswer } from '@infinilabs/ai-answer';
import { Spin } from 'antd';

const AIOverview = (props) => {
  const { config = {}, data, loading, visible, setVisible, theme, onChatContinue } = props;

  if (!visible) return null;

  return (
    <Spin spinning={loading}>
      <AIAnswer
        title="智能解读"
        content={data?.response?.message_chunk || ""}
        onContinue={() => onChatContinue?.()}
        maxHeight={Number.isInteger(Number(config.height)) ? Number(config.height) - 154 : undefined}
        theme={theme}
        containerClass="!border-0 px-16px !pt-16px !pb-16px"
      />
    </Spin>
  )
};

export default AIOverview;
