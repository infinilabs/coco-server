import { useEffect, useRef, useState } from "react";
import { Sparkles } from "lucide-react";
import clsx from "clsx";
import { X } from "lucide-react";

import { ChatMessage } from "../ChatMessage";

const AIOverview = (props) => {
  const { config = {}, data, loading, visible, setVisible } = props;

  const divRef = useRef(null);
  const [shouldScroll, setShouldScroll] = useState(true);

  useEffect(() => {
    if (shouldScroll && divRef.current) {
      divRef.current.scrollTop = divRef.current.scrollHeight;
    }
  }, [data, shouldScroll]);

  const handleScroll = () => {
    const div = divRef.current;
    if (div) {
      const isNearBottom =
        div.scrollHeight - div.scrollTop <= div.clientHeight + 100;

      setShouldScroll(isNearBottom);
    }
  };

  if (!data || !data.response || !visible) return null;

  return (
    <div
      className={`flex flex-col gap-2 relative rounded-3 text-[#333]  dark:text-[#D8D8D8] bg-white dark:bg-[#141414] border border-[var(--ui-search-antd-color-border-secondary)]`}
      style={{
        maxHeight: config.height ? config.height : "auto",
      }}
    >
      {/* <div
        className="absolute top-2 right-2 flex items-center justify-center size-[20px] border rounded-md cursor-pointer dark:border-[#282828]"
        onClick={() => {
          setVisible(false);
        }}
      >
        <X className="size-4" />
      </div> */}
      {config.title && (
        <div className="flex item-center gap-1 pt-6 px-6">
          {config.logo?.light ? (
            <img src={config.logo.light} className="size-4" />
          ) : (
            <Sparkles className="size-4 text-[#881c94]" />
          )}
          {config.title && (
            <span className="text-xs font-semibold">{config.title}</span>
          )}
        </div>
      )}

      <div
        ref={divRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto text-sm px-6 pb-4 mb-2"
      >
        <ChatMessage
          key="current"
          message={{
            _id: "current",
            _source: {
              type: "assistant",
              message: "",
              question: "",
            },
          }}
          {...data}
          isTyping={loading}
          rootClassName="!py-0"
          actionClassName="absolute bottom-4 left-6 !m-0"
          actionIconSize={12}
          showActions={config.showActions}
          output={config.output}
        />
      </div>

      <div
        className={clsx("min-h-[24px]", {
          hidden: loading || config.showActions === false,
        })}
      />
    </div>
  );
};

export default AIOverview;
