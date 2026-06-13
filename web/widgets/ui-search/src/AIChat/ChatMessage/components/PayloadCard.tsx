import { FileText, SquareArrowOutUpRight } from "lucide-react";

import { useDeepResearchDrawer } from "./DeepResearch/DeepResearchDrawerContext";
import { formatDate } from "../utils";

interface PayloadCardProps {
  payload: {
    title?: string;
    url?: string;
    created?: string | number;
    type?: string;
    content?: string;
    [key: string]: any;
  };
  formatUrl?: (data: any) => string;
}

export const PayloadCard = ({ payload, formatUrl }: PayloadCardProps) => {
  const { openDrawer } = useDeepResearchDrawer();

  if (!payload?.title || !payload?.url) return null;

  return (
    <>
      <div className="mt-16px">
        <div
          className="flex items-center gap-3 p-3 bg-gray-50 dark:bg-[#202126] rounded-lg border border-gray-200 dark:border-gray-800 max-w-sm cursor-pointer transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
          onClick={() => openDrawer({
            reportContent: payload.content,
            reportData: payload as any,
            formatUrl,
            showReportOnly: true,
          })}
        >
          <div className="shrink-0">
            <FileText className="w-6 h-6 text-blue-500" />
          </div>
          <div className="flex-1 min-w-0">
            <div
              className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
              title={payload.title}
            >
              {payload.title}
            </div>
            {payload.created && (
              <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                {formatDate(payload.created)}
              </div>
            )}
          </div>
          {payload.url && (
            <a
              href={(formatUrl ? formatUrl({ url: payload.url }) : payload.url)}
              target="_blank"
              rel="noopener noreferrer"
              className="shrink-0 p-2 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-md transition-colors cursor-pointer"
              title="Open Link"
              onClick={(e) => e.stopPropagation()}
            >
              <SquareArrowOutUpRight className="w-5 h-5" />
            </a>
          )}
        </div>
      </div>
    </>
  );
};
