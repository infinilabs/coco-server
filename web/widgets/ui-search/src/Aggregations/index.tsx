import { useEffect, useState } from "react";
import cloneDeep from "lodash/cloneDeep";

import FilterDefaultSvg from "../icons/filter-default.svg"
import { FilterCheckboxGroup, FilterColorPicker, FilterTags } from "./Filter";

interface AggregationItem {
  key: string;
  name?: string;
  icon?: string;
  count: number;
}

interface Aggregation {
  key: string;
  list: AggregationItem[];
}

interface AggregationConfig {
  [key: string]: {
    type?: 'checkbox' | 'color' | 'tag';
    label?: string;
  };
}

interface AggregationsProps {
  config?: AggregationConfig;
  aggregations?: Aggregation[];
  filter?: Record<string, any>;
  onSearch?: (filters: Record<string, any>) => void;
}

export function Aggregations(props: AggregationsProps) {
  const { config = {}, aggregations = [], filter = {}, onSearch } = props;

  const [currentFilters, setCurrentFilters] = useState<Record<string, any>>(filter);

  useEffect(() => {
    setCurrentFilters(filter);
  }, [JSON.stringify(filter)]);

  const onChange = (value: any, aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    newFilters[aggregation.key] = value
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  const onClear = (aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    delete newFilters[aggregation.key];
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  if (!aggregations || aggregations.length === 0) return null

  return (
    <>
      {aggregations.map((aggregation, index) => {
        let count = 0;
        aggregation.list.forEach((item) => (count += item.count));
        const type = config?.[aggregation.key]?.type || 'checkbox';
        const commonProps = {
          defaultExpand: index <= 2,
          title: (config?.[aggregation.key]?.label || aggregation.key)?.toUpperCase() || '',
          value: currentFilters[aggregation.key],
          onChange: (value: any) => {
            onChange(value, aggregation)
          },
          onClear: () => onClear(aggregation)
        }
        let content
        if (type === 'color') {
          content = (
            <FilterColorPicker 
              {...commonProps} 
              onChange={(value) => onChange(value?.toHex(), aggregation)}
            />
          )
        } else if (type === 'tag') {
          content = (
            <FilterTags
              {...commonProps}
              value={commonProps.value || []}
              options={aggregation.list.map((item) => ({
                label: item.name || item.key,
                value: item.key,
              }))}
            />
          )
        } else {
          content = (
            <FilterCheckboxGroup
              {...commonProps}
              value={commonProps.value || []}
              options={aggregation.list.map((item) => ({
                label: item.name || item.key,
                value: item.key,
                icon: item.icon || FilterDefaultSvg,
                count: item.count,
              }))}
            />
          )
        }
        return (
          <div key={aggregation.key} className="mb-24px">
            {content}
          </div>
        );
      })}
    </>
  );
}

export default Aggregations;
