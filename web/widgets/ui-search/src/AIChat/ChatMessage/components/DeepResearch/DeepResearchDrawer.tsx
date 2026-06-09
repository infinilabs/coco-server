import { AnimatePresence, motion } from "motion/react";
import { Button, Segmented } from "antd";
import { Download, Share2, X } from "lucide-react";
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
            className="fixed z-1000 top-20 bottom-20 right-4 flex flex-col rounded-xl overflow-hidden bg-white dark:bg-black shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.1)]"
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
                  style={{ marginBottom: 8 }}
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
                    <a
                      className="flex items-center gap-1 px-2 py-1 rounded-full bg-[#E9F0FE] dark:bg-blue-900/30 text-sm text-[#1784FC] dark:text-blue-400 hover:bg-[#E0E9FD] dark:hover:bg-blue-900/50 border-none outline-none cursor-pointer no-underline"
                      href={reportData?.url ? (formatUrl?.({ url: reportData.url }) || reportData.url) : undefined}
                      download={reportData?.title || true}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <Download className="w-4 h-4" />
                      <span>{t("deepResearch.button.download")}</span>
                    </a>
                    <button
                      type="button"
                      className="flex items-center gap-1 px-2 py-1 rounded-full bg-[#E9F0FE] dark:bg-blue-900/30 text-sm text-[#1784FC] dark:text-blue-400 hover:bg-[#E0E9FD] dark:hover:bg-blue-900/50 border-none outline-none cursor-pointer"
                    >
                      <Share2 className="w-4 h-4" />
                      <span>{t("deepResearch.button.share")}</span>
                    </button>
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

            <div className="pt-6 flex-1 overflow-y-auto bg-white dark:bg-black">
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
