import { useEffect, useMemo, useRef, useState, type FC } from "react";
import { Image as AntdImage, Skeleton } from "antd";
import { useSize } from "ahooks";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

const Image: FC<DocDetailProps> = (props) => {
  const { data, requestHeaders } = props;
  const containerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(containerRef);
  const [failed, setFailed] = useState(false);
  const [imgSrc, setImgSrc] = useState<string | undefined>(data?.metadata?.raw_content);

  useEffect(() => {
    if (!requestHeaders || !data?.metadata?.raw_content) {
      setImgSrc(failed ? data?.thumbnail : data?.metadata?.raw_content);
      return;
    }

    const targetUrl = failed ? data?.thumbnail : data?.metadata?.raw_content;
    if (!targetUrl) return;

    fetch(targetUrl, { headers: requestHeaders })
      .then((res) => res.blob())
      .then((blob) => setImgSrc(URL.createObjectURL(blob)));
  }, [data?.metadata?.raw_content, data?.thumbnail, failed, requestHeaders]);

  const calcHeight = useMemo(() => {
    const { width, height } = data.metadata ?? {};

    if (!containerSize || !width || !height) {
      return undefined;
    }

    return Math.round((containerSize.width * height) / width);
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
          setFailed(true);
        }}
      />
    </div>
  );
};

export default Image;
