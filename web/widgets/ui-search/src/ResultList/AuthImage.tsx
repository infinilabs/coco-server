import { useEffect, useRef, useState } from "react";

function isAssetUrl(url: string): boolean {
  try {
    const parsed = new URL(url);
    return parsed.pathname.startsWith("/assets");
  } catch {
    return false;
  }
}

export function AuthImage(props: any) {
  const { src, requestHeaders, ...rest } = props;
  const needsAuth = !!(src && requestHeaders && !isAssetUrl(src));
  const [blobUrl, setBlobUrl] = useState<string | undefined>(undefined);
  const urlRef = useRef<string | undefined>(undefined);

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
        if (!cancelled) setBlobUrl(undefined);
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
    return <img {...props} />;
  }

  return <img {...rest} src={blobUrl} />;
}
