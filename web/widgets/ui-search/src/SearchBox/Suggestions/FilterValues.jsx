import { Checkbox } from "antd";
import ListContainer from "./ListContainer";

export const SUGGESTION_FILTER_VALUES = "field_values"

export default (props) => {
    const { filter = {}, ...rest } = props;

    const { field = {}, value = [] } = filter || {}
    const { payload = {} } = field || {}
    const { support_multi_select } = payload || {}

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