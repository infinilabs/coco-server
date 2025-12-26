import { Input } from "antd";
import styles from "./index.module.less";
import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { CornerDownLeft } from "lucide-react";
import { Mic } from "lucide-react";

export function SearchBox(props) {

    const { placeholder, query, onSearch } = props;

    const [currentKeyword, setCurrentKeyword] = useState(query)

    useEffect(() => {
        setCurrentKeyword(query)
    }, [query])

    return (
        <div className={`flex w-full h-full items-center justify-center bg-[var(--ui-search-antd-color-bg-container)] ${styles.searchbox} rounded-8px`}>
            <div className="w-full h-48px px-12px py-3px border border-solid border-[var(--ui-search-antd-color-border)] rounded-8px">
                <Input.Search
                    value={currentKeyword}
                    addonBefore={<Search className="relative top-2px w-16px h-16px"/>} 
                    enterButton={<CornerDownLeft className="w-14px h-14px"/>}
                    size="large"
                    onChange={(e) => setCurrentKeyword(e.target.value)}
                    onSearch={(value) => onSearch && onSearch(value)}
                    suffix={(
                        <Mic className="w-16px h-14px"/>
                    )}
                    placeholder={placeholder}
                    autoFocus
                />
            </div>
        </div>
    )
}

export default SearchBox;
