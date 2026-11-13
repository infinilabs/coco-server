import { Empty, Table, Tag, Typography } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { request } from '@/service/request';

interface ChunkItem {
  index: number;
  breadcrumb?: string;
  pages?: { start: number; end: number };
  excerpt?: string;
  vectorized?: boolean;
}

/**
 * 文档分块列表（W13b 读侧）：结构化切块的序号 / 面包屑 / 页区间 /
 * 摘录 / 向量化标记。数据源：GET /document/:id/_chunks。
 * 用于文档详情页的「文档分块」页签；embeddingModel 由调用方传入
 * （文档字段），展示向量模型元信息而不渲染向量值。
 */
export default function ChunkTable({
  docId,
  headers,
  embeddingModel
}: {
  docId?: string;
  headers?: Record<string, string>;
  embeddingModel?: string;
}) {
  const { t } = useTranslation();
  const [chunks, setChunks] = useState<ChunkItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    if (!docId) return;
    setLoading(true);
    try {
      const res: any = await request({
        method: 'get',
        url: `/document/${docId}/_chunks?from=0&size=50`,
        headers
      }).catch(() => null);
      setChunks((res?.data?.chunks as ChunkItem[]) ?? []);
      setTotal(res?.data?.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [docId, headers]);

  useEffect(() => {
    load();
  }, [load]);

  const columns = [
    { title: '#', dataIndex: 'index', width: 56 },
    {
      title: t('page.preview.processing.breadcrumb'),
      dataIndex: 'breadcrumb',
      width: 180,
      ellipsis: true,
      render: (v: string) => v || '-'
    },
    {
      title: t('page.preview.processing.pages'),
      key: 'pages',
      width: 90,
      render: (_: unknown, r: ChunkItem) => (r.pages ? `p${r.pages.start}-${r.pages.end}` : '-')
    },
    {
      title: t('page.preview.processing.excerpt'),
      dataIndex: 'excerpt',
      ellipsis: { showTitle: false },
      render: (v: string) => (
        <Typography.Text ellipsis={{ tooltip: v }} className='text-12px!'>
          {v || '-'}
        </Typography.Text>
      )
    },
    {
      title: t('page.preview.processing.vectorized'),
      dataIndex: 'vectorized',
      width: 80,
      render: (v: boolean) => (v ? <Tag color='green'>✓</Tag> : <Tag color='default'>—</Tag>)
    }
  ];

  // 空态：无分块时给出缺省说明（可能未配置切块/向量化处理器），不渲染空表头
  if (!loading && total === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.datasource.doc.noChunks')} />;
  }

  return (
    <div className='flex flex-col gap-8px'>
      <div className='flex items-center gap-8px flex-wrap'>
        <Typography.Text type='secondary' className='text-12px'>
          {t('page.preview.processing.chunksTitle', { total: String(total) })}
        </Typography.Text>
        {embeddingModel ? (
          <Tag color='geekblue' className='mr-0'>
            {t('page.preview.processing.model')}: {embeddingModel}
          </Tag>
        ) : null}
      </div>
      <Table<ChunkItem>
        rowKey='index'
        size='small'
        loading={loading}
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        columns={columns}
        dataSource={chunks}
        locale={{ emptyText: t('page.searchOps.noData') }}
      />
    </div>
  );
}
