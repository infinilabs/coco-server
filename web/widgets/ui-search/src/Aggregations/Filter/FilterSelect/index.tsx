import { Select, type SelectProps } from "antd";
import FilterCollapse, { type FilterCollapseProps } from "../FilterCollapse";
import type { FC } from "react";

export type FilterSelectProps = FilterCollapseProps & SelectProps;

const FilterSelect: FC<FilterSelectProps> = (props) => {
  return (
    <FilterCollapse {...props}>
      <div className="px-0.5 pb-0.5">
        <Select className="w-full" {...props} />
      </div>
    </FilterCollapse>
  );
};

export default FilterSelect;
