import type { FC, ReactNode } from "react";
import { Typography } from "antd";
import FilterCollapse, { type FilterCollapseProps } from "../FilterCollapse";
import { clsx } from "clsx";

export interface FilterTagOption {
  label: string | React.ReactNode;
  value: string | number;
  icon?: string;
}

export interface FilterTagsProps extends FilterCollapseProps {
  value: Array<string | number>;
  options: FilterTagOption[];
  classNames?: {
    title?: string;
    tag?: string;
    icon?: string;
  };
  onChange?: (value: Array<string | number>) => void;
}

const FilterTags: FC<FilterTagsProps> = (props) => {
  const { value: propsValue, options, classNames, onChange } = props;

  const nameOptions = options.filter((item) => !item.icon);
  const iconOptions = options.filter((item) => item.icon);

  const handleChange = (value: string | number) => {
    if (propsValue.includes(value)) {
      onChange?.(propsValue.filter((v) => v !== value));
    } else {
      onChange?.([...propsValue, value]);
    }
  };

  const getLabelText = (label: string | ReactNode): string => {
    if (typeof label === 'string') return label;
    if (typeof label === 'number') return String(label);
    if (label && typeof label === 'object' && 'props' in label) {
      const children = (label as any).props?.children;
      if (typeof children === 'string') return children;
      if (typeof children === 'number') return String(children);
    }
    return '';
  };

  const getVisualWidth = (label: string | ReactNode) => {
    const str = getLabelText(label);
    return [...str].reduce((sum, char) => sum + (/[\u4e00-\u9fff\u3000-\u303f\uff00-\uffef]/.test(char) ? 2 : 1), 0);
  };

  const shortOptions = nameOptions.filter((item) => getVisualWidth(item.label) <= 8);
  const longOptions = nameOptions.filter((item) => getVisualWidth(item.label) > 8);

  const rows: { items: FilterTagOption[]; fill: boolean }[] = [];

  const shortRemainder = shortOptions.length % 3;
  const fullShortCount = shortOptions.length - shortRemainder;

  for (let i = 0; i < fullShortCount; i += 3) {
    rows.push({ items: shortOptions.slice(i, i + 3), fill: false });
  }

  let longStart = 0;
  if (shortRemainder === 2) {
    rows.push({ items: shortOptions.slice(fullShortCount), fill: true });
  } else if (shortRemainder === 1) {
    if (longOptions.length > 0) {
      rows.push({ items: [shortOptions[fullShortCount], longOptions[0]], fill: true });
      longStart = 1;
    } else {
      rows.push({ items: [shortOptions[fullShortCount]], fill: true });
    }
  }

  const remainingLong = longOptions.slice(longStart);
  const longRemainder = remainingLong.length % 2;
  const fullLongCount = remainingLong.length - longRemainder;

  for (let i = 0; i < fullLongCount; i += 2) {
    rows.push({ items: remainingLong.slice(i, i + 2), fill: false });
  }

  if (longRemainder > 0) {
    rows.push({ items: remainingLong.slice(fullLongCount), fill: true });
  }

  const renderTag = (item: FilterTagOption, style?: React.CSSProperties) => {
    const { label, value } = item;
    return (
      <div
        key={value}
        style={style}
        className={clsx(
          "border border-solid hover:border-[#007EFF] hover:bg-[rgba(0,126,255,0.1)] border-[#F0F0F0] dark:border-[#303030] text-12px text-[#666] inline-flex items-center justify-center h-24px px-1 cursor-pointer rounded-8px transition-colors text-[#666] dark:text-white/80",
          {
            "!border-[#007EFF] !bg-[rgba(0,126,255,0.1)]": propsValue.includes(value),
          },
          classNames?.tag
        )}
        onClick={() => {
          handleChange(value);
        }}
      >
        <Typography.Text
          className="!text-12px !text-inherit !leading-24px"
          ellipsis={{ tooltip: label }}
        >
          {label}
        </Typography.Text>
      </div>
    );
  };

  return (
    <FilterCollapse {...props}>
      <div className="flex flex-col gap-4px">
        {rows.map((row, rowIndex) => (
          <div key={rowIndex} className="flex gap-4px">
            {row.items.map((item) =>
              renderTag(item, row.fill ? { flex: '1 1 0' } : { flex: '1 1 0' })
            )}
          </div>
        ))}
      </div>
      {
        iconOptions.length ? (
          <div className="flex flex-wrap gap-4px mt-2">
            {iconOptions.map((item) => {
              const { label, value, icon } = item;

              return (
                <div
                  key={value}
                  className={clsx(
                    "size-12 rounded-full overflow-hidden cursor-pointer hover:border-primary transition-colors",
                    {
                      "border-primary": propsValue.includes(value),
                    },
                    classNames?.icon
                  )}
                  onClick={() => {
                    handleChange(value);
                  }}
                >
                  <img src={icon} title={getLabelText(label)} className="size-full" />
                </div>
              );
            })}
          </div>
        ) : null
      }
    </FilterCollapse>
  );
};

export default FilterTags;
