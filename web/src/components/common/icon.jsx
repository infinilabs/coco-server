import { Image } from 'antd';
import React from 'react';

import FontIcon from '@/components/common/font_icon';

import normalizeUrl from 'normalize-url';
import { getProxyEndpoint } from '@/components/micro/utils'

const Icon = ({ className, src, style, server, ...rest }) => {
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

  let formatSrc = src;
  if (!src.startsWith('http')) {
    if (src.startsWith('/')) {
      formatSrc = src;
    } else {
      const base = getProxyEndpoint() || server || window.location.origin;
      formatSrc = normalizeUrl(`${base}/${src}`);
    }
  }

  return (
    <Image
      src={formatSrc}
      style={style}
      {...rest}
      preview={false}
    />
  );
};

export default Icon;
