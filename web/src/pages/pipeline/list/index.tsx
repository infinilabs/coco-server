import { PlusOutlined } from '@ant-design/icons';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { Boxes, Pencil, Trash } from 'lucide-react';
import { Button, Card, Dropdown, Modal, Space, Switch, Table, Tag, Tooltip, message } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { MoreOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import useQueryParams from '@/hooks/common/queryParams';
import { deletePipeline, fetchPipelineList, updatePipeline } from '@/service/api/pipeline';
import { fetchDataSourceList } from '@/service/api/data-source';
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

/**
 * 数据源引用管道的两条途径：
 * - document_processing_config.pipeline：字符串，存管道 id（数据源表单的 Pipeline 下拉）
 * - enrichment_pipeline：PipelineConfigV2 对象（JSON 粘贴），按其 id/name 匹配
 */
const pipelineRefMatches = (ds: any, row: PipelineRow): boolean => {
  const dpc = ds?.document_processing_config?.pipeline;
  if (dpc && (dpc === row.id || dpc === row.name)) return true;
  const ep = ds?.enrichment_pipeline;
  if (ep && (ep.id === row.id || ep.name === row.id || ep.name === row.name)) return true;
  return false;
};

/**
 * 数据源列单元格：单行展示（前 3 个 Tag + 「+N」），内容被列宽裁剪
 * （或超出 3 个）时 hover 显示全量数据源名。
 */
function DatasourceCell({ names }: { names: string[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [clipped, setClipped] = useState(false);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const check = () => setClipped(el.scrollWidth > el.clientWidth + 1);
    check();
    // 列宽变化（窗口缩放/字体加载）后重测
    const ro = new ResizeObserver(check);
    ro.observe(el);
    return () => ro.disconnect();
  }, [names]);

  const body = (
    <div ref={containerRef} className="flex items-center gap-4px overflow-hidden">
      {names.slice(0, 3).map(n => (
        <Tag key={n} className="!m-0 flex-none">
          {n}
        </Tag>
      ))}
      {names.length > 3 && (
        <span className="flex-none text-12px text-[var(--ant-color-text-tertiary)]">+{names.length - 3}</span>
      )}
    </div>
  );

  if (names.length <= 3 && !clipped) return body;
  return (
    <Tooltip title={<div className="max-w-360px whitespace-normal">{names.join('、')}</div>}>
      {body}
    </Tooltip>
  );
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
  // 数据源按所引用的管道分组：pipeline id/name -> 数据源名列表
  const [dsUsage, setDsUsage] = useState<Record<string, string[]>>({});

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

  // 一次性拉全量数据源，前端按管道引用分组（数据源量级小，避免每行一次查询）
  const fetchDatasourceUsage = async () => {
    try {
      const res = await fetchDataSourceList({ from: 0, size: 10000 });
      const list = (formatESSearchResult(res?.data).data as any[]) || [];
      const usage: Record<string, string[]> = {};
      list.forEach(ds => {
        if (!ds?.name) return;
        const matched = new Set<string>();
        const dpc = ds?.document_processing_config?.pipeline;
        if (dpc) matched.add(String(dpc));
        const ep = ds?.enrichment_pipeline;
        if (ep?.id) matched.add(String(ep.id));
        if (ep?.name) matched.add(String(ep.name));
        matched.forEach(ref => {
          (usage[ref] = usage[ref] || []).push(ds.name);
        });
      });
      setDsUsage(usage);
    } catch {
      // 数据源权限缺失等场景：列显示为空，不影响列表主功能
    }
  };

  useEffect(() => {
    fetchData();
  }, [queryParams]);

  useEffect(() => {
    fetchDatasourceUsage();
  }, []);

  const onEnabledChange = (row: PipelineRow, value: boolean) => {
    // PUT /pipelines/:id 是整链替换语义（后端强制要求 name），必须回传完整配置，只覆盖 enabled
    updatePipeline(row.id, { ...row, enabled: value }).then(() => {
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
      title: t('page.pipeline.columns.datasources'),
      dataIndex: 'datasources',
      render: (_: any, row: PipelineRow) => {
        const names = dsUsage[row.id] || dsUsage[row.name] || [];
        if (names.length === 0) return <span className="text-[var(--ant-color-text-tertiary)]">—</span>;
        return <DatasourceCell names={names} />;
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
      width: 110,
      hidden: !permissions.update && !permissions.delete,
      render: (_: any, row: PipelineRow) => (
        <Dropdown
          trigger={['hover']}
          menu={{
            items: [
              permissions.update && {
                key: 'edit',
                icon: <Pencil size={14} />,
                label: t('common.edit'),
                onClick: () => navigate(`/pipeline/edit/${row.id}`)
              },
              permissions.delete && {
                key: 'delete',
                icon: <Trash size={14} />,
                label: t('common.delete'),
                danger: true,
                onClick: () =>
                  Modal.confirm({
                    title: t('page.pipeline.deleteConfirm', { name: row.name }),
                    onOk: () => onDelete(row)
                  })
              }
            ].filter(Boolean) as any[]
          }}
        >
          <Button type="text" size="small" icon={<MoreOutlined />} aria-label={t('common.operation')} />
        </Dropdown>
      )
    }
  ];

  return (
    <Card
      className="card-wrapper"
      title={
        <PageHeaderTitle
          description={t('page.pipeline.subtitle')}
          icon={<Boxes size={18} />}
          title={t('page.pipeline.title')}
        />
      }
      extra={
        <Space>
          <span className="text-13px text-[var(--ant-color-text-tertiary)]">{t('common.totalItems', { total })}</span>
          {permissions.create && (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/pipeline/new')}>
              {t('common.add')}
            </Button>
          )}
        </Space>
      }
    >
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} pagination={false} />
    </Card>
  );
}
