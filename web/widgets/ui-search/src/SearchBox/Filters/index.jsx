import { Button, Select, Space } from "antd";
import { useRef, useEffect, useState } from "react"; 
import cloneDeep from "lodash/cloneDeep";
import { OPERATOR_ICONS } from "../Suggestions/Operators";
import styles from "./index.module.less"
import { X } from "lucide-react";

export default function Filters({
  filters,
  onFiltersChange,
  onFilterInputFocus,
  onFilterInputBlur,
  onFilterActiveToggle,
  focusIndex,
  activeIndex,
  className,
  shouldFocusNewFilter = false,
  isHandlingSuggestion = false
}) {
  const filterRefs = useRef([]);
  const prevFilterCountRef = useRef(0);
  const [localShouldFocus, setLocalShouldFocus] = useState(false);
  const lastFilterCountRef = useRef(0);
  const autoFocusedIndexRef = useRef(-1);

  useEffect(() => {
    if (shouldFocusNewFilter) {
      setLocalShouldFocus(true);
      lastFilterCountRef.current = filters.length;
    }
  }, [shouldFocusNewFilter, filters.length]);

  useEffect(() => {
    const currentFilterCount = filters.length;
    
    if (localShouldFocus && currentFilterCount > lastFilterCountRef.current && currentFilterCount > 0) {
      const newFilterIndex = currentFilterCount - 1;
      setTimeout(() => {
        const newInputRef = filterRefs.current[newFilterIndex];
        if (newInputRef) {
          autoFocusedIndexRef.current = newFilterIndex;
          newInputRef.focus();
          if (onFilterInputFocus) {
            onFilterInputFocus(newFilterIndex);
          }
          setLocalShouldFocus(false);
          lastFilterCountRef.current = currentFilterCount;
        }
      }, 0);
    }
    
    prevFilterCountRef.current = currentFilterCount;
  }, [filters, onFilterInputFocus, localShouldFocus]); 

  useEffect(() => {
    if (localShouldFocus && filters.length === lastFilterCountRef.current && filters.length > 0) {
      const newFilterIndex = filters.length - 1;
      setTimeout(() => {
        const newInputRef = filterRefs.current[newFilterIndex];
        if (newInputRef) {
          autoFocusedIndexRef.current = newFilterIndex;
          newInputRef.focus();
          if (onFilterInputFocus) {
            onFilterInputFocus(newFilterIndex);
          }
          setLocalShouldFocus(false);
        }
      }, 100);
    }
  }, [localShouldFocus, filters.length, onFilterInputFocus]);

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
        const shouldTriggerBlur = 
          !isHandlingSuggestion && 
          autoFocusedIndexRef.current !== index &&
          focusIndex === index;
        
        if (shouldTriggerBlur) {
          onFilterInputBlur();
          autoFocusedIndexRef.current = -1;
        }
      }, 150); 
    }
  };

  const setFilterRef = (el, index) => {
    if (!filterRefs.current) filterRefs.current = [];
    filterRefs.current[index] = el;
  };

  if (!filters || filters.length === 0) return null;

  return (
    <div className={`flex flex-wrap items-center gap-4px ${className}`}>
      {filters
        .filter((filter) => !!filter.field?.field_label)
        .map((filter, index) => {
          const isActive = activeIndex === index;
          const isFocused = focusIndex === index;

          return (
            <div 
              key={`filter-${index}-${filter.field?.field_name || filter.field?.field_label}`} 
              className={`flex items-center gap-4px ${styles.item} ${isActive ? styles.active : ""} ${isFocused ? styles.focused : ""}`}
            >
              {isActive && filter.operator && OPERATOR_ICONS[filter.operator]}
              <Space.Compact className="cursor-pointer">
                <Space.Addon 
                  onClick={() => handleFilterAddonClick(index)}
                >
                  {filter.field.field_label}
                </Space.Addon>
                <Select
                  mode="tags"
                  style={{ minWidth: 74 }}
                  styles={{ popup: { root: { display: 'none'} }}}
                  ref={(el) => setFilterRef(el, index)}
                  value={filter.value || []}
                  onFocus={() => handleInputFocus(index)}
                  onBlur={() => handleInputBlur(index)}
                  onMouseDown={(e) => e.preventDefault()}
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