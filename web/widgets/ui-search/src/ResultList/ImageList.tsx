import { memo, useMemo, useRef, useState, useEffect, useCallback, type FC } from "react";
import { Masonry, Tooltip } from "antd";
import { ChevronRight, ImageOff } from "lucide-react";
import { useInViewport, useSize } from "ahooks";
import { useTranslation } from "react-i18next";
import clsx from "clsx";
import ResultDetail from "../ResultDetail";
import LoadingIcon from "../components/LoadingIcon";
import { AuthImage } from "./AuthImage";
import { EndList } from "./EndList";

interface MasonryItemProps {
  data: Record<string, any>;
  onItemClick: (data: Record<string, any>) => void;
  apiConfig?: Record<string, any>;
  /** the tile whose record is shown in the detail preview gets a selection ring */
  isActive?: boolean;
}

const MasonryItem: FC<MasonryItemProps> = (props) => {
  const { data, onItemClick, apiConfig, isActive } = props;
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [inViewport] = useInViewport(imgRef);
  // starts hidden behind the skeleton; only the load event reveals the img, so
  // a missing image never flashes the browser's broken-image icon
  const [loaded, setLoaded] = useState(false);
  const [errored, setErrored] = useState(false);

  const aspectRatio = useMemo(() => {
    const { width, height } = data?.metadata ?? {};
    if (!width || !height) return 4 / 3;
    return width / height;
  }, [data?.metadata]);

  const imgSrc = useMemo(() => {
    if (loaded) return data?.thumbnail;

    return inViewport ? data?.thumbnail : void 0;
  }, [inViewport, loaded, data?.thumbnail]);

  // a card with no thumbnail URL can never load — show the placeholder tile
  // right away instead of an endlessly pulsing skeleton
  const showPlaceholder = errored || !data?.thumbnail;

  const sourceName = data?.source?.name;
  const category = data?.category;

  return (
    <div
      ref={containerRef}
      onClick={() => onItemClick(data)}
      className={clsx(
        "group relative cursor-pointer rounded-lg transition-shadow",
        isActive && "shadow-[0_0_0_2px_var(--ant-color-primary)]"
      )}
    >
      <div
        className="relative w-full rounded-lg overflow-hidden"
        style={{
          aspectRatio,
        }}
      >
        <div
          className={clsx(
            "flex size-full items-center justify-center",
            !loaded && !showPlaceholder ? "bg-black/3 dark:bg-white/4" : ""
          )}
        >
          {!loaded && !showPlaceholder && <LoadingIcon size={28} />}
        </div>

        <AuthImage
          ref={imgRef}
          src={imgSrc}
          alt={data?.title}
          className={clsx(
            "absolute inset-0 size-full object-cover transition",
            {
              "opacity-100": loaded && !showPlaceholder,
              "opacity-0": !loaded || showPlaceholder
            },
          )}
          onLoad={() => {
            setLoaded(true);
          }}
          onError={() => {
            setErrored(true);
          }}
          requestHeaders={apiConfig?.headers}
          loading="lazy"
        />

        <div
          className={clsx(
            "absolute inset-0 size-full flex flex-col items-center justify-center gap-8px bg-[#F7F8FA] text-[#BFBFBF] opacity-0 transition dark:bg-[#1F1F1F] dark:text-[#595959]",
            {
              "opacity-100": showPlaceholder,
            },
          )}
        >
          <ImageOff className="size-8" />
          <div className="text-12px">{t("labels.imageUnavailable")}</div>
        </div>
      </div>

      <div
        className={clsx(
          "absolute left-0 bottom-0 w-full p-3 text-14px text-white opacity-0 transition bg-gradient-to-t from-black/60 to-transparent rounded-b-lg",
          {
            "group-hover:opacity-100": loaded,
          },
        )}
      >
        {data?.title ? (
          // clamp to two lines so a long title can't push the overlay halfway
          // up the image; the full text stays one hover away
          <div className="line-clamp-2 break-words text-3.5 leading-20px" title={data?.title}>
            {data?.title}
          </div>
        ) : null}

        {sourceName || category ? (
          // same treatment as the list view's BreadcrumbsLine: crumbs shrink
          // and ellipsize on one line, the full text on hover
          <div className="mt-4px flex min-w-0 items-center gap-x-4px text-12px text-white/80">
            {sourceName ? (
              <Tooltip title={sourceName}>
                <span className="min-w-0 cursor-default truncate">{sourceName}</span>
              </Tooltip>
            ) : null}
            {sourceName && category ? (
              <ChevronRight className="size-3 shrink-0 opacity-45" />
            ) : null}
            {category ? (
              <Tooltip title={category}>
                <span className="min-w-0 cursor-default truncate">{category}</span>
              </Tooltip>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
};

interface ImageListProps {
  getDetailContainer?: () => HTMLElement;
  data?: Record<string, any>[];
  isMobile?: boolean;
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
  setDetailCollapse?: (v: boolean) => void;
  apiConfig?: Record<string, any>;
  /** clicking a filterable meta entry (category, tag) in the detail drawer re-runs the search */
  onMetaFilter?: (field: string, value: string) => void;
  /**
   * how the open detail renders — 'docked' pins a full-height pane to the
   * right edge beside the grid (wide containers), 'page' takes over the whole
   * area below the header (narrow containers / mobile). There is no floating
   * card variant.
   */
  detailMode?: 'docked' | 'page';
  [key: string]: any;
}

export function ImageList(props: ImageListProps) {
  const { getDetailContainer, data = [], isMobile, loading, hasMore, onLoadMore, setDetailCollapse, apiConfig, onMetaFilter, detailMode = 'page' } = props;
  const { total, settings, onGenerateAnswer, theme } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState<Record<string, any> | undefined>();
  const masonryContainerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(masonryContainerRef);
  const [columns, setColumns] = useState(2);
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
    setDetailCollapse?.(true);
  }, [dataIdentity]);

  useEffect(() => {
    if (!record?.id) return;

    const latestRecord = listData.find((item) => item?.id === record.id);
    if (latestRecord && latestRecord !== record) {
      setRecord(latestRecord);
    }
  }, [listData, record?.id]);

  useEffect(() => {
    const container = getDetailContainer?.();
    if (!container) return;

    const checkAndLoad = () => {
      const { scrollHeight, clientHeight } = container;
      if (scrollHeight - container.scrollTop - clientHeight < 200 && hasMoreRef.current && !loadingRef.current) {
        onLoadMore?.();
      }
    };

    container.addEventListener("scroll", checkAndLoad);
    return () => {
      container.removeEventListener("scroll", checkAndLoad);
    };
  }, [getDetailContainer, onLoadMore]);

  // If content doesn't fill the container (no scrollbar), auto-load more
  useEffect(() => {
    const container = getDetailContainer?.();
    if (!container || !hasMore || loading) return;

    const timer = setTimeout(() => {
      if (container.scrollHeight <= container.clientHeight && hasMoreRef.current && !loadingRef.current) {
        onLoadMore?.();
      }
    }, 200);

    return () => clearTimeout(timer);
  }, [listData.length, hasMore, loading, getDetailContainer, onLoadMore]);

  const calculateColumns = useMemo(() => {
    if (!containerSize?.width) return isMobile ? 1 : 2;
    
    const MIN_ITEM_WIDTH = 300;
    const GUTTER = 16;
    
    let calculatedColumns = Math.floor(containerSize.width / (MIN_ITEM_WIDTH + GUTTER));
    
    calculatedColumns = Math.max(1, Math.min(calculatedColumns, 8));
    
    if (isMobile) {
      calculatedColumns = Math.max(1, calculatedColumns);
    }
    
    return calculatedColumns;
  }, [containerSize?.width, isMobile]);

  useEffect(() => {
    setColumns(calculateColumns);
  }, [calculateColumns]);

  const onOpen = useCallback((record: Record<string, any>) => {
    setRecord(record);
    setOpen(true);
    setDetailCollapse?.(false)
  }, [setDetailCollapse]);

  const onClose = () => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(true)
  };

  const masonryItems = useMemo(() => {
    return listData.filter((item) => item.metadata?.content_category === 'image').map((item, index) => ({ key: item.id || index, data: item }));
  }, [listData]);

  const itemRender = useCallback((item: any) => {
    return (
      <MasonryItem
        data={item.data}
        onItemClick={(item) => onOpen(item)}
        apiConfig={apiConfig}
        isActive={!!record?.id && item.data?.id === record.id}
      />
    );
  }, [onOpen, apiConfig, record]);

  const detail = open && record ? (
    // mirrors the document list's preview: a docked full-height pane beside
    // the grid on wide containers, a full page below the header on narrow
    // ones — never a floating card over the grid
    <div
      className={
        detailMode === 'docked'
          ? 'right-0 bottom-0 fixed z-10 overflow-hidden border-l border-solid border-[var(--ant-color-border-secondary)] bg-[var(--ant-color-bg-container)]'
          : 'left-0 right-0 bottom-0 fixed z-20 overflow-hidden bg-[var(--ant-color-bg-container)]'
      }
      style={detailMode === 'docked' ? { width: 820, top: 64 } : { top: isMobile ? 122 : 64 }}
    >
      <ResultDetail
        inline
        data={record}
        apiConfig={apiConfig}
        theme={theme}
        onClose={onClose}
        onMetaFilter={
          onMetaFilter
            ? (field, value) => {
                // the detail covers the masonry grid — close it so the refreshed results show
                onClose();
                onMetaFilter(field, value);
              }
            : undefined
        }
      />
    </div>
  ) : null;

  return (
    <>
      <div ref={masonryContainerRef} style={{ width: '100%' }}>
        {loading && masonryItems.length === 0 ? (
          // first page in flight — unified loading icon until tiles land
          <div className="flex justify-center py-48px">
            <LoadingIcon size={64} />
          </div>
        ) : (
          <Masonry
            columns={columns}
            gutter={16}
            items={masonryItems}
            itemRender={itemRender}
            fresh
            styles={{
              item: { transition: 'all 0.3s ease' },
            }}
            style={{ width: '100%' }}
          />
        )}
        {loading && hasMore && (
          <div style={{
            textAlign: 'center',
            padding: '16px 0',
            marginTop: '8px',
          }}>
            <LoadingIcon size={40} />
          </div>
        )}
        {!loading && !hasMore && masonryItems.length > 0 && (
          <EndList
            total={total || masonryItems.length}
            settings={settings}
            onGenerateAnswer={onGenerateAnswer}
            padded={false}
          />
        )}
      </div>

      {detail}
    </>
  );
}

export default memo(ImageList);