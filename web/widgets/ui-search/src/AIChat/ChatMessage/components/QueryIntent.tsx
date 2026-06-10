import { ChevronDown, ChevronRight, Loader } from "lucide-react";
import { useState, useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { IChunkData } from "../types/chat";
import CheckIcon from "../../../icons/CheckIcon";

interface QueryIntentProps {
  Detail?: any;
  ChunkData?: IChunkData;
  getSuggestion?: (suggestion: string[]) => void;
  loading?: boolean;
  t?: TFunction;
}

interface IQueryData {
  category: string;
  intent: string;
  query: string[];
  keyword: string[];
  suggestion: string[];
}

export const QueryIntent = ({
  Detail,
  ChunkData,
  getSuggestion,
  loading,
  t: tProp,
}: QueryIntentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const [isThinkingExpanded, setIsThinkingExpanded] = useState(false);

  const [data, setData] = useState<IQueryData | null>(null);

  // 使用 useRef 来避免 useEffect 依赖变化导致的无限循环
  const getSuggestionRef = useRef(getSuggestion);

  useEffect(() => {
    getSuggestionRef.current = getSuggestion;
  }, [getSuggestion]);

  useEffect(() => {
    if (!Detail?.payload) return;
    setData(Detail?.payload);
    if (Detail?.payload?.suggestion && getSuggestionRef.current) {
      getSuggestionRef.current(Detail?.payload?.suggestion);
    }
  }, [Detail?.payload]);

  useEffect(() => {
    if (!ChunkData?.message_chunk) return;
    if (!loading) {
      try {
        const cleanContent = ChunkData.message_chunk.replace(/^"|"$/g, "");
        const allMatches = cleanContent.match(/<JSON>([\s\S]*?)<\/JSON>/g);
        if (allMatches) {
          const lastMatch = allMatches[allMatches.length - 1];
          const jsonString = lastMatch.replace(/<JSON>|<\/JSON>/g, "");
          const data = JSON.parse(jsonString);
          if (data?.suggestion && getSuggestionRef.current) {
            getSuggestionRef.current(data?.suggestion);
          }
          setData(data);
        }
      } catch (error) {
        console.error("Failed to process message chunk in QueryIntent:", error);
      }
    }
  }, [ChunkData?.message_chunk, loading]);

  // Must be after hooks !!!
  if (!ChunkData && !Detail) return null;

  return (
    <div className="space-y-2 mb-8px w-full">
      <button
        onClick={() => setIsThinkingExpanded((prev) => !prev)}
        className="text-[#101010] dark:text-[#F5F5F5] cursor-pointer bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] inline-flex items-center gap-2 px-2 py-2px rounded-12px transition-colors border border-solid border-[#F0F0F0] dark:border-[#303030]"
      >
        <>
          {loading ? (
            <Loader className="w-14px h-14px animate-spin" />
          ) : (
            <CheckIcon className="w-14px h-14px" />
          )}
          <span className={`${loading ? "italic" : ""}`}>
            {t(
              `assistant.message.steps.${
                ChunkData?.chunk_type || Detail.type
              }`
            )}
          </span>
        </>
        {isThinkingExpanded ? (
          <ChevronDown className="w-4 h-4" />
        ) : (
          <ChevronRight className="w-4 h-4" />
        )}
      </button>
      {isThinkingExpanded && (
        <div className="ml-15px pl-6px pt-1 border-l-1 border-[#bbb] dark:border-[#333]">
          <div className="text-[#8b8b8b] dark:text-[#a6a6a6] space-y-2">
            <div className="mb-4 space-y-2 text-xs">
              {data?.keyword ? (
                <div className="flex gap-1">
                  <span className="text-[#999999]">
                    - {t("assistant.message.steps.keywords")}：
                  </span>
                  <div className="flex flex-wrap gap-1">
                    {data?.keyword?.map((keyword, index) => (
                      <span
                        key={keyword + index}
                        className="text-[#333333] dark:text-[#D8D8D8]"
                      >
                        {keyword}
                        {index < 2 && "、"}
                      </span>
                    ))}
                  </div>
                </div>
              ) : null}
              {data?.category ? (
                <div className="flex items-center gap-1">
                  <span className="text-[#999999]">
                    - {t("assistant.message.steps.questionType")}：
                  </span>
                  <span className="text-[#333333] dark:text-[#D8D8D8]">
                    {data?.category}
                  </span>
                </div>
              ) : null}
              {data?.intent ? (
                <div className="flex items-start gap-1">
                  <span className="text-[#999999]">
                    - {t("assistant.message.steps.userIntent")}：
                  </span>
                  <div className="flex-1 text-[#333333] dark:text-[#D8D8D8]">
                    {data?.intent}
                  </div>
                </div>
              ) : null}
              {data?.query ? (
                <div className="flex items-start gap-1">
                  <span className="text-[#999999]">
                    - {t("assistant.message.steps.relatedQuestions")}：
                  </span>
                  <div className="flex-1 flex flex-col text-[#333333] dark:text-[#D8D8D8]">
                    {data?.query?.map((question, qIndex) => (
                      <span key={question + qIndex}>- {question}</span>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
