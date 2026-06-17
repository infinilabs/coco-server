import {Loader, ChevronDown, ChevronRight, Brain } from "lucide-react";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { IChunkData } from "../types/chat";
import CheckIcon from "../../../icons/CheckIcon";

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
    <div className="space-y-2 mb-8px w-full">
      <button
        onClick={() => setIsThinkingExpanded((prev) => !prev)}
        className="text-[12px] text-[#999] dark:text-[#666] cursor-pointer bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] inline-flex items-center gap-2 px-2 py-2px rounded-12px transition-colors border border-solid border-[#F0F0F0] dark:border-[#303030]"
      >
        {loading ? (
          <>
            <Loader className="w-14px h-14px animate-spin text-[#1784FC] shrink-0" />
            <span className="italic">
              {t(`assistant.message.steps.${ChunkData?.chunk_type}`)}
            </span>
          </>
        ) : (
          <>
            <Brain className="w-14px h-14px shrink-0" />
            <span className="">
              {t("assistant.message.steps.thoughtTime")}
            </span>
          </>
        )}
        {isThinkingExpanded ? (
          <ChevronDown className="w-14px h-14px" />
        ) : (
          <ChevronRight className="w-14px h-14px" />
        )}
      </button>
      {isThinkingExpanded && data && (
        <div className="ml-8px pl-8px border-l-1 border-[#F0F0F0] dark:border-[#303030]">
          <div className="text-xs text-[#999] dark:text-[#666] whitespace-pre-wrap">
            {data}
          </div>
        </div>
      )}
    </div>
  );
};
