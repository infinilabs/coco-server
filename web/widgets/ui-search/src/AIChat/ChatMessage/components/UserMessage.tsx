import { useState, useMemo, useRef } from "react";
import type { FC } from "react";
import clsx from "clsx";
import { useAsyncEffect } from "ahooks";
import { Attachments } from "@infinilabs/attachments";

import { CopyButton } from "./Common/CopyButton";

export type AttachmentHit = {
  _id: string;
  _source: Record<string, unknown>;
};

interface UserMessageProps {
  message: string;
  attachments: string[];
  /** Called to fetch attachment metadata by IDs. When omitted, attachments are not rendered. */
  fetchAttachments?: (ids: string[]) => Promise<AttachmentHit[]>;
}

export const UserMessage: FC<UserMessageProps> = (props) => {
  const { message, attachments, fetchAttachments } = props;

  const [showCopyButton, setShowCopyButton] = useState(false);
  const [attachmentData, setAttachmentData] = useState<AttachmentHit[]>([]);

  const handleDoubleClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (typeof window !== "undefined" && typeof document !== "undefined") {
      const selection = window.getSelection();
      const range = document.createRange();

      if (e.currentTarget && selection && range) {
        try {
          range.selectNodeContents(e.currentTarget);
          selection.removeAllRanges();
          selection.addRange(range);
        } catch (error) {
          console.error("Selection failed:", error);
        }
      }
    }
  };

  // Use a content-based key so the effect doesn't re-run when only the array reference changes
  const attachmentsKey = useMemo(() => attachments.join(","), [attachments]);
  const fetchedKeyRef = useRef<string>("");

  useAsyncEffect(async () => {
    try {
      if (attachments.length === 0 || !fetchAttachments) return;
      if (fetchedKeyRef.current === attachmentsKey) return;
      fetchedKeyRef.current = attachmentsKey;
      const hits = await fetchAttachments(attachments);
      setAttachmentData(hits ?? []);
    } catch (error) {
      console.error("Get attachment failed:", String(error));
    }
  }, [attachmentsKey, fetchAttachments]);

  return (
    <>
      {message && (
        <div
          className="flex gap-2 items-center justify-end"
          onMouseEnter={() => setShowCopyButton(true)}
          onMouseLeave={() => setShowCopyButton(false)}
        >
          <div
            className={clsx("size-6 transition", {
              "opacity-0": !showCopyButton,
            })}
          >
            <CopyButton textToCopy={message} />
          </div>
          <div
            className="max-w-[85%] text-left px-3 py-2 bg-[#EEF0F3] dark:bg-[#1E2229]! rounded-8px border border-[#F0F0F0] dark:border-[#303030] font-normal text-14px text-[#333333] dark:text-white cursor-pointer user-select-text whitespace-pre-wrap break-all"
            onDoubleClick={handleDoubleClick}
          >
            {message}
          </div>
        </div>
      )}

      {attachmentData && attachmentData.length > 0 && (
        <div
          className={clsx("w-full", {
            "mt-3": message,
          })}
        >
          <Attachments
            className="justify-end"
            data={attachmentData.map((item) => {
              const bytes = item._source.size as number | undefined;
              let sizeStr: string | undefined;
              if (bytes != null) {
                if (bytes >= 1024 * 1024) {
                  sizeStr = (bytes / (1024 * 1024)).toFixed(1) + " MB";
                } else if (bytes >= 1024) {
                  sizeStr = (bytes / 1024).toFixed(1) + " KB";
                } else {
                  sizeStr = bytes + " B";
                }
              }
              return {
                id: item._source.id as string,
                filename: item._source.name as string,
                extname: item._source.icon as string,
                size: sizeStr,
                status: "uploaded" as const,
              };
            })}
          />
        </div>
      )}
    </>
  );
};
