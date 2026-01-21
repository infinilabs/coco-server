import { memo, useMemo, useRef, useState, useEffect } from "react";
import { Masonry, Skeleton } from "antd";
import { ChevronRight, ImageOff } from "lucide-react";
import { useInViewport, useSize } from "ahooks";
import clsx from "clsx";
import ResultDetail from "../ResultDetail";
import { Spin } from "antd";

const MasonryItem = (props) => {
  const { data, onItemClick } = props;
  const containerRef = useRef(null);
  const containerSize = useSize(containerRef);
  const imgRef = useRef(null);
  const [inViewport] = useInViewport(imgRef);
  const [loaded, setLoaded] = useState(true);
  const [errored, setErrored] = useState(false);

  const calcHeight = () => {
    const { width, height } = data?.metadata ?? {};

    return Math.round((containerSize?.width * height) / width);
  };

  const imgSrc = useMemo(() => {
    if (loaded) return data?.thumbnail;

    return inViewport ? data?.thumbnail : void 0;
  }, [inViewport, loaded, data?.thumbnail]);

  return (
    <div ref={containerRef} onClick={() => onItemClick(data)} className="group relative cursor-pointer">
      <div
        className="relative w-full rounded-lg overflow-hidden"
        style={{
          height: calcHeight() || 0,
        }}
      >
        <Skeleton.Node
          active={!loaded && !errored}
          classNames={{
            root: "size-full!",
            content: "size-full!",
          }}
        />

        <img
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
        />

        <div
          className={clsx(
            "absolute inset-0 size-full flex items-center justify-center opacity-0",
            {
              "opacity-100": errored,
            },
          )}
        >
          <ImageOff className="text-[#999]" />
        </div>
      </div>

      <div
        className={clsx(
          "absolute left-0 bottom-0 w-full p-3 text-white opacity-0 transition",
          {
            "group-hover:opacity-100": loaded,
          },
        )}
      >
        <div className="text-3.5">{data?.title}</div>

        <div className="inline-flex items-center flex-wrap gap-0.5 text-3">
          <span>{data?.source?.name}</span>
          <ChevronRight className="size-3" />
          <span>{data?.category}</span>
        </div>
      </div>
    </div>
  );
};

export function ImageList(props) {
  const { getDetailContainer, data = [], isMobile, loading, hasMore, setDetailCollapse, getRawContent } = props;

  const [open, setOpen] = useState(false);
  const [record, setRecord] = useState();
  const masonryContainerRef = useRef(null);
  const containerSize = useSize(masonryContainerRef);
  const [columns, setColumns] = useState(2);

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

  const onOpen = (record) => {
    setRecord(record);
    setOpen(true);
    setDetailCollapse(false)
  };

  const onClose = () => {
    setOpen(false);
    setRecord();
    setDetailCollapse(true)
  };

  return (
    <>
      <div ref={masonryContainerRef} style={{ width: '100%' }}>
        <Masonry
          columns={columns}
          gutter={16}
          items={data.filter((item) => item.metadata?.content_category === 'image')}
          itemRender={(item) => <MasonryItem data={item} onItemClick={(item) => onOpen(item)}/>}
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
        getRawContent={getRawContent}
      />
    </>
  );
}

export default memo(ImageList);