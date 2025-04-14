import { Image } from 'antd';
import React from 'react';

import FontIcon from '@/components/common/font_icon';

const Icon = ({ className, src, style, ...rest }) => {
  if (!src) {
    return null;
  }
  if (!src.includes('/')) {
    return (
      <FontIcon
        className={className}
        name={src}
        style={style}
        {...rest}
      />
    );
  }
  return (
    <Image
      src={src}
      style={style}
      {...rest}
      preview={false}
    />
  );
};

export default Icon;
