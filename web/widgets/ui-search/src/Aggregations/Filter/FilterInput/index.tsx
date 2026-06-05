import { Input, type InputProps } from "antd";
import FilterCollapse, { type FilterCollapseProps } from "../FilterCollapse";

export type FilterInputProps = FilterCollapseProps & InputProps;

const FilterInput = (props: FilterInputProps) => {
  return (
    <FilterCollapse {...props}>
      <div className="px-0.5 pb-0.5">
        <Input {...props} />
      </div>
    </FilterCollapse>
  );
};

export default FilterInput;
