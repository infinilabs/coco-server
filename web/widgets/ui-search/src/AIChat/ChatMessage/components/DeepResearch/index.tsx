import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import { Loader, Hourglass, CirclePause, Check } from "lucide-react";

import { DeepResearchDrawer } from "./DeepResearchDrawer";
import type {
  StepItem,
  StepStatus,
  StepSearch,
  StepSearchStatus,
  StepSearchHit,
} from "./ResearchStepsContent";
import type { ResearchReportData } from "./ResearchReportContent";
import type { IChunkData } from "../../types/chat";

interface DeepResearchProps {
  Detail?: { type: string; payload?: IChunkData[] };
  ChunkData?: IChunkData[];
  question?: string;
  formatUrl?: (data: any) => string;
  theme?: "light" | "dark";
  t?: TFunction;
}

interface DeepResearchState {
  deepResearchPlans: string[];
  deepResearchCurrentStepIndex: number;
  deepResearchCurrentStepFinished: boolean;
  deepResearchQuery: string;
  deepResearchResultCount: number | undefined;
  deepResearchPlannerStarted: boolean;
  deepResearchResearcherStarted: boolean;
  deepResearchReporterStarted: boolean;
  deepResearchReporterFinished: boolean;
  deepResearchReportData: ResearchReportData | undefined;
  deepResearchSearchMap: Record<
    string,
    { query?: string; resultCount?: number; hits?: StepSearchHit[] }
  >;
}

