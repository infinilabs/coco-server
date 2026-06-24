import { Column } from '@ant-design/charts';
import { useTranslation } from 'react-i18next';

interface HistogramProps {
    data?: { date: string; count: number }[];
    theme?: string;
    onCustomDateRangeChange?: (range: { start?: string; end?: string }) => void;
}

const Histogram = ({ data, theme, onCustomDateRangeChange }: HistogramProps) => {
    const { t } = useTranslation();
    const chartData = Array.isArray(data) ? data : [];
    const maxValue = chartData.length > 0 ? Math.max(...chartData.map((item) => Number(item.count) || 0)) * 1.4 : 0;

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
        interaction: {
            brushXHighlight: true,
        },
        theme: {
            type: theme === 'dark' ? 'dark' : 'light',
        },
        animate: false,
        onReady: ({ chart }: { chart: any }) => {
            chart.on('brush:end', (event: any) => {
                const selection = event.data?.selection?.[0];
                if (Array.isArray(selection) && selection.length >= 2) {
                    const start = selection[0];
                    const end = selection[selection.length - 1];
                    if (start && end) {
                        onCustomDateRangeChange?.({ start, end });
                    }
                }
                chart.emit('brush:remove', { nativeEvent: false });
            });
        },
        tooltip: {
            items: [
                (datum: any, index: number, data: any, column: any) => ({
                    name: t('labels.count'), 
                    value: column.y.value[index],
                }),
            ],
        }
    };

    return (
        <div className="w-full h-86px px-8px bg-[#FAFAFA] dark:bg-[#1F1F1F] rounded-4px border border-[#F0F0F0] dark:border-[#303030]">
            <div className="w-full h-full">
                <Column {...config} />
            </div>
        </div>
    );
};

export default Histogram;