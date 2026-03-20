import { Button } from "antd";
import { Minus, Plus } from "lucide-react";
import ListContainer from "./ListContainer";

export const SUGGESTION_OPERATORS = "suggestion_operators"

export const OPERATOR_ICONS = {
    "and": (
        <Button
            shape="circle"
            className={`!bg-#1784FC !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Plus className="w-8px h-8px" />}
        />
    ),
    "or": (
        <Button
            shape="circle"
            className={`!bg-#8BBD7A !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Minus className="w-8px h-8px rotate-105" />}
        />
    ),
    "not": (
        <Button
            shape="circle"
            className={`!bg-#F15A5A !text-#fff !border-0 !rounded-50% !w-12px !min-w-12px !h-12px !p-0`}
            classNames={{ icon: `w-8px h-8px !text-8px` }}
            icon={<Minus className="w-8px h-8px" />}
        />
    )
}

export default (props) => {
    return (
        <ListContainer
            type={SUGGESTION_OPERATORS}
            title="条件组合"
            data={[
                {
                    suggestion: 'and',
                    source: `满足全部条件`,
                    icon: OPERATOR_ICONS['and'],
                    dividerTitle: '条件组合'
                },
                {
                    suggestion: 'or',
                    source: `满足任一条件`,
                    icon: OPERATOR_ICONS['or'],
                },
                {
                    suggestion: 'not',
                    source: `排除条件`,
                    icon: OPERATOR_ICONS['not'],
                },
            ]}
            {...props}
        />
    )
};