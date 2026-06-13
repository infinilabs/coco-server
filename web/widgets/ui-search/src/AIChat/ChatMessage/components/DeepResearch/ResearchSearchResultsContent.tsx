import { useMemo } from "react";
import SearchResults, {
  type SearchResultsRecord,
} from "../../../../ResultList/SearchResults";
import type { StepSearchHit } from "./ResearchStepsContent";

interface ResearchSearchResultsContentProps {
  hits?: StepSearchHit[];
  theme?: "light" | "dark";
}

export const ResearchSearchResultsContent = ({
  hits,
  theme,
}: ResearchSearchResultsContentProps) => {
  const records = useMemo(() => {
    if (!hits || !Array.isArray(hits)) return [];

    const uniqueHitsMap = new Map<string, StepSearchHit>();
    
    for (const hit of hits) {
      if (!hit || typeof hit !== "object") continue;
      const title = typeof hit.title === "string" ? hit.title : "";
      if (!title) continue;
      
      // Deduplicate by title
      if (!uniqueHitsMap.has(title)) {
        uniqueHitsMap.set(title, hit);
      }
    }

    const result: SearchResultsRecord[] = [];
    
    uniqueHitsMap.forEach((hit) => {
      const title = typeof hit.title === "string" ? hit.title : "";
      const url = typeof hit.url === "string" ? hit.url : undefined;
      const content = typeof hit.content === "string" ? hit.content : undefined;
      
      const record: SearchResultsRecord = {
        title,
        summary: content,
        url,
        source:
          typeof hit.source === "string"
            ? { name: hit.source }
            : undefined,
        metadata:
          typeof hit.score === "number"
            ? { score: hit.score }
            : undefined,
      };
      
      result.push(record);
    });

    return result;
  }, [hits]);

  return (
    <div className="">
      <SearchResults
        section={records}
        theme={theme || "light"}
        hideHeader
        onRecordClick={(record) => {
          if (typeof record.url === "string") window.open(record.url, "_blank");
        }}
        className="[&>div>*+*]:relative [&>div>*+*]:before:content-[''] [&>div>*+*]:before:absolute [&>div>*+*]:before:top-0 [&>div>*+*]:before:left-4 [&>div>*+*]:before:right-4 [&>div>*+*]:before:h-px [&>div>*+*]:before:bg-[#F0F0F0] dark:[&>div>*+*]:before:bg-[#303030]"
      />
    </div>
  );
};
