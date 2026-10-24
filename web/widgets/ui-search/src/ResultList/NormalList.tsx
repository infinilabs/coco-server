import { memo, useEffect, useMemo, useRef, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { useTranslation } from 'react-i18next';
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
                {/* duplicate fold (W12): same content_hash copies collapsed
                    onto this hit by the server; the review stays on the
                    dedup report — here it is show-and-jump only */}
                <DuplicateFold item={item} />
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

/* ---------------- duplicate fold (W12) ---------------- */

interface FoldMember {
  id: string;
  title?: string;
  source?: string;
  updated?: string;
}

function DuplicateFold({ item }: { item: any }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const dupes = item?.metadata?.fingerprint_duplicates;
  const count = dupes?.count ?? 0;
  const members: FoldMember[] = dupes?.members ?? [];
  if (!count || count < 1) return null;

  return (
    <div className='mx-16px mb-4px flex flex-col gap-2px'>
      <button
        type='button'
        className='text-12px text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 text-left'
        onClick={() => setOpen(v => !v)}
      >
        {open
          ? t('labels.duplicatesHide', { count, tier: dupes?.tier ?? 'exact' })
          : t('labels.duplicatesFold', { count, tier: dupes?.tier ?? 'exact' })}
      </button>
      {open && (
        <div className='flex flex-col gap-2px pl-8px border-l border-solid border-slate-200 dark:border-slate-700'>
          {members.map(m => (
            <span key={m.id} className='text-12px text-slate-500 dark:text-slate-400 truncate'>
              · {m.title || m.id}
              {m.source ? ` — ${m.source}` : ''}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
