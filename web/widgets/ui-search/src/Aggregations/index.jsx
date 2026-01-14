import { useState } from "react";
import cloneDeep from "lodash/cloneDeep";

import FilterDefaultSvg from "../icons/filter-default.svg"
import AllSvg from "../icons/all.svg"
import { FilterCheckboxGroup } from "@infinilabs/filter";

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

  return (
    <div>
      {aggregations.map((aggregation, index) => {
        let count = 0;
        aggregation.list.forEach((item) => (count += item.count));
        const filterList = currentFilters[aggregation.key] || [];
        return (
          <div key={aggregation.key} className="mb-24px">
            <FilterCheckboxGroup
              defaultExpand={index === 0} 
              title={<div>{(config?.[aggregation.key]?.displayName || aggregation.key)?.toUpperCase()}</div>}
              value={filterList}
              options={aggregation.list.map((item) => ({
                label: <div>{item.name || item.key}</div>,
                value: item.key,
                icon: item.icon || FilterDefaultSvg,
                count: item.count,
              }))}
              onChange={(value) => {
                onChange(value, aggregation)
              }}
              onClear={() => {
                onClear(aggregation)
              }}
            />
          </div>
        );
      })}
    </div>
  );
}

export default Aggregations;
