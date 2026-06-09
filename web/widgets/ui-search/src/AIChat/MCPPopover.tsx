import { useState, useEffect, useCallback, useRef } from "react";
import { Hammer, Layers, Box, ChevronDown, RefreshCw } from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import { useDebounce } from "ahooks";
import { Popover, Checkbox } from "antd";

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

interface MCPPopoverProps {
  mcp_servers: { enabled?: boolean; visible?: boolean };
  selectedIds: string[];
  onSelectionChange: (ids: string[]) => void;
  isMCPActive: boolean;
  setIsMCPActive: (val: boolean) => void;
  getMCPByServer: (query?: string) => Promise<DataSource[]>;
  shortcut?: string;
  t?: TFunction;
}

export default function MCPPopover({
  mcp_servers,
  selectedIds,
  onSelectionChange,
  isMCPActive,
  setIsMCPActive,
  getMCPByServer,
  t: tProp,
}: MCPPopoverProps) {
  const { t: tOriginal } = useTranslation("ai_chat");
  const t = tProp || tOriginal;

  const [open, setOpen] = useState(false);
  const [page, setPage] = useState(1);
  const [totalPage, setTotalPage] = useState(0);
  const [visibleList, setVisibleList] = useState<DataSource[]>([]);
  const searchInputRef = useRef<InputRef>(null);
  
  const [dataList, setDataList] = useState<DataSource[]>([]);
  const [keyword, setKeyword] = useState("");
  const debouncedKeyword = useDebounce(keyword, { wait: 500 });
  const [isRefreshDataSource, setIsRefreshDataSource] = useState(false);

  const getDataSourceList = useCallback(async () => {
    try {
      setPage(1);
      const res: DataSource[] = await getMCPByServer(debouncedKeyword);

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
      console.error("mcp_search", err);
    }
  }, [debouncedKeyword, getMCPByServer]);

  useEffect(() => {
    if (open) {
      getDataSourceList();
    }
  }, [debouncedKeyword, open, getDataSourceList]);

  useEffect(() => {
    setTotalPage(Math.max(Math.ceil(dataList.length / 10), 1));
  }, [dataList]);

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
    setIsRefreshDataSource(true);
    await getDataSourceList();
    setTimeout(() => {
        setIsRefreshDataSource(false);
    }, 1000);
  };

  if (!mcp_servers?.visible) {
    return null;
  }

  return (
    <div
      className={clsx(
        "flex justify-center items-center gap-1 h-6 px-2 rounded-full transition cursor-pointer",
        !isMCPActive && "hover:bg-[#EDEDED] dark:hover:bg-[#202126]"
      )}
      style={{
        backgroundColor: isMCPActive
          ? 'var(--ant-color-primary-bg)'
          : undefined,
      }}
      onClick={() => setIsMCPActive(!isMCPActive)}
      title={t("search.input.MCP") || "MCP"}
    >
       <Hammer
          className={clsx("size-4", isMCPActive ? "text-[var(--ant-color-primary)]" : "text-#333 dark:text-#666")}
        />

      {isMCPActive && (
        <>
          <span className="text-xs" style={{ color: 'var(--ant-color-primary)' }}>
            {t("search.input.MCP") || "MCP"}
          </span>

          <Popover
            open={open}
            trigger="click"
            onOpenChange={setOpen}
            placement="bottomLeft"
            getPopupContainer={(node) => {
              let el = node.parentElement;
              while (el && el !== document.body && el !== document.documentElement) {
                const { overflowY } = getComputedStyle(el);
                if (overflowY === "auto" || overflowY === "scroll") {
                  const parent = el.parentElement;
                  if (parent && parent !== document.documentElement) {
                    return parent;
                  }
                  return el;
                }
                el = el.parentElement;
              }
              return document.body;
            }}
            content={
              <div
                className="w-[300px] flex flex-col gap-2"
                onClick={(e) => e.stopPropagation()}
              >
                <div className="flex justify-between items-center px-1">
                    <span className="text-sm font-medium">{t("search.input.searchPopover.title") || "Select"}</span>
                    <button
                      className="bg-transparent p-1 hover:bg-black/5 dark:hover:bg-white/10 rounded-md transition-colors"
                      onClick={handleRefresh}
                    >
                      <RefreshIcon
                        className={`size-3 text-[#0287FF] transition-transform duration-1000 ${
                          isRefreshDataSource ? "animate-spin" : ""
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
                      const { id, name, icon } = item;
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
                          <div className="flex items-center gap-2 overflow-hidden">
                            {isAll ? (
                              <Layers className="size-4 text-[#0287FF]" />
                            ) : (
                              icon ? (
                                icon.startsWith("font_") ? (
                                  <FontIcon
                                    name={icon}
                                    className="w-4 h-4 mr-1"
                                  />
                                ) : (
                                  <img
                                    src={icon}
                                    className="w-4 h-4 mr-1"
                                    alt="assistant"
                                    onError={(e) => {
                                      const el = e.currentTarget as HTMLImageElement;
                                      el.style.display = "none";
                                    }}
                                  />
                                )
                              ) : <Box className="size-4 text-muted-foreground" />
                            )}

                            <span className="truncate text-sm">
                              {isAll && name ? t(name) || "All" : name}
                            </span>
                          </div>

                          <div className="flex items-center gap-1">
                            <div className="flex justify-center items-center size-6">
                              <Checkbox
                                checked={checked}
                                indeterminate={isCheckSome}
                                onChange={(e) =>
                                  onSelectDataSource(id, e.target.checked, isAll)
                                }
                              />
                            </div>
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
            }
          >
            <div
              role="button"
              tabIndex={0}
              className="text-[var(--ant-color-primary)] flex items-center justify-center size-4 rounded-sm hover:bg-black/5 dark:hover:bg-white/10"
              onClick={(e) => {
                e.stopPropagation();
              }}
            >
              <ChevronDown size={14} />
            </div>
          </Popover>
        </>
      )}
    </div>
  );
}
