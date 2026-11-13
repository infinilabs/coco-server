import { type FC } from "react";

import loadingSvg from "../icons/file-loading.svg";

interface LoadingIconProps {
  /** pixel size of the square icon */
  size?: number;
  className?: string;
}

/**
 * Unified loading indicator — the self-animating stroke-drawing file icon.
 * The SVG carries its own CSS animation (embedded <style>), so it animates
 * both inline and inside <img> without any external styles.
 */
const LoadingIcon: FC<LoadingIconProps> = ({ size = 48, className = "" }) => (
  <img
    src={loadingSvg}
    alt=""
    aria-hidden="true"
    draggable={false}
    className={className}
    style={{ width: size, height: size }}
  />
);

export default LoadingIcon;
