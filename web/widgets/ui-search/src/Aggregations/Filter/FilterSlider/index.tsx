import { Slider } from "antd";
import type { FilterCollapseProps } from "../FilterCollapse";
import type { ComponentProps, FC } from "react";
import FilterCollapse from "../FilterCollapse";

export type FilterSliderProps = FilterCollapseProps &
  ComponentProps<typeof Slider>;

const FilterSlider: FC<FilterSliderProps> = (props) => {
  return (
    <FilterCollapse {...props}>
      <div className="px-2">
        <Slider {...props} />
      </div>
    </FilterCollapse>
  );
};

export default FilterSlider;
