import { Column } from '@ant-design/charts';
import { useTranslation } from 'react-i18next';
import { useCallback, useRef, useState, type PointerEvent as ReactPointerEvent } from 'react';

interface HistogramProps {
    readonly data?: { date: string; count: number }[];
    readonly theme?: string;
    readonly onCustomDateRangeChange?: (range: { start?: string; end?: string }) => void;
}

// Distance (px) below which a pointer gesture counts as a click (tooltip /
// hover) rather than a brush selection.
const BRUSH_MIN_DRAG = 5;

const Histogram = ({ data, theme, onCustomDateRangeChange }: HistogramProps) => {
    const { t } = useTranslation();
    const chartData = Array.isArray(data) ? data : [];
    const maxValue = chartData.length > 0 ? Math.max(...chartData.map((item) => Number(item.count) || 0)) * 1.4 : 0;

    const containerRef = useRef<HTMLDivElement | null>(null);
    // the @ant-design/charts plot wrapper (v2 passes the instance itself to
    // onReady; its `.chart` is the inner G2 chart used to read plot geometry)
    const chartRef = useRef<any>(null);
    const [brush, setBrush] = useState<{ left: number; width: number } | null>(null);

    // Translate a horizontal pixel span into the bucket keys it covers. The
    // bucket boundaries come from the chart's own plot-area bounds, so hidden
    // axes and the band padding are already accounted for.
    const bucketsFromPixels = useCallback(
        (x0: number, x1: number): { start?: string; end?: string } | null => {
            if (chartData.length === 0) return null;
            try {
                const plot = chartRef.current?.chart?.getContext?.()?.canvas?.document?.getElementsByClassName?.('plot')?.[0];
                const canvas = containerRef.current?.querySelector('canvas');
                if (!plot || !canvas) return null;
                const bounds = plot.getBounds();
                const rect = canvas.getBoundingClientRect();
                const [left, right] = [bounds.min[0], bounds.max[0]];
                const band = (right - left) / chartData.length;
                if (band <= 0) return null;
                // viewport px -> canvas px (canvas renders at CSS scale)
                const toBand = (x: number) => Math.min(chartData.length - 1, Math.max(0, Math.floor((x - rect.left - left) / band)));
                const i0 = toBand(Math.min(x0, x1));
                const i1 = toBand(Math.max(x0, x1));
                const start = chartData[i0]?.date;
                const end = chartData[i1]?.date;
                return start && end ? { start, end } : null;
            } catch {
                return null;
            }
        },
        [chartData]
    );

    const onPointerDown = useCallback(
        (event: ReactPointerEvent<HTMLDivElement>) => {
            if (event.button !== 0 || chartData.length === 0) return;
            const container = containerRef.current;
            if (!container) return;
            const originX = event.clientX;
            const rect = container.getBoundingClientRect();
            let latestX = originX;
            const onMove = (e: PointerEvent) => {
                latestX = e.clientX;
                const left = Math.max(rect.left, Math.min(originX, latestX));
                const right = Math.min(rect.right, Math.max(originX, latestX));
                setBrush({ left: left - rect.left, width: Math.max(0, right - left) });
            };
            const onUp = () => {
                container.removeEventListener('pointermove', onMove);
                container.removeEventListener('pointerup', onUp);
                container.removeEventListener('pointercancel', onUp);
                setBrush(null);
                if (Math.abs(latestX - originX) >= BRUSH_MIN_DRAG) {
                    const range = bucketsFromPixels(originX, latestX);
                    if (range) onCustomDateRangeChange?.(range);
                }
            };
            try {
                // capture keeps the gesture alive when the pointer leaves the
                // chart; synthetic test events carry pointer ids the browser
                // rejects, which is harmless — the listeners below still run
                container.setPointerCapture(event.pointerId);
            } catch {
                // ignore
            }
            container.addEventListener('pointermove', onMove);
            container.addEventListener('pointerup', onUp);
            container.addEventListener('pointercancel', onUp);
        },
        [bucketsFromPixels, onCustomDateRangeChange, chartData.length]
    );

    const config = {
        animation: false,
        data: chartData,
        xField: 'date',
        yField: 'count',
        autoFit: true,
        padding: 0,
        margin: 0,
        style: {
            maxWidth: 16
        },
        scale: {
            y: {
                nice: false,
                domain: [0, maxValue || 1],
            },
            x: {
                padding: 0.5
            },
        },
        axis: {
            x: false,
            y: {
                tickCount: 5,
                grid: true,
                gridLineWidth: 1,
                gridLineDash: [0, 0],
                gridStroke: theme === 'dark' ? '#303030' : '#F0F0F0',
                gridStrokeOpacity: 1,
                line: false,
                tick: false,
                label: false,
                title: false,
            },
        },
        theme: {
            type: theme === 'dark' ? 'dark' : 'light',
        },
        animate: false,
        tooltip: {
            items: [
                (_datum: any, index: number, _data: any, column: any) => ({
                    name: t('labels.count'),
                    value: column.y.value[index],
                }),
            ],
        },
        onReady: (chart: any) => {
            chartRef.current = chart;
        },
    };

    return (
        <div className="w-full h-86px px-8px bg-[#FAFAFA] dark:bg-[#1F1F1F] rounded-4px border border-[#F0F0F0] dark:border-[#303030]">
            <div
                className="relative w-full h-full cursor-crosshair"
                ref={containerRef}
                style={{ touchAction: 'none' }}
                onPointerDown={onPointerDown}
            >
                <Column {...config} />
                {brush ? (
                    <div
                        className="absolute top-0 bottom-0 bg-[#777] opacity-30 pointer-events-none"
                        style={{ left: brush.left, width: brush.width }}
                    />
                ) : null}
            </div>
        </div>
    );
};

export default Histogram;
