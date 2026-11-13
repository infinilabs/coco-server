import { useEffect, useState, type ImgHTMLAttributes, type SyntheticEvent } from 'react';
import classNames from 'classnames';

import { IMAGE_FALLBACK_SRC } from '@/utils/media';

type ProgressiveImageProps = ImgHTMLAttributes<HTMLImageElement>;

/**
 * <img> that never shows the browser's broken-image glyph: while the source
 * is in flight the element is a pulsing gray tile (markdown images carry no
 * dimensions, so a min box keeps the tile visible), and a load failure swaps
 * in the shared fallback art instead of the torn-picture icon.
 */
export default function ProgressiveImage(props: ProgressiveImageProps) {
  const { src, alt, title, className, onLoad, onError, ...rest } = props;
  const [loaded, setLoaded] = useState(false);
  const [failed, setFailed] = useState(false);

  // a new src is a fresh chance — clear both flags from the previous one
  useEffect(() => {
    setLoaded(false);
    setFailed(false);
  }, [src]);

  const pending = !loaded && !failed;

  return (
    <img
      {...rest}
      alt={alt}
      title={failed ? undefined : title ?? alt}
      className={classNames(
        'max-w-full',
        className,
        pending && 'min-h-[96px] min-w-[96px] animate-pulse rounded-6px bg-#EDEEF1 object-contain dark:bg-[#2F3033]',
        failed && 'min-h-[96px] min-w-[96px] rounded-6px'
      )}
      src={failed ? IMAGE_FALLBACK_SRC : src}
      loading="lazy"
      onLoad={(event: SyntheticEvent<HTMLImageElement>) => {
        setLoaded(true);
        onLoad?.(event);
      }}
      onError={(event: SyntheticEvent<HTMLImageElement>) => {
        setFailed(true);
        onError?.(event);
      }}
    />
  );
}
