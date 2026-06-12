import {
  ChevronDown,
  SquareArrowOutUpRight,
  Globe,
  ChevronRight,
} from "lucide-react";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";

import { OpenURLWithBrowser } from "../utils/index";
import type { IChunkData } from "../types/chat";
import CheckIcon from "../../../icons/CheckIcon";

interface FetchSourceProps {
  Detail?: any;
  ChunkData?: IChunkData;
  loading?: boolean;
  formatUrl?: (data: ISourceData) => string;
  t?: TFunction;
}

interface ISourceData {
  category: string;
  icon: string;
  id: string;
  size: number;
  source: {
    type: string;
    name: string;
    id: string;
  };
  summary: string;
  thumbnail: string;
  title: string;
  updated: string | null;
  url: string;
}

export const FetchSource = ({
  Detail,
  ChunkData,
  loading,
  formatUrl,
  t: tProp,
}: FetchSourceProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  const [isSourceExpanded, setIsSourceExpanded] = useState(false);

  const [total, setTotal] = useState(0);
  const [data, setData] = useState<ISourceData[]>([]);

  useEffect(() => {
    if (!Detail?.payload) return;
    setData(Detail?.payload);
    setTotal(Detail?.payload.length);
  }, [Detail?.payload]);

  useEffect(() => {
    if (!ChunkData?.message_chunk) return;

    if (!loading) {
      try {
        const match = ChunkData.message_chunk.match(
          // /\u003cPayload total=(\d+)\u003e/
          /<Payload total=(\d+)>/
        );
        if (match) {
          setTotal(Number(match[1]));
        }

        // const jsonMatch = ChunkData.message_chunk.match(/\[(.*)\]/s);
        const jsonMatch = ChunkData.message_chunk.match(/\[([\s\S]*)\]/);
        if (jsonMatch) {
          const jsonData = JSON.parse(jsonMatch[0]);
          setData(jsonData);
        }
      } catch (e) {
        console.error("Failed to parse fetch source data:", e);
      }
    }
  }, [ChunkData?.message_chunk, loading]);

  const sourceClick = (item: ISourceData) => () => {
    const url = (formatUrl && formatUrl(item)) || item.url;
    if (url) {
      OpenURLWithBrowser(url);
    }
  };

  // Must be after hooks !!!
  if (!ChunkData && !Detail) return null;

  return (
    <div
      className={`mb-8px max-w-full w-full md:w-[610px] ${
        isSourceExpanded
          ? "rounded-12px overflow-hidden border border-solid border-[#F0F0F0] dark:border-[#303030]"
          : ""
      }`}
    >
      <button
        onClick={() => setIsSourceExpanded((prev) => !prev)}
        className={`text-[#101010] dark:text-[#F5F5F5] bg-transparent hover:bg-[#EDEDED] dark:hover:bg-[#3A3A3A] cursor-pointer inline-flex justify-between items-center gap-2 px-2 py-2px transition-colors whitespace-nowrap ${
          isSourceExpanded
            ? "w-full"
            : "rounded-12px border border-solid border-[#F0F0F0] dark:border-[#303030]"
        }`}
      >
        <div className="flex-1 min-w-0 flex items-center gap-2">
          <CheckIcon className="w-14px h-14px shrink-0" />
          <span>
            {t(
              `assistant.message.steps.${ChunkData?.chunk_type || Detail.type}`,
              {
                count: Number(total),
              }
            )}
          </span>
        </div>
        {isSourceExpanded ? (
          <ChevronDown className="w-4 h-4" />
        ) : (
          <ChevronRight className="w-4 h-4" />
        )}
      </button>

      {isSourceExpanded && data?.length > 0 && (
        <>
          {data?.map((item, idx) => (
            <div
              key={idx}
              onClick={sourceClick(item)}
              className="group flex items-center p-2 hover:bg-[#F7F7F7] dark:hover:bg-[#2C2C2C] border-b border-[#F0F0F0] dark:border-[#303030] last:border-b-0 cursor-pointer transition-colors"
            >
              <div className="w-full flex items-center gap-2">
                <div className="w-[75%] mobile:w-full flex items-center gap-1">
                  <Globe className="w-3 h-3 shrink-0" />
                  <div className="text-xs text-[#333333] dark:text-[#D8D8D8] truncate font-normal group-hover:text-[#0072FF] dark:group-hover:text-[#0072FF]">
                    {item.title || item.category}
                  </div>
                </div>
                <div
                  className={`flex-1 mobile:hidden flex items-center justify-end gap-2`}
                >
                  <span className="text-xs text-[#999999] dark:text-[#999999] truncate">
                    {item.source?.name || item?.category}
                  </span>
                  <SquareArrowOutUpRight className="w-3 h-3 text-[#999999] dark:text-[#999999] shrink-0" />
                </div>
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );
};
