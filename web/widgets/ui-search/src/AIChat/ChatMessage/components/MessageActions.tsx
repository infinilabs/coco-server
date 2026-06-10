import { useEffect, useRef, useState } from "react";
import clsx from "clsx";
import {
  Check,
  Copy,
  ThumbsUp,
  ThumbsDown,
  Volume,
  Volume1,
  Volume2,
  RotateCcw,
} from "lucide-react";

import { copyToClipboard } from "../utils";

interface MessageActionsProps {
  id: string;
  content: string;
  question?: string;
  actionClassName?: string;
  actionIconSize?: number;
  copyButtonId?: string;
  onResend?: () => void;
}

const RefreshOnlyIds = ["timedout", "error"];

export const MessageActions = ({
  id,
  content,
  question,
  actionClassName,
  actionIconSize,
  copyButtonId,
  onResend,
}: MessageActionsProps) => {
  const [copied, setCopied] = useState(false);
  const [liked, setLiked] = useState(false);
  const [disliked, setDisliked] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [volumeFrame, setVolumeFrame] = useState(0);
  const [isResending, setIsResending] = useState(false);
  const utteranceRef = useRef<SpeechSynthesisUtterance | null>(null);
  const resumeTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const volumeAnimRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const stoppedByUserRef = useRef(false);

  const isRefreshOnly = RefreshOnlyIds.includes(id);

  useEffect(() => {
    if (isSpeaking) {
      volumeAnimRef.current = setInterval(() => {
        setVolumeFrame((f) => (f + 1) % 3);
      }, 400);
    } else {
      if (volumeAnimRef.current) {
        clearInterval(volumeAnimRef.current);
        volumeAnimRef.current = null;
      }
      setVolumeFrame(0);
    }
    return () => {
      if (volumeAnimRef.current) {
        clearInterval(volumeAnimRef.current);
      }
    };
  }, [isSpeaking]);

  useEffect(() => {
    return () => {
      const synth = typeof window !== "undefined" ? window.speechSynthesis : undefined;
      if (synth) {
        synth.cancel();
      }
      if (resumeTimerRef.current) {
        clearInterval(resumeTimerRef.current);
      }
    };
  }, []);

  const handleCopy = async () => {
    try {
      await copyToClipboard(content);
      setCopied(true);
      const timerID = setTimeout(() => {
        setCopied(false);
        clearTimeout(timerID);
      }, 2000);
    } catch (err) {
      console.error("copy error:", err);
    }
  };

  const handleLike = () => {
    setLiked(!liked);
    setDisliked(false);
  };

  const handleDislike = () => {
    setDisliked(!disliked);
    setLiked(false);
  };

  const handleSpeak = () => {
    const synth = typeof window !== "undefined" ? window.speechSynthesis : undefined;
    if (!synth) return;

    if (isSpeaking) {
      stoppedByUserRef.current = true;
      synth.cancel();
      setIsSpeaking(false);
      utteranceRef.current = null;
      if (resumeTimerRef.current) {
        clearInterval(resumeTimerRef.current);
        resumeTimerRef.current = null;
      }
      return;
    }

    try {
      const stopSpeaking = () => {
        setIsSpeaking(false);
        utteranceRef.current = null;
        if (resumeTimerRef.current) {
          clearInterval(resumeTimerRef.current);
          resumeTimerRef.current = null;
        }
      };

      stoppedByUserRef.current = false;
      synth.cancel();
      const utter = new SpeechSynthesisUtterance(content);
      utter.onend = stopSpeaking;
      utter.onerror = () => {
        if (!stoppedByUserRef.current) {
          stopSpeaking();
        }
      };
      utteranceRef.current = utter;
      synth.speak(utter);
      setIsSpeaking(true);

      // Chrome bug workaround: speech pauses after ~15s without resume
      const isChrome = /Chrome/.test(navigator.userAgent) && !/Edg/.test(navigator.userAgent);
      if (isChrome) {
        resumeTimerRef.current = setInterval(() => {
          if (synth.speaking && !synth.paused) {
            synth.pause();
            synth.resume();
          }
        }, 10000);
      }
    } catch {
      setIsSpeaking(false);
    }
  };

  const handleResend = () => {
    if (onResend) {
      setIsResending(true);
      onResend();
      const timerID = setTimeout(() => {
        setIsResending(false);
        clearTimeout(timerID);
      }, 1000);
    }
  };

  return (
    <div className={clsx("flex items-center gap-4px mt-16px", actionClassName)}>
      {!isRefreshOnly && content && (
        <button
          id={copyButtonId}
          onClick={handleCopy}
          className="bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors"
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
      {question && (
        <button
          onClick={handleResend}
          className={`bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors ${
            isResending ? "animate-spin" : ""
          }`}
        >
          <RotateCcw
            className={`w-4 h-4 ${
              isResending
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
      {!isRefreshOnly && (
        <button
          onClick={handleLike}
          className={`bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors ${
            liked ? "animate-shake" : ""
          }`}
        >
          <ThumbsUp
            className={`w-4 h-4 ${
              liked
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
      {!isRefreshOnly && (
        <button
          onClick={handleDislike}
          className={`bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors ${
            disliked ? "animate-shake" : ""
          }`}
        >
          <ThumbsDown
            className={`w-4 h-4 ${
              disliked
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
      {!isRefreshOnly && content && (
        <>
          <button
            onClick={handleSpeak}
            className={`bg-transparent border-0 cursor-pointer p-4px hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors ${
              isSpeaking ? "text-[#1990FF]" : ""
            }`}
          >
            {isSpeaking
              ? [<Volume className="w-4 h-4" style={{ width: actionIconSize, height: actionIconSize }} />, <Volume1 className="w-4 h-4" style={{ width: actionIconSize, height: actionIconSize }} />, <Volume2 className="w-4 h-4" style={{ width: actionIconSize, height: actionIconSize }} />][volumeFrame]
              : <Volume2
                  className={`w-4 h-4 text-[#666666] dark:text-[#A3A3A3]`}
                  style={{
                    width: actionIconSize,
                    height: actionIconSize,
                  }}
                />
            }
          </button>
        </>
      )}
    </div>
  );
};
