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
  onFilterDelete,
  onFilterValueEdit,
  onFilterComplete,
  onFilterSearch,
  focusIndex,
  activeIndex,
  className = '',
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

  const handleValueAreaClick = (index) => {
    if (onFilterValueEdit) {
      onFilterValueEdit(index);
    }
    // Focus the select input
    setTimeout(() => {
      const ref = filterRefs.current[index];
      if (ref) ref.focus();
    }, 0);
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

  const handleKeyDown = (e, index) => {
    // Prevent antd Select from creating tags on Enter;
    // value selection is handled by the suggestion ListContainer
    if (e.key === 'Enter') {
      e.preventDefault();
      return;
    }
    // Backspace on empty value to delete filter
    if (e.key === 'Backspace') {
      const filter = filters[index];
      const hasValue = filter?.value && (Array.isArray(filter.value) ? filter.value.length > 0 : !!filter.value);
      const hasSearchText = e.target?.value?.length > 0;
      if (!hasValue && !hasSearchText) {
        e.preventDefault();
        if (onFilterDelete) {
          onFilterDelete(index);
        }
      }
    }
  };

  const setFilterRef = (el, index) => {
    if (!filterRefs.current) filterRefs.current = [];
    filterRefs.current[index] = el;
  };

  if (!filters || filters.length === 0) return null;

  const hasMultipleFilters = filters.filter(f => !!f.field?.field_label).length > 1;

  return (
    <div className={`flex flex-wrap items-center gap-4px ${className}`}>
      {filters
        .filter((filter) => !!filter.field?.field_label)
        .map((filter, index) => {
          const isActive = activeIndex === index;
          const isFocused = focusIndex === index;
          const operator = filter.operator || 'and';
          // // and/not always visible; or visible when active
          // const showOperator = operator !== 'or' || isActive;
          const showOperator = isActive;

          return (
            <div 
              key={`filter-${index}-${filter.field?.field_name || filter.field?.field_label}`} 
              className={`flex items-center gap-4px ${styles.item} ${isActive ? styles.active : ""} ${isFocused ? styles.focused : ""}`}
            >
              {showOperator && OPERATOR_ICONS[operator]}
              <Space.Compact className="cursor-pointer">
                <Space.Addon 
                  onClick={() => handleFilterAddonClick(index)}
                  className="border-[#F0F0F0] dark:border-[#303030]"
                >
                  {filter.field.field_label}
                </Space.Addon>
                <Select
                  className="border-[#F0F0F0] dark:border-[#303030]"
                  mode="tags"
                  maxTagCount={3}
                  maxTagPlaceholder={() => '...'}
                  style={{ minWidth: 'auto' }}
                  styles={{ popup: { root: { display: 'none'} }, content: { flexWrap: 'nowrap' } }}
                  ref={(el) => setFilterRef(el, index)}
                  value={filter.value || []}
                  onFocus={() => handleInputFocus(index)}
                  onBlur={() => handleInputBlur(index)}
                  onSearch={(val) => onFilterSearch && onFilterSearch(val)}
                  onMouseDown={(e) => {
                    // Allow clicks on tag remove button to pass through
                    if (e.target.closest('.ant-select-selection-item-remove')) {
                      return;
                    }
                    e.preventDefault();
                    handleValueAreaClick(index);
                  }}
                  onDeselect={(removedValue) => {
                    // Remove the deselected value and enter edit mode
                    const newFilters = cloneDeep(filters);
                    const f = newFilters[index];
                    if (Array.isArray(f.value)) {
                      f.value = f.value.filter(v => v !== removedValue);
                    }
                    onFiltersChange(newFilters);
                    if (onFilterValueEdit) {
                      onFilterValueEdit(index);
                    }
                    setTimeout(() => {
                      const ref = filterRefs.current[index];
                      if (ref) ref.focus();
                    }, 0);
                  }}
                  onInputKeyDown={(e) => handleKeyDown(e, index)}
                  options={[]}
                  suffixIcon={null}
                />
              </Space.Compact>
              {isActive && (
                <Button
                    shape="circle"
                    className={`!bg-#999 !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
                    classNames={{ icon: `w-8px h-8px !text-8px` }}
                    icon={<X className="w-8px h-8px" />}
                    onClick={() => {
                      if (onFilterDelete) {
                        onFilterDelete(index);
                      } else {
                        const newFilters = cloneDeep(filters);
                        newFilters.splice(index, 1);
                        onFiltersChange(newFilters);
                        handleFilterAddonClick(-1);
                      }
                    }}
                />
              )}
            </div>
          );
        })}
    </div>
  );
}