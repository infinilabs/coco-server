import {
  Search,
  FileText,
  ChevronDown,
  BookOpen,
  List,
  ChevronUp,
  Hourglass,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import CheckIcon from "../../../../icons/CheckIcon";
import EditIcon from "../../../../icons/EditIcon";

/**
 * 步骤状态类型定义
 * done: 已完成
 * in_progress: 进行中
 * pending: 等待中
 */
export type StepStatus = "done" | "in_progress" | "pending";

/**
 * 搜索状态类型定义
 * done: 搜索完成
 * searching: 正在搜索
 */
export type StepSearchStatus = "done" | "searching";

/**
 * 搜索结果条目接口
 */
export interface StepSearchHit {
  // 来源名称
  source?: string;
  // 标题
  title: string;
  // 链接地址
  url?: string;
  // 内容摘要
  content?: string;
  // 相关性得分
  score?: number;
}

/**
 * 搜索任务接口
 */
export interface StepSearch {
  // 唯一标识
  id: string;
  // 搜索查询语句
  query: string;
  // 结果数量
  resultCount?: number;
  // 状态
  status: StepSearchStatus;
  // 备注信息
  note?: string;
  // 搜索结果列表
  hits?: StepSearchHit[];
}

/**
 * 研究步骤条目接口
 */
export interface StepItem {
  // 步骤唯一标识
  id: string;
  // 步骤标题
  title: string;
  // 步骤描述
  description?: string;
  // 步骤状态
  status: StepStatus;
  // 该步骤下的搜索任务
  searches?: StepSearch[];
  // 是否显示优化计划
  showOptimizePlan?: boolean;
}

interface ResearchStepsContentProps {
  // 步骤列表数据
  steps?: StepItem[];
  // 计划生成状态
  plannerStatus?: StepStatus;
  // 执行状态
  executionStatus?: StepStatus;
  // 报告生成状态
  reportStatus?: StepStatus;
  isEnd?: boolean;
  t?: TFunction;
}

/**
 * 深层研究步骤内容组件
 * 展示研究计划、执行步骤和报告生成的全过程状态
 */
export const ResearchStepsContent = ({
  steps,
  plannerStatus,
  executionStatus,
  reportStatus,
  isEnd,
  t: tProp,
}: ResearchStepsContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const rawData = steps ?? [];

  // First step defaults to in_progress if not done and not ended
  const data = rawData.map((step, index) => {
    if (index === 0 && !isEnd && step.status !== "done") {
      return { ...step, status: "in_progress" as StepStatus };
    }
    return step;
  });
  // 控制展开的搜索结果集合
  const [expandedSearches, setExpandedSearches] = useState<Set<string>>(
    new Set(),
  );
  // 控制计划列表的展开/折叠
  const [plansExpanded, setPlansExpanded] = useState(false);

  /**
   * 切换搜索结果的展开状态
   */
  const toggleSearch = (searchId: string) => {
    setExpandedSearches((prev) => {
      const next = new Set(prev);
      if (next.has(searchId)) {
        next.delete(searchId);
      } else {
        next.add(searchId);
      }
      return next;
    });
  };

  // 找到第一个非 pending 状态的步骤索引，用于连接线的起始位置
  const firstActiveIndex = data.findIndex((step) => step.status !== "pending");

  // 找到最后一个非 pending 状态的步骤索引，用于连接线的结束位置
  const lastActiveIndex = data.reduce(
    (lastIndex, step, index) => (step.status !== "pending" ? index : lastIndex),
    -1,
  );

  // 自动计算执行状态：如果有步骤进行中则为 in_progress，有步骤完成则为 done，否则为 pending
  const autoExecutionStatus: StepStatus = (() => {
    if (!data.length) return "pending";
    if (data.some((step) => step.status === "in_progress")) {
      return "in_progress";
    }
    if (data.every((step) => step.status === "done")) {
      return "done";
    }
    if (data.some((step) => step.status === "done")) {
      return "in_progress";
    }
    return "pending";
  })();

  const planner = plannerStatus ?? (data.length ? "done" : "pending");
  const execution = executionStatus ?? autoExecutionStatus;
  const report = reportStatus ?? "pending";

  return (
    <div className="space-y-6 pr-6 px-24px pb-24px">
      <div className="text-sm text-[#333] dark:text-[#E5E7EB]">
        {t("deepResearch.steps.intro")}
      </div>

      {/* 计划生成阶段 */}
      <div className="space-y-4">
        <div
          className={`flex items-center gap-2 text-base font-medium ${
            planner === "pending"
              ? "text-[#999] dark:text-[#666]"
              : "text-[#333] dark:text-[#E5E7EB]"
          }`}
        >
          {planner === "in_progress" ? (
            <Hourglass className={`w-5 h-5 text-[#1784FC] ${isEnd ? "" : "animate-spin"}`} />
          ) : (
            <EditIcon
              className={`w-5 h-5 ${
                planner === "pending" ? "text-[#999] dark:text-[#666]" : "text-[#1784FC]"
              }`}
            />
          )}
          {t("deepResearch.steps.planTitle")}
        </div>
        <div className=" text-[#999] dark:text-[#666] text-sm mb-2">
          {t("deepResearch.steps.planDescription")}
        </div>

        {/* 计划列表展开开关 */}
        <div
          className="border border-[#F0F0F0] dark:border-[#303030] rounded-lg p-3 bg-transparent flex justify-between items-center cursor-pointer hover:bg-gray-50 dark:hover:bg-[#111827] transition-colors"
          onClick={() => setPlansExpanded((prev) => !prev)}
        >
          <div className="flex items-center gap-2 text-sm text-[#333] dark:text-[#E5E7EB]">
            <List className="w-4 h-4 text-[#1784FC]" />
            {t("deepResearch.steps.generatedPlans")}
          </div>
          {plansExpanded ? (
            <ChevronUp className="w-4 h-4 text-gray-400" />
          ) : (
            <ChevronDown className="w-4 h-4 text-gray-400" />
          )}
        </div>

        {/* 计划列表详情 */}
        {plansExpanded && data.length > 0 && (
          <div className="mt-2 space-y-1 rounded-lg bg-transparent border border-[#F0F0F0] dark:border-[#303030] p-3">
            {data.map((step, index) => (
              <div key={step.id} className="flex items-start gap-2 text-sm">
                <span className="text-[#999] dark:text-[#666]">
                  {index + 1}.
                </span>
                <span className="text-[#333] dark:text-[#E5E7EB]">
                  {step.title}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 执行阶段 */}
      <div className="">
        <div
          className={`flex items-center gap-2 text-base font-medium mb-4 ${
            execution === "pending"
              ? "text-[#999] dark:text-[#666]"
              : "text-[#333] dark:text-[#E5E7EB]"
          }`}
        >
          <List className="w-5 h-5 text-[#1784FC]" />
          {t("deepResearch.steps.executeTitle")}
        </div>

        {/* 步骤列表渲染 */}
        {data.map((step, index) => {
          const isActive = step.status !== "pending";
          const isFirstActive = index === firstActiveIndex;
          const isLastActive = index === lastActiveIndex;

          const isSimplePending = step.status === "pending" && !step.searches;

          return (
            <div
              key={step.id}
              className={`relative ${
                isSimplePending ? "pl-6 pb-4" : "pl-6 pb-8 last:pb-0"
              }`}
            >
              {/* 步骤之间的连接虚线 */}
              {isActive &&
                firstActiveIndex !== -1 &&
                lastActiveIndex !== -1 && (
                  <div
                    className="absolute left-2 border-l border-dashed border-[#018AE5]/20 dark:border-gray-700"
                    style={{
                      top: isFirstActive ? 10 : 16,
                      bottom: isLastActive ? 10 : -12,
                    }}
                  />
                )}

              {/* 完成状态图标 */}
              {step.status === "done" && (
                <CheckIcon className="absolute left-0 top-2px w-4 h-4 text-[#1784FC]" />
              )}

              {/* 进行中状态图标 */}
              {step.status === "in_progress" && (
                <Hourglass className={`absolute left-0 top-2px w-4 h-4 text-[#1784FC] ${isEnd ? "" : "animate-spin"}`} />
              )}

              {
                isSimplePending && (
                  <List className="absolute left-0 top-2px w-4 h-4 text-[#C8C8C8]" />
                )
              }

              {/* 等待中的简化展示 */}
              {isSimplePending ? (
                <>
                  <div className="text-sm text-[#999] dark:text-[#666]">
                    {step.title}
                  </div>
                </>
              ) : (
                <>
                  {/* 步骤标题 */}
                  <h3
                    className={`text-sm font-bold mb-16px ${
                      step.status === "pending"
                        ? "text-[#999] dark:text-[#666]"
                        : "text-[#333] dark:text-[#E5E7EB]"
                    }`}
                  >
                    {step.title}
                  </h3>

                  {/* 步骤描述 */}
                  {step.description && (
                    <p className="text-[#999] dark:text-[#666] text-sm mb-8px">
                      {step.description}
                    </p>
                  )}

                  {/* 搜索任务列表 */}
                  {step.searches && (
                    <div className="space-y-3">
                      {step.searches.map((search) =>
                        search.status === "searching" ? (
                          // 正在搜索状态
                          <div
                            key={search.id}
                            className="flex items-center justify-between border border-blue-100 dark:border-blue-900/30 bg-blue-50/30 dark:bg-blue-900/10 rounded-lg p-3"
                          >
                            <div className="flex items-center gap-2 text-sm overflow-hidden">
                              <Search className="w-4 h-4 text-[#1784FC] animate-pulse shrink-0" />
                              <span className="text-[#333] dark:text-[#E5E7EB] shrink-0">
                                {t("deepResearch.steps.searching")}
                              </span>
                              {
                                search.query && (
                                  <span className="text-[#999] dark:text-[#666] truncate">
                                    ｜ {search.query}
                                  </span>
                                )
                              }
                            </div>
                          </div>
                        ) : (
                          // 搜索完成状态
                          <div key={search.id}>
                            <div
                              className="flex items-center justify-between border border-[#F0F0F0] dark:border-[#303030] rounded-lg p-3 bg-transparent cursor-pointer hover:bg-gray-100 dark:hover:bg-[#1F2937] transition-colors"
                              onClick={() => toggleSearch(search.id)}
                            >
                              <div className="flex items-center gap-2 text-sm overflow-hidden">
                                <Search className="w-4 h-4 text-[#1784FC] shrink-0" />
                                <span className="text-[#333] dark:text-[#E5E7EB] shrink-0">
                                  {t("deepResearch.steps.searchTitle")}
                                </span>
                                {
                                  search.query && (
                                    <span className="text-[#999] dark:text-[#666] truncate">
                                      ｜ {search.query}
                                    </span>
                                  )
                                }
                              </div>
                              {typeof search.resultCount === "number" && (
                                <div className="flex items-center gap-8px">
                                  <div className="min-w-24px flex px-1 items-center justify-center rounded-12px border border-solid border-[rgba(1,138,229,0.21)] bg-transparent text-xs font-medium text-[#1784FC] dark:text-[#7EC2FF]">
                                    {search.resultCount}
                                  </div>
                                  {expandedSearches.has(search.id) ? (
                                    <ChevronUp className="w-4 h-4 text-[#999] dark:text-[#666]" />
                                  ) : (
                                    <ChevronDown className="w-4 h-4 text-[#999] dark:text-[#666]" />
                                  )}
                                </div>
                              )}
                            </div>

                            {/* 展开的搜索结果详情 */}
                            {expandedSearches.has(search.id) && search.hits && (
                              <div className="mt-2 space-y-2 pl-4">
                                {search.hits.map((hit, idx) => (
                                  <div
                                    key={idx}
                                    className="p-3 bg-transparent border border-[#F0F0F0] dark:border-[#303030] rounded-lg hover:border-blue-200 dark:hover:border-blue-800 transition-colors cursor-pointer group"
                                    onClick={() =>
                                      hit.url && window.open(hit.url, "_blank")
                                    }
                                  >
                                    <div className="flex items-start gap-2">
                                      <div className="min-w-0 flex-1">
                                        <div className="text-sm font-medium text-[#333] dark:text-[#E5E7EB] group-hover:text-[#1784FC] transition-colors line-clamp-1">
                                          {hit.title}
                                        </div>
                                        {hit.content && (
                                          <div className="text-xs text-[#999] dark:text-[#666] mt-1 line-clamp-2">
                                            {hit.content}
                                          </div>
                                        )}
                                      </div>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            )}

                            {search.note && (
                              <p className="text-[#999] dark:text-[#666] text-sm mt-4 mb-2">
                                {search.note}
                              </p>
                            )}
                          </div>
                        ),
                      )}

                      {/* 优化计划提示 */}
                      {step.showOptimizePlan && (
                        <div className="border border-[#F0F0F0] dark:border-[#303030] rounded-lg p-3 bg-transparent flex justify-between items-center cursor-pointer hover:bg-gray-50 dark:hover:bg-[#111827] transition-colors">
                          <div className="flex items-center gap-2 text-sm">
                            <FileText className="w-4 h-4 text-[#1784FC]" />
                            <span className="text-[#333] dark:text-[#E5E7EB]">
                              {t("deepResearch.steps.optimizePlan")}
                            </span>
                          </div>
                          <ChevronDown className="w-4 h-4 text-[#999] dark:text-[#666]" />
                        </div>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>
          );
        })}

        {/* 报告生成阶段 */}
        <div className="">
          <div
            className={`flex items-center gap-2 text-base font-medium mb-4 ${
              report === "pending"
                ? "text-[#999] dark:text-[#666]"
                : "text-[#333] dark:text-[#E5E7EB]"
            }`}
          >
            {report === "in_progress" ? (
             <Hourglass className={`w-5 h-5 text-[#1784FC] ${isEnd ? "" : "animate-spin"}`} />
            ) : (
              <BookOpen
                className={`w-5 h-5 ${
                  report === "pending" ? "text-[#999] dark:text-[#666]" : "text-[#1784FC]"
                }`}
              />
            )}
            {t("deepResearch.steps.reportTitle")}
          </div>
        </div>
      </div>
    </div>
  );
};
