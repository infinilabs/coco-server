import { useEffect, useRef, useState } from "react";
import { Copy, Check, ThumbsUp, ThumbsDown, Volume, Volume1, Volume2 } from "lucide-react";

export type AIAnswerActionsProps = {
  copyText?: string;
  onCopy?: (text: string) => void;
  onLike?: (liked: boolean) => void;
  onDislike?: (disliked: boolean) => void;
  onSpeak?: () => void;
  theme?: "light" | "dark" | "auto";
};

export function AIAnswerActions({
  copyText = "",
  onCopy,
  onLike,
  onDislike,
  onSpeak,
  theme = "auto"
}: AIAnswerActionsProps) {
  const [liked, setLiked] = useState(false);
  const [disliked, setDisliked] = useState(false);
  const [likePulse, setLikePulse] = useState(false);
  const [dislikePulse, setDislikePulse] = useState(false);
  const [copied, setCopied] = useState(false);
  const [speaking, setSpeaking] = useState(false);
  const [volumeFrame, setVolumeFrame] = useState(0);
  const utteranceRef = useRef<SpeechSynthesisUtterance | null>(null);
  const resumeTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const volumeAnimRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const stoppedByUserRef = useRef(false);

  useEffect(() => {
    if (speaking) {
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
  }, [speaking]);

  const baseBtnClass =
    "bg-transparent border-0 inline-flex items-center justify-center p-1 hover:bg-black/5 dark:hover:bg-white/5 rounded-lg transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-300 dark:focus-visible:ring-slate-600";
  const defaultBtnClass =
    "text-[#666] hover:bg-slate-50 dark:text-white/80 dark:hover:bg-slate-800";

  const activeBtnClass = "text-[#1677ff]";

  useEffect(() => {
    if (likePulse) {
      const t = setTimeout(() => setLikePulse(false), 220);
      return () => clearTimeout(t);
    }
  }, [likePulse]);

  useEffect(() => {
    if (dislikePulse) {
      const t = setTimeout(() => setDislikePulse(false), 220);
      return () => clearTimeout(t);
    }
  }, [dislikePulse]);
  useEffect(() => {
    if (copied) {
      const t = setTimeout(() => setCopied(false), 1200);
      return () => clearTimeout(t);
    }
  }, [copied]);

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

  return (
    <div className={theme === "dark" ? "dark" : undefined}>
      <div className="flex">
        <button
          type="button"
          className={`${baseBtnClass} ${defaultBtnClass}`}
          onClick={async () => {
            try {
              if (copyText) {
                if (navigator.clipboard && window.isSecureContext) {
                  await navigator.clipboard.writeText(copyText);
                } else {
                  const textarea = document.createElement("textarea");
                  textarea.value = copyText;
                  textarea.style.position = "fixed";
                  textarea.style.opacity = "0";
                  document.body.appendChild(textarea);
                  textarea.select();
                  document.execCommand("copy");
                  document.body.removeChild(textarea);
                }
              }
              onCopy?.(copyText);
            } catch {
              onCopy?.(copyText);
            }
            setCopied(true);
          }}
        >
          {copied ? <Check className="h-4 w-4 text-[#1677ff]" /> : <Copy className="h-4 w-4" />}
        </button>

        <button
          type="button"
          className={`${baseBtnClass} ${liked ? activeBtnClass : defaultBtnClass
            } transition-transform ${likePulse ? "scale-110" : "scale-100"
            }`}
          onClick={() => {
            const next = !liked;
            setLiked(next);
            setDisliked(false);
            setLikePulse(true);
            onLike?.(next);
          }}
        >
          <ThumbsUp className="h-4 w-4" />
        </button>

        <button
          type="button"
          className={`${baseBtnClass} ${disliked ? activeBtnClass : defaultBtnClass
            } transition-transform ${dislikePulse ? "scale-110" : "scale-100"
            }`}
          onClick={() => {
            const next = !disliked;
            setDisliked(next);
            setLiked(false);
            setDislikePulse(true);
            onDislike?.(next);
          }}
        >
          <ThumbsDown className="h-4 w-4" />
        </button>

        <button
          type="button"
          className={`${baseBtnClass} ${speaking ? activeBtnClass : defaultBtnClass}`}
          onClick={() => {
            if (onSpeak) {
              onSpeak();
              return;
            }
            const synth = typeof window !== "undefined" ? window.speechSynthesis : undefined;
            if (!synth) return;

            if (speaking) {
              stoppedByUserRef.current = true;
              synth.cancel();
              setSpeaking(false);
              utteranceRef.current = null;
              if (resumeTimerRef.current) {
                clearInterval(resumeTimerRef.current);
                resumeTimerRef.current = null;
              }
              return;
            }

            try {
              const stopSpeaking = () => {
                setSpeaking(false);
                utteranceRef.current = null;
                if (resumeTimerRef.current) {
                  clearInterval(resumeTimerRef.current);
                  resumeTimerRef.current = null;
                }
              };

              stoppedByUserRef.current = false;
              synth.cancel();
              const utter = new SpeechSynthesisUtterance(copyText || "");
              utter.onend = stopSpeaking;
              utter.onerror = () => {
                // Only reset state if not already handled by user stop
                if (!stoppedByUserRef.current) {
                  stopSpeaking();
                }
              };
              utteranceRef.current = utter;
              synth.speak(utter);
              setSpeaking(true);

              // Chrome bug workaround: speech pauses after ~15s without resume
              // Only apply on Chrome; Safari doesn't have this bug and pause/resume can disrupt playback
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
              setSpeaking(false);
            }
          }}
        >
          {speaking
            ? [<Volume className="h-4 w-4" />, <Volume1 className="h-4 w-4" />, <Volume2 className="h-4 w-4" />][volumeFrame]
            : <Volume2 className="h-4 w-4" />}
        </button>
      </div>
    </div>
  );
}
