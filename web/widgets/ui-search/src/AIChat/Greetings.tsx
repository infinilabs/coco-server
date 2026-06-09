import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { ChatMessage } from "./ChatMessage/components";

import { useChatStore } from "./stores/chatStore";

type Assistant = {
  _source?: {
    chat_settings?: {
      greeting_message?: string;
    };
  };
};

interface GreetingsProps {
  t?: TFunction;
}

export const Greetings = ({ t: tProp }: GreetingsProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const currentAssistant = useChatStore(
    (state) => state.currentAssistant as Assistant | null
  );

  return (
    <ChatMessage
      key={"greetings"}
      message={{
        _id: "greetings",
        _source: {
          type: "assistant",
          message:
            currentAssistant?._source?.chat_settings?.greeting_message ||
            t("assistant.chat.greetings"),
        },
      }}
      currentAssistant={currentAssistant}
      t={t}
    />
  );
};
