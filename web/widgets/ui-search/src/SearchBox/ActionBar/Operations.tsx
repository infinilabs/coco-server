import { Badge, Button, Space, Upload } from "antd";
import { MessageCircle, Paperclip, Search } from "lucide-react";
import { type FC } from "react";
import { useTranslation } from "react-i18next";
import DeepresearchIcon from "../../icons/DeepresearchIcon";

export function getFileNameAndExt(fileName: string | undefined): string | undefined {
    if (!fileName) return;

    const lastDotIndex = fileName.lastIndexOf('.');
    if (lastDotIndex <= 0) {
        return;
    }
    return fileName.slice(lastDotIndex + 1);
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

interface OperationsProps {
    size?: number;
    disabled?: boolean;
    onSearch?: () => void;
    attachments?: any[];
    setAttachments?: (updater: (list: any[]) => any[]) => void;
    onAttachmentUpload?: (files: File[], cb: (res: any) => void) => void;
    action_type?: string;
}

const Operations: FC<OperationsProps> = (props) => {
    const { size = 32, disabled = false, onSearch, attachments = [], setAttachments, onAttachmentUpload, action_type } = props;

    const btnStyle = { minWidth: size, width: size, height: size }
    const { t } = useTranslation();

    const getActionIcon = () => {
        switch (action_type) {
            case 'deepthink':
                return <MessageCircle className="w-14px h-14px" />;
            case 'deepresearch':
                return <DeepresearchIcon className="w-14px h-14px" />;
            default:
                return <Search className="w-14px h-14px" />;
        }
    };

    return (
        <Space size={4} className="!leading-none">
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
                        size: formatBytes(f.size || 0),
                        status: 'uploading',
                    }))
                    setAttachments?.((list) => [...list, ...attachments]);
                    const files = (fileList || []).map(f => f.originFileObj || f);
                    fileList.forEach((file, i) => {
                        const localId = attachments[i].id;
                        onAttachmentUpload?.([file.originFileObj || file] as any, (res: any) => {
                            const serverIds = res?.result?.attachments || res?.attachments || [];
                            setAttachments?.((list) =>
                                list.map((a) => {
                                    if (a.id !== localId) return a;
                                    if (!res?.acknowledged || serverIds.length === 0) {
                                        return {
                                            ...a,
                                            status: "failed",
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
                        onMouseDown={(e) => e.preventDefault()}
                    />
                </Badge>
            </Upload>
            {/* <Button
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
            /> */}
            <Button
                style={btnStyle}
                className={`border-0 ml-4px !rounded-50%`}
                classNames={{ icon: `w-14px h-14px !text-14px` }}
                disabled={disabled}
                type="primary"
                shape="circle"
                icon={getActionIcon()}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => onSearch && onSearch()}
            />
        </Space>
    )
};

export default Operations;