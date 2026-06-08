import { Drawer, Tooltip } from "antd";
import { SquareArrowOutUpRight, X } from "lucide-react";
import { ActionButton, DocDetail } from "./DocDetail";
import { type FC } from "react";
import { useTranslation } from "react-i18next";
import { filesize } from 'filesize';
import dayjs from "dayjs";
import { isString } from "lodash";

interface ResultDetailProps {
    getContainer?: () => HTMLElement;
    data?: Record<string, any>;
    isMobile?: boolean;
    open?: boolean;
    onClose?: () => void;
    getRawContent?: (data: Record<string, any>) => string;
    apiConfig?: Record<string, any>;
}

export const formatDate = (value: string | number) => {
    const targetDate = dayjs(value);
    const dateTime = targetDate.format('YYYY-MM-DD HH:mm:ss');
    const timezone = `(GMT${targetDate.format('ZZ')})`;
    return `${dateTime} ${timezone}`;
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
    const { getContainer, data = {}, isMobile, open, onClose, getRawContent, apiConfig } = props;
    const { t } = useTranslation();

    return (
        <Drawer
            onClose={onClose}
            open={open}
            size={isMobile ? 'large' : 800}
            closeIcon={null}
            getContainer={getContainer}
            destroyOnHidden
            classNames={{
                wrapper: `!overflow-hidden ${isMobile ? '!left-12px !right-12px !w-[calc(100%-24px)]' : '!right-24px'} !top-146px !bottom-24px !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`,
                body: '!p-24px !rounded-12px !overflow-hidden !h-full'
            }}
            mask={false}
            maskClosable={false}
        >
            <X className="color-[#bbb] cursor-pointer absolute right-24px top-24px z-1" onClick={onClose} />
            <DocDetail 
                mode="embedded"
                requestHeaders={apiConfig?.headers}
                data={{
                    ...(data || {}),
                    url: getRawContent ? getRawContent(data) : data?.url,
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
                }} 
                actionButtons={[
                    <ActionButton onClick={() => {
                        if (data?.url?.startsWith('http')) {
                            window.open(data.url)
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
                        aiInterpretation: t('labels.aiInterpretation')
                    }
                }}
            />
        </Drawer>
    );
}

export default ResultDetail;
