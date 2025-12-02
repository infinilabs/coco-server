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
  let formatSrc = src
  if (!src.startsWith('http')) {
    formatSrc = normalizeUrl(`${getProxyEndpoint() || server}/${src}`)
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
