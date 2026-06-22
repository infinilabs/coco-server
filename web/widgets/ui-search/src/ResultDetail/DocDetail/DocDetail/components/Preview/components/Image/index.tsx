import { useEffect, useMemo, useRef, useState, type FC } from "react";
import { Image as AntdImage, Skeleton } from "antd";
import { useSize } from "ahooks";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

interface ImageProps extends DocDetailProps {
  onLoadingChange?: (loading: boolean) => void;
  onLoadError?: (error: Error) => void;
}

const Image: FC<ImageProps> = (props) => {
  const { data, requestHeaders, onLoadingChange, onLoadError } = props;
  const containerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(containerRef);
  const [imgSrc, setImgSrc] = useState<string | undefined>();

  useEffect(() => {
    const targetUrl = data?.metadata?.raw_content || data?.thumbnail;
    if (!targetUrl) return;

    onLoadingChange?.(true);
    fetch(targetUrl, { headers: requestHeaders })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => setImgSrc(URL.createObjectURL(blob)))
      .catch((e) => {
        onLoadError?.(e instanceof Error ? e : new Error(String(e)));
      })
      .finally(() => onLoadingChange?.(false));
  }, [data?.metadata?.raw_content, data?.thumbnail, requestHeaders]);

  const calcHeight = useMemo(() => {
    if (!containerSize) return undefined;

    const { width, height } = data.metadata ?? {};

    if (width && height) {
      return Math.round((containerSize.width * height) / width);
    }

    return Math.round(containerSize.width * 3 / 4);
  }, [containerSize?.width, data?.metadata?.width, data?.metadata?.height]);

  return (
    <div ref={containerRef}>
      <AntdImage
        preview={false}
        width={containerSize?.width}
        height={calcHeight}
        placeholder={
          <Skeleton.Node
            active
            classNames={{
              root: "size-full!",
              content: "size-full!",
            }}
          />
        }
        src={imgSrc}
        onError={() => {
          onLoadError?.(new Error("Image load failed"));
        }}
      />
    </div>
  );
};

export default Image;
