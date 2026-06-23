import type { RefObject } from "react";
import clsx from "clsx";
import { ArrowDownToLine } from "lucide-react";
import { FloatButton } from "antd";

interface ScrollToBottomBtnProps {
  scrollRef: RefObject<HTMLDivElement | null>;
  isAtBottom: boolean;
  className?: string;
  onScrollToBottom?: () => void;
}

export const ScrollToBottomBtn = ({
  scrollRef,
  isAtBottom,
  className,
  onScrollToBottom,
}: ScrollToBottomBtnProps) => {
  return (
    <FloatButton
      className={clsx(
        "!absolute !right-4 !bottom-8 !border-[#F0F0F0] !dark:border-[#303030]",
        { "!hidden": isAtBottom },
        className
      )}
      icon={<ArrowDownToLine className="size-18px" />}
      onClick={() => {
        scrollRef.current?.scrollTo({
          top: scrollRef.current?.scrollHeight,
          behavior: "auto",
        });
        onScrollToBottom?.();
      }}
    />
  );
};
