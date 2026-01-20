import { Drawer } from "antd";
import styles from "./index.module.less";
import { SquareArrowOutUpRight, X } from "lucide-react";
import { ActionButton, DocDetail } from "@infinilabs/doc-detail";

export function ResultDetail(props) {
    const { getContainer, data = {}, isMobile, open, onClose, getRawContent } = props;

    return (
        <Drawer
            onClose={onClose}
            open={open}
            width={isMobile ? '100%' : 800}
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
                        Open Source
                    </ActionButton>,
                ]}
            />
        </Drawer>
    );
}

export default ResultDetail;
