import { SyncOutlined } from '@ant-design/icons';
import PageHeaderTitle from '@/components/stateless/common/PageHeaderTitle';
import { RadioTower } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, Descriptions, Drawer, Empty, Popconfirm, Radio, Space, Spin, Table, Tabs, Tag, Tooltip, message } from 'antd';
import { useEffect, useState } from 'react';

import { Deployments } from './Deployments';
import {
  approveGatewayInstance,
  fetchGatewayInstanceDetail,
  fetchGatewayInstances,
  forgetGatewayInstance,
  proxyGatewayInstance,
  type GatewayInstance,
  type GatewayInstanceDetail
} from '@/service/api/gateway';

const ONLINE_WINDOW_MS = 90 * 1000;

function formatSync(ms?: number): string {
  if (!ms) return '-';
  return new Date(ms).toLocaleString();
}

export function Component() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    read: hasAuth('coco#gateway_instance/read'),
    search: hasAuth('coco#gateway_instance/search'),
    update: hasAuth('coco#gateway_instance/update'),
    remove: hasAuth('coco#gateway_instance/delete')
  };

  const [data, setData] = useState<GatewayInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [typeFilter, setTypeFilter] = useState<string>('');
  const [detailId, setDetailId] = useState<string>();
  const [detail, setDetail] = useState<GatewayInstanceDetail>();
  const [detailLoading, setDetailLoading] = useState(false);
  const [liveInfo, setLiveInfo] = useState<any>();
  const [liveLoading, setLiveLoading] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res: any = await fetchGatewayInstances(typeFilter || undefined);
      if (res?.data?.items) {
        setData(res.data.items);
      } else if ((res as any)?.error) {
        message.error((res as any).error);
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, [typeFilter]);

  // keep online dots fresh while the page is open
  useEffect(() => {
    const timer = setInterval(fetchData, 15000);
    return () => clearInterval(timer);
  }, [typeFilter]);

  const openDetail = async (id: string) => {
    setDetailId(id);
    setDetail(undefined);
    setLiveInfo(undefined);
    setDetailLoading(true);
    try {
      const res: any = await fetchGatewayInstanceDetail(id);
      const d = res?.data;
      setDetail(d);
      // live version info through the reverse channel; failure is expected
      // for offline instances — the registration record is the source of truth
      if (d?.reverse_connected) {
        setLiveLoading(true);
        const info: any = await proxyGatewayInstance(id, '/_info');
        setLiveInfo(info?.data?.data ?? info?.data);
        setLiveLoading(false);
      }
    } finally {
      setDetailLoading(false);
    }
  };

  const onApprove = async (row: GatewayInstance) => {
    const res: any = await approveGatewayInstance(row.id);
    if (res?.data) {
      message.success(t('page.gateway.approveDone', { name: row.alias || row.name }));
      fetchData();
    }
  };

  const onForget = async (row: GatewayInstance) => {
    const res: any = await forgetGatewayInstance(row.id);
    if (res?.data) {
      message.success(t('page.gateway.forgetDone', { name: row.alias || row.name }));
      fetchData();
    }
  };

  const columns: any[] = [
    {
      title: t('page.gateway.columns.name'),
      dataIndex: 'name',
      render: (_: any, row: GatewayInstance) => (
        <Space>
          {permissions.read ? (
            <Button type="link" className="!px-0" onClick={() => openDetail(row.id)}>
              {row.alias || row.name}
            </Button>
          ) : (
            <span>{row.alias || row.name}</span>
          )}
          {row.alias && <span className="text-12px text-[var(--ant-color-text-tertiary)]">{row.name}</span>}
        </Space>
      )
    },
    {
      title: t('page.gateway.columns.type'),
      dataIndex: 'type',
      width: 110,
      render: (v: string) => <Tag>{v || '-'}</Tag>
    },
    {
      title: t('page.gateway.columns.status'),
      dataIndex: 'status',
      width: 110,
      render: (v: string) =>
        v === 'approved' ? <Tag color="success">{t('page.gateway.status.approved')}</Tag> : <Tag color="warning">{t('page.gateway.status.pending')}</Tag>
    },
    {
      title: t('page.gateway.columns.online'),
      dataIndex: 'online',
      width: 100,
      render: (online: boolean, row: GatewayInstance) => (
        <Tooltip title={t('page.gateway.lastSync', { time: formatSync(row.last_sync) })}>
          <Badge status={online ? 'success' : 'default'} text={online ? t('page.gateway.online') : t('page.gateway.offline')} />
        </Tooltip>
      )
    },
    {
      title: t('page.gateway.columns.reverse'),
      dataIndex: 'reverse_connected',
      width: 100,
      render: (v: boolean) =>
        v ? (
          <Space size={4}>
            <RadioTower size={14} />
            <span>{t('page.gateway.reverseConnected')}</span>
          </Space>
        ) : (
          <span className="text-[var(--ant-color-text-tertiary)]">—</span>
        )
    },
    {
      title: t('page.gateway.columns.version'),
      dataIndex: 'version',
      width: 100,
      render: (v: string) => v || '-'
    },
    {
      title: t('page.gateway.columns.endpoint'),
      dataIndex: 'endpoint',
      ellipsis: true,
      render: (v: string) => v || '-'
    },
    {
      title: t('page.gateway.columns.groups'),
      dataIndex: 'groups',
      width: 140,
      render: (v: string[]) =>
        v?.length ? (
          <Space size={4} wrap>
            {v.map(g => (
              <Tag key={g} color="blue">
                {g}
              </Tag>
            ))}
          </Space>
        ) : (
          '-'
        )
    },
    {
      title: t('common.operation'),
      key: 'actions',
      width: 170,
      render: (_: any, row: GatewayInstance) => (
        <Space>
          {permissions.update && row.status !== 'approved' && (
            <Popconfirm title={t('page.gateway.approveConfirm', { name: row.alias || row.name })} onConfirm={() => onApprove(row)}>
              <Button type="link" size="small" className="!px-0">
                {t('page.gateway.approve')}
              </Button>
            </Popconfirm>
          )}
          {permissions.remove && (
            <Popconfirm title={t('page.gateway.forgetConfirm', { name: row.alias || row.name })} onConfirm={() => onForget(row)}>
              <Button type="link" size="small" danger className="!px-0">
                {t('page.gateway.forget')}
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
        <PageHeaderTitle
          description={t('page.gateway.subtitle')}
          icon={<RadioTower size={18} />}
          title={t('page.gateway.title')}
        />
      }
    >
      <Tabs
        defaultActiveKey="instances"
        items={[
          {
            key: 'instances',
            label: t('page.gateway.tabInstances'),
            children: (
              <>
                <div className="mb-12px flex items-center justify-between">
                  <Radio.Group value={typeFilter} onChange={e => setTypeFilter(e.target.value)} size="small">
                    <Radio.Button value="">{t('page.gateway.filterAll')}</Radio.Button>
                    <Radio.Button value="gateway">{t('page.gateway.filterGateway')}</Radio.Button>
                    <Radio.Button value="agent">{t('page.gateway.filterAgent')}</Radio.Button>
                  </Radio.Group>
                  <Space>
                    <span className="text-13px text-[var(--ant-color-text-tertiary)]">
                      {t('common.totalItems', { total: data.length })}
                    </span>
                    <Button icon={<SyncOutlined />} onClick={fetchData} />
                  </Space>
                </div>
                <Table rowKey="id" loading={loading} columns={columns} dataSource={data} pagination={false} />
              </>
            )
          },
          {
            key: 'deployments',
            label: t('page.gateway.tabDeployments'),
            children: <Deployments onRefreshTargets={fetchData} />
          }
        ]}
      />
      <Drawer
        destroyOnClose
        open={!!detailId}
        title={detail ? detail.alias || detail.name : detailId}
        width={640}
        onClose={() => setDetailId(undefined)}
      >
        <Spin spinning={detailLoading}>
          {detail ? (
            <Space direction="vertical" size="middle" className="w-full">
              <Descriptions
                bordered
                column={2}
                size="small"
                items={[
                  { key: 'id', label: 'ID', children: <span className="font-mono text-12px">{detail.id}</span> },
                  { key: 'type', label: t('page.gateway.columns.type'), children: detail.type || '-' },
                  {
                    key: 'status',
                    label: t('page.gateway.columns.status'),
                    children:
                      detail.status === 'approved' ? (
                        <Tag color="success">{t('page.gateway.status.approved')}</Tag>
                      ) : (
                        <Tag color="warning">{t('page.gateway.status.pending')}</Tag>
                      )
                  },
                  {
                    key: 'online',
                    label: t('page.gateway.columns.online'),
                    children: (
                      <Badge
                        status={detail.online ? 'success' : 'default'}
                        text={`${detail.online ? t('page.gateway.online') : t('page.gateway.offline')} · ${formatSync(detail.last_sync)}`}
                      />
                    )
                  },
                  { key: 'version', label: t('page.gateway.columns.version'), children: detail.version || '-' },
                  { key: 'endpoint', label: 'Endpoint', children: detail.endpoint || '-' },
                  { key: 'ip', label: 'IP', children: detail.ip || '-' },
                  { key: 'hostname', label: t('page.gateway.columns.hostname'), children: detail.hostname || '-' },
                  { key: 'os', label: 'OS', children: detail.os || '-' },
                  { key: 'reverse', label: t('page.gateway.columns.reverse'), children: detail.reverse_connected ? t('page.gateway.reverseConnected') : '—' },
                  ...(detail.groups?.length
                    ? [{ key: 'groups', label: t('page.gateway.columns.groups'), children: detail.groups.join(', ') }]
                    : []),
                  ...(detail.note ? [{ key: 'note', label: t('page.gateway.columns.note'), children: detail.note, span: 2 }] : [])
                ]}
              />
              {detail.labels && Object.keys(detail.labels).length > 0 && (
                <Descriptions
                  bordered
                  column={1}
                  size="small"
                  title={t('page.gateway.labels')}
                  items={Object.entries(detail.labels).map(([k, v]) => ({
                    key: k,
                    label: <span className="font-mono text-12px">{k}</span>,
                    children: <span className="font-mono text-12px">{v}</span>
                  }))}
                />
              )}
              <div>
                <div className="mb-8px font-500">{t('page.gateway.assignedConfigs')}</div>
                {detail.assigned_configs?.length ? (
                  <Table
                    rowKey="name"
                    size="small"
                    pagination={false}
                    dataSource={detail.assigned_configs}
                    columns={[
                      { title: t('page.gateway.configName'), dataIndex: 'name' },
                      { title: t('page.gateway.configVersion'), dataIndex: 'version', width: 80 },
                      { title: t('page.gateway.configSize'), dataIndex: 'size', width: 100, render: (v: number) => `${v} B` },
                      { title: t('page.gateway.configTarget'), dataIndex: 'target', width: 140 }
                    ]}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.noConfigs')} />
                )}
              </div>
              {detail.reverse_connected && (
                <div>
                  <div className="mb-8px font-500">{t('page.gateway.liveInfo')}</div>
                  <Spin spinning={liveLoading}>
                    {liveInfo ? (
                      <pre className="m-0 max-h-280px overflow-auto rounded-4px bg-[var(--ant-color-fill-tertiary)] p-8px text-12px">
                        {JSON.stringify(liveInfo, null, 2)}
                      </pre>
                    ) : (
                      <span className="text-[var(--ant-color-text-tertiary)]">—</span>
                    )}
                  </Spin>
                </div>
              )}
            </Space>
          ) : (
            <Empty />
          )}
        </Spin>
      </Drawer>
    </Card>
  );
}
