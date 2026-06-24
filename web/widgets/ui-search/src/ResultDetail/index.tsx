import { Tooltip } from "antd";
import { SquareArrowOutUpRight, X } from "lucide-react";
import { ActionButton, DocDetail } from "./DocDetail";
import { type FC, useMemo } from "react";
import CommonDrawer from "../Layout/CommonDrawer";
import { useTranslation } from "react-i18next";
import { filesize } from 'filesize';
import dayjs from "dayjs";
import { isString } from "lodash";
import { formatDate } from "../utils/date";

interface ResultDetailProps {
    getContainer?: () => HTMLElement;
    data?: Record<string, any>;
    isMobile?: boolean;
    open?: boolean;
    onClose?: () => void;
    apiConfig?: Record<string, any>;
    theme?: "light" | "dark" | "auto";
}

export function DateTime(props: { value: string | number, showTooltip?: boolean }) {
    const { value, showTooltip = true } = props;
    if (!value || !dayjs(value).isValid()) return "-"
    
    const formatValue = formatDate(value)

    if (showTooltip) {
        return (
            <Tooltip title={isString(value) ? value : formatDate(value)}>
                {formatValue}
            </Tooltip>
        )
    }

    return formatValue
}

export const ResultDetail: FC<ResultDetailProps> = (props) => {
    const { getContainer, data = {}, isMobile, open, onClose, apiConfig, theme } = props;
    const { t } = useTranslation();
    const detailData = useMemo<Record<string, any>>(() => ({
        ...(data || {}),
        size: filesize(data?.size ?? 0),
        created: data?.created ? (
            <DateTime
                showTooltip={false}
                value={data?.created}
            />
        ) : null,
        updated: data?.updated ? (
            <DateTime
                showTooltip={false}
                value={data?.updated}
            />
        ) : null
    }), [data]);

    return (
        <CommonDrawer
            placement="right"
            onClose={onClose}
            open={open}
            size={isMobile ? undefined : 800}
            getContainer={getContainer}
            destroyOnHidden
            clickOutsideToClose={isMobile ? true : false}
            classNames={{
                wrapper: `${isMobile ? '!left-0px !right-0px !w-full !top-122px !bottom-0px' : '!right-24px !top-146px !bottom-24px'}`,
                body: '!p-24px !overflow-hidden !h-full'
            }}
        >
            <X className="color-[#bbb] cursor-pointer absolute right-24px top-24px z-1" onClick={onClose} />
            <DocDetail 
                mode="embedded"
                theme={theme}
                requestHeaders={apiConfig?.headers}
                data={detailData} 
                actionButtons={[
                    <ActionButton onClick={() => {
                        if (detailData?.url?.startsWith('http')) {
                            window.open(detailData.url)
                        }
                    }} key="open" icon={<SquareArrowOutUpRight />}>
                        {t('labels.openSource')}
                    </ActionButton>,
                ]}
                i18n={{
                    labels: {
                        type: t('labels.type'),
                        size: t('labels.size'),
                        createdBy: t('labels.createdBy'),
                        createdAt: t('labels.createdAt'),
                        updatedAt: t('labels.updatedAt'),
                        preview: t('labels.preview'),
                        previewUnavailableTitle: t('labels.previewUnavailableTitle'),
                        previewUnavailableDescription: t('labels.previewUnavailableDescription'),
                        openSource: t('labels.openSourceFromPreview'),
                        aiInterpretation: t('labels.aiInterpretation')
                    }
                }}
            />
        </CommonDrawer>
    );
}

export default ResultDetail;
