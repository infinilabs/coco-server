import { memo, useEffect, useMemo, useRef } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import styles from './NormalList.module.less';
import LoadingIcon from '../components/LoadingIcon';
import { EndList } from './EndList';
import SkeletonList from './SkeletonList';
import SearchResults from './SearchResults';

interface NormalListProps {
  /** scroll container for virtualization (the widget root) */
  readonly getDetailContainer?: () => HTMLElement;
  readonly data?: Record<string, any>[];
  readonly isMobile?: boolean;
  readonly query?: string;
  readonly loading?: boolean;
  readonly hasMore?: boolean;
  readonly onLoadMore?: () => void;
  /** the record currently shown in the layout's preview pane (state owned by the page) */
  readonly detailRecord?: Record<string, any>;
  readonly onOpenDetail?: (record: Record<string, any>) => void;
  readonly onCloseDetail?: () => void;
  readonly apiConfig?: Record<string, any>;
  readonly theme?: 'auto' | 'dark' | 'light';
  [key: string]: any;
}

// real cards measure 131–153px (summary length drives the difference); 144 is
// the calibrated midpoint. The estimate is refined from measured rows and kept
// module-level, so a later search's first paint positions items almost exactly
// and the virtualizer's measure pass no longer shifts them
const ESTIMATED_ITEM_HEIGHT = 144;
let calibratedItemHeight = ESTIMATED_ITEM_HEIGHT;

export function NormalList(props: NormalListProps) {
  const {
    getDetailContainer,
    data = [],
    loading,
    hasMore,
    onLoadMore,
    detailRecord,
    onOpenDetail,
    onCloseDetail,
    apiConfig
  } = props;
  const { total, settings, onGenerateAnswer } = props;

  const listWrapperRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(loading);
  const hasMoreRef = useRef(hasMore);
  const appIntegrationId = apiConfig?.headers?.['APP-INTEGRATION-ID'] || apiConfig?.headers?.['app-integration-id'];
  const listData = useMemo<Record<string, any>[]>(
    () =>
      data.map(item => ({
        ...item,
        url: appIntegrationId && item?.url ? `${item.url}?app-integration-id=${appIntegrationId}` : item?.url
      })),
    [appIntegrationId, data]
  );
  const dataIdentity = useMemo(() => listData.map(item => item?.id).join('|'), [listData]);

  useEffect(() => {
    loadingRef.current = loading;
  }, [loading]);
  useEffect(() => {
    hasMoreRef.current = hasMore;
  }, [hasMore]);

  useEffect(() => {
    onCloseDetail?.();
  }, [dataIdentity, onCloseDetail]);

  useEffect(() => {
    if (!detailRecord?.id) return;

    const latestRecord = listData.find(item => item?.id === detailRecord.id);
    if (latestRecord && latestRecord !== detailRecord) {
      onOpenDetail?.(latestRecord);
    }
  }, [listData, detailRecord, onOpenDetail]);

  const scrollElement = getDetailContainer?.() ?? null;

  const virtualizer = useVirtualizer({
    count: listData.length,
    getScrollElement: () => scrollElement,
    estimateSize: () => calibratedItemHeight,
    overscan: 5,
    scrollMargin: listWrapperRef.current?.offsetTop ?? 0
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
    onOpenDetail?.(record);
  };

  // first page still in flight — placeholder rows that mirror the result
  // cards' boxes keep the page's shape (a lone centered spinner leaves the
  // whole upper list area blank while the first page loads)
  if (loading && listData.length === 0) {
    return (
      <div className={styles.list} ref={listWrapperRef}>
        <SkeletonList />
      </div>
    );
  }

  return (
    <div
      className={styles.list}
      ref={listWrapperRef}
    >
      <div
        style={{
          height: virtualizer.getTotalSize(),
          width: '100%',
          position: 'relative'
        }}
      >
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            transform: `translateY(${(virtualItems[0]?.start ?? 0) - virtualizer.options.scrollMargin}px)`
          }}
        >
          {virtualItems.map(virtualRow => {
            const item = listData[virtualRow.index];
            if (!item) return null;
            const isActive = item.id === detailRecord?.id;
            return (
              <div
                data-index={virtualRow.index}
                data-result-id={item.id}
                key={item.id}
                ref={el => {
                  if (el) {
                    const height = el.getBoundingClientRect().height;
                    if (height > 0) {
                      calibratedItemHeight = Math.round((calibratedItemHeight + height) / 2);
                    }
                  }
                  return virtualizer.measureElement(el);
                }}
              >
                <SearchResults
                  requestHeaders={apiConfig?.headers}
                  section={
                    {
                      ...item,
                      isActive,
                      href: item?.url
                    } as any
                  }
                  onRecordClick={(record: any) => {
                    onOpen(record);
                  }}
                />
                {virtualRow.index < listData.length - 1 && (
                  <div className='mx-16px border-b border-solid border-slate-200/70 dark:border-slate-700/50' />
                )}
              </div>
            );
          })}
        </div>
      </div>
      {loading && hasMore && (
        <div
          style={{
            textAlign: 'center',
            padding: '16px 0',
            marginTop: '8px'
          }}
        >
          <LoadingIcon size={40} />
        </div>
      )}
      {!loading && !hasMore && listData.length > 0 && (
        <EndList
          settings={settings}
          total={total || listData.length}
          onGenerateAnswer={onGenerateAnswer}
        />
      )}
    </div>
  );
}

export default memo(NormalList);
