import { Button, Space, Upload } from "antd";
import { cloneDeep } from "lodash";
import { Image, Mic, Paperclip, Search } from "lucide-react";

export function getFileNameAndExt(fileName) {
  if (!fileName) return;
  
  const lastDotIndex = fileName.lastIndexOf('.');
  if (lastDotIndex <= 0) { 
    return;
  }
  return fileName.slice(lastDotIndex+1);
}

export default (props) => {
    const { size = 32, disabled = false, onSearch, attachments = [], setAttachments } = props;

    const btnStyle = { minWidth: size, width: size, height: size }

    console.log('attachments', attachments)
    
    return (
        <Space size={4} styles={{ item: { lineHeight: 1 } }}>
            <Upload 
                name={'attachments'}
                action={''}
                showUploadList={false}
                fileList={attachments.map((item) => item.file)}
                beforeUpload={(file, fileList) => {
                    setAttachments((prev) => {
                        const newAttachments = cloneDeep(prev);
                        const index = newAttachments.findIndex((item) => item.id === file.uid);
                        if (index === -1) {
                            newAttachments.push({
                                id: file.uid,
                                filename: file.name,
                                extname: getFileNameAndExt(file.name),
                                type: file.type,
                                size: file.size,
                                status: 'uploading',
                                file,
                            })
                        } else {
                            newAttachments[index] = {
                                id: file.uid,
                                filename: file.name,
                                extname: getFileNameAndExt(file.name),
                                type: file.type,
                                size: file.size,
                                status: 'uploading',
                                file,
                            }
                        }
                        return newAttachments
                    })
                    
                    const reader = new FileReader();
                    reader.readAsDataURL(file);
                    reader.onload = () => {
                        setAttachments((prev) => {
                            const newAttachments = cloneDeep(prev);
                            const index = newAttachments.findIndex((item) => item.id === file.uid);
                            if (index !== -1) {
                                newAttachments[index] = {
                                    id: file.uid,
                                    filename: file.name,
                                    extname: getFileNameAndExt(file.name),
                                    type: file.type,
                                    size: file.size,
                                    status: 'uploaded',
                                    file,
                                }
                            }
                            return newAttachments
                        })
                    };
                    return false
                }}
            >
                <Button
                    style={btnStyle}
                    classNames={{ icon: `w-16px h-16px !text-16px` }}
                    icon={<Paperclip className="w-16px h-16px" />}
                    type="text"
                    shape="circle"
                />
            </Upload>
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