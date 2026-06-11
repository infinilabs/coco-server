import { FileText, SquareArrowOutUpRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from "@infinilabs/markdown";

import { formatDate } from "../../utils";

export interface ResearchReportData {
  title?: string;
  url?: string;
  created?: string;
  attachment?: string;
  format?: string;
}

export interface ResearchReportContentProps {
  content?: string;
  data?: ResearchReportData;
  formatUrl?: (data: any) => string;
  t?: TFunction;
}

export const ResearchReportContent = ({
  content,
  data,
  formatUrl,
  t: tProp,
}: ResearchReportContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  if (!content && !data) {
    return (
      <div className="px-6 max-w-[730px] h-full flex flex-col items-center justify-center text-center">
        <div className="mb-2 text-base font-medium text-[#333333] dark:text-[#E5E7EB]">
          {t("deepResearch.report.generatingTitle")}
        </div>
        <div className="text-sm text-[#999999] dark:text-[#A6A6A6] max-w-[520px] leading-relaxed">
          {t("deepResearch.report.generatingDescription")}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-[730px]">
      {/* {data && (
        <div className="mb-6 p-4 border border-gray-200 dark:border-gray-800 rounded-xl bg-gray-50 dark:bg-gray-900 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center text-blue-600 dark:text-blue-400">
              <FileText className="w-6 h-6" />
            </div>
            <div>
              <div className="font-medium text-gray-900 dark:text-gray-100">
                {data.title || t("deepResearch.report.defaultTitle")}
              </div>
              <div className="text-xs text-gray-500">
                {data.created ? formatDate(data.created) : ""}
              </div>
            </div>
          </div>
          {data.url && (
            <a
              href={formatUrl ? formatUrl({ url: data.url }) : data.url}
              target="_blank"
              rel="noopener noreferrer"
              className="p-2 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors cursor-pointer"
              title={t("deepResearch.button.view")}
            >
              <SquareArrowOutUpRight className="w-5 h-5" />
            </a>
          )}
        </div>
      )} */}
      {content && (
        data?.format === "html" ? (
          <iframe
            srcDoc={content}
            className="w-full border-0 rounded-lg"
            style={{ minHeight: 600 }}
            sandbox="allow-same-origin"
            title="research-report"
          />
        ) : (
          <Markdown content={content} />
        )
      )}

      {!content && data?.url && (
        data?.format === "html" ? (
          <iframe
            src={formatUrl ? formatUrl({ url: data.url }) : data.url}
            className="w-full border-0 rounded-lg"
            style={{ minHeight: 600 }}
            sandbox="allow-same-origin allow-scripts"
            title="research-report"
          />
        ) : (
          <Markdown url={formatUrl ? formatUrl({ url: data.url }) : data.url} />
        )
      )}
    </div>
  );
};
