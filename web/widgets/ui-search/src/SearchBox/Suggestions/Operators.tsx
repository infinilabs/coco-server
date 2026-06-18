import { Button } from "antd";
import { Minus, Plus } from "lucide-react";
import ListContainer from "./ListContainer";
import { type FC, type RefObject } from "react";
import { useTranslation } from "react-i18next";

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

interface OperatorsProps {
    currentOperator?: string;
    onItemClick?: (item: any) => void;
    language?: string;
    keyboardRootRef?: RefObject<HTMLElement | null>;
}

const Operators: FC<OperatorsProps> = ({ currentOperator, ...props }) => {
    const { t } = useTranslation();
    const conditionGroupLabel = t('labels.conditionGroup');
    const data = [
        {
            suggestion: 'and',
            source: t('labels.satisfyAll'),
            icon: OPERATOR_ICONS['and'],
            dividerTitle: conditionGroupLabel
        },
        // {
        //     suggestion: 'or',
        //     source: t('labels.matchAnyCondition'),
        //     icon: OPERATOR_ICONS['or'],
        // },
        {
            suggestion: 'not',
            source: t('labels.excludeCondition'),
            icon: OPERATOR_ICONS['not'],
        },
    ];
    const defaultActiveIndex = currentOperator ? data.findIndex(d => d.suggestion === currentOperator) : 0;
    return (
        <ListContainer
            type={SUGGESTION_OPERATORS}
            title={conditionGroupLabel}
            data={data}
            defaultActiveIndex={defaultActiveIndex >= 0 ? defaultActiveIndex : 0}
            {...props}
        />
    )
};

export default Operators;