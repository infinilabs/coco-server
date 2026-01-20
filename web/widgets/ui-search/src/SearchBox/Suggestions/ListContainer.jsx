import { Button, List, Typography } from "antd";
import { CornerDownLeft } from "lucide-react";
import { useState, useEffect, useRef, forwardRef, useImperativeHandle } from "react";

import styles from "./index.module.less";
import BasicIcon from "../../BasicIcon";
import InfiniteScroll from 'react-infinite-scroll-component';

const ListContainer = forwardRef((props, ref) => {
  const { 
    type, 
    title, 
    data = [], 
    onItemClick, 
    loadNext, 
    renderPrefix, 
    defaultRows = 5,
    useGlobalKeydown = false,
    globalActiveIndex = -1,
    onGlobalSelect
  } = props;

  const [activeIndex, setActiveIndex] = useState(-1);
  const itemRefs = useRef([]);
  const [dataSource, setDataSource] = useState([]);
  const hasMoreRefs = useRef(true);
  const [hasMore, setHasMore] = useState(true);
  const scrollContainerRef = useRef(null);
  const keyDirectionRef = useRef('none');
  const scrollBoundaryRef = useRef({
    isAtTop: false,
    isAtBottom: false
  });

  // 同步全局激活索引
  useEffect(() => {
    if (useGlobalKeydown) {
      setActiveIndex(globalActiveIndex);
    }
  }, [useGlobalKeydown, globalActiveIndex]);

  // 核心：渐进式滚动逻辑
  useEffect(() => {
    if (
      activeIndex === -1 || 
      activeIndex >= itemRefs.current.length ||
      !scrollContainerRef.current ||
      !itemRefs.current[activeIndex]
    ) {
      scrollBoundaryRef.current = { isAtTop: false, isAtBottom: false };
      keyDirectionRef.current = 'none';
      return;
    }

    const container = scrollContainerRef.current;
    const activeElement = itemRefs.current[activeIndex];
    const containerRect = container.getBoundingClientRect();
    const elementRect = activeElement.getBoundingClientRect();
    
    const elementTopInContainer = elementRect.top - containerRect.top + container.scrollTop;
    const elementBottomInContainer = elementTopInContainer + elementRect.height;
    const containerHeight = containerRect.height;
    const containerScrollTop = container.scrollTop;
    const direction = keyDirectionRef.current;

    let targetScrollTop = containerScrollTop;
    const boundary = scrollBoundaryRef.current;

    const isElementInView = elementTopInContainer >= containerScrollTop && elementBottomInContainer <= containerScrollTop + containerHeight;
    if (direction === 'none' || (isElementInView && !boundary.isAtTop && !boundary.isAtBottom)) {
      scrollBoundaryRef.current = { isAtTop: false, isAtBottom: false };
    }

    if (direction === 'down') {
      if (boundary.isAtBottom) {
        targetScrollTop = elementBottomInContainer - containerHeight;
      } else {
        const willReachBottom = elementBottomInContainer > containerScrollTop + containerHeight;
        if (willReachBottom) {
          targetScrollTop = elementBottomInContainer - containerHeight;
          scrollBoundaryRef.current.isAtBottom = true;
        } else {
          targetScrollTop = containerScrollTop;
        }
      }
    } else if (direction === 'up') {
      if (boundary.isAtTop) {
        targetScrollTop = elementTopInContainer;
      } else {
        const willReachTop = elementTopInContainer < containerScrollTop;
        if (willReachTop) {
          targetScrollTop = elementTopInContainer;
          scrollBoundaryRef.current.isAtTop = true;
        } else {
          targetScrollTop = containerScrollTop;
        }
      }
    } else {
      if (elementBottomInContainer > containerScrollTop + containerHeight) {
        targetScrollTop = elementBottomInContainer - containerHeight;
      } else if (elementTopInContainer < containerScrollTop) {
        targetScrollTop = elementTopInContainer;
      }
    }

    targetScrollTop = Math.max(0, Math.min(targetScrollTop, container.scrollHeight - containerHeight));
    if (Math.abs(containerScrollTop - targetScrollTop) > 1) {
      container.scrollTo({
        top: targetScrollTop,
        behavior: 'smooth'
      });
    }

  }, [activeIndex]);

  useEffect(() => {
    if (loadNext) {
      if (data.length > 0) {
        setDataSource((prev) => [...prev, ...data].filter(item => !!item?.suggestion));
        hasMoreRefs.current = true;
        setHasMore(true);
      } else {
        hasMoreRefs.current = false;
        setHasMore(false);
      }
    } else {
      setDataSource(data.filter(item => !!item?.suggestion));
    }
  }, [data]);

  // 本地键盘事件：移除取模运算，限制索引边界
  useEffect(() => {
    if (useGlobalKeydown || !onItemClick) return;

    const handleKeyDown = (e) => {
      if (![38, 40, 13].includes(e.keyCode)) return;

      const totalItems = dataSource.length;
      if (totalItems === 0) return;

      e.preventDefault();

      switch (e.keyCode) {
        case 40: // 下键：索引最大到 totalItems - 1
          keyDirectionRef.current = 'down';
          setActiveIndex((prev) => {
            // 移除 % 循环 → 边界限制：不能超过最后一项索引
            if (prev === -1) return 0;
            if (prev >= totalItems - 1) return totalItems - 1;
            return prev + 1;
          });
          break;
        case 38: // 上键：索引最小到 0
          keyDirectionRef.current = 'up';
          setActiveIndex((prev) => {
            // 移除 % 循环 → 边界限制：不能小于 0
            if (prev <= 0) return 0;
            if (prev === -1) return -1;
            return prev - 1;
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
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [useGlobalKeydown, dataSource, activeIndex, onItemClick]);

  useEffect(() => {
    itemRefs.current = itemRefs.current.slice(0, dataSource.length);
  }, [dataSource]);

  useImperativeHandle(ref, () => ({
    triggerItemClick: (index) => {
      if (index >= 0 && index < itemRefs.current.length) {
        itemRefs.current[index]?.click();
      }
    },
    getListLength: () => dataSource.length,
    scrollToActiveItem: (index, direction = 'none') => {
      keyDirectionRef.current = direction;
      setActiveIndex(index);
    },
    setKeyDirection: (direction) => {
      keyDirectionRef.current = direction;
    }
  }));

  if (dataSource.length === 0) return null;

  const scrollID = `${type}-scroll-container`;

  const handleItemClick = (item, index) => {
    keyDirectionRef.current = 'none';
    scrollBoundaryRef.current = { isAtTop: false, isAtBottom: false };
    
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
        <div className="py-11px px-16px text-12px text-[var(--ui-search-antd-color-text-description)]">
          {title}
        </div>
      )}
      <div
        ref={scrollContainerRef}
        id={scrollID}
        className="px-8px mb-12px overflow-auto"
        style={{ 
          maxHeight: 40 * defaultRows,
          scrollBehavior: 'smooth'
        }}
      >
        <InfiniteScroll
          dataLength={dataSource.length}
          next={() => loadNext && hasMore && hasMoreRefs.current && loadNext()}
          hasMore={hasMore}
          scrollableTarget={scrollID}
          scrollThreshold={1}
          useWindow={false}
        >
          <List
            itemLayout="vertical"
            size="large"
            dataSource={dataSource}
            renderItem={(item, index) => {
              const isActive = activeIndex === index;
              return (
                <div key={index}>
                  <div
                    ref={el => itemRefs.current[index] = el}
                    className={`${styles.listItem} ${isActive ? styles.active : ''} cursor-pointer relative h-40px pl-8px pr-40px flex flex-nowrap items-center rounded-8px 
                    hover:bg-[rgba(233,240,254,1)] 
                    ${isActive ? "bg-[rgba(233,240,254,1)]" : ""}`}
                    onClick={() => handleItemClick(item, index)}
                  >
                    {renderPrefix?.(item)}
                    {item.icon && (
                      <BasicIcon 
                        className={"w-16px h-16px mr-8px text-[var(--ui-search-antd-color-text-description)] flex-shrink-0"} 
                        icon={item.icon}
                      />
                    )}
                    <div className="mr-12px flex-shrink-1 max-w-[100%] min-w-0">
                      <div className="truncate whitespace-nowrap">{item.suggestion}</div>
                    </div>
                    {item.source && (
                      <Typography.Text type="secondary" style={{ width: 200 }} ellipsis>
                        {item.source}
                      </Typography.Text>
                    )}
                    {onItemClick && (
                      <Button
                        className={`${styles.enter} absolute right-8px top-8px !w-24px !h-24px rounded-8px border-0`}
                        classNames={{ icon: `w-14px h-14px !text-14px` }}
                        size="small"
                        icon={<CornerDownLeft className="w-14px h-14px" />}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleItemClick(item, index);
                        }}
                      />
                    )}
                  </div>
                </div>
              );
            }}
          />
        </InfiniteScroll>
      </div>
    </>
  );
});

export default ListContainer;