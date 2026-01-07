import { Button, Dropdown, Space } from "antd";
import { ChevronDown, Globe, Hammer, Search } from "lucide-react";

export default (props) => {
    const { selectedItem } = props; 

    if (selectedItem?.action === 'deepthink') {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4}>
                        <Search className="w-16px h-16px"/>
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
    } else if (selectedItem?.action === 'deepresearch') {
        return (
            <Space size={0}>
                <Button className="!px-12px rounded-16px !text-#1784FC !bg-[rgba(204,232,250,1)] !border-0 mr-4px">
                    <Space size={4}>
                        <Search className="w-16px h-16px"/>
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
        return (
            <div>
                <Dropdown menu={{ items: [
                    {
                        key: '1',
                        label: '混合搜索',
                    },
                    {
                        key: '2',
                        label: '关键词搜索',
                    },
                    {
                        key: '3',
                        label: '语义搜索',
                    },
                ]}}>
                    <Button className="!px-12px rounded-16px text-[var(--ui-search-antd-color-text-description)]">
                        <Space size={4}>
                            <Search className="w-16px h-16px"/>
                            混合搜索
                            <ChevronDown className="w-20px h-20px" />
                        </Space>
                    </Button>
                </Dropdown>
            </div>
        )
    }
};