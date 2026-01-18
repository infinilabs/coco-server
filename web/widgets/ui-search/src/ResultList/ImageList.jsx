import { memo, useMemo, useRef, useState } from "react";
import { Masonry, Skeleton } from "antd";
import { ChevronRight, ImageOff } from "lucide-react";
import { useInViewport, useSize } from "ahooks";
import clsx from "clsx";

const MasonryItem = (props) => {
  const { data } = props;
  const containerRef = useRef(null);
  const containerSize = useSize(containerRef);
  const imgRef = useRef(null);
  const [inViewport] = useInViewport(imgRef);
  const [loaded, setLoaded] = useState(false);
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
    <div ref={containerRef} className="group relative cursor-pointer">
      <div
        className="relative w-full rounded-lg overflow-hidden"
        style={{
          height: calcHeight() || 0,
        }}
      >
        <Skeleton.Node
          active={!errored}
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
            "absolute inset-0 size-full object-cover opacity-0 transition",
            {
              "opacity-100": loaded,
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
  const { getDetailContainer, data = [], isMobile, loading, hasMore } = props;

  return (
    <Masonry
      columns={{
        xl: 6,
        lg: 5,
        md: 4,
        sm: 3,
        xs: 2,
      }}
      gutter={16}
      // items={data.filter((item) => item.metadata?.content_category === 'image')}
      items={data}
      itemRender={(item) => <MasonryItem data={item} />}
    />
  );
}

export default memo(ImageList);
