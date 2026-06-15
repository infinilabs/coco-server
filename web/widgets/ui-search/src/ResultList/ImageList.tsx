import { memo, useMemo, useRef, useState, useEffect, useCallback, type FC } from "react";
import { Masonry, Skeleton, Spin } from "antd";
import { ChevronRight, ImageOff } from "lucide-react";
import { useInViewport, useSize } from "ahooks";
import clsx from "clsx";
import ResultDetail from "../ResultDetail";
import { AuthImage } from "./AuthImage";

interface MasonryItemProps {
  data: Record<string, any>;
  onItemClick: (data: Record<string, any>) => void;
  apiConfig?: Record<string, any>;
}

const MasonryItem: FC<MasonryItemProps> = (props) => {
  const { data, onItemClick, apiConfig } = props;
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [inViewport] = useInViewport(imgRef);
  const [loaded, setLoaded] = useState(true);
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

  return (
    <div ref={containerRef} onClick={() => onItemClick(data)} className="group relative cursor-pointer">
      <div
        className="relative w-full rounded-lg overflow-hidden"
        style={{
          aspectRatio,
        }}
      >
        <Skeleton.Node
          active={!loaded && !errored}
          classNames={{
            root: "size-full!",
            content: "size-full!",
          }}
        />

        <AuthImage
          ref={imgRef}
          src={imgSrc}
          alt={data?.title}
          className={clsx(
            "absolute inset-0 size-full object-cover transition",
            {
              "opacity-100": loaded,
              "opacity-0": !loaded
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
            "absolute inset-0 size-full flex items-center justify-center opacity-0",
            {
              "opacity-100": errored,
            },
          )}
        >
          <ImageOff className="text-[#999] dark:text-[#666]" />
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
        <div className="text-3.5">{data?.title}</div>

        <div className="inline-flex items-center flex-wrap gap-0.5 text-12px">
          <span>{data?.source?.name}</span>
          <ChevronRight className="size-3" />
          <span>{data?.category}</span>
        </div>
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
  [key: string]: any;
}

export function ImageList(props: ImageListProps) {
  const { getDetailContainer, data = [], isMobile, loading, hasMore, onLoadMore, setDetailCollapse, apiConfig } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState<Record<string, any> | undefined>();
  const masonryContainerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(masonryContainerRef);
  const [columns, setColumns] = useState(2);
  const loadingRef = useRef(loading);
  const hasMoreRef = useRef(hasMore);

  useEffect(() => { loadingRef.current = loading; }, [loading]);
  useEffect(() => { hasMoreRef.current = hasMore; }, [hasMore]);

  useEffect(() => {
    setOpen(false);
    setRecord(undefined);
    setDetailCollapse?.(true);
  }, [data]);

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
  }, [data.length, hasMore, loading, getDetailContainer, onLoadMore]);

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
    return data.filter((item) => item.metadata?.content_category === 'image').map((item, index) => ({ key: item.id || index, data: item }));
  }, [data]);

  const itemRender = useCallback((item: any) => {
    return <MasonryItem data={item.data} onItemClick={(item) => onOpen(item)} apiConfig={apiConfig} />;
  }, [onOpen, apiConfig]);

  return (
    <>
      <div ref={masonryContainerRef} style={{ width: '100%' }}>
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
        {loading && hasMore && (
          <div style={{
            textAlign: 'center',
            padding: '16px 0',
            marginTop: '8px',
          }}>
            <Spin />
          </div>
        )}
      </div>
      
      <ResultDetail 
        getContainer={getDetailContainer}
        open={open}
        onClose={onClose}
        data={record || {}}
        isMobile={isMobile}
        apiConfig={apiConfig}
      />
    </>
  );
}

export default memo(ImageList);