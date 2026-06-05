import { useState, type FC } from "react";
import { Checkbox } from "antd";
import { motion, AnimatePresence } from "motion/react";

import type { FilterCollapseProps } from "../FilterCollapse";
import FilterCollapse from "../FilterCollapse";
import { ChevronDown } from "lucide-react";
import { clsx } from "clsx";

export interface FilterCheckboxGroupOption {
  label: string;
  value: string | number;
  icon?: string;
  count: number;
}

export interface FilterCheckboxGroupProps extends FilterCollapseProps {
  value: Array<string | number>;
  options: FilterCheckboxGroupOption[];
  i18n?: {
    labels?: {
      more?: string;
    };
  };
  classNames?: {
    title?: string;
    icon?: string;
    label?: string;
    count?: string;
    more?: string;
  };
  onChange?: (value: Array<string | number>) => void;
}

const FilterCheckboxGroup: FC<FilterCheckboxGroupProps> = (props) => {
  const { options, value: propsValue, i18n, classNames, onChange } = props;
  const [expandMore, setExpandMore] = useState(false);

  const renderOptions = (options: FilterCheckboxGroupOption[]) => {
    return (
      <div className="flex flex-col gap-16px">
        {options.map((item) => {
          const { label, value, icon, count } = item;

          return (
            <div key={value} className="flex items-center justify-between">
              <Checkbox
                className="inline-flex items-center"
                checked={propsValue.includes(value)}
                onChange={(event) => {
                  const checked = event.target.checked;

                  if (checked) {
                    onChange?.([...propsValue, value]);
                  } else {
                    onChange?.(propsValue.filter((item) => item !== value));
                  }
                }}
              >
                <div className="flex items-center gap-5px">
                  {icon && (
                    <img
                      src={icon}
                      alt={label}
                      className={clsx("w-14px h-14px", classNames?.icon)}
                    />
                  )}

                  <span
                    className={clsx(
                      "text-[#666] dark:text-white/80",
                      classNames?.label
                    )}
                  >
                    {label}
                  </span>
                </div>
              </Checkbox>

              <span
                className={clsx(
                  "text-[#666] dark:text-white/80",
                  classNames?.count
                )}
              >
                {count}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  const handleExpandMore = () => {
    setExpandMore((prev) => !prev);
  };

  return (
    <FilterCollapse {...props}>
      {renderOptions(options.slice(0, 5))}

      <AnimatePresence>
        {expandMore && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="overflow-hidden"
          >
            <div className="mt-4">{renderOptions(options.slice(5))}</div>
          </motion.div>
        )}
      </AnimatePresence>

      {options.length > 5 && (
        <div
          className={clsx(
            "inline-flex items-center gap-2 mt-4 text-primary cursor-pointer",
            classNames?.more
          )}
          onClick={handleExpandMore}
        >
          <ChevronDown
            className={clsx("size-4 transition", {
              "-scale-y-100": expandMore,
            })}
          />

          <span>{i18n?.labels?.more ?? "More"}</span>
        </div>
      )}
    </FilterCollapse>
  );
};

export default FilterCheckboxGroup;
