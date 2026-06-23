import type { FC } from "react";
import { ColorPicker, type ColorPickerProps } from "antd";
import FilterCollapse, { type FilterCollapseProps } from "../FilterCollapse";

export type FilterColorPickerProps = FilterCollapseProps &
  ColorPickerProps & {
    i18n?: {
      labels?: {
        presetColors?: string;
      };
    };
  };

const PRESET_COLORS = [
  "#FF0000",
  "#FFA500",
  "#FFFF00",
  "#00FF00",
  "#00FFFF",
  "#0000FF",
  "#8000FF",
  "#FF00FF",
  "#FFFFFF",
  "#808080",
  "#000000",
  "#964B00",
];

const FilterColorPicker: FC<FilterColorPickerProps> = (props) => {
  const { presets, i18n, ...rest } = props;

  return (
    <FilterCollapse {...props}>
      <ColorPicker
        presets={[
          {
            label: i18n?.labels?.presetColors ?? "Preset Colors",
            colors: PRESET_COLORS,
          },
          ...(presets ?? []),
        ]}
        {...rest}
      />
    </FilterCollapse>
  );
};

export default FilterColorPicker;
