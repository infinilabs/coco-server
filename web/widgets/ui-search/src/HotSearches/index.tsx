import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';
import { RotateCw, Flame } from "lucide-react";

interface HotSearchRow {
    query: string;
    count: number;
    last_seen?: string;
}

interface HotSearchesProps {
    /** host hook: GET /query/_hot — top search-log queries (搜索设置 · 启用热搜 gates server-side) */
    onHotSearches?: (callback: (res: { items?: HotSearchRow[] }) => void) => void;
    /** click a hot term → run it as a search */
    onPick?: (query: string) => void;
}

const CAPSULE_H = 30;   // h-[30px]
const GAP_Y = 5;        // gap-y-5px → row pitch 35px

// hot-queries list under the home search box: rank + term + hit count.
// renders nothing when the endpoint is disabled (empty items) or fails.
// one page = however many capsules fill the first two rows of the flow layout
// (width-adaptive, not a fixed count); more than one page → a refresh icon in
// the section header pages through them (wraps back to the first page).
export default function HotSearches(props: HotSearchesProps) {
    const { onHotSearches, onPick } = props;
    const [items, setItems] = useState<HotSearchRow[]>([]);
    const [page, setPage] = useState(0);
    const [pageSize, setPageSize] = useState<number | null>(null);
    const { t } = useTranslation();
    const hasFetched = useRef(false);
    const measureRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (hasFetched.current || !onHotSearches) return;
        hasFetched.current = true;
        onHotSearches(res => {
            if (Array.isArray(res?.items)) {
                setItems(res.items.filter(row => row?.query));
            }
        });
    }, []);

    // page size = capsules inside the first two visual rows of the hidden
    // measure flow (same width as the real one), re-measured on resize
    useLayoutEffect(() => {
        if (items.length === 0) return;
        const measure = () => {
            const el = measureRef.current;
            if (!el) return;
            const base = el.getBoundingClientRect().top;
            let count = 0;
            for (const cap of Array.from(el.children) as HTMLElement[]) {
                // rows 0 & 1 have tops 0 / CAPSULE_H + GAP_Y
                if (cap.getBoundingClientRect().top - base <= CAPSULE_H + GAP_Y + 1) count++;
                else break;
            }
            if (count > 0) setPageSize(count);
        };
        measure();
        window.addEventListener('resize', measure);
        return () => window.removeEventListener('resize', measure);
    }, [items]);

    if (items.length === 0) return null;

    // first paint falls back to 10; useLayoutEffect measures before paint so
    // the fallback is never visible
    const size = Math.max(pageSize ?? 10, 1);
    const pageCount = Math.ceil(items.length / size);
    const safePage = Math.min(page, pageCount - 1);
    const visible = items.slice(safePage * size, safePage * size + size);

    return (
        <div className="recommends-container mt-24px">
            <div className="flex items-center h-[24px] mb-[10px]">
                <div className="flex items-center gap-6px text-14px text-[rgba(25,25,26,1)] dark:text-[rgba(230,230,230,1)]">
                    <Flame className="w-14px h-14px shrink-0" />
                    {t('labels.hotSearches')}
                    {pageCount > 1 && (
                        <RotateCw
                            className="w-14px h-14px ml-6px cursor-pointer text-[rgba(25,25,26,0.45)] dark:text-[rgba(230,230,230,0.45)] hover:text-[rgba(25,25,26,0.8)] dark:hover:text-[rgba(230,230,230,0.8)]"
                            onClick={() => setPage((safePage + 1) % pageCount)}
                        />
                    )}
                </div>
            </div>
            {/* tag-style flow: capsule width follows its content, wraps like chips */}
            <div className="flex flex-wrap gap-x-6px gap-y-5px">
                {visible.map((row, index) => {
                    // rank is global (across pages), so the red dots stay with the overall top 3
                    const rank = safePage * size + index;
                    return (
                    <div
                        key={`${row.query}-${index}`}
                        title={row.query}
                        className="bg-[rgba(241,241,241,1)] dark:bg-[rgba(55,55,55,1)] text-[#666666] dark:text-[rgba(230,230,230,1)] h-[30px] rounded-[15px] px-[12px] text-14px
                            inline-flex items-center cursor-pointer max-w-full overflow-hidden whitespace-nowrap gap-6px"
                        onClick={() => onPick?.(row.query)}
                    >
                        {/* rank dot: overall top 3 red, rest gray (same palette as the old numbers) */}
                        <span className={`w-6px h-6px rounded-full shrink-0 ${rank < 3 ? 'bg-[#f5222d] dark:bg-[#ff7875]' : 'bg-[rgba(25,25,26,0.45)] dark:bg-[rgba(230,230,230,0.45)]'}`} />
                        <span className="min-w-0 overflow-hidden text-ellipsis">{row.query}</span>
                        <span className="text-12px text-[rgba(25,25,26,0.35)] dark:text-[rgba(230,230,230,0.35)] shrink-0">{row.count}</span>
                    </div>
                    );
                })}
            </div>
            {/* hidden twin flow holding ALL items: used to measure how many
                capsules fill two rows at the current width; h-0 +
                overflow-hidden keeps it visually absent */}
            <div
                ref={measureRef}
                aria-hidden
                className="h-0 overflow-hidden flex flex-wrap gap-x-6px gap-y-5px"
            >
                {items.map((row, index) => (
                    <div
                        key={index}
                        className="bg-white text-[#666666] h-[30px] rounded-[15px] px-[12px] text-14px inline-flex items-center whitespace-nowrap gap-6px shrink-0"
                    >
                        <span className="w-6px h-6px rounded-full shrink-0" />
                        <span>{row.query}</span>
                        <span className="text-12px">{row.count}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}
