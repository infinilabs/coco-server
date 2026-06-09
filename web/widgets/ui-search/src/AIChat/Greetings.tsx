import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from "@infinilabs/markdown";

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

  const message =
    currentAssistant?._source?.chat_settings?.greeting_message ||
    t("assistant.chat.greetings");

  return (
    <div className="w-full py-8 pl-7">
      <div className="cm-markdown prose dark:prose-invert prose-sm max-w-none">
        <Markdown content={message} />
      </div>
    </div>
  );
};
