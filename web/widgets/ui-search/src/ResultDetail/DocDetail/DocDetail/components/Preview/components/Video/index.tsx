import { useEffect, useState, type FC } from "react";

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
    if (!requestHeaders) {
      setSrc(url);
      return;
    }

    onLoadingChange?.(true);
    fetch(url, { headers: requestHeaders })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => setSrc(URL.createObjectURL(blob)))
      .catch((e) => {
        onLoadError?.(e instanceof Error ? e : new Error(String(e)));
      })
      .finally(() => onLoadingChange?.(false));
  }, [url, requestHeaders]);

  return <video src={src} className="w-full" controls />;
};

export default Video;
