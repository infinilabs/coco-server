import { PlusOutlined } from '@ant-design/icons';
import { Boxes, Pencil, Trash } from 'lucide-react';
import { Button, Card, Popconfirm, Space, Switch, Table, message } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import useQueryParams from '@/hooks/common/queryParams';
import { deletePipeline, fetchPipelineList, updatePipeline } from '@/service/api/pipeline';
import { formatESSearchResult } from '@/service/request/es';

interface PipelineRow {
  id: string;
  name: string;
  description?: string;
  enabled?: boolean;
  processor?: Record<string, any>[];
  created?: string;
  updated?: string;
}

function entryNames(row: PipelineRow): string[] {
  return (row.processor ?? []).map((entry) => ('if' in entry ? 'if/else' : Object.keys(entry)[0] ?? '?'));
}

export function Component() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [queryParams, setQueryParams] = useQueryParams();
  const { hasAuth } = useAuth();

  const permissions = {
    create: hasAuth('generic#pipeline/create'),
    update: hasAuth('generic#pipeline/update'),
    delete: hasAuth('generic#pipeline/delete')
  };

  const [data, setData] = useState<PipelineRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res = await fetchPipelineList({ ...queryParams, size: 50 });
      if (res?.data) {
        const result = formatESSearchResult(res.data);
        setData(result.data ?? []);
        setTotal(result.total ?? 0);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [queryParams]);

  const onEnabledChange = (row: PipelineRow, value: boolean) => {
    updatePipeline(row.id, { enabled: value }).then(() => {
      message.success(t('common.updateSuccess'));
      setQueryParams((old: any) => ({ ...old, t: Date.now() }));
    });
  };

  const onDelete = (row: PipelineRow) => {
    deletePipeline(row.id).then(() => {
      message.success(t('common.deleteSuccess'));
      setQueryParams((old: any) => ({ ...old, t: Date.now() }));
    });
  };

  const columns: any[] = [
    {
      title: t('page.pipeline.columns.name'),
      dataIndex: 'name',
      render: (_: any, row: PipelineRow) => (
        <Button type="link" className="!px-0" onClick={() => navigate(`/pipeline/edit/${row.id}`)}>
          {row.name}
        </Button>
      )
    },
    {
      title: t('page.pipeline.columns.enabled'),
      dataIndex: 'enabled',
      width: 100,
      render: (_: any, row: PipelineRow) => (
        <Switch size="small" checked={!!row.enabled} disabled={!permissions.update} onChange={(v) => onEnabledChange(row, v)} />
      )
    },
    {
      title: t('page.pipeline.columns.processors'),
      dataIndex: 'processor',
      render: (_: any, row: PipelineRow) => {
        const names = entryNames(row);
        if (names.length === 0) return <span className="text-[var(--ant-color-text-tertiary)]">—</span>;
        return (
          <Space size={4} wrap>
            {names.slice(0, 4).map((n, i) => (
              <span key={`${n}-${i}`} className="font-mono text-12px px-4px py-1px rounded-4px bg-[var(--ant-color-fill-tertiary)]">
                {n}
              </span>
            ))}
            {names.length > 4 && <span className="text-12px text-[var(--ant-color-text-tertiary)]">+{names.length - 4}</span>}
          </Space>
        );
      }
    },
    {
      title: t('page.pipeline.columns.description'),
      dataIndex: 'description',
      ellipsis: true
    },
    {
      title: t('common.operation'),
      key: 'actions',
      width: 140,
      hidden: !permissions.update && !permissions.delete,
      render: (_: any, row: PipelineRow) => (
        <Space>
          {permissions.update && (
            <Button type="text" size="small" icon={<Pencil size={14} />} onClick={() => navigate(`/pipeline/edit/${row.id}`)}>
              {t('common.edit')}
            </Button>
          )}
          {permissions.delete && (
            <Popconfirm title={t('page.pipeline.deleteConfirm', { name: row.name })} onConfirm={() => onDelete(row)}>
              <Button type="text" size="small" danger icon={<Trash size={14} />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  return (
    <Card
      className="card-wrapper"
      title={
        <span className="flex items-center gap-8px">
          <Boxes size={18} />
          {t('page.pipeline.title')}
          <span className="text-13px font-normal text-[var(--ant-color-text-tertiary)]">{t('common.totalItems', { total })}</span>
        </span>
      }
      extra={
        permissions.create && (
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/pipeline/new')}>
            {t('common.add')}
          </Button>
        )
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} pagination={false} />
    </Card>
  );
}
