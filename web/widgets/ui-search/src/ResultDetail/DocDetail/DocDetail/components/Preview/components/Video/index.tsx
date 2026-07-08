import { useEffect, useState, type FC } from "react";

import { isBlobUrl } from "../../utils";

interface VideoProps {
  url: string;
  requestHeaders?: Record<string, string>;
  onLoadingChange?: (loading: boolean) => void;
  onLoadError?: (error: Error) => void;
}

const Video: FC<VideoProps> = (props) => {
  const { url, requestHeaders, onLoadingChange, onLoadError } = props;

  const [src, setSrc] = useState<string>(url);

  useEffect(() => {
    if (!url) return;

    if (isBlobUrl(url)) {
      setSrc(url);
      onLoadingChange?.(false);
      return;
    }

    if (!requestHeaders) {
      setSrc(url);
      return;
    }

    onLoadingChange?.(true);
    let objectUrl = "";

    fetch(url, { headers: requestHeaders })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        setSrc(objectUrl);
      })
      .catch((e) => {
        onLoadError?.(e instanceof Error ? e : new Error(String(e)));
      })
      .finally(() => onLoadingChange?.(false));

    return () => {
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [url, requestHeaders]);

  return <video src={src} className="w-full" controls />;
};

export default Video;
