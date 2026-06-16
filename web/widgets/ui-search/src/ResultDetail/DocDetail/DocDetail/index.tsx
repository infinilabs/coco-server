import { Collapse, Typography } from "antd";
import { ChevronDown, ChevronRight, Dot, Minus } from "lucide-react";
import { motion } from "motion/react";

import { useMemo, useState, type FC, type HTMLAttributes, type ReactNode } from "react";
import Preview from "./components/Preview";
import AIInterpretation from "./components/AIInterpretation";
import { clsx } from "clsx";
import PreviewIcon from "../../../icons/PreviewIcon";
import AIInsightIcon from "../../../icons/AIInsightIcon";
import { AuthImage } from "../../../ResultList/AuthImage";

const { Text } = Typography;

export type MetadataContentType =
  | "image"
  | "video"
  | "markdown"
  | "pdf"
  | "docx"
  | "pptx"
  | "xlsx";

export interface DocDetailProps extends HTMLAttributes<HTMLDivElement> {
  data: {
    id?: string;
    created?: ReactNode;
    updated?: ReactNode;
    _system?: {
      owner_id?: string;
      parent_path?: string;
      tenant_id?: string;
    };
    metadata?: {
      colors?: string[];
      content_type?: MetadataContentType;
      height?: number;
      mime_type?: string;
      users?: null | unknown;
      width?: number;
      raw_content?: string;
    };
    source?: {
      type?: string;
      name?: string;
      id?: string;
    };
    type?: string;
    category?: string;
    title?: string;
    summary?: string;
    icon?: string;
    thumbnail?: string;
    cover?: string;
    tags?: string[];
    url?: string;
    size?: ReactNode;
    owner?: {
      type?: string;
      id?: string;
      icon?: string;
      title?: string;
      subtitle?: string;
      cover?: string;
    };
    ai_insights?: {
      text?: string;
    };
  };
  i18n?: {
    labels?: {
      type?: string;
      size?: string;
      createdBy?: string;
      createdAt?: string;
      updatedAt?: string;
      preview?: string;
      aiInterpretation?: string;
    };
  };
  requestHeaders?: Record<string, string>;
  actionButtons?: ReactNode[];
  mode?: "embedded" | "standalone";
  theme?: "light" | "dark" | "auto";
}

const DocDetail: FC<DocDetailProps> = (props) => {
  const { data, i18n, actionButtons, requestHeaders, className, mode, theme, ...rest } =
    props;

  const [expandMore, setExpandMore] = useState(false);

  const moreInfo = [
    {
      label: i18n?.labels?.type ?? "Type",
      value: data?.type,
    },
    {
      label: i18n?.labels?.size ?? "Size",
      value: data?.size,
    },
    {
      label: i18n?.labels?.createdBy ?? "Created By",
      value: data?.owner?.title,
    },
    {
      label: i18n?.labels?.createdAt ?? "Created At",
      value: data?.created,
    },
    {
      label: i18n?.labels?.updatedAt ?? "Updated At",
      value: data?.updated,
    },
  ];

  const contentType = data?.metadata?.content_type;
  const isInlinePreview = contentType === "image" || contentType === "video";
  const hasCollapsiblePreview =
    !!contentType && !!data?.metadata?.raw_content && !isInlinePreview;

  const collapseItems = useMemo(() => {
    const items: { key: string; label: ReactNode | string; children: ReactNode }[] = [];

    if (hasCollapsiblePreview) {
      items.push({
        key: "preview",
        label: (
          <div className="inline-flex items-center gap-8px">
            <PreviewIcon size={16} className="shrink-0 text-[--ant-color-primary]"/>
            <div className="text-#333 dark:text-#666 text-16px leading-22px">
              {i18n?.labels?.preview ?? "Preview"}
            </div>
          </div>
        ),
        children: <Preview {...props} />,
      });
    }

    if (data?.ai_insights?.text) {
      items.push({
        key: "ai-interpretation",
        label: (
          <div className="inline-flex items-center gap-8px">
            <AIInsightIcon size={16} className="shrink-0 text-[--ant-color-primary]"/>
            <div className="text-#333 dark:text-#666 text-16px leading-22px">
              {i18n?.labels?.aiInterpretation ?? "AI Interpretation"}
            </div>
          </div>
        ),
        children: <AIInterpretation {...props} />,
      });
    }

    return items;
  }, [hasCollapsiblePreview, data?.ai_insights?.text, i18n, props]);

  return (
    <div
      className={clsx("flex flex-col h-full overflow-hidden", className)}
      {...rest}
    >
      <div className={clsx("text-20px text-[#1A0CAB] dark:text-[#8AB4F8] break-words", mode === "embedded" ? "pr-24px" : "")}>
        <AuthImage
          src={data?.icon}
          className={"w-20px h-20px inline-block align-middle mr-8px"}
          requestHeaders={requestHeaders}
        />
        <span className="align-middle">{data?.title}</span>
      </div>

      <div className="flex items-center justify-between my-2">
        <Text
          className="inline-flex items-center gap-0.5 text-3 text-[#666] dark:text-white/80"
        >
          <div>{data?.source?.name ?? "-"}</div>
          {
            data?.category && (
              <>
                <ChevronRight className="size-3" />
                <div>{data?.category ?? "-"}</div>
              </>
            )
          }
          {
            data?.owner?.title && (
              <>
                <Minus className="size-3 rotate-90" />
                <div>{data?.owner?.title ?? "-"}</div>
              </>
            )
          }
          {
            data?.updated && (
              <>
                <Dot className="size-3" />
                <div>{data?.updated ?? "-"}</div>
              </>
            )
          }

          <ChevronDown
            className={clsx(
              "ml-2 size-3 hover:text-[--ant-color-primary] transition cursor-pointer",
              {
                "-scale-y-100": expandMore,
              }
            )}
            onClick={() => {
              setExpandMore((prev) => !prev);
            }}
          />
        </Text>

        <div className="inline-flex gap-2">{actionButtons}</div>
      </div>

      <motion.div
        className="bg-black/3 dark:bg-white/4 rounded-lg overflow-hidden"
        initial={false}
        animate={{
          height: expandMore ? "auto" : 0,
          opacity: expandMore ? 1 : 0,
          marginBottom: expandMore ? "1rem" : 0,
        }}
      >
        <div className="flex flex-wrap gap-row-2 p-4">
          {moreInfo.map((item) => {
            const { label, value } = item;

            return (
              <div
                key={label}
                className="w-1/2 inline-flex items-center <sm:w-full"
              >
                <Text type="secondary" className="w-24">
                  {label}
                </Text>

                <span className="text-3.5">{value ?? "-"}</span>
              </div>
            );
          })}
        </div>
      </motion.div>

      <div className="flex flex-col gap-4 flex-1 overflow-auto">
        {isInlinePreview && <Preview {...props} />}
        {collapseItems.length > 0 && (
          <Collapse
            size="small"
            defaultActiveKey={[collapseItems[0]?.key]}
            classNames={{
              root: "bg-transparent border-[#F0F0F0] dark:border-[#303030] [&_.ant-collapse-panel]:border-[#F0F0F0]! dark:[&_.ant-collapse-panel]:border-[#303030]!",
              body: "p-24px!",
              header: "px-16px! py-9px!",
              title: "flex items-center",
              icon: "text-[#999]! dark:text-[#666]! [&_.ant-collapse-arrow]:text-16px!"
            }}
            items={collapseItems}
            expandIconPlacement="end"
          />
        )}
      </div>
    </div>
  );
};

export default DocDetail;
