import { Button, Input } from "antd";
import styles from "./index.module.less";
import { useEffect, useState } from "react";
import { CornerDownLeft } from "lucide-react";
import { Mic } from "lucide-react";
import { Dropdown } from "antd";
import { Space } from "antd";
import { ChevronDown, Search } from "lucide-react";
import { List } from "antd";
import { MessageCircle } from "lucide-react";

export function SearchBox(props) {

    const { placeholder, query, onSearch, minimize = false } = props;

    const [currentKeyword, setCurrentKeyword] = useState(query)

    useEffect(() => {
        setCurrentKeyword(query)
    }, [query])

    const isSearchDisabled = (currentKeyword || '').trim().length === 0

    return minimize ? (
        <div className={`flex w-full h-full items-center justify-center bg-[var(--ui-search-antd-color-bg-container)] ${styles.searchbox} rounded-8px`}>
            <div className="w-full h-48px px-12px py-3px border border-solid border-[var(--ui-search-antd-color-border)] rounded-8px">
                <Input.Search
                    value={currentKeyword}
                    addonBefore={<Search className="relative top-2px w-16px h-16px" />}
                    enterButton={<CornerDownLeft className="w-14px h-14px" />}
                    size="large"
                    onChange={(e) => setCurrentKeyword(e.target.value)}
                    onSearch={(value) => onSearch && onSearch(value)}
                    suffix={(
                        <Mic className="w-16px h-14px" />
                    )}
                    placeholder={placeholder}
                    autoFocus
                />
            </div>
        </div>
    ) : (
        <div className={`pt-16px pb-12px rounded-12px overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] border border-[rgba(235,235,235,1)] ${styles.searchbox}`}>
            <Input.TextArea
                placeholder={placeholder}
                autoSize={{ minRows: isSearchDisabled ? 2 : 1, maxRows: 6 }}
                classNames={{ textarea: '!text-16px !px-16px !pt-0 !pb-0 !mb-14px !bg-transparent' }}
                onChange={(e) => setCurrentKeyword(e.target.value)}
            />
            {
                !isSearchDisabled && (
                    <List
                        className="px-8px mb-20px"
                        itemLayout="vertical"
                        size="large"
                        pagination={false}
                        dataSource={[
                            {
                                icon: <Search className="w-16px h-16px"/>,
                                desc: `快速查找 | 直达文件与结果`
                            },
                            {
                                icon: <MessageCircle className="w-16px h-16px"/>,
                                desc: `深度思考 | AI 提炼，结论优先`
                            },
                            {
                                icon: <Search className="w-16px h-16px"/>,
                                desc: `深度研究 | 多步推理，综合分析`
                            },
                        ]}
                        renderItem={(item, index) => {
                            return (
                                <div className="h-40px px-8px flex items-center">
                                    <span className="mr-8px text-[var(--ui-search-antd-color-text-description)]">{item.icon}</span>
                                    <span className="mr-12px">{currentKeyword}</span>
                                    <span className="text-[var(--ui-search-antd-color-text-description)]">-{item.desc}</span>
                                </div>
                            )
                        }}
                    />
                )
            }
            {
                !isSearchDisabled && (
                    <List
                        header={"搜索建议"}
                        className="px-8px mb-20px"
                        itemLayout="vertical"
                        size="large"
                        pagination={false}
                        dataSource={[
                            {
                                icon: <Search className="w-16px h-16px"/>,
                                desc: `常用`
                            },
                            {
                                icon: <Search className="w-16px h-16px"/>,
                                desc: `最近`
                            },
                        ]}
                        renderItem={(item, index) => {
                            return (
                                <div className="h-40px px-8px flex items-center">
                                    <span className="mr-8px text-[var(--ui-search-antd-color-text-description)]">{item.icon}</span>
                                    <span className="mr-12px">{currentKeyword}</span>
                                    <span className="text-[var(--ui-search-antd-color-text-description)]">-{item.desc}</span>
                                </div>
                            )
                        }}
                    />
                )
            }
            <div className="flex justify-between items-center px-12px">
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
                <div>
                    <Button 
                        className={`border-0`}
                        classNames={{ icon: `w-14px h-14px !text-14px` }}
                        disabled={isSearchDisabled} 
                        type="primary" 
                        shape="circle" 
                        icon={<Search className="w-14px h-14px"/>}
                        onClick={() => onSearch && onSearch(currentKeyword)} 
                    />
                </div>
            </div>
        </div>
    )
}

export default SearchBox;
