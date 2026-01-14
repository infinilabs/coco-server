import { Button, Select, Space } from "antd";
import { useRef, useEffect } from "react";
import cloneDeep from "lodash/cloneDeep";
import { OPERATOR_ICONS } from "../Suggestions/Operators";
import styles from "./index.module.less"
import { X } from "lucide-react";

export default function Filters({
  filters,
  onFiltersChange,
  onFilterInputFocus,
  onFilterInputBlur,
  // 替换：接收切换逻辑
  onFilterActiveToggle,
  focusIndex,
  activeIndex,
  className
}) {
  const filterRefs = useRef([]);
  const prevFilterCountRef = useRef(0);

  useEffect(() => {
    const currentFilterCount = filters.length;
    if (currentFilterCount > prevFilterCountRef.current) {
      const newFilterIndex = currentFilterCount - 1;
      setTimeout(() => {
        const newInputRef = filterRefs.current[newFilterIndex];
        if (newInputRef) {
          newInputRef.focus();
          if (onFilterInputFocus) {
            onFilterInputFocus(newFilterIndex);
          }
        }
      }, 0);
    }
    prevFilterCountRef.current = currentFilterCount;
  }, [filters, onFilterInputFocus]);

  const handleFiltersChange = (index, key, value) => {
    const newFilters = cloneDeep(filters);
    newFilters[index][key] = value;
    onFiltersChange(newFilters);
  };

  // 核心修改：点击Addon切换激活状态
  const handleFilterAddonClick = (index) => {
    if (onFilterActiveToggle) {
      onFilterActiveToggle(index);
    }
  };

  const handleInputFocus = (index) => {
    if (onFilterInputFocus) {
      onFilterInputFocus(index);
    }
  };

  const handleInputBlur = (index) => {
    if (onFilterInputBlur) {
      setTimeout(() => {
        if (focusIndex === index) {
          onFilterInputBlur();
        }
      }, 0);
    }
  };

  if (!filters || filters.length === 0) return null;

  return (
    <div className={`flex flex-wrap items-center gap-4px ${className}`}>
      {filters
        .filter((filter) => !!filter.field?.suggestion)
        .map((filter, index) => {
          // 判断当前filter是否处于激活状态
          const isActive = activeIndex === index;

          return (
            <div 
              key={index} 
              className={`flex items-center gap-4px ${styles.item} ${isActive ? styles.active : ""}`}
            >
              {/* 仅激活状态且有operator时显示图标 */}
              {isActive && filter.operator && OPERATOR_ICONS[filter.operator]}
              <Space.Compact className="cursor-pointer">
                <Space.Addon 
                  onClick={() => handleFilterAddonClick(index)}
                >
                  {filter.field.suggestion}
                </Space.Addon>
                <Select
                  mode="tags"
                  style={{ minWidth: 74 }}
                  styles={{ popup: { root: { display: 'none'} }}}
                  ref={(el) => {
                    filterRefs.current[index] = el;
                  }}
                  value={filter.value || []}
                  onChange={(value) => handleFiltersChange(index, 'value', value)}
                  onFocus={() => handleInputFocus(index)}
                  onBlur={() => handleInputBlur(index)}
                  options={[]}
                  maxTagCount={3}
                  suffixIcon={null}
                  maxTagTextLength={10}
                />
              </Space.Compact>
              {isActive && (
                <Button
                    shape="circle"
                    className={`!bg-#999 !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
                    classNames={{ icon: `w-8px h-8px !text-8px` }}
                    icon={<X className="w-8px h-8px" />}
                    onClick={() => {
                      const newFilters = cloneDeep(filters);
                      newFilters.splice(index, 1);
                      onFiltersChange(newFilters);
                      handleFilterAddonClick(-1);
                    }}
                />
              )}
            </div>
          );
        })}
    </div>
  );
}
