import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Copy, Check } from "lucide-react";

interface ExpandJsonProps {
    content: string;
    maxHeight?: number;
}

export const ExpandJson = ({ content, maxHeight = 100 }: ExpandJsonProps) => {
    const [expanded, setExpanded] = useState(false);
    const [copied, setCopied] = useState(false);
    const { t } = useTranslation();

    const handleCopy = () => {
        navigator.clipboard.writeText(content).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        });
    };

    return (
        <div>
            <pre
                className="m-0 text-12px leading-20px text-[#333] dark:text-[#E5E7EB] whitespace-pre-wrap break-all overflow-hidden"
                style={{ maxHeight: expanded ? "none" : `${maxHeight}px` }}
            >
                {content}
            </pre>
            {!expanded && <div className="pl-16px text-[#333] dark:text-[#E5E7EB] text-12px">...</div>}
            <div className="flex items-center gap-8px">
                <button
                    onClick={() => setExpanded((prev) => !prev)}
                    type="button"
                    className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px"
                >
                    {expanded ? t("labels.collapse", "收起") : t("labels.expand", "展开")}
                </button>
                <button
                    onClick={handleCopy}
                    type="button"
                    className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px inline-flex items-center gap-2px"
                >
                    {copied ? <Check className="w-12px h-12px" /> : <Copy className="w-12px h-12px" />}
                </button>
            </div>
        </div>
    );
};
