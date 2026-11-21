import { History, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useState, type FC, type MouseEvent } from "react";
import { loadRecentSearches, removeRecentSearch } from "../../utils/recentSearches";

interface RecentSearchesProps {
  onSelect: (query: string) => void;
}

// Local, most-recent-first list of executed queries shown above the tips when
// the search box is focused empty. Renders nothing until the first search has
// been recorded, so the panel keeps its original look on a fresh install.
const RecentSearches: FC<RecentSearchesProps> = ({ onSelect }) => {
  const { t } = useTranslation();
  const [items, setItems] = useState<string[]>(() => loadRecentSearches());

  if (items.length === 0) return null;

  const handleRemove = (event: MouseEvent, query: string) => {
    event.stopPropagation();
    removeRecentSearch(query);
    setItems(loadRecentSearches());
  };

  return (
    <div>
      <div className="py-14px px-12px text-12px text-[var(--ant-color-text-description)]">
        {t("labels.recentSearches")}
      </div>
      <div className="px-4px mb-12px">
        {items.map((query) => (
          <div
            key={query}
            className="group relative h-40px pl-8px pr-32px flex flex-nowrap items-center rounded-8px cursor-pointer hover:bg-[rgba(233,240,254,1)] dark:hover:bg-[rgba(255,255,255,0.05)]"
            onClick={() => onSelect(query)}
          >
            <History className="w-16px h-16px mr-8px flex-shrink-0 text-[var(--ant-color-text-description)]" />
            <div className="flex-1 min-w-0 leading-22px truncate whitespace-nowrap">{query}</div>
            <button
              type="button"
              aria-label={t("labels.removeRecentSearch")}
              className="absolute right-8px top-8px w-24px h-24px flex items-center justify-center rounded-8px border-0 bg-transparent p-0 cursor-pointer text-[var(--ant-color-text-description)] opacity-0 group-hover:opacity-100 hover:bg-[rgba(0,0,0,0.06)] dark:hover:bg-[rgba(255,255,255,0.1)]"
              onClick={(event) => handleRemove(event, query)}
            >
              <X className="w-14px h-14px" />
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

export default RecentSearches;
