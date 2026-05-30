import { Badge, Button, Space, Upload } from "antd";
import { cloneDeep } from "lodash";
import { Image, Mic, Paperclip, Search } from "lucide-react";
import { useTranslation } from "react-i18next";

export function getFileNameAndExt(fileName) {
    if (!fileName) return;

    const lastDotIndex = fileName.lastIndexOf('.');
    if (lastDotIndex <= 0) {
        return;
    }
    return fileName.slice(lastDotIndex + 1);
}

function formatBytes(n) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export default (props) => {
    const { size = 32, disabled = false, onSearch, attachments = [], setAttachments, onAttachmentUpload } = props;

    const btnStyle = { minWidth: size, width: size, height: size }
    const { t } = useTranslation();

    return (
        <Space size={4} styles={{ item: { lineHeight: 1 } }}>
            <Upload
                name={'attachments'}
                multiple
                showUploadList={false}
                fileList={[]}
                beforeUpload={() => false}
                onChange={({ fileList }) => {
                    const attachments = (fileList || []).map((f, index) => ({
                        id: f.uid,
                        filename: f.name,
                        extname: getFileNameAndExt(f.name),
                        type: f.type,
                        size: formatBytes(f.size),
                        status: 'uploading',
                    }))
                    setAttachments((list) => [...list, ...attachments]);
                    const files = (fileList || []).map(f => f.originFileObj || f);
                    fileList.forEach((file, i) => {
                        const localId = attachments[i].id;
                        onAttachmentUpload([file.originFileObj || file], (res) => {
                            const serverIds = res?.result?.attachments || res?.attachments || [];
                            setAttachments((list) =>
                                list.map((a) => {
                                    if (a.id !== localId) return a;
                                    if (!res?.acknowledged || serverIds.length === 0) {
                                        return {
                                            ...a,
                                            status: "error",
                                            error: res?.error?.message || t("search.input.attachment_upload_failed") || "Upload failed",
                                        };
                                    }
                                    return { ...a, status: "uploaded", id: serverIds[0] };
                                })
                            );
                        });
                    });
                }}
            >
                <Badge count={attachments.length} size="small" classNames={{ indicator: '!text-10px'}}>
                    <Button
                        style={btnStyle}
                        classNames={{ icon: `w-16px h-16px !text-16px flex items-center justify-center` }}
                        icon={<Paperclip className="w-16px h-16px" />}
                        type="text"
                        shape="circle"
                    />
                </Badge>
            </Upload>
            <Button
                style={btnStyle}
                classNames={{ icon: `w-16px h-16px !text-16px flex items-center justify-center` }}
                icon={<Image className="w-16px h-16px" />}
                type="text"
                shape="circle"
                disabled
            />
            <Button
                style={btnStyle}
                classNames={{ icon: `w-16px h-16px !text-16px flex items-center justify-center` }}
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
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => onSearch && onSearch()}
            />
        </Space>
    )
};