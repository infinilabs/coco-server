import { ChatContent as BaseChatContent } from "@infinilabs/chat-message";

const ChatContent = (props) => {
  const {
    activeChat,
    messages: propMessages,
    query_intent,
    tools,
    fetch_source,
    pick_source,
    deep_read,
    think,
    response,
    timedoutShow,
    Question,
    handleSendMessage,
    formatUrl,
    registerStreamHandler,
    onStream,
    apiConfig,
    locale = "en",
    theme = "light",
  } = props;

  const effectiveActiveChat =
    activeChat ||
    (Array.isArray(propMessages) && propMessages.length > 0
      ? {
          _id: "chat",
          messages: propMessages,
          _source: {
            id: "chat",
          },
        }
      : undefined);

  const getFileUrl = (path) => path;

  return (
    <BaseChatContent
      activeChat={effectiveActiveChat}
      query_intent={query_intent}
      tools={tools}
      fetch_source={fetch_source}
      pick_source={pick_source}
      deep_read={deep_read}
      think={think}
      response={response}
      timedoutShow={timedoutShow}
      Question={Question}
      handleSendMessage={handleSendMessage}
      getFileUrl={getFileUrl}
      formatUrl={formatUrl}
      theme={theme}
      locale={locale}
      registerStreamHandler={registerStreamHandler}
      onStream={onStream}
      apiConfig={apiConfig}
    />
  );
};

export default ChatContent;
