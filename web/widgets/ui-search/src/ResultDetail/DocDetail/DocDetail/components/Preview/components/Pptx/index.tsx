import { useEffect, useRef, type FC } from "react";
import { init } from "pptx-preview";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";


interface PptxProps extends DocDetailProps {
  url: string;
}

const Pptx: FC<PptxProps> = (props) => {
  const { url, requestHeaders } = props;

  const containerRef = useRef<HTMLDivElement>(null);

  const renderPptx = async () => {
    if (!containerRef.current) return;

    containerRef.current.innerHTML = "";

    const width = containerRef.current.clientWidth;
    const height = Math.round(width * (9 / 16));

    const pptx = init(containerRef.current, {
      width,
      height,
    });

    const response = await fetch(url, {
      headers: requestHeaders,
    });

    if (!response.ok) return;

    const arrayBuffer = await response.arrayBuffer();

    if (arrayBuffer.byteLength === 0) return;

    pptx.preview(arrayBuffer);
  };

  useEffect(() => {
    renderPptx();
  }, [url]);

  return (
    <div
      ref={containerRef}
      className="w-full [&_.pptx-preview-wrapper]:(overflow-hidden)"
    />
  );
};

export default Pptx;
