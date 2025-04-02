import React from 'react';
import FontIcon from '@/components/common/font_icon';
import { Image } from 'antd';

const Icon = ({ src, className, style, ...rest }) => {
  if(!src){
    return null;
  }
  if(!src.includes("/")){
    return <FontIcon name={src} className={className} style={style} {...rest} />;
  }
  return (
    <Image src={src} style={style} {...rest} preview={false}/>
  );
};

export default Icon;