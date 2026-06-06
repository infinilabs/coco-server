import { Checkbox } from "antd";
import ListContainer from "./ListContainer";
import { type FC } from "react";

export const SUGGESTION_FILTER_VALUES = "field_values"

interface FilterValuesProps {
    filter?: { field?: { support_multi_select?: boolean }; value?: string[] } | null;
    onComplete?: () => void;
    data?: any[];
    onItemClick?: (item: any) => void;
    language?: string;
    resetKey?: string;
}

const FilterValues: FC<FilterValuesProps> = (props) => {
    const { filter = {}, onComplete, ...rest } = props;

    const { field = {}, value = [] } = filter || {}
    const { support_multi_select } = field || {}

    return (
        <ListContainer
            type={SUGGESTION_FILTER_VALUES}
            title="过滤条件"
            {...rest}
            renderPrefix={(item) => {
                if (!support_multi_select) return null;
                return (
                    support_multi_select && (
                        <div className="mr-8px flex-shrink-0">
                            <Checkbox checked={value.findIndex((v) => v === item.suggestion) !== -1}></Checkbox>
                        </div>
                    )
                )
            }}
        />
    )
};

export default FilterValues;