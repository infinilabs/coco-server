import { AnimatePresence, motion } from "motion/react";
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

interface DeepResearchDrawerProps {
  open: boolean;
  onClose: () => void;
  defaultActiveTab?: string;
  steps?: StepItem[];
  plannerStatus?: StepStatus;
  executionStatus?: StepStatus;
  reportStatus?: StepStatus;
  reportData?: ResearchReportData;
  reportContent?: string;
  searchHits?: StepSearchHit[];
  formatUrl?: (data: any) => string;
  theme?: "light" | "dark";
  showReportOnly?: boolean;
  t?: TFunction;
}

export const DeepResearchDrawer = ({
  open,
  onClose,
  defaultActiveTab,
  steps,
  plannerStatus,
  executionStatus,
  reportStatus,
  reportData,
  reportContent,
  searchHits,
  formatUrl,
  theme,
  showReportOnly = false,
  t: tProp,
}: DeepResearchDrawerProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [activeTab, setActiveTab] = useState(defaultActiveTab || t("deepResearch.tab.steps"));

  useEffect(() => {
    if (showReportOnly) {
      setActiveTab(t("deepResearch.tab.report"));
    } else if (defaultActiveTab) {
      setActiveTab(defaultActiveTab);
    }
  }, [defaultActiveTab, showReportOnly, t]);

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            className="fixed inset-0 z-999 bg-black/20"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />
          <motion.div
            className="fixed z-1000 top-[100px] bottom-[40px] right-4 flex flex-col rounded-xl overflow-hidden bg-white dark:bg-black shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.1)]"
            initial={{
              width: 0,
              height: 0,
              opacity: 0,
              padding: 0,
            }}
            animate={{
              width: 800,
              height: "auto",
              opacity: 1,
              padding: 24,
            }}
            exit={{
              width: 0,
              height: 0,
              opacity: 0,
              padding: 0,
            }}
          >
            <div className="flex items-center justify-between">
              {showReportOnly ? (
                <div className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                  {t("deepResearch.tab.report")}
                </div>
              ) : (
                <Segmented
                  className="cm-deep-research-segmented"
                  value={activeTab}
                  onChange={(val) => setActiveTab(val as string)}
                  options={[
                    t("deepResearch.tab.report"),
                    t("deepResearch.tab.steps"),
                    t("deepResearch.tab.searchResults"),
                  ]}
                />
              )}
              <div className="flex items-center gap-2">
                {activeTab === t("deepResearch.tab.report") && (
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
                <Button
                  type="text"
                  onClick={onClose}
                  className="text-[#999] hover:text-gray-600 flex items-center justify-center"
                >
                  <X className="w-5 h-5" />
                </Button>
              </div>
            </div>

            <div className={`${activeTab === t("deepResearch.tab.searchResults") ? 'py-8px' : 'py-6'} flex-1 overflow-y-auto bg-white dark:bg-black`}>
              {activeTab === t("deepResearch.tab.report") && (
                <ResearchReportContent
                  content={reportContent}
                  data={reportData}
                  formatUrl={formatUrl}
                  t={t}
                />
              )}
              {activeTab === t("deepResearch.tab.steps") && (
                <ResearchStepsContent
                  steps={steps}
                  plannerStatus={plannerStatus}
                  executionStatus={executionStatus}
                  reportStatus={reportStatus}
                  t={t}
                />
              )}
              {activeTab === t("deepResearch.tab.searchResults") && (
                <ResearchSearchResultsContent hits={searchHits} theme={theme} />
              )}
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
