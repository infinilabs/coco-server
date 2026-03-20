import { Button, Dropdown, Space } from "antd";
import { ChevronDown, Globe, Hammer, Search } from "lucide-react";

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

    if (actionType === ACTION_TYPE_DEEPTHINK) {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4}>
                        <Search className="w-16px h-16px" />
                        深度思考
                    </Space>
                </Button>
                <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Globe strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                />
                <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Hammer strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                />
            </Space>
        )
    } else if (actionType === ACTION_TYPE_DEEPSEARCH) {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4}>
                        <Search className="w-16px h-16px" />
                        深度研究
                    </Space>
                </Button>
                <Button
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Globe strokeWidth={1} className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                />
            </Space>
        )
    } else {
        const items = [
            {
                key: ACTION_TYPE_SEARCH_HYBRID,
                label: '混合搜索',
            },
            {
                key: ACTION_TYPE_SEARCH_KEYWORD,
                label: '关键词搜索',
            },
            {
                key: ACTION_TYPE_SEARCH_SEMANTIC,
                label: '语义搜索',
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
                >
                    <Button 
                        className="!px-12px rounded-16px text-[var(--ui-search-antd-color-text-description)]"
                        onClick={(e) => {
                            e.stopPropagation();
                            if (onButtonClick) {
                                onButtonClick();
                            }
                        }}
                    >
                        <Space size={4}>
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