import { Button, Dropdown, Space } from "antd";
import { Brain, ChevronDown, Globe, Hammer, Search, Telescope } from "lucide-react";
import { useTranslation } from 'react-i18next';

export const ACTION_TYPE_SEARCH = 'search'
export const ACTION_TYPE_SEARCH_HYBRID = 'hybrid'
export const ACTION_TYPE_SEARCH_KEYWORD = 'keyword'
export const ACTION_TYPE_SEARCH_SEMANTIC = 'semantic'
export const ACTION_TYPE_DEEPTHINK = 'deepthink'
export const ACTION_TYPE_DEEPSEARCH = 'deepresearch'

export default (props) => {
    const { 
        actionType, 
        searchType = ACTION_TYPE_SEARCH_KEYWORD, 
        onSearchTypeChange,
        onButtonClick,
        onDropdownClose 
    } = props;

    const { t } = useTranslation();

    if (actionType === ACTION_TYPE_DEEPTHINK) {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4} className="!leading-none">
                        <Telescope className="w-16px h-16px" />
                        {t('labels.deepThink')}
                    </Space>
                </Button>
                {/* <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Globe strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                    disabled
                />
                <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Hammer strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                    disabled
                /> */}
            </Space>
        )
    } else if (actionType === ACTION_TYPE_DEEPSEARCH) {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4} className="!leading-none">
                        <Brain className="w-16px h-16px" />
                        {t('labels.deepResearch')}
                    </Space>
                </Button>
                {/* <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Globe strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                    disabled
                /> */}
            </Space>
        )
    } else {
        const items = [
            {
                key: ACTION_TYPE_SEARCH_HYBRID,
                label: t('labels.hybrid'),
            },
            {
                key: ACTION_TYPE_SEARCH_KEYWORD,
                label: t('labels.keyword'),
            },
            {
                key: ACTION_TYPE_SEARCH_SEMANTIC,
                label: t('labels.semantic'),
            },
        ]
        
        const handleVisibleChange = (visible) => {
            if (!visible && onDropdownClose) {
                onDropdownClose();
            }
        };
        
        return (
            <div>
                <Dropdown 
                    menu={{ 
                        items, 
                        onClick: ({key}) => {
                            onSearchTypeChange(key);
                        } 
                    }}
                    trigger={['click']}
                    placement="bottomLeft"
                    onOpenChange={handleVisibleChange} 
                    getPopupContainer={(trigger) => trigger.parentElement}
                >
                    <Button 
                        className="border-[#F0F0F0] dark:border-[#303030] !px-12px rounded-16px text-[var(--ant-color-text-description)]"
                        onClick={(e) => {
                            e.stopPropagation();
                            if (onButtonClick) {
                                onButtonClick();
                            }
                        }}
                    >
                        <Space size={4} className="!leading-none">
                            <Search className="w-16px h-16px" />
                            {items.find((item) => item.key === searchType)?.label}
                            <ChevronDown className="w-20px h-20px" />
                        </Space>
                    </Button>
                </Dropdown>
            </div>
        )
    }
};