import { Typography } from "antd";
import { type CSSProperties, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

interface ExpandTextProps {
    readonly content: string;
    readonly rows?: number;
    readonly className?: string;
    readonly maxHeight?: number;
    readonly variant?: "json" | "text";
}

export const ExpandText = ({ content, rows = 3, className = "!mb-0 leading-20px text-12px text-[#333] dark:text-[#E5E7EB] whitespace-pre-wrap", maxHeight = 100, variant = "text" }: ExpandTextProps) => {
    const [expanded, setExpanded] = useState(false);
    const [showButton, setShowButton] = useState(false);
    const [isOverflow, setIsOverflow] = useState(false);
    const preRef = useRef<HTMLPreElement>(null);
    const { t } = useTranslation();
    const isJson = variant === "json";
    const textStyle: CSSProperties = {
        overflowWrap: "anywhere",
        wordBreak: "break-word",
    };
    const jsonStyle: CSSProperties = {
        overflowWrap: "anywhere",
        wordBreak: "break-all",
        maxHeight: expanded ? "none" : `${maxHeight}px`,
    };
    const canToggle = isJson ? isOverflow : showButton;

    useEffect(() => {
        if (!isJson) {
            setIsOverflow(false);
            return;
        }
        if (preRef.current) {
            setIsOverflow(preRef.current.scrollHeight > maxHeight);
        }
    }, [content, isJson, maxHeight]);

    if (isJson) {
        return (
            <div className="min-w-0 max-w-full">
                <pre
                    ref={preRef}
                    className="m-0 min-w-0 max-w-full text-12px leading-20px text-[#333] dark:text-[#E5E7EB] whitespace-pre-wrap break-all overflow-hidden"
                    style={jsonStyle}
                >
                    {content}
                </pre>
                {!expanded && isOverflow && <div className="pl-16px text-[#333] dark:text-[#E5E7EB] text-12px">...</div>}
                {canToggle && (
                    <div className="flex items-center gap-8px mt-2px">
                        <button
                            onClick={() => setExpanded((prev) => !prev)}
                            type="button"
                            className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px"
                        >
                            {expanded ? t("labels.collapse") : t("labels.expand")}
                        </button>
                    </div>
                )}
            </div>
        );
    }

    return (
        <div className="min-w-0 max-w-full">
            <Typography.Paragraph
                ellipsis={{
                    rows,
                    expandable: false,
                    onEllipsis: (ellipsis) => setShowButton(ellipsis),
                }}
                className={className}
                style={expanded ? { ...textStyle, display: "none" } : textStyle}
            >
                {content}
            </Typography.Paragraph>
            {expanded && (
                <div className={className} style={textStyle}>
                    {content}
                </div>
            )}
            <div className="flex items-center gap-8px mt-2px">
                {canToggle && (
                    <button
                        onClick={() => setExpanded((prev) => !prev)}
                        type="button"
                        className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px"
                    >
                        {expanded ? t("labels.collapse") : t("labels.expand")}
                    </button>
                )}
            </div>
        </div>
    );
};
