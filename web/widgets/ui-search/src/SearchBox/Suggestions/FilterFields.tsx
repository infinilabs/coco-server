import ListContainer from "./ListContainer";
import { type FC } from "react";

export const SUGGESTION_FILTER_FIELDS = "field_names";

interface FilterFieldsProps {
  data?: any[];
  onItemClick?: (item: any) => void;
  loadNext?: () => void;
  language?: string;
  resetKey?: string;
}

const FilterFields: FC<FilterFieldsProps> = (props) => {

  return (
    <ListContainer
      type={SUGGESTION_FILTER_FIELDS}
      title="过滤条件"
      {...props}
    />
  )
}

export default FilterFields;