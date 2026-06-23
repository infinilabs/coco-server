import { Button } from "antd";
import { useTranslation } from "react-i18next";

interface EndListProps {
    total?: number;
    settings?: Record<string, any>;
    onGenerateAnswer?: () => void;
    padded?: boolean;
}

export const EndList = ({ total = 0, settings, onGenerateAnswer, padded = true }: EndListProps) => {
    const { t } = useTranslation();

    return (
        <div className={`w-full pt-32px pb-8px ${padded ? 'px-16px' : ''} flex flex-col items-center text-center`}>
            <div className="relative w-full flex items-center justify-center">
                <div className="absolute inset-x-0 top-1/2 h-1px bg-[#e8e8e8] dark:bg-white/10" />
                <div className="relative min-w-200px h-28px px-32px rounded-16px border border-[#F0F0F0] dark:border-[#303030] bg-[rgb(var(--ui-search--layout-bg-color))] flex items-center justify-center text-14px text-[#666] dark:text-white/80">
                    {t('labels.endListReached')}
                </div>
            </div>
            <div className="mt-24px text-14px text-[#999] dark:text-[#666]">
                {t('labels.endListTip', { count: total })}
            </div>
            {settings?.deep_think_assistant_entity?.id && (
                <Button className="mt-28px min-w-120px h-38px rounded-20px font-500" variant="solid" color="primary" onClick={onGenerateAnswer}>
                    {t('labels.askAI')}
                </Button>
            )}
        </div>
    );
};

