import { AIAnswer } from '@infinilabs/ai-answer';

const AIOverview = (props) => {
  const { config = {}, data, loading, visible, setVisible, theme, onChatContinue } = props;

  if (!data || !data.response || !visible) return null;

  return (
    <AIAnswer
      title="智能解读"
      content={data.response.message_chunk || ""}
      onContinue={() => onChatContinue?.()}
      maxHeight={Number.isInteger(Number(config.height)) ? Number(config.height) - 154 : undefined}
      // theme={theme}
    />
  )
};

export default AIOverview;
