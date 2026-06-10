import { Loader, Hammer, ChevronDown, ChevronUp, ChevronRight } from "lucide-react";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from '@infinilabs/markdown';

import type { IChunkData } from "../types/chat";
import CheckIcon from "../../../icons/CheckIcon";

interface CallToolsProps {
  Detail?: any;
  ChunkData?: IChunkData;
  loading?: boolean;
  t?: TFunction;
}

export const CallTools = ({ Detail, ChunkData, loading, t: tProp }: CallToolsProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [isThinkingExpanded, setIsThinkingExpanded] = useState(false);

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
        className="text-[#101010] dark:text-[#F5F5F5] cursor-pointer bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] inline-flex items-center gap-2 px-2 py-2px rounded-12px transition-colors border border-solid border-[#F0F0F0] dark:border-[#303030]"
      >
        <>
          {loading ? (
            <Loader className="w-14px h-14px animate-spin" />
          ) : (
            <CheckIcon className="w-14px h-14px" />
          )}
          <span className={`${loading ? "italic" : ""}`}>
            {t(`assistant.message.steps.${ChunkData?.chunk_type}`)}
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
          <div className="text-[#8b8b8b] dark:text-[#a6a6a6] space-y-2 cm-markdown">
            <Markdown content={data || ""} />
            {/* {data?.split("\n").map(
              (paragraph, idx) =>
                paragraph.trim() && (
                  <p key={idx} className="text-sm">
                    {paragraph}
                  </p>
                )
            )} */}
          </div>
        </div>
      )}
    </div>
  );
};
