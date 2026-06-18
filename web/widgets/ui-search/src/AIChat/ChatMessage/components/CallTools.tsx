import { ChevronDown, ChevronRight, Hammer, Loader } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import type { IChunkData } from "../types/chat";
import { ExpandText } from "./ExpandText";

interface CallToolsProps {
  readonly Detail?: any;
  readonly ChunkData?: IChunkData;
  readonly loading?: boolean;
  readonly t?: TFunction;
}

export const CallTools = ({ Detail, ChunkData, loading, t: tProp }: CallToolsProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [isThinkingExpanded, setIsThinkingExpanded] = useState(false);

  const [data, setData] = useState<{ arguments: string; name: string; result: string}[]>([]);

  useEffect(() => {
    if (!Detail?.payload) return;
    setData(Detail.payload);
  }, [Detail?.payload]);

  useEffect(() => {
    if (!ChunkData?.tool_call_message_chunk) return;
    try {
      const parsed = JSON.parse(ChunkData.tool_call_message_chunk);
      setData((prev) => [...prev, parsed]);
    } catch {
      
    }
  }, [ChunkData?.tool_call_message_chunk]);

  if (!ChunkData && !Detail) return null;

  const renderContent = (text: string) => {
    try {
      const parsed = JSON.parse(text);
      if (typeof parsed === 'object' && parsed !== null) {
        return <ExpandText content={JSON.stringify(parsed, null, 2)} variant="json" />;
      }
      return <ExpandText content={text} />;
    } catch {
      return <ExpandText content={text} />;
    }
  };

  return (
    <div className="space-y-2 mb-8px w-full">
      <button
        onClick={() => setIsThinkingExpanded((prev) => !prev)}
        className="text-[12px] text-[#999] dark:text-[#666] cursor-pointer bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] inline-flex items-center gap-2 px-2 py-2px rounded-12px transition-colors border border-solid border-[#F0F0F0] dark:border-[#303030]"
      >
        <>
          {loading ? (
            <Loader className="w-14px h-14px animate-spin text-[#1784FC] shrink-0" />
          ) : (
            <Hammer className="w-14px h-14px shrink-0" />
          )}
          <span>
            {t(
              `assistant.message.steps.${ChunkData?.chunk_type || Detail.type}`,
              {
                count: Number(data.length || 0),
              }
            )}
          </span>
        </>
        {isThinkingExpanded ? (
          <ChevronDown className="w-14px h-14px" />
        ) : (
          <ChevronRight className="w-14px h-14px" />
        )}
      </button>
      {isThinkingExpanded && data.length > 0 && (
        <div className="ml-8px pl-8px border-l-1 border-[#F0F0F0] dark:border-[#303030]">
          <div className="space-y-8px">
            {
              data.map((item, index) => (
                <div key={index} className="text-[#333] dark:text-[#E5E7EB] text-12px rounded-8px border border-[#F0F0F0] dark:border-[#303030] p-12px">
                  <div className="mb-12px font-semibold">{item.name}</div>
                  <div className="mb-4px text-[#999] dark:text-[#666]">{t('labels.arguments')}</div>
                  <div className="pb-8px mb-8px border-b border-[#F0F0F0] dark:border-[#303030]">
                    {renderContent(item.arguments)}
                  </div>
                  <div className="mb-4px text-[#999] dark:text-[#666]">{t('labels.result')}</div>
                  <div className="">
                    {renderContent(item.result)}
                  </div>
                </div>
              ))
            }
          </div>
        </div>
      )}
    </div>
  );
};
