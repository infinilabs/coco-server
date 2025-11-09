import { Button, Descriptions, Drawer, Tag } from "antd";
import styles from "./index.module.less";
import Markdown from "./Markdown";
import { Bot, SquareArrowOutUpRight, Tags, X } from "lucide-react";

import dayjs from "dayjs";
import relativeTime from 'dayjs/plugin/relativeTime';
import { getDocument } from "@/service/api";
dayjs.extend(relativeTime);

export const isWithin7Days = (date) => {
    const targetDate = dayjs(date);
    const now = dayjs();
    const diffInMs = now.diff(targetDate);
    return diffInMs <= 7 * 24 * 60 * 60 * 1000; 
}

export const formatDate = (date) => {
    const targetDate = dayjs(date);
    return isWithin7Days(date) ? targetDate.fromNow() : targetDate.format('YYYY-MM-DD HH:mm:ss')
}

export default function DocumentDrawer(props) {
    const { getContainer, data = {}, isMobile, open, onClose } = props;

    const { t } = useTranslation();

    return (
        <>
            <Drawer
                title={data.source?.name || ' '}
                onClose={onClose}
                open={open}
                width={isMobile ? '100%' : 724}
                closeIcon={null}
                extra={(
                    <X className="color-[#bbb] cursor-pointer" onClick={onClose}/>
                )}
                rootClassName={styles.detail}
                getContainer={getContainer}
            >
                <div className="h-full overflow-auto px-24px">
                    <div className="color-[#027ffe] text-16px mb-8px">
                        {data.title}
                    </div>
                    <div className="color-[#999] mb-16px">
                        {data.url?.startsWith('http') ? <a onClick={() => data.url && window.open(data.url, '_blank')}>{data.url}</a> : data.url }
                    </div>
                    {
                        data.tags?.length > 0 && (
                            <div className="color-[#999] mb-16px flex items-center gap-8px mb-24px flex-wrap">
                                <Tags className="text-24px"/>
                                {data.tags.map((t, i) => <Tag className="bg-#E8E8E8 color-#101010 border-0" key={i}>{t}</Tag>)}
                            </div>
                        )
                    }
                    {
                        data.thumbnail && (
                            <div className={`flex justify-center items-center w-full bg-#F6F8FA rounded-lg mb-16px`}>
                                <img src={data.thumbnail} className="max-w-full max-h-full object-contain"/>
                            </div>
                        )
                    }
                    <div className="leading-[24px] text-12px">
                        <Markdown content={data.content} />
                    </div>
                </div>
                <div className="absolute bottom-0 w-full px-24px">
                    <div className="bg-#f5f5f5 rounded-20px mb-24px py-24px px-16px">
                        <Descriptions column={2} colon={false} items={[
                            {
                                key: 'type',
                                label: t('page.datasource.labels.type'),
                                children: data.type || '-',
                            },
                            {
                                key: 'size',
                                label: t('page.datasource.labels.size'),
                                children: data.size || '-',
                            },
                            {
                                key: 'created',
                                label: t('page.datasource.labels.created'),
                                children: data.created ? (isWithin7Days(data.created) ? formatDate(data.created) : data.created) : '-',
                            },
                            {
                                key: 'created_by',
                                label: t('page.datasource.labels.createdBy'),
                                children: data.owner?.username || '-',
                            },
                            {
                                key: 'updated',
                                label: t('page.datasource.labels.updated'),
                                children: data.last_updated_by?.timestamp ? (isWithin7Days(data.last_updated_by?.timestamp) ? formatDate(data.last_updated_by?.timestamp) : data.last_updated_by?.timestamp) : '-',
                            },
                            {
                                key: 'updated_by',
                                label: t('page.datasource.labels.updatedBy'),
                                children: data.last_updated_by?.user?.username || '-',
                            },
                            ]} 
                        />
                    </div>
                    {/* <div className="flex gap-8px">
                        <Button size="large" className="w-50% rounded-36px" onClick={() => data.url && window.open(data.url, '_blank')}><SquareArrowOutUpRight className="w-14px"/> Open</Button>
                        <Button size="large" type="primary" className="w-50% rounded-36px"><Bot className="w-14px"/> AI 解读</Button>
                    </div> */}
                </div>
            </Drawer>
        </>
    );
}
