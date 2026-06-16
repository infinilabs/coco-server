import { ChevronDown, ChevronRight, Loader } from "lucide-react";
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
    }
  }, [ChunkData?.message_chunk]);
console.log("chunk data", ChunkData);
console.log("Detail data", Detail);
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
              {t(
                `assistant.message.steps.${
                  ChunkData?.chunk_type || Detail?.type
                }`
              )}
            </span>
          </>
        ) : (
          <>
            <ReadingIcon size={14} className="shrink-0" />
            <span className="">
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
          <ChevronDown className="w-14px h-14px" />
        ) : (
          <ChevronRight className="w-14px h-14px" />
        )}
      </button>
      {isThinkingExpanded && (data?.length > 0 || description) && (
        <div className="ml-8px pl-8px border-l-1 border-[#F0F0F0] dark:border-[#303030]">
          <div className="space-y-2">
            <div className="mb-4 space-y-3 text-xs">
              {data?.map((item, index) => (
                <div key={`${item}-${index}`} className="flex flex-col gap-2">
                  <div className="text-xs text-[#999] dark:text-[#666]">
                    - {item}
                  </div>
                </div>
              ))}
              {description?.split("\n").map(
                (paragraph, idx) =>
                  paragraph.trim() && (
                    <p key={idx} className="text-xs text-[#999] dark:text-[#666]">
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
