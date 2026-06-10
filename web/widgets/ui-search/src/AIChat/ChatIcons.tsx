import React from "react";
import {
  Send,
  Square,
  Paperclip,
  //Mic
} from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import { useChatStore } from "./stores/chatStore";
import type { TFunction } from "i18next";
import EnterIcon from "../icons/EnterIcon";

export interface SendMessageParams {
  message?: string;
  attachments?: string[];
  search?: boolean;
  deep_thinking?: boolean;
  datasource?: string[],
  mcp?: boolean,
  mcp_servers?: string[],
}

interface ChatIconsProps {
  curChatEnd: boolean;
  inputValue: string;
  onSend: (params: SendMessageParams) => void;
  onCancel: () => void;
  disabledChange?: () => void;
  speechSupported?: boolean;
  listening?: boolean;
  onVoiceToggle?: () => void;
  /** Triggered when the user clicks the paperclip / attachment button. */
  onAttachClick?: () => void;
  /** When true, the attachment button is hidden (e.g. host did not wire it). */
  hideAttachment?: boolean;
  /** When true, the send button is enabled even when the input is empty (e.g. attachments present). */
  canSendWithoutText?: boolean;
  /** Number of currently attached/pending files. Shown as a small badge on the paperclip. */
  attachmentCount?: number;
  /** When true, sending is blocked (e.g. attachments still uploading). */
  disableSend?: boolean;
  t?: TFunction;
}

const ChatIcons: React.FC<ChatIconsProps> = ({
  curChatEnd,
  inputValue = "",
  onSend,
  onCancel,
  disabledChange,
  onAttachClick,
  hideAttachment = false,
  canSendWithoutText = false,
  attachmentCount = 0,
  disableSend = false,
  // speechSupported = false,
  // listening = false,
  // onVoiceToggle,
  t: tProp,
}) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const renderSendButton = () => {
    const canSend = !!inputValue?.trim() || (canSendWithoutText && !disableSend);
    if (curChatEnd) {
      return (
        <div className="flex items-center gap-1">
          {!hideAttachment && (
            <button
              type="button"
              className={clsx(
                "cursor-pointer bg-transparent border-0 relative flex items-center justify-center rounded-full shrink-0 transition-colors",
                onAttachClick
                  ? "text-[#999] hover:text-[#666] dark:text-[#666] dark:hover:text-[#999] cursor-pointer"
                  : "text-[#ccc] dark:text-[#444] cursor-not-allowed"
              )}
              style={{ width: "32px", height: "32px" }}
              title={t("search.input.attachment") || "Attachment"}
              onClick={() => onAttachClick?.()}
              disabled={!onAttachClick}
            >
              <Paperclip size={16} />
              {attachmentCount > 0 && (
                <span
                  className="absolute -top-0.5 -right-0.5 min-w-[14px] h-[14px] px-[3px] rounded-full bg-[#027FFE] text-white text-[10px] leading-[14px] text-center"
                  style={{ pointerEvents: "none" }}
                >
                  {attachmentCount}
                </span>
              )}
            </button>
          )}
          {/* {speechSupported && (
            <button
              className={clsx(
                "flex items-center justify-center rounded-full shrink-0 transition-colors",
                {
                  "text-[#027FFE] bg-[#027FFE]/10": listening,
                  "text-[#999] hover:text-[#666] dark:text-[#666] dark:hover:text-[#999]":
                    !listening,
                }
              )}
              style={{ width: "32px", height: "32px" }}
              title={
                listening
                  ? t("search.input.stop_voice") || "Stop Voice"
                  : t("search.input.voice") || "Voice"
              }
              onClick={onVoiceToggle}
            >
              <Mic size={20} className={clsx({ "animate-pulse": listening })} />
            </button>
          )} */}
          <button
            className={clsx(
              "border-0 flex items-center justify-center rounded-full transition-colors p-0 shrink-0 aspect-square cursor-pointer",
              {
                "bg-[#027FFE] text-white": canSend,
                "bg-[#0000000a] dark:bg-[#ffffff0a] text-[#ccc] dark:text-[#666] cursor-not-allowed":
                  !canSend,
              }
            )}
            style={{ width: "24px", height: "24px" }}
            type="submit"
            onClick={() => {
              onSend({
                message: inputValue?.trim() || "",
              });
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSend({
                  message: inputValue?.trim() || "",
                });
              }
            }}
            disabled={!canSend}
            title={`${t("search.input.send") || "Send"} (Enter)`}
          >
            <EnterIcon size={14} />
          </button>
        </div>
      );
    }

    return (
      <button
        className="border-0 flex items-center justify-center rounded-full shrink-0 cursor-pointer bg-[#0072FF] transition-colors"
        style={{ width: "24px", height: "24px" }}
        onClick={() => {
          onCancel();
          disabledChange?.();
        }}
        title={t("search.input.stop") || "Stop"}
      >
        <Square
          size={10}
          strokeWidth={2}
          className="text-white fill-white"
          aria-label={t("search.input.stop") || "Stop"}
        />
      </button>
    );
  };

  return <>{renderSendButton()}</>;
};

export default ChatIcons;
