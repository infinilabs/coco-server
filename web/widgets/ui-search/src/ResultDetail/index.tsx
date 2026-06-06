import { Drawer } from "antd";
import styles from "./index.module.less";
import { SquareArrowOutUpRight, X } from "lucide-react";
import { ActionButton, DocDetail } from "@infinilabs/doc-detail";
import { type FC } from "react";
import { useTranslation } from "react-i18next";

interface ResultDetailProps {
    getContainer?: () => HTMLElement;
    data?: Record<string, any>;
    isMobile?: boolean;
    open?: boolean;
    onClose?: () => void;
    getRawContent?: (data: Record<string, any>) => string;
    apiConfig?: Record<string, any>;
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
            rootClassName={styles.detail}
            getContainer={getContainer}
            destroyOnHidden
            classNames={{
                wrapper: `!overflow-hidden ${isMobile ? '!left-12px !right-12px !w-[calc(100%-24px)]' : '!right-24px'} !top-146px !bottom-24px !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`,
                body: '!p-24px !rounded-12px'
            }}
            mask={false}
            maskClosable={false}
        >
            <X className="color-[#bbb] cursor-pointer absolute right-24px top-24px z-1" onClick={onClose} />
            <DocDetail 
                requestHeaders={apiConfig?.headers}
                data={{
                    ...(data || {}),
                    url: getRawContent ? getRawContent(data) : data?.url
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
            />
        </Drawer>
    );
}

export default ResultDetail;
