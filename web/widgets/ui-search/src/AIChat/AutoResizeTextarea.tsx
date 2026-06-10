import { useBoolean } from "ahooks";
import {
  useImperativeHandle,
  forwardRef,
  type KeyboardEvent,
  useCallback,
  type ChangeEvent,
  useRef,
  useEffect,
} from "react";
import { useTranslation } from "react-i18next";

import { clsx } from "clsx"
import { TFunction } from "i18next";

const MAX_HEIGHT = 240;

interface AutoResizeTextareaProps {
  input: string;
  setInput: (value: string) => void;
  handleKeyDown?: (e: KeyboardEvent<HTMLTextAreaElement>) => void;
  chatPlaceholder?: string;
  lineCount?: number;
  onLineCountChange?: (lineCount: number) => void;
  firstLineMaxWidth: number;
  disabled?: boolean;
  t?: TFunction;
}

// Forward ref to allow parent to interact with this component
const AutoResizeTextarea = forwardRef<
  { reset: () => void; focus: () => void },
  AutoResizeTextareaProps
>(
  (
    {
      input = "",
      setInput,
      handleKeyDown,
      chatPlaceholder,
      lineCount,
      onLineCountChange,
      firstLineMaxWidth,
      disabled,
      t: tProp,
    },
    ref
  ) => {
    const { t: tOriginal } = useTranslation();
    const t = tProp || tOriginal;
    const [isComposition, { setTrue, setFalse }] = useBoolean();
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    // Expose methods to the parent via ref
    useImperativeHandle(ref, () => ({
      reset: () => {
        setInput("");
      },
      focus: () => {
        textareaRef.current?.focus();
      },
    }));

    const handleKeyPress = (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (isComposition) {
        return event.stopPropagation();
      }

      handleKeyDown?.(event);
    };

    useEffect(() => {
      const textarea = textareaRef.current;

      if (!textarea) return;

      textarea.style.height = "auto";
      textarea.style.minHeight = "auto";

      const computedStyle = getComputedStyle(textarea);
      const lineHeight = parseInt(computedStyle.lineHeight);
      const scrollHeight = textarea.scrollHeight;

      let height = lineHeight;
      let minHeight = lineHeight;

      if (scrollHeight > lineHeight) {
        minHeight = lineHeight * 2;
        height = Math.min(Math.max(minHeight, scrollHeight), MAX_HEIGHT);
      }

      textarea.style.height = `${height}px`;
      textarea.style.minHeight = `${minHeight}px`;

      onLineCountChange?.(Math.round(height / lineHeight));
    }, [input, firstLineMaxWidth, onLineCountChange]);

    const handleChange = useCallback(
      (event: ChangeEvent<HTMLTextAreaElement>) => {
        setInput(event.currentTarget.value);
      },
      [setInput]
    );

    return (
      <>
        <textarea
          ref={textareaRef}
          id="chat-textarea"
          autoFocus
          autoComplete="off"
          autoCapitalize="none"
          spellCheck="false"
          className={clsx(
            "break-all border-0 auto-resize-textarea text-base flex-1 outline-none w-full min-w-[200px] bg-transparent custom-scrollbar resize-none overflow-y-auto",
            {
              "overflow-y-hidden": lineCount === 1,
            }
          )}
          style={{
            resize: "none",
            color: 'var(--ant-color-text)',
          }}
          placeholder={chatPlaceholder || t("search.textarea.placeholder")}
          aria-label={t("search.textarea.ariaLabel")}
          value={input}
          onChange={handleChange}
          onKeyDown={handleKeyPress}
          onCompositionStart={setTrue}
          onCompositionEnd={() => {
            setTimeout(setFalse, 0);
          }}
          rows={1}
          disabled={disabled}
        />
      </>
    );
  }
);

export default AutoResizeTextarea;
