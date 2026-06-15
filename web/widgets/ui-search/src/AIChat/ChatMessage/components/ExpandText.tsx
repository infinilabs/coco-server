import { Typography } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";

interface ExpandTextProps {
  children: React.ReactNode;
  rows?: number;
  className?: string;
}

export const ExpandText = ({ children, rows = 3, className = "!mb-0 leading-20px text-12px text-[#333] dark:text-[#E5E7EB]" }: ExpandTextProps) => {
  const [expanded, setExpanded] = useState(false);
  const [showButton, setShowButton] = useState(false);
  const { t } = useTranslation();

  return (
    <div>
      <Typography.Paragraph
        ellipsis={{
          rows,
          expandable: false,
          onEllipsis: (ellipsis) => setShowButton(ellipsis),
        }}
        className={className}
        style={expanded ? { display: "none" } : undefined}
      >
        {children}
      </Typography.Paragraph>
      {expanded && (
        <div className={className}>
          {children}
        </div>
      )}
      {showButton && (
        <button
          onClick={() => setExpanded((prev) => !prev)}
          type="button"
          className="border-0 bg-transparent p-0 text-[var(--ant-color-link)] hover:text-[var(--ant-color-link-hover)] no-underline outline-none cursor-pointer transition-all duration-[var(--ant-motion-duration-slow)] select-none text-12px"
        >
          {expanded ? t("labels.collapse") : t("labels.expand")}
        </button>
      )}
    </div>
  );
};
