import { Image, Select, Tag } from 'antd';
import { ReactSVG } from 'react-svg';

import './icon.css';
import FontIcon from '@/components/common/font_icon';

const strictUrlRegex = /^(https?):\/\/((([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,})|(localhost)|(\d{1,3}(\.\d{1,3}){3}))(:\d{1,5})?(\/[^\s]*)?$/;
export const IconSelector = ({ className, icons = [], onChange, value }) => {
  const onInnerChange = (value) => {
   onChange?.(value.length > 0 ? value[0]: null);
  }

  const tagRender = (props) => {
    const { label, closable, onClose } = props;
    return <Tag className='inline-flex items-center' closable={closable} onClose={onClose}><div className="inline-flex items-center gap-3px">
        {strictUrlRegex.test(label) && <Image
          height="1em"
          preview={false}
          src={label}
          width="1em"
        />}
      <span className="overflow-hidden text-ellipsis">{label}</span>
    </div>
    </Tag> 
  }

  return (
    <Select
      mode='tags'
      maxCount={1}
      className={className}
      popupMatchSelectWidth={false}
      showSearch={true}
      value={value}
      onChange={onInnerChange}
      tagRender={tagRender}
      optionRender={({ label, value }) => {
        if(strictUrlRegex.test(value)) {
          return <div className="inline-flex items-center gap-3px">
            <Image
              height="1em"
              preview={false}
              src={label}
              width="1em"
            />
            <span className="overflow-hidden text-ellipsis">{label}</span>
          </div>
        }
       return label;
      }}
    >
      {icons.map((icon, index) => {
        return (
          <Select.Option
            item={icon}
            value={icon.path}
            key={index}
          >
            <div className="inline-flex items-center gap-3px">
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
