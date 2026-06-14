import { Button, Spin, Typography } from "antd";
import { useState, useEffect, useRef, forwardRef, useImperativeHandle, useCallback, type ReactNode } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";

import styles from "./index.module.less";
import BasicIcon from "../../BasicIcon";
import { DEFAULT_SUGGESTIONS_SIZE } from "../useSearchBox";
import EnterIcon from "../../icons/EnterIcon";

interface ListContainerProps {
  type?: string;
  title?: string;
  data?: any[];
  onItemClick?: (item: any) => void;
  loadNext?: () => void;
  renderPrefix?: (item: any) => ReactNode;
  defaultRows?: number;
  useGlobalKeydown?: boolean;
  globalActiveIndex?: number;
  onGlobalSelect?: (index: number) => void;
  className?: string;
  defaultActiveIndex?: number;
  language?: string;
  resetKey?: string;
}

const ITEM_HEIGHT = 40;

const ListContainer = forwardRef<any, ListContainerProps>((props, ref) => {
  const { 
    type, 
    title, 
    data = [], 
    onItemClick, 
    loadNext, 
    renderPrefix, 
    defaultRows = DEFAULT_SUGGESTIONS_SIZE,
    useGlobalKeydown = false,
    globalActiveIndex = 0,
    onGlobalSelect,
    className = '',
    defaultActiveIndex = 0,
    language,
    resetKey
  } = props;
    
  const lang = language ? (language.startsWith('zh') ? 'zh' : 'en') : 'en';

  const getItemDescription = (item: any) => {
    const fd = item?.payload?.field_description;
    let desc = null;
    if (fd) {
      if (typeof fd === 'string') desc = fd;
      else if (typeof fd === 'object') desc = fd[lang] || fd.en || fd.zh || null;
    }
    if (!desc) desc = item?.source || null;
    return desc;
  };

  const [activeIndex, setActiveIndex] = useState(() => {
    if (useGlobalKeydown && typeof globalActiveIndex === 'number' && globalActiveIndex >= 0) return globalActiveIndex;
    return defaultActiveIndex;
  });

  const [dataSource, setDataSource] = useState<any[]>([]);
  const hasMoreRefs = useRef(true);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef(false);
  const scrollContainerRef = useRef<HTMLDivElement | null>(null);
  const keyDirectionRef = useRef('none');
  const [isKeyboardNav, setIsKeyboardNav] = useState(false);

  const virtualizer = useVirtualizer({
    count: dataSource.length,
    getScrollElement: () => scrollContainerRef.current,
    estimateSize: () => ITEM_HEIGHT,
    overscan: 5,
  });

  const scrollToIndex = useCallback((index: number, direction: string) => {
    if (index < 0 || index >= dataSource.length) return;
    const align = direction === 'down' ? 'end' : direction === 'up' ? 'start' : 'auto';
    virtualizer.scrollToIndex(index, { align, behavior: 'smooth' });
  }, [dataSource.length, virtualizer]);

  useEffect(() => {
    if (useGlobalKeydown) {
      const idx = typeof globalActiveIndex === 'number' ? globalActiveIndex : defaultActiveIndex;
      setActiveIndex(idx);
      scrollToIndex(idx, keyDirectionRef.current);
    }
  }, [globalActiveIndex, defaultActiveIndex, useGlobalKeydown, scrollToIndex]);

  // Only scroll when activeIndex changes due to user interaction (keyboard/mouse),
  // not when dataSource grows (which would reset scroll to top).
  const prevActiveIndexRef = useRef(activeIndex);
  useEffect(() => {
    if (activeIndex !== prevActiveIndexRef.current) {
      prevActiveIndexRef.current = activeIndex;
      if (activeIndex >= 0 && activeIndex < dataSource.length) {
        scrollToIndex(activeIndex, keyDirectionRef.current);
      }
    }
  }, [activeIndex]);

  // Trigger loadNext when scrolling near the end
  useEffect(() => {
    if (!loadNext || !hasMore || !hasMoreRefs.current || loadingRef.current) return;

    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      if (!hasMoreRefs.current || loadingRef.current) return;
      const { scrollTop, scrollHeight, clientHeight } = container;
      if (scrollTop + clientHeight >= scrollHeight - ITEM_HEIGHT) {
        loadingRef.current = true;
        loadNext();
      }
    };

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [loadNext, hasMore, dataSource.length]);

  useEffect(() => {
    if (loadNext) {
      if (data.length > 0) {
        setDataSource((prev) => [...prev, ...data].filter(item => !!item?.suggestion));
        const hasMore = data.length >= defaultRows;
        hasMoreRefs.current = hasMore;
        setHasMore(hasMore);
      } else {
        hasMoreRefs.current = false;
        setHasMore(false);
      }
      loadingRef.current = false;
    } else {
      setDataSource(data.filter(item => !!item?.suggestion));
    }
  }, [data, defaultRows]);

  // Reset dataSource when suggestion context changes (e.g., suggestion type or field trigger)
  useEffect(() => {
    if (typeof resetKey === 'undefined') return;
    setDataSource([]);
    hasMoreRefs.current = true;
    setHasMore(true);
    loadingRef.current = false;
  }, [resetKey]);

  useEffect(() => {
    if (useGlobalKeydown || !onItemClick) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (![38, 40, 13].includes(e.keyCode)) return;

      const totalItems = dataSource.length;
      if (totalItems === 0) return;

      e.preventDefault();
      // Stop propagation in capture phase to prevent antd Select (in tags mode)
      // from also processing Enter and creating its own tag.
      e.stopPropagation();

      setIsKeyboardNav(true);

      switch (e.keyCode) {
        case 40: 
          keyDirectionRef.current = 'down';
          setActiveIndex((prev) => {
            if (prev === -1) return 0;
            if (prev >= totalItems - 1) return totalItems - 1;
            return prev + 1;
          });
          break;
        case 38: 
          keyDirectionRef.current = 'up';
          setActiveIndex((prev) => {
            if (prev <= 0) return 0;
            if (prev === -1) return -1;
            return prev - 1;
          });
          break;
        case 13: 
          if (activeIndex >= 0 && activeIndex < totalItems) {
            keyDirectionRef.current = 'none';
            onItemClick(dataSource[activeIndex]);
          }
          break;
        default:
          break;
      }
    };

    // Use capture phase so this handler fires BEFORE antd Select's internal
    // keydown handler (which is attached to the input in bubble phase).
    document.addEventListener("keydown", handleKeyDown, true);
    return () => document.removeEventListener("keydown", handleKeyDown, true);
  }, [useGlobalKeydown, dataSource, activeIndex, onItemClick]);

  useImperativeHandle(ref, () => ({
    triggerItemClick: (index: number) => {
      if (index >= 0 && index < dataSource.length) {
        handleItemClick(dataSource[index], index);
      }
    },
    getListLength: () => dataSource.length,
    scrollToActiveItem: (index: number, direction = 'none') => {
      keyDirectionRef.current = direction;
      if (direction === 'up' || direction === 'down') {
        setIsKeyboardNav(true);
      }
      setActiveIndex(index);
    },
    setKeyDirection: (direction: string) => {
      keyDirectionRef.current = direction;
    }
  }));

  if (dataSource.length === 0) return null;

  const handleItemClick = (item: any, index: number) => {
    keyDirectionRef.current = 'none';
    
    if (useGlobalKeydown && onGlobalSelect) {
      onGlobalSelect(index);
    } else {
      setActiveIndex(index);
    }
    onItemClick?.(item);
  };

  return (
    <>
      {title && (
        <div className="py-14px px-12px text-12px text-[var(--ant-color-text-description)]">
          {title}
        </div>
      )}
      <div
        ref={scrollContainerRef}
        className={`px-4px mb-12px overflow-auto ${className}`}
        style={{ 
          maxHeight: ITEM_HEIGHT * defaultRows,
        }}
        onMouseMove={() => {
          if (isKeyboardNav) setIsKeyboardNav(false);
        }}
      >
        <div
          style={{
            height: `${virtualizer.getTotalSize()}px`,
            width: '100%',
            position: 'relative',
          }}
        >
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const index = virtualRow.index;
            const item = dataSource[index];
            const isActive = activeIndex === index && !!onItemClick;
            const desc = getItemDescription(item);
            return (
              <div
                key={virtualRow.key}
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  width: '100%',
                  height: `${virtualRow.size}px`,
                  transform: `translateY(${virtualRow.start}px)`,
                }}
              >
                <div
                  className={`${styles.listItem} ${isActive ? styles.active : ''} relative h-40px pl-8px flex flex-nowrap items-center rounded-8px 
                  ${onItemClick ? 'cursor-pointer' : ''} 
                  ${isActive ? "bg-[rgba(233,240,254,1)] dark:bg-[rgba(255,255,255,0.05)] pr-40px" : "pr-8px"}`}
                  onClick={() => handleItemClick(item, index)}
                  onMouseEnter={() => {
                    if (!onItemClick) return;
                    if (isKeyboardNav) return;
                    keyDirectionRef.current = 'none';
                    if (useGlobalKeydown && onGlobalSelect) {
                      onGlobalSelect(index);
                    } else {
                      setActiveIndex(index);
                    }
                  }}
                >
                  {renderPrefix?.(item)}
                  {item.icon && (
                    <BasicIcon 
                      className={"flex justify-center items-center w-16px h-16px mr-8px text-[var(--ant-color-text-description)] flex-shrink-0"} 
                      icon={item.icon}
                    />
                  )}
                  <div className="mr-12px flex-1 min-w-0">
                    <div className="leading-22px truncate whitespace-nowrap">{item.suggestion}</div>
                  </div>
                  {desc && (
                    <Typography.Text type="secondary" className="flex-shrink-0" >
                      {desc}
                    </Typography.Text>
                  )}
                  {onItemClick && isActive && (
                    <Button
                      className={`${styles.enter} absolute right-8px top-8px !w-24px !h-24px rounded-8px !border-0 dark:!bg-[rgb(var(--ui-search--layout-bg-color))] !shadow-none`}
                      classNames={{ icon: `w-14px h-14px !text-14px` }}
                      size="small"
                      icon={<EnterIcon className="w-14px h-14px !text-[#333] dark:!text-#666" />}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleItemClick(item, index);
                      }}
                    />
                  )}
                </div>
              </div>
            );
          })}
        </div>
        {loadNext && hasMore && (
          <div className="flex justify-center py-12px text-[var(--ant-color-text-description)]">
            <Spin size="small" />
          </div>
        )}
      </div>
    </>
  );
});

export default ListContainer;