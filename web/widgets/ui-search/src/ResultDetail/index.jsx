import { Button, Descriptions, Drawer, Tag } from "antd";
import { cloneElement, useState } from "react";
import styles from "./index.module.less";
import ImageSvg from "../icons/image.svg"
import { formatDate, isWithin7Days } from "../utils/date";
import { X } from "lucide-react";
import { Tags } from "lucide-react";
import { SquareArrowOutUpRight } from "lucide-react";
import { Bot } from "lucide-react";
import Markdown from "../ChatMessage/Markdown";

export function ResultDetail(props) {
    const { getContainer, data = {}, isMobile, children } = props;
    const [open, setOpen] = useState(false);

    const showDrawer = () => {
        setOpen(true);
    };

    const onClose = () => {
        setOpen(false);
    };

    return (
        <>
            {
                cloneElement(children, {
                    onClick: showDrawer
                })
            }
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
                        {data.url}
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
                    <div className={`flex justify-center items-center w-full bg-#F6F8FA rounded-lg mb-16px ${data.thumbnail ? "" : "h-300px"}`}>
                        {
                            data.thumbnail ? (
                                <img src={data.thumbnail} className="max-w-full max-h-full object-contain"/>
                            ) : (
                                <img src={ImageSvg} className="w-64px h-64px"/>
                            )
                        }
                    </div>
                    <div className="leading-[24px] text-12px">
                        <Markdown content={data.content} />
                    </div>
                </div>
                <div className="absolute bottom-0 w-full pb-24px px-24px">
                    <div className="bg-#f5f5f5 rounded-20px mb-24px py-24px px-16px">
                        <Descriptions column={2} colon={false} items={[
                            {
                                key: 'type',
                                label: 'Type',
                                children: data.type || '-',
                            },
                            {
                                key: 'size',
                                label: 'Size',
                                children: data.size || '-',
                            },
                            {
                                key: 'created_at',
                                label: 'Created At',
                                children: data.created ? (isWithin7Days(data.created) ? formatDate(data.created) : data.created) : '-',
                            },
                            {
                                key: 'created_by',
                                label: 'Created by',
                                children: data.owner?.username || '-',
                            },
                            {
                                key: 'updated_at',
                                label: 'Updated at',
                                children: data.last_updated_by?.timestamp ? (isWithin7Days(data.last_updated_by?.timestamp) ? formatDate(data.last_updated_by?.timestamp) : data.last_updated_by?.timestamp) : '-',
                            },
                            {
                                key: 'updated_by',
                                label: 'Updated by',
                                children: data.last_updated_by?.user?.username || '-',
                            },
                            ]} 
                        />
                    </div>
                    <div className="flex gap-8px">
                        <Button size="large" className="w-50% rounded-36px" onClick={() => data.url && window.open(data.url, '_blank')}><SquareArrowOutUpRight className="w-14px"/> Open</Button>
                        <Button size="large" type="primary" className="w-50% rounded-36px"><Bot className="w-14px"/> AI 解读</Button>
                    </div>
                </div>
            </Drawer>
        </>
    );
}

export default ResultDetail;
