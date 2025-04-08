import React from 'react';

import { useIconfontScript } from '@/hooks/common/script';

const FontIcon = ({ className, name, style, ...rest }) => {
  useIconfontScript();
  return (
    <svg
      className={`icon ${className || ''}`}
      style={style}
      {...rest}
    >
      <use xlinkHref={`#${name}`} />
    </svg>
  );
};

export default FontIcon;
