import type { RefObject } from "react";
import clsx from "clsx";
import { ArrowDown } from "lucide-react";

import { Button, FloatButton } from "antd";

interface ScrollToBottomProps {
  scrollRef: RefObject<HTMLDivElement>;
  isAtBottom: boolean;
}

const ScrollToBottom = ({ scrollRef, isAtBottom }: ScrollToBottomProps) => {
  return (
    <FloatButton
      className={clsx(
        "!absolute !right-4 !bottom-8 !border-[#F0F0F0] !dark:border-[#303030]",
        { "!hidden": isAtBottom }
      )}
      icon={<ArrowDown className="size-18px" />}
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
