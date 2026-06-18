import { Button, Select, Space } from "antd";
import { forwardRef, useImperativeHandle, useRef, useEffect, useState, useCallback, type KeyboardEvent } from "react"; 
import { OPERATOR_ICONS } from "../Suggestions/Operators";
import styles from "./index.module.less"
import { X } from "lucide-react";

interface FilterItem {
  field?: { field_label?: string; field_name?: string };
  operator?: keyof typeof OPERATOR_ICONS;
  value?: string[];
}

interface FiltersProps {
  filters: FilterItem[];
  onFiltersChange: (filters: FilterItem[]) => void;
  onFilterInputFocus?: (index: number) => void;
  onFilterInputBlur?: () => void;
  onFilterActiveToggle?: (index: number) => void;
  onFilterDelete?: (index: number) => void;
  onFilterValueEdit?: (index: number) => void;
  onFilterComplete?: () => void;
  onFilterSearch?: (val: string) => void;
  filterSearchValue?: string;
  focusIndex?: number;
  activeIndex?: number;
  className?: string;
  shouldFocusNewFilter?: boolean;
  isHandlingSuggestion?: boolean;
}

const Filters = forwardRef<any, FiltersProps>(({
  filters,
  onFiltersChange,
  onFilterInputFocus,
  onFilterInputBlur,
  onFilterActiveToggle,
  onFilterDelete,
  onFilterValueEdit,
  onFilterComplete,
  onFilterSearch,
  filterSearchValue = '',
  focusIndex,
  activeIndex,
  className = '',
  shouldFocusNewFilter = false,
  isHandlingSuggestion = false
}, ref) => {
  const filterRefs = useRef<any[]>([]);
  const prevFilterCountRef = useRef(0);
  const [localShouldFocus, setLocalShouldFocus] = useState(false);
  const lastFilterCountRef = useRef(0);
  const autoFocusedIndexRef = useRef(-1);

  const focusFilterInput = useCallback((index: number, autoFocus = false) => {
    const inputRef = filterRefs.current[index];
    if (!inputRef) return false;

    if (autoFocus) autoFocusedIndexRef.current = index;
    inputRef.focus();
    onFilterInputFocus?.(index);
    return true;
  }, [onFilterInputFocus]);

  const focusFilterInputOnNextFrame = useCallback((index: number, autoFocus = false) => {
    requestAnimationFrame(() => {
      focusFilterInput(index, autoFocus);
    });
  }, [focusFilterInput]);

  useImperativeHandle(ref, () => ({
    focusFilterInput: (index: number) => focusFilterInputOnNextFrame(index),
  }), [focusFilterInputOnNextFrame]);

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
      requestAnimationFrame(() => {
        if (focusFilterInput(newFilterIndex, true)) {
          setLocalShouldFocus(false);
          lastFilterCountRef.current = currentFilterCount;
        }
      });
    }
    
    prevFilterCountRef.current = currentFilterCount;
  }, [filters, localShouldFocus, focusFilterInput]); 

  useEffect(() => {
    if (localShouldFocus && filters.length === lastFilterCountRef.current && filters.length > 0) {
      const newFilterIndex = filters.length - 1;
      requestAnimationFrame(() => {
        if (focusFilterInput(newFilterIndex, true)) {
          setLocalShouldFocus(false);
        }
      });
    }
  }, [localShouldFocus, filters.length, focusFilterInput]);

  const handleFilterAddonClick = (index: number) => {
    if (onFilterActiveToggle) {
      onFilterActiveToggle(index);
    }
  };

  const handleFilterAddonMouseDown = (e: React.MouseEvent, index: number) => {
    e.preventDefault();
    handleFilterAddonClick(index);
    filterRefs.current.forEach(ref => ref?.blur?.());
  };

  const handleValueAreaClick = (index: number) => {
    if (onFilterValueEdit) {
      onFilterValueEdit(index);
    }
    // Focus the select input
    focusFilterInputOnNextFrame(index);
  };

  const handleInputFocus = (index: number) => {
    if (onFilterInputFocus) {
      onFilterInputFocus(index);
    }
  };

  const handleInputBlur = (index: number) => {
    if (onFilterInputBlur) {
      const shouldTriggerBlur = 
        !isHandlingSuggestion && 
        autoFocusedIndexRef.current !== index &&
        focusIndex === index;
      
      if (shouldTriggerBlur) {
        onFilterInputBlur();
        autoFocusedIndexRef.current = -1;
      }
    }
  };

  const handleKeyDown = (e: any, index: number) => {
    // Prevent antd Select from consuming Enter;
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

  const setFilterRef = (el: any, index: number) => {
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
                  onMouseDown={(e) => handleFilterAddonMouseDown(e, index)}
                  className="border-[#F0F0F0] dark:border-[#303030]"
                >
                  {filter.field?.field_label}
                </Space.Addon>
                <Select
                  className="border-[#F0F0F0] dark:border-[#303030]"
                  mode="multiple"
                  maxTagCount={3}
                  maxTagPlaceholder={() => '...'}
                  style={{ minWidth: 'auto' }}
                  styles={{ popup: { root: { display: 'none'} }, content: { flexWrap: 'nowrap' } }}
                  ref={(el) => setFilterRef(el, index)}
                  value={filter.value || []}
                  searchValue={isFocused ? filterSearchValue : undefined}
                  autoClearSearchValue={false}
                  onFocus={() => handleInputFocus(index)}
                  onBlur={() => handleInputBlur(index)}
                  onSearch={(val) => onFilterSearch && onFilterSearch(val)}
                  onMouseDown={(e) => {
                    // Allow clicks on tag remove button to pass through
                    if ((e.target as HTMLElement).closest('.ant-select-selection-item-remove')) {
                      return;
                    }
                    e.preventDefault();
                    handleValueAreaClick(index);
                  }}
                  onDeselect={(removedValue) => {
                    // Remove the deselected value and enter edit mode
                    const newFilters = filters.map((filterItem, filterIndex) => {
                      if (filterIndex !== index) return filterItem;
                      return {
                        ...filterItem,
                        value: Array.isArray(filterItem.value) ? filterItem.value.filter(v => v !== removedValue) : filterItem.value
                      };
                    });
                    onFiltersChange(newFilters);
                    if (onFilterValueEdit) {
                      onFilterValueEdit(index);
                    }
                    focusFilterInputOnNextFrame(index);
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
                        const newFilters = filters.filter((_, filterIndex) => filterIndex !== index);
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
});

export default Filters;