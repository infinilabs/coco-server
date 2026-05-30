import { useState } from "react";
import cloneDeep from "lodash/cloneDeep";

import FilterDefaultSvg from "../icons/filter-default.svg"
import { FilterCheckboxGroup, FilterColorPicker, FilterTags } from "@infinilabs/filter";

export function Aggregations(props) {
  const { config = {}, aggregations = [], filter = {}, onSearch } = props;

  const [currentFilters, setCurrentFilters] = useState(filter);

  const onChange = (value, aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    newFilters[aggregation.key] = value
    setCurrentFilters(newFilters);
    onSearch(newFilters);
  };

  const onClear = (aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    delete newFilters[aggregation.key];
    setCurrentFilters(newFilters);
    onSearch(newFilters);
  };

  if (!aggregations || aggregations.length === 0) return null

  return (
    <div>
      {aggregations.map((aggregation, index) => {
        let count = 0;
        aggregation.list.forEach((item) => (count += item.count));
        const type = config?.[aggregation.key]?.type || 'checkbox';
        const commonProps = {
          defaultExpand: index <= 2,
          title: <div>{(config?.[aggregation.key]?.label || aggregation.key)?.toUpperCase()}</div>,
          value: currentFilters[aggregation.key],
          onChange: (value) => {
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
                label: <div>{item.name || item.key}</div>,
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
                label: <div>{item.name || item.key}</div>,
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
    </div>
  );
}

export default Aggregations;
