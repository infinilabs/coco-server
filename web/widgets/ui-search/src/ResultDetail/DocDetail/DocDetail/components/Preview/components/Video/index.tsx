import { useEffect, useState, type FC } from "react";

interface VideoProps {
  url: string;
  requestHeaders?: Record<string, string>;
}

const Video: FC<VideoProps> = (props) => {
  const { url, requestHeaders } = props;

  const [src, setSrc] = useState<string>(url);

  useEffect(() => {
    if (!requestHeaders) {
      setSrc(url);
      return;
    }

    fetch(url, { headers: requestHeaders })
      .then((res) => res.blob())
      .then((blob) => setSrc(URL.createObjectURL(blob)));
  }, [url, requestHeaders]);

  return <video src={src} className="w-full" controls />;
};

export default Video;
