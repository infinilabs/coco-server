import { AIAnswer } from '@infinilabs/ai-answer';

const AIOverview = (props) => {
  const { config = {}, data, loading, visible, setVisible, theme } = props;

  if (!data || !data.response || !visible) return null;

  return (
    <AIAnswer
      title="智能解读"
      content={data.response.message_chunk || ""}
      onContinue={() => console.log('点击了继续追问')}
      maxHeight={Number.isInteger(Number(config.height)) ? Number(config.height) - 154 : undefined}
      // theme={theme}
    />
  )
};

export default AIOverview;
