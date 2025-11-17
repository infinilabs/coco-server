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
      className={`flex flex-col gap-2 relative rounded-3 text-[#333]  dark:text-[#D8D8D8] bg-white dark:bg-[#141414] shadow-[0_4px_8px_rgba(0,0,0,0.1)] dark:shadow-[0_4px_20px_rgba(255,255,255,0.2)]`}
      style={{
        maxHeight: config.height ? config.height : "auto",
      }}
    >
      <div
        className="absolute top-2 right-2 flex items-center justify-center size-[20px] border rounded-md cursor-pointer dark:border-[#282828]"
        onClick={() => {
          setVisible(false);
        }}
      >
        <X className="size-4" />
      </div>
      {config.title && (
        <div className="flex item-center gap-1 pt-4 px-4">
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
        className="flex-1 overflow-auto text-sm px-4 pb-4"
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
          actionClassName="absolute bottom-2 left-3 !m-0"
          actionIconSize={12}
          showActions={config.showActions}
          output={config.output}
        />
      </div>

      <div
        className={clsx("min-h-[20px]", {
          hidden: loading || config.showActions === false,
        })}
      />
    </div>
  );
};

export default AIOverview;
