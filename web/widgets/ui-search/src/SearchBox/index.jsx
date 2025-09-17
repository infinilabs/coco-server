import { Input } from "antd";
import styles from "./index.module.less";
import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { CornerDownLeft } from "lucide-react";
import { Mic } from "lucide-react";

export function SearchBox(props) {

    const { placeholder, keyword, onSearch } = props;

    const [currentKeyword, setCurrentKeyword] = useState(keyword)

    useEffect(() => {
        setCurrentKeyword(keyword)
    }, [keyword])

    return (
        <div className={`flex w-full h-full items-center justify-center bg-#fff ${styles.searchbox}`}>
            <div className="w-full h-48px px-12px py-4px border border-solid border-[rgba(235,235,235,1)] rounded-8px">
                <Input.Search
                    value={currentKeyword}
                    addonBefore={<Search className="w-16px h-16px"/>} 
                    enterButton={<CornerDownLeft className="w-14px h-14px"/>}
                    size="large"
                    onChange={(e) => setCurrentKeyword(e.target.value)}
                    onSearch={(value) => onSearch(value)}
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
