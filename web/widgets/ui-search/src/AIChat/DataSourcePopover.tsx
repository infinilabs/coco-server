import { useState, useEffect, useLayoutEffect, useCallback, useRef, type ReactNode } from "react";
import { Layers, Box, ChevronDown } from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import { useDebounce } from "ahooks";
import { Checkbox } from "antd";

import NoDataImage from "./NoDataImage";
import Pagination from "./Common/Pagination";
import { Input, type InputRef } from "antd";
import type { TFunction } from "i18next";
import FontIcon from "./FontIcon";
import RefreshIcon from "../icons/RefreshIcon";

export interface DataSource {
    id: string;
    name?: string;
    icon?: string;
    [key: string]: unknown;
}

interface DataSourcePopoverProps {
    visible?: boolean;
    selectedIds: string[];
    onSelectionChange: (ids: string[]) => void;
    isActive: boolean;
    setIsActive: (val: boolean) => void;
    fetchData: (query?: string) => Promise<DataSource[]>;
    icon: ReactNode;
    label: string;
    title?: string;
    showItemIcon?: boolean;
    t?: TFunction;
}

export default function DataSourcePopover({
    visible,
    selectedIds,
    onSelectionChange,
    isActive,
    setIsActive,
    fetchData,
    icon,
    label,
    title,
    showItemIcon = false,
    t: tProp,
}: DataSourcePopoverProps) {
    const { t: tOriginal } = useTranslation();
    const t = tProp || tOriginal;

    const [open, setOpen] = useState(false);
    const [page, setPage] = useState(1);
    const [totalPage, setTotalPage] = useState(0);
    const [visibleList, setVisibleList] = useState<DataSource[]>([]);
    const searchInputRef = useRef<InputRef>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const triggerRef = useRef<HTMLDivElement>(null);

    const [dataList, setDataList] = useState<DataSource[]>([]);
    const [keyword, setKeyword] = useState("");
    const debouncedKeyword = useDebounce(keyword, { wait: 500 });
    const [isRefreshing, setIsRefreshing] = useState(false);
    const hasAutoSelectedRef = useRef(false);

    // Calculate panel position to stay within .ui-search bounds
    const [panelStyle, setPanelStyle] = useState<React.CSSProperties>({});

    const updatePanelPosition = useCallback(() => {
        if (!triggerRef.current) return;
        const container = triggerRef.current.closest('.ui-search') as HTMLElement | null;
        if (!container) return;
        const containerRect = container.getBoundingClientRect();
        const triggerRect = triggerRef.current.getBoundingClientRect();
        const parentRect = (triggerRef.current.offsetParent as HTMLElement | null)?.getBoundingClientRect();
        if (!parentRect) return;

        const panelPadding = 24;
        const maxW = 300 + panelPadding;
        const rightSpace = containerRect.right - triggerRect.left;
        const leftSpace = triggerRect.right - containerRect.left;
        const allSpace = rightSpace + leftSpace - 16;
        let left = 0;
        let panelW = maxW;
        if (rightSpace >= maxW) {
            left = triggerRect.left;
        } else if (leftSpace >= maxW) {
            left = triggerRect.right - maxW;
        } else {
            if (allSpace >= maxW) {
                left = (containerRect.width - maxW) / 2 + containerRect.left;
            } else {
                left = containerRect.left;
                panelW = allSpace;
            }
        }
        setPanelStyle({ width: panelW, left: left - parentRect.left });
    }, []);

    useLayoutEffect(() => {
        if (!open) return;
        updatePanelPosition();
        const ro = new ResizeObserver(updatePanelPosition);
        const container = triggerRef.current?.closest('.ui-search') as HTMLElement | null;
        if (container) ro.observe(container);
        return () => ro.disconnect();
    }, [open, updatePanelPosition]);

    // Close panel when clicking outside
    useEffect(() => {
        if (!open) return;
        const handleClickOutside = (e: MouseEvent) => {
            if (
                panelRef.current && !panelRef.current.contains(e.target as Node) &&
                triggerRef.current && !triggerRef.current.contains(e.target as Node)
            ) {
                setOpen(false);
            }
        };
        // Use the closest root (shadow DOM or document) to capture clicks
        const root = triggerRef.current?.getRootNode() as Document | ShadowRoot || document;
        root.addEventListener("mousedown", handleClickOutside as EventListener);
        return () => root.removeEventListener("mousedown", handleClickOutside as EventListener);
    }, [open]);

    const getDataList = useCallback(async () => {
        try {
            setPage(1);
            const res: DataSource[] = await fetchData(debouncedKeyword);

            if (!res || res.length === 0) {
                setDataList([]);
                return;
            }
            const data = res.length
                ? [
                    {
                        id: "all",
                        name: "search.input.searchPopover.allScope",
                    },
                    ...res,
                ]
                : [];

            setDataList(data);
        } catch (err) {
            setDataList([]);
            console.error("datasource_popover_fetch", err);
        }
    }, [debouncedKeyword, fetchData]);

    useEffect(() => {
        if (open) {
            getDataList();
        }
    }, [debouncedKeyword, open, getDataList]);

    useEffect(() => {
        setTotalPage(Math.max(Math.ceil(dataList.length / 10), 1));
    }, [dataList]);

    useEffect(() => {
        if (isActive && !hasAutoSelectedRef.current && dataList.length > 1) {
            const allIds = dataList.slice(1).map((item) => item.id);
            onSelectionChange(allIds);
            hasAutoSelectedRef.current = true;
        }
    }, [isActive, dataList, onSelectionChange]);

    useEffect(() => {
        if (dataList.length === 0) {
            return setVisibleList([]);
        }

        const startIndex = (page - 1) * 9;
        const endIndex = startIndex + 9;

        const list = [
            dataList[0],
            ...dataList.slice(1).slice(startIndex, endIndex),
        ];

        setVisibleList(list);
    }, [dataList, page]);

    const onSelectDataSource = useCallback(
        (id: string, checked: boolean, isAll: boolean) => {
            const nextSourceDataIds = new Set(selectedIds);

            const ids = isAll ? visibleList.slice(1).map((item) => item.id) : [id];

            for (const id of ids) {
                if (checked) {
                    nextSourceDataIds.add(id);
                } else {
                    nextSourceDataIds.delete(id);
                }
            }

            onSelectionChange(Array.from(nextSourceDataIds));
        },
        [visibleList, selectedIds, onSelectionChange]
    );

    const handlePrev = () => {
        if (page === 1) return;
        setPage(page - 1);
    };

    const handleNext = () => {
        if (page === totalPage) return;
        setPage(page + 1);
    };

    const handleRefresh = async () => {
        setIsRefreshing(true);
        await getDataList();
        setTimeout(() => {
            setIsRefreshing(false);
        }, 1000);
    };

    if (!visible) {
        return null;
    }

    const renderItemIcon = (item: DataSource, isAll: boolean) => {
        if (isAll) {
            return <Layers className="size-4 text-[#0287FF]" />;
        }

        if (!showItemIcon) {
            return <Box className="size-4 text-muted-foreground" />;
        }

        const { icon: itemIcon } = item;
        if (!itemIcon) {
            return <Box className="size-4 text-muted-foreground" />;
        }

        if (itemIcon.startsWith("font_")) {
            return <FontIcon name={itemIcon} className="w-4 h-4 mr-1" />;
        }

        return (
            <img
                src={itemIcon}
                className="w-4 h-4 mr-1"
                alt="icon"
                onError={(e) => {
                    const el = e.currentTarget as HTMLImageElement;
                    el.style.display = "none";
                }}
            />
        );
    };

    return (
        <div
            className={clsx(
                "relative flex justify-center items-center gap-1 h-6 px-2 rounded-full transition cursor-pointer",
                !isActive && "hover:bg-[#EDEDED] dark:hover:bg-[#202126]"
            )}
            style={{
                backgroundColor: isActive
                    ? 'var(--ant-color-primary-bg)'
                    : undefined,
            }}
            onClick={() => setIsActive(!isActive)}
            title={title || label}
        >
            {icon}

            {isActive && (
                <>
                    <span className="text-xs" style={{ color: 'var(--ant-color-primary)' }}>
                        {label}
                    </span>

                    <div
                        ref={triggerRef}
                        role="button"
                        tabIndex={0}
                        className="text-[var(--ant-color-primary)] flex items-center justify-center size-4 rounded-sm hover:bg-black/5 dark:hover:bg-white/10"
                        onClick={(e) => {
                            e.stopPropagation();
                            setOpen(!open);
                        }}
                    >
                        <ChevronDown size={14} />
                    </div>

                    {open && (
                        <div
                            ref={panelRef}
                            className="absolute bottom-full mb-2 z-50 rounded-lg bg-[var(--ant-color-bg-elevated,#fff)] shadow-[0_6px_16px_0_rgba(0,0,0,0.08),0_3px_6px_-4px_rgba(0,0,0,0.12),0_9px_28px_8px_rgba(0,0,0,0.05)] p-3"
                            style={panelStyle}
                            onClick={(e) => e.stopPropagation()}
                        >
                            <div className="flex flex-col gap-2">
                                <div className="flex justify-between items-center px-1">
                                    <span className="text-sm font-medium">{t("search.input.searchPopover.title") || "Select"}</span>
                                    <button
                                        className="cursor-pointer bg-transparent border-0 p-1 hover:bg-black/5 dark:hover:bg-white/10 rounded-md transition-colors"
                                        onClick={handleRefresh}
                                    >
                                        <RefreshIcon
                                            className={`size-3 text-[#0287FF] transition-transform duration-1000 ${isRefreshing ? "animate-spin" : ""
                                                }`}
                                        />
                                    </button>
                                </div>

                                <div className="flex items-center gap-2 px-2 py-1 border rounded-md border border-solid border-[#F0F0F0] dark:border-[#303030]">
                                    <Input
                                        autoFocus
                                        autoCorrect="off"
                                        value={keyword}
                                        ref={searchInputRef}
                                        className="h-6 p-0 border-0 shadow-none focus-visible:ring-0 focus:shadow-none"
                                        variant="borderless"
                                        placeholder={
                                            t("search.input.searchPopover.placeholder") || "Search..."
                                        }
                                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                                            setKeyword(e.target.value);
                                        }}
                                    />
                                </div>

                                {visibleList.length > 0 ? (
                                    <ul className="flex flex-col gap-1 p-0 m-0 list-none">
                                        {visibleList.map((item, index) => {
                                            const { id, name } = item;
                                            const isAll = index === 0;

                                            const checked = isAll
                                                ? visibleList.slice(1).every((vItem) => selectedIds.includes(vItem.id))
                                                : selectedIds.includes(id);

                                            const isCheckSome = isAll && !checked &&
                                                visibleList.slice(1).some((vItem) => selectedIds.includes(vItem.id));

                                            return (
                                                <li
                                                    key={id}
                                                    className="flex justify-between items-center px-2 py-1 hover:bg-black/5 dark:hover:bg-white/10 rounded-sm cursor-pointer"
                                                    onClick={() => {
                                                        onSelectDataSource(id, !checked, isAll);
                                                    }}
                                                >
                                                    <div className="flex items-center gap-2 overflow-hidden min-w-0 flex-1">
                                                        <div className="shrink-0">{renderItemIcon(item, isAll)}</div>

                                                        <span className="truncate text-sm">
                                                            {isAll && name ? t(name) || "All" : name}
                                                        </span>
                                                    </div>

                                                    <div className="shrink-0 flex justify-center items-center size-6">
                                                        <Checkbox
                                                            checked={checked}
                                                            indeterminate={isCheckSome}
                                                            onChange={(e) =>
                                                                onSelectDataSource(id, e.target.checked, isAll)
                                                            }
                                                        />
                                                    </div>
                                                </li>
                                            );
                                        })}
                                    </ul>
                                ) : (
                                    <div className="flex items-center justify-center py-4">
                                        <NoDataImage />
                                    </div>
                                )}

                                {visibleList.length > 0 && (
                                    <Pagination
                                        current={page}
                                        totalPage={totalPage}
                                        onPrev={handlePrev}
                                        onNext={handleNext}
                                        className="border-t-[#F0F0F0] dark:border-t-[#303030]"
                                    />
                                )}
                            </div>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}
