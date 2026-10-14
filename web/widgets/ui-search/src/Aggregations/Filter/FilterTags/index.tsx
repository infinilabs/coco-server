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

  const renderTag = (item: FilterTagOption) => {
    const { label, value } = item;
    return (
      <div
        key={value}
        className={clsx(
          "min-w-0 max-w-full border border-solid hover:border-[#007EFF] hover:bg-[rgba(0,126,255,0.1)] border-[#F0F0F0] dark:border-[#303030] text-12px inline-flex items-center justify-center h-24px px-1 cursor-pointer rounded-8px transition-colors text-[#666] dark:text-white/80",
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
      {/* tags size to their labels and wrap — no fixed per-row count, so the
          chips never clip regardless of how narrow the facet rail is */}
      <div className="flex flex-wrap gap-4px">
        {nameOptions.map(item => renderTag(item))}
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
