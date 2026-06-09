import { useEffect, useRef, useState, useCallback, type CSSProperties } from "react";
import { XMarkdown } from "@ant-design/x-markdown";
import AIOverviewIcon from "../../icons/AIOverviewIcon";
import clsx from "clsx";
import "./index.css";

import { AIContinueButton } from "./AIContinueButton";
import { AIExpandToggle } from "./AIExpandToggle";
import { AIAnswerActions } from "./AIAnswerActions";

export type AIAnswerProps = {
  title?: string;
  content: string;
  maxHeight?: number;
  expandText?: string;
  collapseText?: string;
  continueLabel?: string;
  theme?: "light" | "dark" | "auto";
  onContinue?: () => void;
  containerClass?: string;
  headerClass?: string;
  titleClass?: string;
  containerStyle?: CSSProperties;
  headerStyle?: CSSProperties;
  titleStyle?: CSSProperties;
  loading?: boolean;
};

export function AIAnswer({
  title = "智能解读",
  content,
  maxHeight = 180,
  expandText = "展开更多",
  collapseText = "收起",
  continueLabel = "继续追问",
  theme = "auto",
  onContinue,
  containerClass,
  headerClass,
  titleClass,
  containerStyle,
  headerStyle,
  titleStyle,
  loading = false,
}: AIAnswerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);
  const [overflow, setOverflow] = useState(false);
  const [bodyMaxHeight, setBodyMaxHeight] = useState<number | undefined>(undefined);
  const [collapsedState, setCollapsedState] = useState(true);
  const [userInteracted, setUserInteracted] = useState(false);
  const collapsed = userInteracted ? collapsedState : overflow;
  const showToggle = overflow;

  const getFixedHeight = useCallback(() => {
    const container = containerRef.current;
    const body = bodyRef.current;
    if (!container || !body) return 0;
    const cs = getComputedStyle(container);
    let fixed = parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom);
    for (const child of Array.from(container.children)) {
      if (child === body) continue;
      const el = child as HTMLElement;
      const childCs = getComputedStyle(el);
      fixed += el.offsetHeight + parseFloat(childCs.marginTop) + parseFloat(childCs.marginBottom);
    }
    return fixed;
  }, []);

  useEffect(() => {
    const body = bodyRef.current;
    if (!body) return;

    const TOGGLE_HEIGHT = 32;

    const measure = () => {
      const bodyNatural = body.scrollHeight;
      if (bodyNatural === 0) {
        setBodyMaxHeight(undefined);
        setOverflow(false);
        return;
      }
      const fixedHeight = getFixedHeight();
      const totalFixed = fixedHeight + TOGGLE_HEIGHT;
      const budget = maxHeight - totalFixed;
      const needsCollapse = bodyNatural + totalFixed > maxHeight;

      if (needsCollapse) {
        setBodyMaxHeight(Math.max(60, budget));
        setOverflow(true);
      } else {
        setBodyMaxHeight(undefined);
        setOverflow(false);
      }
    };

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(body);
    return () => observer.disconnect();
  }, [content, maxHeight, getFixedHeight]);

  return (
    <div className={theme === "dark" ? "dark" : undefined}>
      <div
        ref={containerRef}
        className={clsx(
          "p-6 rounded-xl border border-[#EBEBEB] bg-transparent text-[#333] dark:text-#666 dark:border-slate-700 dark:bg-transparent",
          containerClass
        )}
        style={containerStyle}
      >
        <div
          className={clsx("mb-4 flex items-center gap-2", headerClass)}
          style={headerStyle}
        >
          <AIOverviewIcon
            className="flex-none text-24px text-[#1784FC]"
          />
          <span
            className={clsx(
              "font-semibold text-base text-[#19191A] dark:text-white",
              titleClass
            )}
            style={titleStyle}
          >
            {title}
          </span>
        </div>
        <div
          ref={bodyRef}
          className="relative overflow-hidden"
          style={{
            maxHeight: collapsed && showToggle && bodyMaxHeight != null ? bodyMaxHeight : undefined,
            maskImage: collapsed && showToggle
              ? "linear-gradient(to bottom, black calc(100% - 40px), transparent 100%)"
              : undefined,
            WebkitMaskImage: collapsed && showToggle
              ? "linear-gradient(to bottom, black calc(100% - 40px), transparent 100%)"
              : undefined,
          }}
        >
          {content && !loading ? (
            <XMarkdown content={content}/>
          ) : (
            <span
              className="animate-typing inline-block w-1.5 h-5 ml-0.5 -mb-0.5 bg-[#666666] dark:bg-[#A3A3A3] rounded-sm "
            />
          )}
        </div>
        {showToggle ? (
          <AIExpandToggle
            collapsed={collapsed}
            expandText={expandText}
            collapseText={collapseText}
            onToggle={() => {
              setUserInteracted(true);
              setCollapsedState(!collapsed);
            }}
          />
        ) : null}
        {content && !loading ? (
          <div className="mt-4 flex items-center justify-between">
            <AIAnswerActions
              copyText={content}
              onCopy={() => { }}
              onLike={() => { }}
              onDislike={() => { }}
              theme={theme}
            />
            <AIContinueButton label={continueLabel} onClick={onContinue} />
          </div>
        ) : null}
      </div>
    </div>
  );
}
