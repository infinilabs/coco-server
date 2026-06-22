import { Tag } from "antd";
import { Lightbulb } from "lucide-react";
import ListContainer from "./ListContainer";
import { useTranslation } from 'react-i18next';
import { type FC } from "react";

export const SUGGESTION_TIPS = "suggestion_tips"

const Tips: FC = () => {
    const { t } = useTranslation();

    return (
        <ListContainer
            type={SUGGESTION_TIPS}
            title={t('labels.searchTips')}
            data={[
                {
                    icon: <Lightbulb className="w-16px h-16px" />,
                    suggestion: (
                        <span>
                            {t('labels.advancedFilterTipPart1')} <Tag>/</Tag> {t('labels.advancedFilterTipPart1Suffix')}
                            {' '}
                            {t('labels.advancedFilterTipOr')} <Tag>{t('labels.fieldName')}</Tag> + <Tag>:</Tag> {t('labels.advancedFilterTipConvert')}
                        </span>
                    ),
                },
            ]}
            defaultRows={1}
        />
    )
};

export default Tips;