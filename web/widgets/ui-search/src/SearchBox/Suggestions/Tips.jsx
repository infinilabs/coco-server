import { List, Tag } from "antd";
import styles from "./index.module.less";
import { Lightbulb } from "lucide-react";

export const SUGGESTION_TIPS = "suggestion_tips"

export default (props) => {
    const { } = props;

    return (
        <List
            className="px-8px mb-12px"
            itemLayout="vertical"
            size="large"
            pagination={false}
            dataSource={[
                {
                    icon: <Lightbulb className="w-16px h-16px" />,
                    suggestion: <span>按 <Tag>/</Tag> 启用高级字段过滤，或直接输入 <Tag>字段名</Tag> + <Tag>:</Tag> 转为条件</span>,
                }
            ]}
            renderItem={(item, index) => {
                return (
                    <>
                        <div className="py-11px px-8px text-12px text-[var(--ui-search-antd-color-text-description)]">
                            搜索 Tips
                        </div>
                        <div
                            className={`${styles.listItem} relative h-40px pl-8px pr-40px flex flex-nowrap items-center rounded-8px`}
                        >
                            {
                                item.icon && (
                                    <div className="mr-8px text-[var(--ui-search-antd-color-text-description)] flex-shrink-0">
                                        {item.icon}
                                    </div>
                                )
                            }

                            <div className="mr-12px flex-shrink-1 max-w-[100%] min-w-0">
                                <div className="truncate whitespace-nowrap">
                                    {item.suggestion}
                                </div>
                            </div>

                            {
                                item.source && (
                                    <div className="w-210px text-[var(--ui-search-antd-color-text-description)] flex-shrink-0">
                                        {item.source}
                                    </div>
                                )
                            }
                        </div>
                    </>
                );
            }}
        />
    );
};