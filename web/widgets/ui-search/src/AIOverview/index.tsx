import { AIAnswer } from './AIAnswer';
import { useTranslation } from 'react-i18next';

interface AIOverviewConfig {
    height?: string | number;
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
}

const AIOverview = (props: AIOverviewProps) => {
    const { config = {}, data, loading, visible, theme, onChatContinue, isReplyEnd } = props;
    const { t } = useTranslation();

    if (!visible) return null;

    return (
        <AIAnswer
            title={t("labels.aiOverview")}
            content={data?.response?.message_chunk || ""}
            onContinue={() => onChatContinue?.()}
            maxHeight={Number.isInteger(Number(config.height)) ? Number(config.height) : undefined}
            theme={theme}
            containerClass="!border-0 px-16px !pt-16px !pb-16px"
            continueLabel={t("labels.continueAsk")}
            expandText={t("labels.expandMore")}
            collapseText={t("labels.collapse")}
            loading={loading}
            isReplyEnd={isReplyEnd}
        />
    );
};

export default AIOverview;
