import type { RefObject } from "react";
import clsx from "clsx";
import { ArrowDownToLine } from "lucide-react";

import { FloatButton } from "antd";

interface ScrollToBottomProps {
  scrollRef: RefObject<HTMLDivElement>;
  isAtBottom: boolean;
}<ArrowDownToLine />

const ScrollToBottom = ({ scrollRef, isAtBottom }: ScrollToBottomProps) => {
  return (
    <FloatButton
      className={clsx(
        "!absolute !right-4 !bottom-8 !border-[#F0F0F0] !dark:border-[#303030]",
        { "!hidden": isAtBottom }
      )}
      icon={<ArrowDownToLine className="size-18px" />}
      onClick={() => {
        scrollRef.current?.scrollTo({
          top: scrollRef.current?.scrollHeight,
          behavior: "smooth",
        });
      }}
    />
  );
};

export default ScrollToBottom;
