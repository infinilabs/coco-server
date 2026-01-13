import { Button, List } from "antd";
import { CornerDownLeft, Minus, Plus, Slash } from "lucide-react";
import { useState, useEffect, useRef, useMemo } from "react";

import styles from "./index.module.less";

export const SUGGESTION_OPERATORS = "suggestion_operators"

export const OPERATOR_ICONS = {
    "and": (
        <Button
            shape="circle"
            className={`!bg-#1784FC !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Plus className="w-8px h-8px" />}
        />
    ),
    "or": (
        <Button
            shape="circle"
            className={`!bg-#8BBD7A !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Minus className="w-8px h-8px rotate-105" />}
        />
    ),
    "not": (
        <Button
            shape="circle"
            className={`!bg-#F15A5A !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Minus className="w-8px h-8px" />}
        />
    )
}

export default (props) => {
    const { onItemClick } = props;

    const [activeIndex, setActiveIndex] = useState(0);
    const itemRefs = useRef([]);

    const data = [
        {
            suggestion: 'and',
            source: `满足全部条件`,
            icon: OPERATOR_ICONS['and'],
            dividerTitle: '条件组合'
        },
        {
            suggestion: 'or',
            source: `满足任一条件`,
            icon: OPERATOR_ICONS['or'],
        },
        {
            suggestion: 'not',
            source: `排除条件`,
            icon: OPERATOR_ICONS['not'],
        },
    ]

    useEffect(() => {
        const handleKeyDown = (e) => {
            if (![38, 40, 13].includes(e.keyCode)) return;

            const totalItems = data.length;
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
    }, [data, activeIndex]);

    return (
        <List
            className="px-8px mb-12px"
            itemLayout="vertical"
            size="large"
            pagination={false}
            dataSource={data}
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
                            }}
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

                            <Button
                                className={`${styles.enter} absolute right-8px top-8px !w-24px !h-24px rounded-8px border-0`}
                                classNames={{ icon: `w-14px h-14px !text-14px` }}
                                size="small"
                                icon={<CornerDownLeft className="w-14px h-14px" />}
                                onClick={(e) => {
                                    e.stopPropagation();
                                    onItemClick?.(item);
                                }}
                            />
                        </div>
                    </>
                );
            }}
        />
    );
};