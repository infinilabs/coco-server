import { useState } from "react";
import clsx from "clsx";
import {
  Check,
  Copy,
  Volume2,
} from "lucide-react";

const RefreshOnlyIds = ["timedout", "error"];

export const MessageActions = ({
  id,
  content,
  actionClassName,
  actionIconSize,
  copyButtonId,
}) => {
  const [copied, setCopied] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);

  const isRefreshOnly = RefreshOnlyIds.includes(id);

  const handleCopy = async () => {
    try {
      // await copyToClipboard(content);
      setCopied(true);
      const timerID = setTimeout(() => {
        setCopied(false);
        clearTimeout(timerID);
      }, 2000);
    } catch (err) {
      console.error("copy error:", err);
    }
  };


  const handleSpeak = () => {
    if ("speechSynthesis" in window) {
      if (isSpeaking) {
        window.speechSynthesis.cancel();
        setIsSpeaking(false);
        return;
      }

      const utterance = new SpeechSynthesisUtterance(content);
      utterance.lang = "zh-CN";

      utterance.onend = () => {
        setIsSpeaking(false);
      };

      setIsSpeaking(true);
      window.speechSynthesis.speak(utterance);
    }
  };

  const commonCls = 'p-1 hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors border-0 bg-transparent cursor-pointer'

  return (
    <div className={clsx("flex items-center gap-1 mt-2", actionClassName)}>
      {!isRefreshOnly && (
        <button
          id={copyButtonId}
          onClick={handleCopy}
          className={commonCls}
        >
          {copied ? (
            <Check
              className="w-4 h-4 text-[#38C200] dark:text-[#38C200]"
              style={{
                width: actionIconSize,
                height: actionIconSize,
              }}
            />
          ) : (
            <Copy
              className="w-4 h-4 text-[#666666] dark:text-[#A3A3A3]"
              style={{
                width: actionIconSize,
                height: actionIconSize,
              }}
            />
          )}
        </button>
      )}
      {!isRefreshOnly && (
        <button
          onClick={handleSpeak}
          className={commonCls}
        >
          <Volume2
            className={`w-4 h-4 ${
              isSpeaking
                ? "text-[#1990FF] dark:text-[#1990FF]"
                : "text-[#666666] dark:text-[#A3A3A3]"
            }`}
            style={{
              width: actionIconSize,
              height: actionIconSize,
            }}
          />
        </button>
      )}
    </div>
  );
};
