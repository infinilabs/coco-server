import { Spin } from "antd";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import ResultDetail from "../ResultDetail";
import styles from "./NormalList.module.less";
import { EndList } from "./EndList";
import SearchResults from "./SearchResults";

interface NormalListProps {
  getDetailContainer?: () => HTMLElement;
  data?: Record<string, any>[];
  isMobile?: boolean;
  query?: string;
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
  setDetailCollapse?: (v: boolean) => void;
  apiConfig?: Record<string, any>;
  theme?: "light" | "dark" | "auto";
  [key: string]: any;
}

const ESTIMATED_ITEM_HEIGHT = 120;

export function NormalList(props: NormalListProps) {
  const {
    getDetailContainer,
    data = [],
    isMobile,
    query,
    loading,
    hasMore,
    onLoadMore,
    setDetailCollapse,
    apiConfig,
    theme,
  } = props;
  const { total, settings, onGenerateAnswer } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState<Record<string, any> | undefined>();
  const listWrapperRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(loading);
  const hasMoreRef = useRef(hasMore);
  const appIntegrationId = apiConfig?.headers?.['APP-INTEGRATION-ID'] || apiConfig?.headers?.['app-integration-id'];
  const listData = useMemo<Record<string, any>[]>(() => data.map((item) => ({
    ...item,
    url: appIntegrationId && item?.url ? `${item.url}?app-integration-id=${appIntegrationId}` : item?.url,
  })), [appIntegrationId, data]);
  const dataIdentity = useMemo(() => listData.map((item) => item?.id).join('|'), [listData]);

  useEffect(() => { loadingRef.current = loading; }, [loading]);
  useEffect(() => { hasMoreRef.current = hasMore; }, [hasMore]);

  useEffect(() => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(false);
  }, [dataIdentity]);

  useEffect(() => {
    if (!record?.id) return;

    const latestRecord = listData.find((item) => item?.id === record.id);
    if (latestRecord && latestRecord !== record) {
      setRecord(latestRecord);
    }
  }, [listData, record?.id]);

  const scrollElement = getDetailContainer?.() ?? null;

  const virtualizer = useVirtualizer({
    count: listData.length,
    getScrollElement: () => scrollElement,
    estimateSize: () => ESTIMATED_ITEM_HEIGHT,
    overscan: 5,
    scrollMargin: listWrapperRef.current?.offsetTop ?? 0,
  });

  const virtualItems = virtualizer.getVirtualItems();

  useEffect(() => {
    const lastItem = virtualizer.getVirtualItems().at(-1);
    if (!lastItem) return;
    if (lastItem.index >= listData.length - 3 && hasMoreRef.current && !loadingRef.current) {
      onLoadMore?.();
    }
  }, [virtualizer.getVirtualItems(), listData.length, onLoadMore]);

  const onOpen = (record: Record<string, any>) => {
    setRecord(record);
    setOpen(true);
    setDetailCollapse?.(true);
  };

  const onClose = () => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(false);
  };

  return (
    <>
      <div className={styles.list} ref={listWrapperRef}>
        <div
          style={{
            height: virtualizer.getTotalSize(),
            width: "100%",
            position: "relative",
          }}
        >
          <div
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              width: "100%",
              transform: `translateY(${(virtualItems[0]?.start ?? 0) - virtualizer.options.scrollMargin}px)`,
            }}
          >
            {virtualItems.map((virtualRow) => {
              const item = listData[virtualRow.index];
              if (!item) return null;
              const isActive = item.id === record?.id;
              return (
                <div
                  key={item.id}
                  data-index={virtualRow.index}
                  ref={virtualizer.measureElement}
                >
                  <SearchResults
                    section={{
                      ...item,
                      isActive,
                      href: item?.url,
                    } as any}
                    onRecordClick={(record: any) => {
                      onOpen(record);
                    }}
                    requestHeaders={apiConfig?.headers}
                  />
                </div>
              );
            })}
          </div>
        </div>
        {loading && hasMore && (
          <div style={{
            textAlign: 'center',
            padding: '16px 0',
            marginTop: '8px',
          }}>
            <Spin />
          </div>
        )}
        {!loading && !hasMore && listData.length > 0 && (
          <EndList
            total={total || listData.length}
            settings={settings}
            onGenerateAnswer={onGenerateAnswer}
          />
        )}
      </div>
      <ResultDetail
        getContainer={getDetailContainer}
        open={open}
        onClose={onClose}
        data={record || {}}
        isMobile={isMobile}
        apiConfig={apiConfig}
        theme={theme}
      />
    </>
  );
}

export default memo(NormalList);