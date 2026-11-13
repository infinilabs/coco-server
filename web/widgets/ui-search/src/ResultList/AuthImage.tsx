import { useEffect, useRef, useState, forwardRef, type ImgHTMLAttributes, type SyntheticEvent } from "react";
import clsx from "clsx";

import { IMAGE_FALLBACK_SRC, IMAGE_LOADING_SRC } from "../utils/media";

function isAssetUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.pathname.startsWith("/assets");
  } catch {
    return false;
  }
}

interface AuthImageProps extends ImgHTMLAttributes<HTMLImageElement> {
  src?: string;
  requestHeaders?: Record<string, string>;
}

export const AuthImage = forwardRef<HTMLImageElement, AuthImageProps>((props, ref) => {
  const { src, requestHeaders, onError, onLoad, ...rest } = props;
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

    fetch(src, { headers: requestHeaders })
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

  if (!needsAuth) {
    return (
      <img
        ref={ref}
        {...rest}
        src={failed ? IMAGE_FALLBACK_SRC : src}
        onError={(event) => {
          if (!failed) setFailed(true);
          onError?.(event);
        }}
      />
    );
  }

  // while the authorized blob is still being fetched there is no src to
  // decode; a src-less <img> draws the browser's broken-image glyph with the
  // alt text, so decode a transparent pixel over a pulsing tile instead
  const loading = !blobUrl && !failed;

  // the loading pixel decodes instantly — forwarding that as the caller's
  // onLoad would drop their skeleton/fade-in before any real pixels exist
  const handleLoad = (event: SyntheticEvent<HTMLImageElement>) => {
    if (loading) return;
    onLoad?.(event);
  };

  return (
    <img
      ref={ref}
      {...rest}
      className={clsx(rest.className, loading && "animate-pulse bg-slate-200/70 dark:bg-slate-700/50")}
      src={failed ? IMAGE_FALLBACK_SRC : blobUrl || IMAGE_LOADING_SRC}
      onLoad={handleLoad}
      onError={(event) => {
        // undecodable blob (e.g. an HTML error page served with 200) — swap in
        // the fallback art rather than letting the browser draw its own
        if (!failed && blobUrl) setFailed(true);
        onError?.(event);
      }}
    />
  );
})
