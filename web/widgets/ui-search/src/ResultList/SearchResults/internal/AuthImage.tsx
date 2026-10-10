import { useContext, useEffect, useRef, useState } from "react";
import type React from "react";
import clsx from "clsx";

import { RequestHeadersContext } from "./RequestHeadersContext";
import { IMAGE_FALLBACK_SRC, IMAGE_LOADING_SRC } from "../../../utils/media";

type AuthImageProps = React.ImgHTMLAttributes<HTMLImageElement>;

function isAssetUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.pathname.startsWith("/assets");
  } catch {
    return false;
  }
}

export function AuthImage(props: AuthImageProps) {
  const requestHeaders = useContext(RequestHeadersContext);
  const { src, onError, onLoad, ...rest } = props;
  const needsAuth = !!(src && requestHeaders && !isAssetUrl(src));
  const [blobUrl, setBlobUrl] = useState<string | undefined>(undefined);
  const [failed, setFailed] = useState(false);
  const urlRef = useRef<string | undefined>(undefined);

  // a new src is a fresh chance — clear the fallback from the previous one
  useEffect(() => {
    setFailed(false);
  }, [src]);

  useEffect(() => {
    if (!src || !needsAuth) return;

    let cancelled = false;

    fetch(src, { headers: requestHeaders! })
      .then((res) => res.blob())
      .then((blob) => {
        if (cancelled) return;
        const url = URL.createObjectURL(blob);
        urlRef.current = url;
        setBlobUrl(url);
      })
      .catch(() => {
        // show the designed fallback instead of an empty/blank frame
        if (!cancelled) setFailed(true);
      });

    return () => {
      cancelled = true;
      if (urlRef.current) {
        URL.revokeObjectURL(urlRef.current);
        urlRef.current = undefined;
      }
    };
  }, [src, needsAuth, requestHeaders]);

  const handleImageError = (event: React.SyntheticEvent<HTMLImageElement>) => {
    // missing remote image or undecodable blob — swap in the fallback art
    // rather than letting the browser draw its broken-image icon
    if (!failed && (!needsAuth || blobUrl)) setFailed(true);
    onError?.(event);
  };

  // while the authorized blob is still being fetched there is no src to
  // decode; a src-less <img> draws the browser's broken-image glyph with the
  // alt text, so decode a transparent pixel over a pulsing tile instead
  const loading = !blobUrl && !failed;

  // the loading pixel decodes instantly — forwarding that as the caller's
  // onLoad would drop their skeleton/fade-in before any real pixels exist
  const handleLoad = (event: React.SyntheticEvent<HTMLImageElement>) => {
    if (loading) return;
    onLoad?.(event);
  };

  if (!needsAuth) {
    return <img {...props} src={failed ? IMAGE_FALLBACK_SRC : src} onError={handleImageError} />;
  }

  return (
    <img
      {...rest}
      className={clsx(rest.className, loading && "animate-pulse bg-slate-200/70 dark:bg-slate-700/50")}
      src={failed ? IMAGE_FALLBACK_SRC : blobUrl || IMAGE_LOADING_SRC}
      onLoad={handleLoad}
      onError={handleImageError}
    />
  );
}
