import { useEffect, useRef, useState } from "react";
import { useTranslation } from 'react-i18next';
import { Search } from "lucide-react";

interface Recommendation {
    title: string;
    url?: string;
    score: number;
}

interface RecommendsProps {
    onRecommend?: (callback: (res: { recommendations?: Recommendation[] }) => void) => void;
    showTitle?: boolean;
    onDataLoaded?: (hasData: boolean) => void;
    /** click a recommendation → run it as a search (overrides the url window.open fallback) */
    onPick?: (title: string) => void;
    /** cap visible pills; more topics than this rotate automatically (home page uses 3) */
    visibleCount?: number;
    /** rotation interval in ms (default 8000); rotation pauses while hovered */
    rotateMs?: number;
}

const ROTATE_MS = 8000;
// pill geometry for the sliding track: h-40px pills, 8px gaps
const PILL_H = 40;
const GAP = 8;
const STEP = PILL_H + GAP;

export default function Recommends(props: RecommendsProps) {
    const { onRecommend, showTitle = false, onDataLoaded, onPick, visibleCount, rotateMs = ROTATE_MS } = props;
    const [data, setData] = useState<Recommendation[]>([]);
    // single-item-step continuous carousel: pos is the index of the pill at the
    // top of the viewport; it may transiently equal data.length (the cloned
    // head) before the seamless snap-back resets it to 0 — always slides up
    const [pos, setPos] = useState(0);
    const [instant, setInstant] = useState(false);
    const [paused, setPaused] = useState(false);
    const { t } = useTranslation();
    const hasFetched = useRef(false);

    useEffect(() => {
        if (hasFetched.current) return;
        hasFetched.current = true;
        onRecommend?.(res => {
            if (Array.isArray(res?.recommendations)) {
                const sorted = res.recommendations.sort((a, b) => b.score - a.score);
                setData(sorted);
                onDataLoaded?.(sorted.length > 0);
            } else {
                onDataLoaded?.(false);
            }
        });
    }, []);

    const rotates = Boolean(visibleCount) && data.length > (visibleCount ?? 0);
    const viewportH = visibleCount ? visibleCount! * PILL_H + (visibleCount! - 1) * GAP : 0;

    // tick: advance one pill; hover pauses so a pill the user is about to
    // click never rotates away under the cursor
    useEffect(() => {
        if (!rotates || paused) return;
        const id = setInterval(() => setPos(p => p + 1), rotateMs);
        return () => clearInterval(id);
    }, [rotates, paused, rotateMs]);

    // seamless loop: after sliding onto the cloned head (pos === data.length
    // looks identical to pos 0), kill the transition, jump back to 0, then
    // re-enable the transition two frames later so the jump never paints
    useEffect(() => {
        if (!rotates || pos < data.length) return;
        const t = setTimeout(() => {
            setInstant(true);
            setPos(0);
        }, 650);
        return () => clearTimeout(t);
    }, [pos, data.length, rotates]);

    useEffect(() => {
        if (!instant) return;
        const id = requestAnimationFrame(() => requestAnimationFrame(() => setInstant(false)));
        return () => cancelAnimationFrame(id);
    }, [instant]);

    if (data.length === 0) return null;

    // clone the head onto the tail so the wrap-around position shows the same
    // pills as the start — the snap back to 0 is pixel-identical
    const trackItems = rotates ? [...data, ...data.slice(0, visibleCount!)] : data;
    const offset = Math.min(pos, data.length);

    const renderPill = (item: Recommendation, index: number) => (
        <div key={index}>
            <div
                title={item.title}
                className="bg-[rgba(241,241,241,1)] dark:bg-[rgba(55,55,55,1)] text-[#666666] dark:text-[rgba(230,230,230,1)] leading-[40px] h-[40px] px-[12px] rounded-[12px]
                    inline-flex items-center align-middle cursor-pointer max-w-full overflow-hidden whitespace-nowrap gap-6px"
                onClick={() => {
                    if (onPick) {
                        onPick(item.title)
                    } else if (item.url?.startsWith('http')) {
                        window.open(item.url)
                    }
                }}
            >
                <Search className="w-14px h-14px shrink-0 opacity-60" />
                <span className="min-w-0 overflow-hidden text-ellipsis">{item.title}</span>
            </div>
        </div>
    );

    return (
        <div className="recommends-container" onMouseEnter={() => setPaused(true)} onMouseLeave={() => setPaused(false)}>
            {
                showTitle && (
                    <div className="text-16px text-[rgba(25,25,26,1)] dark:text-[rgba(230,230,230,1)] mb-16px">{t('labels.relatedSearch')}</div>
                )
            }
            {rotates ? (
                <div className="overflow-hidden" style={{ height: viewportH }}>
                    <div
                        style={{
                            transform: `translateY(-${offset * STEP}px)`,
                            transition: instant ? 'none' : 'transform 600ms cubic-bezier(0.22, 1, 0.36, 1)'
                        }}
                    >
                        <div className="flex flex-col" style={{ gap: GAP }}>
                            {trackItems.map((item, index) => renderPill(item, index))}
                        </div>
                    </div>
                </div>
            ) : (
                <div className="flex flex-col" style={{ gap: GAP }}>
                    {data.map((item, index) => renderPill(item, index))}
                </div>
            )}
        </div>
    );
}
