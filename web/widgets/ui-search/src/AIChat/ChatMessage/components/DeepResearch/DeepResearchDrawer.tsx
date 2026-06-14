import { Button, Segmented } from "antd";
import { Download, SquareArrowOutUpRight, X } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import { ResearchStepsContent } from "./ResearchStepsContent";
import type { StepItem, StepStatus, StepSearchHit } from "./ResearchStepsContent";
import {
  ResearchReportContent,
  type ResearchReportData,
} from "./ResearchReportContent";
import { ResearchSearchResultsContent } from "./ResearchSearchResultsContent";
import { ActionButton } from "../../../../ResultDetail/DocDetail";
import CommonDrawer from "../../../../Layout/CommonDrawer";

const TAB_KEYS = {
  REPORT: "report",
  STEPS: "steps",
  SEARCH_RESULTS: "searchResults",
} as const;

type TabKey = (typeof TAB_KEYS)[keyof typeof TAB_KEYS];

interface DeepResearchDrawerProps {
  open: boolean;
  onClose: () => void;
  defaultActiveTab?: string;
  revision?: number;
  steps?: StepItem[];
  plannerStatus?: StepStatus;
  executionStatus?: StepStatus;
  reportStatus?: StepStatus;
  reportData?: ResearchReportData;
  reportContent?: string;
  searchHits?: StepSearchHit[];
  formatUrl?: (data: any) => string;
  requestHeaders?: Record<string, string>;
  theme?: "light" | "dark";
  isMobile?: boolean;
  showReportOnly?: boolean;
  t?: TFunction;
  isEnd?: boolean;
}

export const DeepResearchDrawer = ({
  open,
  onClose,
  defaultActiveTab,
  revision,
  steps,
  plannerStatus,
  executionStatus,
  reportStatus,
  reportData,
  reportContent,
  searchHits,
  formatUrl,
  requestHeaders,
  theme,
  isMobile,
  showReportOnly = false,
  isEnd,
  t: tProp,
}: DeepResearchDrawerProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [activeTab, setActiveTab] = useState<TabKey>(
    (defaultActiveTab as TabKey) || TAB_KEYS.STEPS
  );

  useEffect(() => {
    if (showReportOnly) {
      setActiveTab(TAB_KEYS.REPORT);
    } else if (defaultActiveTab) {
      setActiveTab(defaultActiveTab as TabKey);
    } else {
      setActiveTab(TAB_KEYS.STEPS);
    }
  }, [revision]);

  return (
    <CommonDrawer
      placement="right"
      open={open}
      onClose={onClose}
      size={800}
      clickOutsideToClose={isMobile ? true : false}
      classNames={{
        wrapper: `${isMobile ? '!left-0px !right-0px !w-full !top-64px !bottom-0px' : '!right-24px !top-88px !bottom-24px'}`,
        body: '!p-0px !overflow-hidden !h-full',
      }}
    >
      <div className="py-24px pl-24px pr-62px flex items-center justify-between flex-wrap gap-y-12px relative">
        {showReportOnly ? (
          <div className="text-lg font-medium text-gray-900 dark:text-gray-100">
            {t("deepResearch.tab.report")}
          </div>
        ) : (
          <Segmented
            value={activeTab}
            onChange={(val) => setActiveTab(val as TabKey)}
            options={[
              { label: t("deepResearch.tab.report"), value: TAB_KEYS.REPORT },
              { label: t("deepResearch.tab.steps"), value: TAB_KEYS.STEPS },
              { label: t("deepResearch.tab.searchResults"), value: TAB_KEYS.SEARCH_RESULTS },
            ]}
            classNames={{
              root: "!p-4px !bg-transparent border border-[#F0F0F0] dark:border-[#303030] rounded-8px",
              item: "h-32px !rounded-8px !bg-white dark:!bg-black [&:not(:last-child)]:mr-4px [&.ant-segmented-item-selected]:!bg-[rgba(1,138,229,0.09)] dark:[&.ant-segmented-item-selected]:!bg-[rgba(100,181,246,0.2)] [&:not(.ant-segmented-item-selected)]:hover:!bg-[rgba(1,138,229,0.09)] dark:[&:not(.ant-segmented-item-selected)]:hover:!bg-[rgba(100,181,246,0.2)]",
              label: "!px-16px h-full !rounded-8px text-16px text-[#333] dark:text-[#E5E7EB] [.ant-segmented-item-selected>&]:!text-[#1784FC] dark:[.ant-segmented-item-selected>&]:!text-[#7EC2FF] [.ant-segmented-item:not(.ant-segmented-item-selected):hover>&]:!text-[#1784FC] dark:[.ant-segmented-item:not(.ant-segmented-item-selected):hover>&]:!text-[#7EC2FF]",
            }}
          />
        )}
        <div className="flex items-center gap-2">
          {activeTab === TAB_KEYS.REPORT && (
            <>
              <ActionButton
                className="bg-[#E9F0FE] dark:bg-blue-900/30"
                onClick={() => {
                  const url = reportData?.url ? (formatUrl?.({ url: reportData.url }) || reportData.url) : undefined;
                  if (url) {
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = (reportData?.title as string) || '';
                    a.target = '_blank';
                    a.rel = 'noopener noreferrer';
                    a.click();
                  }
                }}
                key="download"
                icon={<Download className="w-4 h-4" />}
              >
                {t("deepResearch.button.download")}
              </ActionButton>
              <ActionButton
                className="bg-[#E9F0FE] dark:bg-blue-900/30"
                onClick={() => {
                  const url = reportData?.url ? (formatUrl?.({ url: reportData.url }) || reportData.url) : undefined;
                  if (url) {
                    window.open(url, '_blank', 'noopener,noreferrer');
                  }
                }}
                key="source"
                icon={<SquareArrowOutUpRight className="w-4 h-4" />}
              >
                {t('labels.openSource')}
              </ActionButton>
            </>
          )}
        </div>
        <div className="absolute top-29px right-21px">
          <Button
            type="text"
            onClick={onClose}
            className="!px-3px text-[#999] hover:text-gray-600 flex items-center justify-center"
          >
            <X className="w-24px h-24px" />
          </Button>
        </div>
      </div>

      <div className={`${activeTab === TAB_KEYS.SEARCH_RESULTS ? 'pb-8px px-8px' : 'px-24px pb-24px'} flex-1 overflow-y-auto bg-white dark:bg-black`}>
        {activeTab === TAB_KEYS.REPORT && (
          <ResearchReportContent
            content={reportContent}
            data={reportData}
            formatUrl={formatUrl}
            requestHeaders={requestHeaders}
            t={t}
          />
        )}
        {activeTab === TAB_KEYS.STEPS && (
          <ResearchStepsContent
            steps={steps}
            plannerStatus={plannerStatus}
            executionStatus={executionStatus}
            reportStatus={reportStatus}
            isEnd={isEnd}
            t={t}
          />
        )}
        {activeTab === TAB_KEYS.SEARCH_RESULTS && (
          <ResearchSearchResultsContent hits={searchHits} theme={theme} />
        )}
      </div>
    </CommonDrawer>
  );
};
