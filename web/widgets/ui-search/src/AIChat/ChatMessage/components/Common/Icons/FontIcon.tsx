import type { FC, CSSProperties } from "react";

interface FontIconProps {
  name: string;
  className?: string;
  style?: CSSProperties;
}

const FontIcon: FC<FontIconProps> = ({ name, className, style, ...rest }) => {
  return (
    <svg className={`icon dark:drop-shadow-[0_0_6px_rgb(255,255,255)] ${className || ""}`} style={style} {...rest}>
      <use xlinkHref={`#${name}`} />
    </svg>
  );
};

export default FontIcon;
