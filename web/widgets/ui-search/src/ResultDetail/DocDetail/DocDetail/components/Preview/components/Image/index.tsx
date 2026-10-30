import { useEffect, useMemo, useRef, useState, type FC } from "react";
import { Image as AntdImage } from "antd";
import LoadingIcon from "../../../../../../../components/LoadingIcon";
import { filesize } from "filesize";
import { useSize } from "ahooks";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";
import { isBlobUrl } from "../../utils";
import {
  createNamedObjectUrl,
  ensureFilenameExtension,
  extensionFromMime,
  IMAGE_FALLBACK_SRC,
  revokeObjectUrlLater
} from "../../../../../../../utils/media";

interface ImageProps extends DocDetailProps {
  onLoadError?: (error: Error) => void;
}

const Image: FC<ImageProps> = (props) => {
  const { data, requestHeaders, onLoadError } = props;
  const containerRef = useRef<HTMLDivElement>(null);
  const containerSize = useSize(containerRef);
  const [imgSrc, setImgSrc] = useState<string | undefined>();

  useEffect(() => {
    const targetUrl = data?.metadata?.raw_content || data?.thumbnail;
    if (!targetUrl) return;

    if (isBlobUrl(targetUrl)) {
      setImgSrc(targetUrl);
      return;
    }

    // images render their own calm placeholder below, so they don't drive the
    // shared full-area loading overlay (which is built for heavier documents)
    let objectUrl = "";

    fetch(targetUrl, { headers: requestHeaders })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => {
        // a misrouted request (e.g. dev proxy miss falling through to the SPA,
        // or the external-URL JSON wrapper) still resolves with 200 + HTML/JSON;
        // handing that to <img> yields an undecodable blob URL — surface it here
        if (blob.type.startsWith("text/html") || blob.type.startsWith("application/json")) {
          throw new Error(`Unexpected content type: ${blob.type || "unknown"}`);
        }
        // carry the document name + extension in the URL fragment so an
        // address copied from the viewer (or a later save) isn't a bare UUID
        const { file_extension, mime_type } = data?.metadata ?? {};
        const ext = file_extension?.replace(/^\./, "") || extensionFromMime(mime_type) || extensionFromMime(blob.type);
        objectUrl = createNamedObjectUrl(blob, ensureFilenameExtension(data?.title || "image", ext));
        setImgSrc(objectUrl);
      })
      .catch((e) => {
        onLoadError?.(e instanceof Error ? e : new Error(String(e)));
      });

    return () => {
      if (objectUrl) {
        // deferred, not immediate: revoking on unmount would kill the blob for
        // a tab the user just opened via "open in new tab" from the viewer
        revokeObjectUrlLater(objectUrl);
      }
    };
  }, [data?.metadata?.raw_content, data?.thumbnail, data?.title, data?.metadata?.file_extension, data?.metadata?.mime_type, requestHeaders]);

  // fit inside the measured box while keeping the document's aspect ratio —
  // callers cap the available height (e.g. a quarter of the detail pane for
  // inline image previews), so a width-derived height alone can overflow it
  const fittedSize = useMemo(() => {
    if (!containerSize?.width) return undefined;

    const { width, height } = data.metadata ?? {};
    const aspect = width && height ? width / height : 4 / 3;

    let w = containerSize.width;
    let h = w / aspect;

    const available = containerSize.height;
    if (available && h > available) {
      h = available;
      w = h * aspect;
    }

    return { width: Math.round(w), height: Math.round(h) };
  }, [containerSize?.width, containerSize?.height, data?.metadata?.width, data?.metadata?.height]);

  // compact info line shown as a badge over the image corner: dimensions,
  // format, file size — whatever the document actually carries
  const infoLine = useMemo(() => {
    const { width, height, file_extension, mime_type } = data?.metadata ?? {};
    const parts: string[] = [];
    if (width && height) parts.push(`${width} × ${height}`);
    const format = String(file_extension || mime_type?.split("/")?.[1] || "").toUpperCase();
    if (format) parts.push(format);
    if (typeof data?.size === "number" && data.size > 0) parts.push(filesize(data.size));
    else if (typeof data?.size === "string" && data.size) parts.push(data.size);
    return parts.length ? parts.join(" · ") : undefined;
  }, [
    data?.metadata?.width,
    data?.metadata?.height,
    data?.metadata?.file_extension,
    data?.metadata?.mime_type,
    data?.size
  ]);

  return (
    <div
      ref={containerRef}
      className='flex h-full w-full items-center justify-center'
    >
      {imgSrc ? (
        <div
          className='relative'
          style={{ width: fittedSize?.width, height: fittedSize?.height }}
        >
          <AntdImage
            // the inline preview is capped to a fraction of the pane, so keep
            // click-to-zoom available for reading the full-size image
            width={fittedSize?.width}
            height={fittedSize?.height}
            src={imgSrc}
            // shown in both the inline box and the zoom overlay until the
            // image decodes, so neither flashes a broken-image icon
            placeholder={(
              <div className="flex size-full items-center justify-center bg-black/3 dark:bg-white/4">
                <LoadingIcon size={28} />
              </div>
            )}
            // decode failures (truncated blob, exotic format) draw the shared
            // placeholder art instead of the browser's broken-image glyph
            fallback={IMAGE_FALLBACK_SRC}
            onError={() => {
              onLoadError?.(new Error("Image load failed"));
            }}
          />
          {infoLine && (
            <div className='absolute bottom-6px left-6px z-1 rounded-md bg-black/45 px-6px py-2px text-11px leading-14px text-white/90 backdrop-blur-sm'>
              {infoLine}
            </div>
          )}
        </div>
      ) : (
        // rendered while the blob URL is being fetched; a self-drawn panel is
        // used because antd Image drops its placeholder once the empty src is
        // flagged invalid, leaving a blank flash otherwise
        <div
          className="flex items-center justify-center rounded-lg bg-black/3 dark:bg-white/4"
          style={{ width: fittedSize?.width, height: fittedSize?.height }}
        >
          <LoadingIcon size={40} />
        </div>
      )}
    </div>
  );
};

export default Image;
