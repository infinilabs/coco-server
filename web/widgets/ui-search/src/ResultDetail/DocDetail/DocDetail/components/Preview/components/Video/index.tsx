import { useEffect, useState, type FC } from "react";

interface VideoProps {
  url: string;
  requestHeaders?: Record<string, string>;
  onLoadingChange?: (loading: boolean) => void;
}

const Video: FC<VideoProps> = (props) => {
  const { url, requestHeaders, onLoadingChange } = props;

  const [src, setSrc] = useState<string>(url);

  useEffect(() => {
    if (!requestHeaders) {
      setSrc(url);
      return;
    }

    onLoadingChange?.(true);
    fetch(url, { headers: requestHeaders })
      .then((res) => res.blob())
      .then((blob) => setSrc(URL.createObjectURL(blob)))
      .catch(() => {})
      .finally(() => onLoadingChange?.(false));
  }, [url, requestHeaders]);

  return <video src={src} className="w-full" controls />;
};

export default Video;
