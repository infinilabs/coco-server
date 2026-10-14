import { Button } from "antd";
import { useTranslation } from "react-i18next";
import loadingFailedSvg from "../icons/file-loading-failed.svg";

interface EmptyListProps {
    query?: string;
    settings?: Record<string, any>;
    variant?: "search" | "filtered";
    onClearFilters?: () => void;
    onGenerateAnswer?: () => void;
}

export const EmptyList = ({ query, settings, variant = "search", onClearFilters, onGenerateAnswer }: EmptyListProps) => {
    const { t } = useTranslation();
    const keyword = query?.trim();
    const isFiltered = variant === "filtered";

    return (
        <div className="px-6 mt-48px py-64px flex flex-col items-center justify-center text-center min-h-280px">
            <img className="mb-24px w-80px h-80px" src={loadingFailedSvg} alt="" />
            <div className="text-16px font-medium text-[#4B5563] dark:text-[#C4C9CF] max-w-[600px]">
                {isFiltered ? t('labels.emptyFilteredResult') : keyword ? t('labels.emptyResultWithQuery', { query: keyword }) : t('labels.emptyResult')}
            </div>
            {isFiltered ? (
                <Button className="mt-24px min-w-120px rounded-20px" variant="solid" color="primary" onClick={onClearFilters}>
                    {t('labels.clearFilters')}
                </Button>
            ) : settings?.deep_think_assistant_entity?.id && (
                <div className="mt-8px text-14px text-[#6B7280] dark:text-[#9CA3AF]">
                    <div>{t('labels.emptyResultAIOverviewTip')}</div>
                    <Button className="mt-24px min-w-120px rounded-20px" variant="solid" color="primary" onClick={onGenerateAnswer}>
                        {t('labels.generateAnswer')}
                    </Button>
                </div>
            )}
        </div>
    );
};

