import { useState, useRef, useEffect } from "react";
import { useTranslation } from "react-i18next";

interface ExpandJsonProps {
    content: string;
    maxHeight?: number;
}

export const ExpandJson = ({ content, maxHeight = 100 }: ExpandJsonProps) => {
    const [expanded, setExpanded] = useState(false);
    const [isOverflow, setIsOverflow] = useState(false);
    const preRef = useRef<HTMLPreElement>(null);
    const { t } = useTranslation();

    useEffect(() => {
        if (preRef.current) {
            setIsOverflow(preRef.current.scrollHeight > maxHeight);
        }
    }, [content, maxHeight]);

    return (
        <div>
            <pre
                ref={preRef}
                className="m-0 text-12px leading-20px text-[#333] dark:text-[#E5E7EB] whitespace-pre-wrap break-all overflow-hidden"
                style={{ maxHeight: expanded ? "none" : `${maxHeight}px` }}
            >
                {content}
            </pre>
            {!expanded && isOverflow && <div className="pl-16px text-[#333] dark:text-[#E5E7EB] text-12px">...</div>}
            {isOverflow && (
                <div className="flex items-center gap-8px mt-2px">
                    <button
                        onClick={() => setExpanded((prev) => !prev)}
                        type="button"
                        className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px"
                    >
                        {expanded ? t("labels.collapse", "收起") : t("labels.expand", "展开")}
                    </button>
                </div>
            )}
        </div>
    );
};
