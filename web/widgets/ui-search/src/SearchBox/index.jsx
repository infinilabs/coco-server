import { Button, Input } from "antd";
import styles from "./index.module.less";
import { useCallback, useEffect, useMemo, useState } from "react";
import { CornerDownLeft, Image, Paperclip } from "lucide-react";
import { Mic } from "lucide-react";
import { Dropdown } from "antd";
import { Space } from "antd";
import { ChevronDown, Search } from "lucide-react";
import QuickAccess from "./QuickAccess";
import SearchActions from "./SearchActions";

export function SearchBox(props) {

    const { placeholder, query, onSearch, minimize = false } = props;

    const [currentKeyword, setCurrentKeyword] = useState(query)
    const [selectedItem, setSelectedItem] = useState()

    useEffect(() => {
        setCurrentKeyword(query)
    }, [query])

    const calculateCharLength = useCallback((str) => {
        if (!str) return 0;
        let totalLength = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charAt(i);
            if (/[\u4e00-\u9fa5\u3000-\u303f\uff00-\uffef]/.test(char)) {
                totalLength += 2;
            } else {
                totalLength += 1;
            }
        }
        return totalLength;
    }, []);

    const showSuggestion = useMemo(() => {
        return calculateCharLength(currentKeyword) < 40;
    }, [currentKeyword])

    const hasKeyword = useMemo(() => {
        return (currentKeyword || '').trim().length > 0
    }, [currentKeyword])

    return minimize ? (
        <div className={`flex w-full h-full items-center justify-center bg-[rgba(243,244,246,1)] ${styles.searchbox} rounded-8px`}>
            <div className="w-full h-48px px-12px rounded-12px">
                <Space.Compact className="items-center w-full h-full flex">
                    <Search
                        className="relative top-2px w-16px h-16px flex-shrink-0 text-#999"
                    />
                    <Input.Search
                        value={currentKeyword}
                        enterButton={<Search className="w-14px h-14px" />}
                        size="large"
                        onChange={(e) => setCurrentKeyword(e.target.value)}
                        onSearch={(value) => onSearch && onSearch(value)}
                        suffix={(
                            <Space size={0}>
                                <Button
                                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                                    className="w-24px h-24px"
                                    icon={<Paperclip strokeWidth={1} className="w-16px h-16px" />}
                                    type="text"
                                    shape="circle"
                                    disabled
                                />
                                <Button
                                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                                    className="w-24px h-24px"
                                    icon={<Image strokeWidth={1} className="w-16px h-16px" />}
                                    type="text"
                                    shape="circle"
                                    disabled
                                />
                                <Button
                                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                                    className="w-24px h-24px"
                                    icon={<Mic strokeWidth={1} className="w-16px h-16px" />}
                                    type="text"
                                    shape="circle"
                                    disabled
                                />
                            </Space>
                            // <Mic className="w-16px h-14px flex-shrink-0" /> 
                        )}
                        placeholder={placeholder}
                        autoFocus
                        className="flex-1 w-full"
                    />
                </Space.Compact>
            </div>
        </div>
    ) : (
        <div className={`pt-16px pb-12px rounded-12px overflow-hidden shadow-[0_2px_20px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)] border border-[rgba(235,235,235,1)] dark:border-[rgba(50,50,50,1)] ${styles.searchbox}`}>
            <Input.TextArea
                placeholder={placeholder}
                autoSize={{ minRows: hasKeyword ? 1 : 2, maxRows: 6 }}
                classNames={{ textarea: '!text-16px !px-16px !pt-0 !pb-0 !mb-14px !bg-transparent' }}
                onChange={(e) => setCurrentKeyword(e.target.value)}
            />
            {
                hasKeyword && (
                    <QuickAccess
                        keyword={currentKeyword}
                        suggestions={showSuggestion ? [
                            {
                                icon: <Search className="w-16px h-16px" />,
                                keyword: '搜索建议 1',
                                desc: `常用`,
                                type: 'suggestion'
                            },
                            {
                                icon: <Search className="w-16px h-16px" />,
                                keyword: '搜索建议 2',
                                desc: `最近`,
                                type: 'suggestion'
                            },
                        ] : []}
                        onSelectItem={(item) => {
                            setSelectedItem(item)
                        }}
                        onClickItem={(item) => {
                            onSearch(item.keyword || currentKeyword)
                        }}
                    />
                )
            }
            <div className="flex justify-between items-center px-12px">
                <SearchActions selectedItem={selectedItem} />
                <Space size={0}>
                    <Button
                        classNames={{ icon: `w-16px h-16px !text-16px` }}
                        icon={<Paperclip strokeWidth={1} className="w-16px h-16px" />}
                        type="text"
                        shape="circle"
                        disabled
                    />
                    <Button
                        classNames={{ icon: `w-16px h-16px !text-16px` }}
                        icon={<Image strokeWidth={1} className="w-16px h-16px" />}
                        type="text"
                        shape="circle"
                        disabled
                    />
                    <Button
                        classNames={{ icon: `w-16px h-16px !text-16px` }}
                        icon={<Mic strokeWidth={1} className="w-16px h-16px" />}
                        type="text"
                        shape="circle"
                        disabled
                    />
                    <Button
                        className={`border-0 ml-8px`}
                        classNames={{ icon: `w-14px h-14px !text-14px` }}
                        disabled={!hasKeyword}
                        type="primary"
                        shape="circle"
                        icon={<Search className="w-14px h-14px" />}
                        onClick={() => onSearch && onSearch(currentKeyword)}
                    />
                </Space>
            </div>
        </div>
    )
}

export default SearchBox;
