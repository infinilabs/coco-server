import { Button, Dropdown, Space } from "antd";
import { Brain, ChevronDown, Search, Telescope } from "lucide-react";
import { useTranslation } from 'react-i18next';
import { type FC } from "react";

export const ACTION_TYPE_SEARCH = 'search'
export const ACTION_TYPE_SEARCH_HYBRID = 'hybrid'
export const ACTION_TYPE_SEARCH_KEYWORD = 'keyword'
export const ACTION_TYPE_SEARCH_SEMANTIC = 'semantic'
export const ACTION_TYPE_DEEPTHINK = 'deepthink'
export const ACTION_TYPE_DEEPSEARCH = 'deepresearch'
export const DEFAULT_SEARCH_FUZZINESS = 3
export const MIN_SEARCH_FUZZINESS = 0
export const MAX_SEARCH_FUZZINESS = 5
export const SORT_BEST_MATCH = '_score:desc'
export const SORT_CREATED_DESC = 'created:desc'
export const SORT_CREATED_ASC = 'created:asc'
export const SORT_UPDATED_DESC = 'updated:desc'
export const DEFAULT_SEARCH_SORT = SORT_BEST_MATCH

const SEARCH_SORTS = new Set([SORT_BEST_MATCH, SORT_CREATED_DESC, SORT_CREATED_ASC, SORT_UPDATED_DESC])

export const normalizeSearchFuzziness = (value: unknown) => {
    const fuzziness = Number(value);
    if (!Number.isFinite(fuzziness)) return DEFAULT_SEARCH_FUZZINESS;
    return Math.min(MAX_SEARCH_FUZZINESS, Math.max(MIN_SEARCH_FUZZINESS, Math.round(fuzziness)));
}

export const normalizeSearchSort = (value: unknown): string => {
    if (typeof value !== 'string') return DEFAULT_SEARCH_SORT;
    const sorts = value.split(',').filter((item) => SEARCH_SORTS.has(item));
    return sorts.length > 0 ? sorts.join(',') : DEFAULT_SEARCH_SORT;
}

interface SearchActionsProps {
    actionType?: string;
    searchType?: string;
    onSearchTypeChange?: (key: string) => void;
    onButtonClick?: () => void;
    onDropdownClose?: () => void;
}

const SearchActions: FC<SearchActionsProps> = (props) => {
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
                        <Brain className="w-16px h-16px" />
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
                        <Telescope className="w-16px h-16px" />
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
        
        const handleVisibleChange = (visible: boolean) => {
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
                            onSearchTypeChange?.(key);
                        } 
                    }}
                    trigger={['click']}
                    placement="bottomLeft"
                    onOpenChange={handleVisibleChange} 
                    getPopupContainer={(trigger) => trigger.parentElement!}
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

export default SearchActions;