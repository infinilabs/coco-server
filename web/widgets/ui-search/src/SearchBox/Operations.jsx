import { Button, Space } from "antd";
import { Image, Mic, Paperclip, Search } from "lucide-react";

export default (props) => {
    const { size = 32, disabled = false, onSearch } = props; 

    const btnStyle = { minWidth: size, width: size, height: size }

    return (
        <Space size={4} styles={{ item: { lineHeight: 1 } }}>
            <Button
                style={btnStyle}
                classNames={{ icon: `w-16px h-16px !text-16px` }}
                icon={<Paperclip className="w-16px h-16px" />}
                type="text"
                shape="circle"
                disabled
            />
            <Button
                style={btnStyle}
                classNames={{ icon: `w-16px h-16px !text-16px` }}
                icon={<Image className="w-16px h-16px" />}
                type="text"
                shape="circle"
                disabled
            />
            <Button
                style={btnStyle}
                classNames={{ icon: `w-16px h-16px !text-16px` }}
                icon={<Mic className="w-16px h-16px" />}
                type="text"
                shape="circle"
                disabled
            />
            <Button
                style={btnStyle}
                className={`border-0 ml-4px !rounded-50%`}
                classNames={{ icon: `w-14px h-14px !text-14px` }}
                disabled={disabled}
                type="primary"
                shape="circle"
                icon={<Search className="w-14px h-14px" />}
                onClick={() => onSearch && onSearch()}
            />
        </Space>
    )
};