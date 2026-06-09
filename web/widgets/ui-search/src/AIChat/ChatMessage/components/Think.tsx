import {Loader, Brain, ChevronDown, ChevronUp } from "lucide-react";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { IChunkData } from "../types/chat";

interface ThinkProps {
  Detail?: any;
  ChunkData?: IChunkData;
  loading?: boolean;
  t?: TFunction;
}

export const Think = ({ Detail, ChunkData, loading, t: tProp }: ThinkProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [isThinkingExpanded, setIsThinkingExpanded] = useState(true);

  const [data, setData] = useState("");

  useEffect(() => {
    if (!Detail?.description) return;
    setData(Detail?.description);
  }, [Detail?.description]);

  useEffect(() => {
    if (!ChunkData?.message_chunk) return;
    setData(ChunkData?.message_chunk);
  }, [ChunkData?.message_chunk, data]);

  // Must be after hooks !!!
  if (!ChunkData && !Detail) return null;

  return (
    <div className="space-y-2 mb-3 w-full">
      <button
        onClick={() => setIsThinkingExpanded((prev) => !prev)}
        className="cursor-pointer bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] inline-flex items-center gap-2 px-2 py-1 rounded-xl transition-colors border border-solid border-[#F0F0F0] dark:border-[#303030]"
      >
        {loading ? (
          <>
            <Loader className="w-4 h-4 animate-spin text-[#1990FF]" />
            <span className="text-xs text-[#999999] italic">
              {t(`assistant.message.steps.${ChunkData?.chunk_type}`)}
            </span>
          </>
        ) : (
          <>
            <Brain className="w-4 h-4 text-[#38C200]" />
            <span className="text-xs text-[#999999]">
              {t("assistant.message.steps.thoughtTime")}
            </span>
          </>
        )}
        {isThinkingExpanded ? (
          <ChevronUp className="w-4 h-4 text-[#999999]" />
        ) : (
          <ChevronDown className="w-4 h-4 text-[#999999]" />
        )}
      </button>
      {isThinkingExpanded && (
        <div className="pl-2 pt-1 border-l-2 border-[#F0F0F0] dark:border-[#303030]">
          <div className="text-[#8b8b8b] dark:text-[#a6a6a6] space-y-2">
            {data?.split("\n").map(
              (paragraph, idx) =>
                paragraph.trim() && (
                  <p key={idx} className="text-sm">
                    {paragraph}
                  </p>
                )
            )}
          </div>
        </div>
      )}
    </div>
  );
};
