import { Button, Checkbox, List } from "antd";
import { CornerDownLeft } from "lucide-react";
import { useState, useEffect, useRef, useMemo } from "react";

import styles from "./index.module.less";

export const SUGGESTION_FILTER_VALUES = "field_values"

export default (props) => {
    const { data = [], filter = {}, onItemClick } = props;

    const [activeIndex, setActiveIndex] = useState(0);
    const itemRefs = useRef([]);

    const formatData = useMemo(() => {
        return data.filter(item => !!item).map((item, index) => ({
        ...item,
        dividerTitle: index === 0 ? '过滤条件' : ''
        }));
    }, [data]);

    useEffect(() => {
        const handleKeyDown = (e) => {
            if (![38, 40, 13].includes(e.keyCode)) return;

            const totalItems = formatData.length;
            if (totalItems === 0) return;

            e.preventDefault();

            switch (e.keyCode) {
                case 40: // down
                    setActiveIndex((prev) => {
                        let index;
                        if (prev === -1) {
                            index = 0;
                        } else {
                            index = (prev + 1) % totalItems;
                        }
                        return index;
                    });
                    break;
                case 38: // up
                    if (activeIndex === -1) return;
                    setActiveIndex((prev) => {
                        const index = (prev - 1 + totalItems) % totalItems;
                        return index;
                    });
                    break;
                case 13: // enter
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
    }, [formatData, activeIndex]); 

    const { field = {}, value = [] } = filter || {}
    const { payload = {} } = field || {}
    const { support_multi_select } = payload || {}

    return (
        <List
            className="px-8px mb-12px"
            itemLayout="vertical"
            size="large"
            pagination={false}
            dataSource={formatData}
            renderItem={(item, index) => {
                const isActive = activeIndex === index;
                
                return (
                    <>
                        {item.dividerTitle && (
                            <div className="py-11px px-8px text-12px text-[var(--ui-search-antd-color-text-description)]">
                                {item.dividerTitle}
                            </div>
                        )}
                        <div
                            ref={(el) => (itemRefs.current[index] = el)}
                            className={`${styles.listItem} ${isActive ? styles.active : ''} cursor-pointer relative h-40px pl-8px pr-40px flex flex-nowrap items-center rounded-8px 
                hover:bg-[rgba(233,240,254,1)] 
                ${isActive ? "bg-[rgba(233,240,254,1)]" : ""}`}
                            onClick={() => {
                                onItemClick?.(item)
                                setActiveIndex(index)
                            }}
                        >
                            {
                                support_multi_select && (
                                    <div className="mr-8px flex-shrink-0">
                                        <Checkbox checked={value.findIndex((v) => v === item.suggestion) !== -1}></Checkbox>
                                    </div>
                                )
                            }

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

                            <Button
                                className={`${styles.enter} absolute right-8px top-8px !w-24px !h-24px rounded-8px border-0`}
                                classNames={{ icon: `w-14px h-14px !text-14px` }}
                                size="small"
                                icon={<CornerDownLeft className="w-14px h-14px" />}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onItemClick?.(item);
                                    setActiveIndex(index)
                                }}
                            />
                        </div>
                    </>
                );
            }}
        />
    );
};