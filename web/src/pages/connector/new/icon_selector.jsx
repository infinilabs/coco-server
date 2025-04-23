import { Image, Select } from 'antd';
import { ReactSVG } from 'react-svg';

import './icon.css';
import FontIcon from '@/components/common/font_icon';

export const IconSelector = ({ className, icons = [], onChange, value }) => {
  return (
    <Select
      className={className}
      popupMatchSelectWidth={false}
      showSearch={true}
      value={value}
      onChange={onChange}
    >
      {icons.map((icon, index) => {
        return (
          <Select.Option
            item={icon}
            value={icon.path}
            key={index}
          >
            <div className="flex items-center gap-3px">
              {icon.source === 'fonts' ? (
                <FontIcon name={icon.path} />
              ) : icon.path.endsWith('.svg') ? (
                <ReactSVG
                  className="limitw"
                  src={icon.path}
                />
              ) : (
                <Image
                  height="1em"
                  preview={false}
                  src={icon.path}
                  width="1em"
                />
              )}

              <span className="overflow-hidden text-ellipsis">{icon.name}</span>
            </div>
          </Select.Option>
        );
      })}
    </Select>
  );
};
