import { ChevronDown, ChevronUp, Loader } from "lucide-react";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { IChunkData } from "../types/chat";
import ReadingIcon from "../icons/Reading";

interface DeepReadeProps {
  Detail?: any;
  ChunkData?: IChunkData;
  loading?: boolean;
  t?: TFunction;
}

export const DeepRead = ({
  Detail,
  ChunkData,
  loading,
  t: tProp,
}: DeepReadeProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const [isThinkingExpanded, setIsThinkingExpanded] = useState(false);

  const [data, setData] = useState<string[]>([]);
  const [description, setDescription] = useState("");

  useEffect(() => {
    if (!Detail?.description) return;
    setDescription(Detail?.description);
  }, [Detail?.description]);

  useEffect(() => {
    if (!ChunkData?.message_chunk) return;
    try {
      if (ChunkData.message_chunk.includes("&")) {
        const contentArray = ChunkData.message_chunk.split("&").filter(Boolean);
        setData(contentArray);
      } else {
        setData([ChunkData.message_chunk]);
      }
    } catch (e) {
      console.error("Failed to parse query data:", e);
    }
  }, [ChunkData?.message_chunk]);

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
              {t(
                `assistant.message.steps.${
                  ChunkData?.chunk_type || Detail?.type
                }`
              )}
            </span>
          </>
        ) : (
          <>
            <ReadingIcon className="w-4 h-4 text-[#38C200]" />
            <span className="text-xs text-[#999999]">
              {t(
                `assistant.message.steps.${
                  ChunkData?.chunk_type || Detail?.type
                }`,
                {
                  count: Number(data.length),
                }
              )}
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
            <div className="mb-4 space-y-3 text-xs">
              {data?.map((item, index) => (
                <div key={`${item}-${index}`} className="flex flex-col gap-2">
                  <div className="text-xs text-[#999999] dark:text-[#808080]">
                    - {item}
                  </div>
                </div>
              ))}
              {description?.split("\n").map(
                (paragraph, idx) =>
                  paragraph.trim() && (
                    <p key={idx} className="text-sm">
                      {paragraph}
                    </p>
                  )
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
