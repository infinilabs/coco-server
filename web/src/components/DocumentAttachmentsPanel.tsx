import { Empty, List, Spin } from 'antd';
import { Download, FileText } from 'lucide-react';
import { useEffect, useState } from 'react';
import { request } from '@/service/request';

interface AttachmentInfo {
  id: string;
  name?: string;
  size?: number;
  mime_type?: string;
  created?: string;
}

const formatSize = (size?: number) => {
  if (!size) return '';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(0)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
};

/**
 * 文档附件面板:列出该文档富化时收集的全部附件(名称/大小/日期),
 * 并提供下载。数据来自 /attachment/_search 按 id 批量查询。
 */
export function DocumentAttachmentsPanel({
  docId,
  headers
}: {
  docId: string;
  headers?: Record<string, string>;
}) {
  const [loading, setLoading] = useState(true);
  const [attachments, setAttachments] = useState<AttachmentInfo[]>([]);

  useEffect(() => {
    if (!docId) return;
    let disposed = false;
    setLoading(true);

    request({ method: 'get', url: `/document/${docId}`, headers })
      .then(res => {
        const ids: string[] = res?.data?.attachments || [];
        if (!ids.length) {
          setAttachments([]);
          return;
        }
        const body = { attachments: ids };
        return request({
          method: 'post',
          url: '/attachment/_search',
          data: body,
          headers
        }).then(res2 => {
          const es = res2?.data || {};
          const hits: any[] = es?.hits?.hits || [];
          setAttachments(
            (hits as any[]).map(h => ({ id: h._id, ...(h._source || {}) })) as AttachmentInfo[]
          );
        });
      })
      .catch(() => setAttachments([]))
      .finally(() => {
        if (!disposed) setLoading(false);
      });

    return () => {
      disposed = true;
    };
  }, [docId, headers]);

  if (loading) return <div className='flex justify-center py-8'><Spin /></div>;
  if (!attachments.length)
    return <Empty description='暂无附件' image={Empty.PRESENTED_IMAGE_SIMPLE} />;

  return (
    <List
      dataSource={attachments}
      renderItem={att => (
        <List.Item
          actions={[
            <a
              aria-label='下载'
              href={`/attachment/${att.id}`}
              key='download'
              rel='noreferrer'
              target='_blank'
            >
              <Download size={14} />
            </a>
          ]}
        >
          <List.Item.Meta
            avatar={<FileText size={18} className='mt-2px text-[var(--ant-color-text-tertiary)]' />}
            description={
              <span className='text-xs text-[var(--ant-color-text-tertiary)]'>
                {formatSize(att.size)} {att.created ? `· ${att.created}` : ''}
              </span>
            }
            title={
              <a href={`/attachment/${att.id}`} rel='noreferrer' target='_blank'>
                {att.name || att.id}
              </a>
            }
          />
        </List.Item>
      )}
    />
  );
}
