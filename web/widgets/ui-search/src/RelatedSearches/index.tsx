import { useEffect, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';
import { Search } from "lucide-react";

interface RelatedRow {
    query: string;
    count: number;
}

interface RelatedSearchesProps {
    /** the active search query — related searches are conditioned on it */
    query?: string;
    /** host hook: GET /query/_related?q= — search-log lexical neighbors */
    onRelatedSearches?: (query: string, callback: (res: { items?: RelatedRow[] }) => void) => void;
    /** click a related query → run it as a search */
    onPick?: (query: string) => void;
    /** host visibility hook: whether the sidebar column should render */
    onDataLoaded?: (hasData: boolean) => void;
}

// result-page sidebar "related searches": real query-conditioned suggestions
// from search-log telemetry (方案 B · 词面关联). Renders nothing while the
// endpoint has no lexical neighbors for the current query — the operator's
// static recommend topics are a separate feature (home page only).
export default function RelatedSearches(props: RelatedSearchesProps) {
    const { query, onRelatedSearches, onPick, onDataLoaded } = props;
    const [items, setItems] = useState<RelatedRow[]>([]);
    const { t } = useTranslation();
    const dataLoadedRef = useRef(onDataLoaded);
    dataLoadedRef.current = onDataLoaded;

    useEffect(() => {
        const q = (query || '').trim();
        if (!q || !onRelatedSearches) {
            setItems([]);
            dataLoadedRef.current?.(false);
            return;
        }
        let cancelled = false;
        onRelatedSearches(q, res => {
            if (cancelled) return;
            const rows = Array.isArray(res?.items) ? res.items.filter(row => row?.query && row.query.toLowerCase() !== q.toLowerCase()) : [];
            setItems(rows);
            dataLoadedRef.current?.(rows.length > 0);
        });
        return () => { cancelled = true; };
    }, [query, onRelatedSearches]);

    if (items.length === 0) return null;

    return (
        <div className="related-searches-container">
            <div className="text-16px text-[rgba(25,25,26,1)] dark:text-[rgba(230,230,230,1)] mb-16px">{t('labels.relatedSearch')}</div>
            <div className="flex flex-col gap-8px">
                {items.map((row, index) => (
                    <div
                        key={`${row.query}-${index}`}
                        title={row.query}
                        className="bg-[rgba(241,241,241,1)] dark:bg-[rgba(55,55,55,1)] text-[#666666] dark:text-[rgba(230,230,230,1)] leading-[36px] h-[36px] px-[12px] rounded-[12px]
                            flex items-center cursor-pointer max-w-full overflow-hidden whitespace-nowrap gap-8px
                            hover:bg-[rgba(25,25,26,0.06)] dark:hover:bg-[rgba(55,55,55,0.9)]"
                        onClick={() => onPick?.(row.query)}
                    >
                        <Search className="w-14px h-14px shrink-0 opacity-60" />
                        <span className="min-w-0 overflow-hidden text-ellipsis">{row.query}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}
