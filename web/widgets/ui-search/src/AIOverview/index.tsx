import { AIAnswer } from './AIAnswer';
import { useTranslation } from 'react-i18next';

interface AIOverviewConfig {
    title?: string;
    height?: string | number;
    logo?: {
        light?: string;
    };
    [key: string]: unknown;
}

interface AIOverviewData {
    response?: {
        message_chunk?: string;
    };
    [key: string]: unknown;
}

interface AIOverviewProps {
    config?: AIOverviewConfig;
    data?: AIOverviewData;
    loading?: boolean;
    visible?: boolean;
    setVisible?: (visible: boolean) => void;
    theme?: "light" | "dark" | "auto";
    onChatContinue?: () => void;
    isReplyEnd?: boolean;
    requestHeaders?: Record<string, string>;
}

const AIOverview = (props: AIOverviewProps) => {
    const { config = {}, data, loading, visible, theme, onChatContinue, isReplyEnd, requestHeaders } = props;
    const { t } = useTranslation();

    if (!visible) return null;

    return (
        <AIAnswer
            title={config?.title || t("labels.aiOverview")}
            content={data?.response?.message_chunk || ""}
            logo={config?.logo?.light}
            onContinue={() => onChatContinue?.()}
            maxHeight={Number.isInteger(Number(config.height)) ? Number(config.height) : undefined}
            theme={theme}
            containerClass="!border-0 px-16px !pt-16px !pb-16px"
            continueLabel={t("labels.continueAsk")}
            expandText={t("labels.expandMore")}
            collapseText={t("labels.collapse")}
            loading={loading}
            isReplyEnd={isReplyEnd}
            requestHeaders={requestHeaders}
        />
    );
};

export default AIOverview;
