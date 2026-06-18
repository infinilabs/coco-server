import ListContainer from "./ListContainer";
import { type FC, type RefObject } from "react";
import { useTranslation } from "react-i18next";

export const SUGGESTION_FILTER_FIELDS = "field_names";

interface FilterFieldsProps {
  data?: any[];
  onItemClick?: (item: any) => void;
  loadNext?: () => void;
  language?: string;
  resetKey?: string;
  keyboardRootRef?: RefObject<HTMLElement | null>;
}

const FilterFields: FC<FilterFieldsProps> = (props) => {
  const { t } = useTranslation();
  return (
    <ListContainer
      type={SUGGESTION_FILTER_FIELDS}
      title={t('labels.filterTitle')}
      {...props}
    />
  )
}

export default FilterFields;