const deriveDeepResearchState = (chunks: IChunkData[]): DeepResearchState => {
  const state: DeepResearchState = {
    deepResearchPlans: [],
    deepResearchCurrentStepIndex: -1,
    deepResearchCurrentStepFinished: false,
    deepResearchQuery: "",
    deepResearchResultCount: undefined,
    deepResearchPlannerStarted: false,
    deepResearchResearcherStarted: false,
    deepResearchReporterStarted: false,
    deepResearchReporterFinished: false,
    deepResearchReportData: undefined,
    deepResearchSearchMap: {},
  };

  chunks.forEach((chunkData) => {
    if (chunkData.chunk_type === "research_planner_start") {
      state.deepResearchPlans = [];
      state.deepResearchCurrentStepIndex = -1;
      state.deepResearchCurrentStepFinished = false;
      state.deepResearchQuery = "";
      state.deepResearchResultCount = undefined;
      state.deepResearchPlannerStarted = true;
      state.deepResearchResearcherStarted = false;
      state.deepResearchReporterStarted = false;
      state.deepResearchReporterFinished = false;
      state.deepResearchReportData = undefined;
      state.deepResearchSearchMap = {};
      return;
    }

    if (chunkData.chunk_type === "research_planner_end") {
      if (typeof chunkData.message_chunk === "string") {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          if (Array.isArray(payload)) {
            const plans = payload.map((item) => String(item));
            state.deepResearchPlans = plans;
            state.deepResearchCurrentStepIndex = plans.length > 0 ? 0 : -1;
          }
          state.deepResearchPlannerStarted = false;
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_start") {
      if (
        typeof chunkData.message_chunk === "string" &&
        chunkData.message_chunk
      ) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText =
            typeof payload?.plan === "string" ? payload.plan : "";
          if (planText) {
            state.deepResearchResearcherStarted = true;
            state.deepResearchCurrentStepFinished = false;

            let index = state.deepResearchPlans.findIndex(
              (title) => title === planText,
            );

            // If plan not found, add it to the list
            if (index === -1) {
              state.deepResearchPlans.push(planText);
              index = state.deepResearchPlans.length - 1;
            }

            state.deepResearchCurrentStepIndex = index;
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_step_start") {
      if (
        typeof chunkData.message_chunk === "string" &&
        chunkData.message_chunk
      ) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText =
            typeof payload?.plan === "string" ? payload.plan : "";
          const stepQuery = payload?.step?.payload?.query;

          // Ensure plan exists in state if not already
          if (planText && !state.deepResearchPlans.includes(planText)) {
            state.deepResearchPlans.push(planText);
            if (state.deepResearchCurrentStepIndex === -1) {
              state.deepResearchCurrentStepIndex =
                state.deepResearchPlans.length - 1;
            }
          }

          if (typeof stepQuery === "string") {
            state.deepResearchQuery = stepQuery;
          }
          state.deepResearchResultCount = undefined;
          if (planText && typeof stepQuery === "string") {
            const prevInfo = state.deepResearchSearchMap[planText] ?? {};
            state.deepResearchSearchMap[planText] = {
              ...prevInfo,
              query: stepQuery,
            };
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_step_end") {
      if (
        typeof chunkData.message_chunk === "string" &&
        chunkData.message_chunk
      ) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          const planText =
            typeof payload?.plan === "string" ? payload.plan : "";
          const hits = payload?.step?.payload?.hits;
          if (Array.isArray(hits)) {
            state.deepResearchResultCount = hits.length;
            if (planText) {
              const prevInfo = state.deepResearchSearchMap[planText] ?? {};
              state.deepResearchSearchMap[planText] = {
                ...prevInfo,
                resultCount: hits.length,
                hits: hits,
              };
            }
          }
        } catch (error) {
          console.error(error);
        }
      }
      return;
    }

    if (chunkData.chunk_type === "research_researcher_end") {
      state.deepResearchQuery = "";
      state.deepResearchCurrentStepFinished = true;
      return;
    }

    if (chunkData.chunk_type === "research_reporter_start") {
      state.deepResearchReporterStarted = true;
      return;
    }

    if (chunkData.chunk_type === "research_reporter_end") {
      state.deepResearchReporterStarted = true;
      state.deepResearchReporterFinished = true;
      if (
        typeof chunkData.message_chunk === "string" &&
        chunkData.message_chunk
      ) {
        try {
          const payload = JSON.parse(chunkData.message_chunk);
          state.deepResearchReportData = payload;
        } catch (error) {
          console.error(error);
        }
      }
    }
  });

  return state;
};

export const DeepResearch = ({
  Detail,
  ChunkData = [],
  question,
  formatUrl,
  theme,
  t: tProp,
}: DeepResearchProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [isDrawerOpen, setIsDrawerOpen] = useState(true);
  const [drawerDefaultTab, setDrawerDefaultTab] = useState(
    t("deepResearch.tab.steps"),
  );

  // Merge persisted detail chunks (from ES history) with live streaming chunks.
  // Detail.payload contains the saved chunks; ChunkData contains real-time ones.
  const allChunks = useMemo(() => {
    const saved = Detail?.payload ?? [];
    if (ChunkData.length > 0) return ChunkData;
    return saved;
  }, [Detail?.payload, ChunkData]);

  const {
    deepResearchPlans,
    deepResearchCurrentStepIndex,
    deepResearchCurrentStepFinished,
    deepResearchQuery,
    deepResearchResultCount,
    deepResearchPlannerStarted,
    deepResearchResearcherStarted,
    deepResearchReporterStarted,
    deepResearchReporterFinished,
    deepResearchReportData,
    deepResearchSearchMap,
  } = useMemo(() => deriveDeepResearchState(allChunks), [allChunks]);

  const hasDeepResearchPlan =
    deepResearchPlans.length > 0 &&
    deepResearchCurrentStepIndex >= 0 &&
    deepResearchCurrentStepIndex < deepResearchPlans.length;

  const stepTitle = hasDeepResearchPlan
    ? deepResearchPlans[deepResearchCurrentStepIndex]
    : "";

  const deepResearchPlanningProgress = deepResearchPlans.length > 0 ? 1 : 0;

  const deepResearchExecutionProgress = hasDeepResearchPlan
    ? (deepResearchCurrentStepIndex + 1) / deepResearchPlans.length
    : 0;

  const deepResearchReportProgress = deepResearchReporterFinished
    ? 1
    : deepResearchReporterStarted
      ? 0.5
      : 0;

  const progress =
    (deepResearchPlanningProgress +
      deepResearchExecutionProgress +
      deepResearchReportProgress) /
    3;

  const statusText = useMemo(() => {
    if (deepResearchReporterFinished) {
      if (typeof deepResearchResultCount === "number") {
        return `深度研究完成 · 找到 ${deepResearchResultCount} 条相关结果`;
      }
      return "深度研究完成";
    }
    if (deepResearchReporterStarted) {
      return "正在编写研究报告";
    }
    if (deepResearchResearcherStarted) {
      return "正在执行研究计划";
    }
    if (deepResearchPlans.length > 0) {
      return "正在规划研究计划";
    }
    return undefined;
  }, [
    deepResearchReporterFinished,
    deepResearchResultCount,
    deepResearchReporterStarted,
    deepResearchResearcherStarted,
    deepResearchPlans.length,
  ]);

  const steps = useMemo<StepItem[]>(() => {
    if (!deepResearchPlans.length) return [];

    return deepResearchPlans.map((title, index) => {
      let status: StepStatus = "pending";

      if (deepResearchReporterFinished || deepResearchReporterStarted) {
        status = "done";
      } else if (deepResearchResearcherStarted) {
        if (index < deepResearchCurrentStepIndex) {
          status = "done";
        } else if (index === deepResearchCurrentStepIndex) {
          status = deepResearchCurrentStepFinished ? "done" : "in_progress";
        }
      }

      const searchInfo = deepResearchSearchMap[title];
      const searches: StepSearch[] | undefined = searchInfo?.query
        ? [
            {
              id: `step-${index + 1}-search-1`,
              query: searchInfo.query,
              resultCount: searchInfo.resultCount,
              status:
                typeof searchInfo.resultCount === "number"
                  ? ("done" as StepSearchStatus)
                  : ("searching" as StepSearchStatus),
              hits: searchInfo.hits,
            },
          ]
        : undefined;

      return {
        id: `step-${index + 1}`,
        title,
        status,
        searches,
        showOptimizePlan: false,
      };
    });
  }, [
    deepResearchPlans,
    deepResearchReporterFinished,
    deepResearchReporterStarted,
    deepResearchResearcherStarted,
    deepResearchSearchMap,
    deepResearchCurrentStepIndex,
    deepResearchCurrentStepFinished,
  ]);

  const searchHits = useMemo(() => {
    const allHits: StepSearchHit[] = [];
    Object.values(deepResearchSearchMap).forEach((info) => {
      if (info.hits && Array.isArray(info.hits)) {
        allHits.push(...info.hits);
      }
    });
    return allHits;
  }, [deepResearchSearchMap]);

  const plannerStatus: StepStatus = deepResearchPlans.length
    ? "done"
    : deepResearchPlannerStarted
      ? "in_progress"
      : "pending";

  const executionStatus: StepStatus = useMemo(() => {
    if (!steps.length) return "pending";
    if (steps.some((step) => step.status === "in_progress")) {
      return "in_progress";
    }
    if (steps.some((step) => step.status === "done")) {
      return "done";
    }
    return "pending";
  }, [steps]);

  const reportStatus: StepStatus = deepResearchReporterFinished
    ? "done"
    : deepResearchReporterStarted
      ? "in_progress"
      : "pending";

  const normalizedProgress = useMemo(() => {
    if (typeof progress !== "number" || Number.isNaN(progress)) return 0;
    if (progress < 0) return 0;
    if (progress > 1) return 1;
    return progress;
  }, [progress]);

  const displayStatus = useMemo(() => {
    if (statusText) return statusText;
    if (normalizedProgress >= 1) {
      if (typeof deepResearchResultCount === "number") {
        return t("deepResearch.status.completedWithCount", {
          count: deepResearchResultCount,
        });
      }
      return t("deepResearch.status.completed");
    }
    if (normalizedProgress > 0) {
      return t("deepResearch.status.running");
    }
    return t("deepResearch.status.preparing");
  }, [statusText, normalizedProgress, deepResearchResultCount, t]);

  // 如果没有数据，不显示组件
  if (!ChunkData.length) {
    return null;
  }

  return (
    <>
      <div
        className="w-full my-3 cursor-pointer"
        onClick={() => {
          setDrawerDefaultTab(t("deepResearch.tab.steps"));
          setIsDrawerOpen(true);
        }}
      >
        <div className="w-full rounded-xl border border-[#EEF0F3] bg-[#F3F4F6] dark:border-[#1D3A6F] dark:bg-[#020817] p-4">
          <div className="flex items-center gap-2 mb-4">
            <Hourglass className="h-4 w-4 text-[#148EFF] cm-hourglass-rotate" />
            <div className="text-sm font-medium text-[#333] dark:text-[#E5E7EB] truncate">
              {stepTitle || "——"}
            </div>
          </div>

          <div className="mt-2 flex items-center justify-between rounded-full bg-white px-3 py-2 text-sm text-[#333] dark:bg-[#111827] dark:text-[#D1D5DB]">
            <div className="flex min-w-0 items-center gap-2 flex-1">
              {normalizedProgress >= 1 ? (
                <Check className="h-4 w-4 text-[#22C55E] shrink-0" />
              ) : (
                <Loader className="h-4 w-4 animate-spin text-[#148EFF] shrink-0" />
              )}
              <div className="flex min-w-0 items-center flex-1">
                <span className="whitespace-nowrap shrink-0">
                  {displayStatus}
                </span>
                <span className="text-[#999] dark:text-[#A6A6A6] truncate ml-1">
                  ｜ {deepResearchQuery || question}
                </span>
              </div>
            </div>
            <div className="ml-2 flex items-center gap-2 shrink-0">
              {normalizedProgress < 1 &&
              typeof deepResearchResultCount === "number" ? (
                <div className="flex py-0.5 px-1 items-center justify-center rounded-full border border-solid border-[#018AE5] bg-white text-xs font-medium text-[#018AE5] dark:bg-[#020617]">
                  {deepResearchResultCount}
                </div>
              ) : null}
              {normalizedProgress >= 1 && (
                <button
                  type="button"
                  className="px-3 py-1 text-xs font-medium rounded-full bg-[#E9F0FE] dark:bg-blue-900/30 text-[#1784FC] dark:text-blue-400 hover:bg-[#E0E9FD] dark:hover:bg-blue-900/50 cursor-pointer"
                  onClick={(e) => {
                    e.stopPropagation();
                    setDrawerDefaultTab(t("deepResearch.tab.report"));
                    setIsDrawerOpen(true);
                  }}
                >
                  {t("deepResearch.button.view")}
                </button>
              )}
            </div>
          </div>

          <div className="mt-5 w-full flex items-center gap-2 overflow-hidden">
            <div className="h-2 rounded-full flex-1 items-center bg-white dark:bg-[#1F2937]">
              <div
                className={`h-full rounded-full transition-all ${
                  normalizedProgress >= 1 ? "bg-[#00C868]" : "bg-[#1784FC]"
                }`}
                style={{ width: `${normalizedProgress * 100}%` }}
              />
            </div>
            <div className="flex items-center justify-center cursor-not-allowed opacity-50">
              {normalizedProgress >= 1 ? (
                <Check className="h-4 w-4 text-[#22C55E]" />
              ) : (
                <CirclePause className="h-4 w-4 text-[#1784FC]" />
              )}
            </div>
          </div>
        </div>
      </div>
      <DeepResearchDrawer
        open={isDrawerOpen}
        onClose={() => setIsDrawerOpen(false)}
        defaultActiveTab={drawerDefaultTab}
        steps={steps}
        plannerStatus={plannerStatus}
        executionStatus={executionStatus}
        reportStatus={reportStatus}
        reportData={deepResearchReportData}
        searchHits={searchHits}
        formatUrl={formatUrl}
        theme={theme}
        t={t}
      />
    </>
  );
};
