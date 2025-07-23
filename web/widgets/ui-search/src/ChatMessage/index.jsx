import { memo } from "react";
import Markdown from "./Markdown";
import { MessageActions } from "./MessageActions";

export const ChatMessage = memo(function ChatMessage({
  message,
  isTyping,
  response,
  onResend,
  actionClassName,
  actionIconSize,
  copyButtonId,
  showActions = true,
  output = 'markdown'
}) {

  const messageContent = message?._source?.message || "";
  const question = message?._source?.question || "";

  const currentContent = messageContent || response?.message_chunk || ""

  const showMessageActions = showActions && isTyping === false && !!currentContent;

  const renderContent = () => {
    let content;
    if (output === 'markdown') {
      content = (
        <Markdown
          content={currentContent}
          loading={isTyping}
          onDoubleClickCapture={() => {}}
        />
      )
    } else if (output === 'html') {
      content = <div dangerouslySetInnerHTML={{ __html: currentContent }} />;
    } else if (output === 'text') {
      content = currentContent
    }
    return (
      <>
        {content}
        {isTyping && (
          <div className="inline-block w-1.5 h-5 ml-0.5 -mb-0.5 bg-[#666666] dark:bg-[#A3A3A3] rounded-sm animate-typing" />
        )}
        {showMessageActions && (
          <MessageActions
            id={message._id}
            content={currentContent}
            question={question}
            actionClassName={actionClassName}
            actionIconSize={actionIconSize}
            copyButtonId={copyButtonId}
            onResend={() => {
              onResend && onResend(question);
            }}
          />
        )}
      </>
    );
  };

  return (
    <div className={"flex justify-start"}>
      <div className={`flex gap-4 w-full`}>
        <div className={`w-full space-y-2 text-left`}>
          <div className="w-full max-w-none">
            <div className="w-full text-[#333] leading-relaxed">
              {renderContent()}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
});
