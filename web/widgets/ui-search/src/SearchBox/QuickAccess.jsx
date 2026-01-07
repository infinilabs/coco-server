import { Button, List } from "antd";
import { CornerDownLeft, MessageCircle, Search } from "lucide-react";
import { useState, useEffect, useRef } from "react";

import styles from "./index.module.less";

export default (props) => {
    const { keyword, suggestions = [], onSelectItem, onClickItem } = props;

    const mergedData = [
        {
            action: "search",
            icon: <Search className="w-16px h-16px" />,
            desc: `快速查找 | 直达文件与结果`,
            type: "quickAccess",
        },
        {
            action: "deepthink",
            icon: <MessageCircle className="w-16px h-16px" />,
            desc: `深度思考 | AI 提炼，结论优先`,
            type: "quickAccess",
        },
        {
            action: "deepresearch",
            icon: <Search className="w-16px h-16px" />,
            desc: `深度研究 | 多步推理，综合分析`,
            type: "quickAccess",
        },
        ...suggestions.map((item, index) => ({ ...item, isFirst: index === 0 })),
    ];

    const [activeIndex, setActiveIndex] = useState(0);
    const itemRefs = useRef([]);

    useEffect(() => {
        const handleKeyDown = (e) => {
            if (![38, 40, 13].includes(e.keyCode)) return;

            const totalItems = mergedData.length;
            if (totalItems === 0) return;

            e.preventDefault();

            switch (e.keyCode) {
                case 40: // 下键
                    setActiveIndex((prev) => {
                        let index;
                        if (prev === -1) {
                            // 未选中时，按下键直接选中第一项
                            index = 0;
                        } else {
                            // 已选中时，循环切换下一项
                            index = (prev + 1) % totalItems;
                        }
                        onSelectItem?.(mergedData[index]);
                        return index;
                    });
                    break;
                case 38: // 上键
                    // 未选中时，上键直接不生效
                    if (activeIndex === -1) return;
                    setActiveIndex((prev) => {
                        const index = (prev - 1 + totalItems) % totalItems;
                        onSelectItem?.(mergedData[index]);
                        return index;
                    });
                    break;
                case 13: // 回车
                    if (activeIndex >= 0 && activeIndex < totalItems) {
                        itemRefs.current[activeIndex]?.click();
                    }
                    break;
                default:
                    break;
            }
        };

        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [mergedData.length, onSelectItem, activeIndex]); 

    return (
        <List
            className="px-8px mb-12px"
            itemLayout="vertical"
            size="large"
            pagination={false}
            dataSource={mergedData}
            renderItem={(item, index) => {
                const isActive = activeIndex === index;

                return (
                    <>
                        {item.type === "suggestion" && item.isFirst && (
                            <div className="py-11px px-8px text-12px text-[var(--ui-search-antd-color-text-description)]">
                                搜索建议
                            </div>
                        )}
                        <div
                            ref={(el) => (itemRefs.current[index] = el)}
                            className={`${styles.listItem} ${isActive ? styles.active : ''} cursor-pointer relative h-40px pl-8px pr-40px flex flex-nowrap items-center rounded-8px 
                hover:bg-[rgba(233,240,254,1)] 
                ${isActive ? "bg-[rgba(233,240,254,1)]" : ""}`}
                            onClick={() => onClickItem?.(item)}
                        >
                            <div className="mr-8px text-[var(--ui-search-antd-color-text-description)] flex-shrink-0">
                                {item.icon}
                            </div>

                            <div className="mr-12px flex-shrink-1 max-w-[100%] min-w-0">
                                <div className="truncate whitespace-nowrap">
                                    {item.type === "quickAccess" ? keyword : item.keyword}
                                </div>
                            </div>

                            <div className="w-210px text-[var(--ui-search-antd-color-text-description)] flex-shrink-0">
                                -{item.desc}
                            </div>

                            <Button
                                className={`${styles.enter} absolute right-8px top-8px !w-24px !h-24px rounded-8px border-0`}
                                classNames={{ icon: `w-14px h-14px !text-14px` }}
                                size="small"
                                icon={<CornerDownLeft className="w-14px h-14px" />}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onClickItem?.(item);
                                }}
                            />
                        </div>
                    </>
                );
            }}
        />
    );
};