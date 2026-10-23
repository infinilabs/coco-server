import { ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Button, Empty, Select, Space, Spin, Table, Tag, Tooltip, message } from 'antd';
import { useEffect, useState } from 'react';

import { proxyGatewayInstance } from '@/service/api/gateway';

/**
 * Observability panels fetched through the instance's reverse channel
 * (read-only whitelisted endpoints). Every panel degrades gracefully when
 * the instance is offline: the caller renders the not-connected notice and
 * skips fetching entirely.
 */

function NotConnected({ t }: { t: any }) {
  return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.obs.reverseOnly')} />;
}

function bytes(n?: number): string {
  if (n == null) return '-';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

// ─── queues ───────────────────────────────────────────────────────────────

export function QueuePanel({ id, connected }: { id: string; connected: boolean }) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchQueues = async () => {
    setLoading(true);
    try {
      const res: any = await proxyGatewayInstance(id, '/queue/stats');
      const data = res?.data?.data;
      const out: any[] = [];
      // shape: {queue: {<backend>: {<name>: stats}}}
      const byType = data?.queue ?? {};
      Object.entries(byType).forEach(([type, names]: [string, any]) => {
        Object.entries(names ?? {}).forEach(([name, stats]: [string, any]) => {
          out.push({
            key: `${type}/${name}`,
            type,
            name,
            depth: stats?.depth,
            size: stats?.size,
            consumers: Array.isArray(stats?.consumers) ? stats.consumers.length : stats?.consumer_count,
            error: stats?.error,
            raw: stats
          });
        });
      });
      setRows(out);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (connected) fetchQueues();
  }, [id, connected]);

  if (!connected) return <NotConnected t={t} />;
  return (
    <Spin spinning={loading}>
      <div className="mb-8px text-right">
        <Button size="small" icon={<ReloadOutlined />} onClick={fetchQueues} />
      </div>
      <Table
        rowKey="key"
        size="small"
        pagination={false}
        dataSource={rows}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.obs.noQueues')} /> }}
        columns={[
          { title: t('page.gateway.obs.queueBackend'), dataIndex: 'type', width: 110 },
          { title: t('page.gateway.obs.queueName'), dataIndex: 'name' },
          {
            title: t('page.gateway.obs.queueDepth'),
            dataIndex: 'depth',
            width: 100,
            render: (v: number, r: any) =>
              r.error ? <Tag color="error">{t('page.gateway.obs.statsError')}</Tag> : v ?? '-'
          },
          { title: t('page.gateway.obs.queueSize'), dataIndex: 'size', width: 110, render: (v: number) => bytes(v) },
          {
            title: t('page.gateway.obs.queueConsumers'),
            dataIndex: 'consumers',
            width: 90,
            render: (v: number) => v ?? '-'
          }
        ]}
        expandable={{
          rowExpandable: r => !r.error,
          expandedRowRender: r => (
            <pre className="m-0 max-h-240px overflow-auto text-12px">{JSON.stringify(r.raw, null, 2)}</pre>
          )
        }}
      />
    </Spin>
  );
}

// ─── logs ─────────────────────────────────────────────────────────────────

export function LogPanel({ id, connected }: { id: string; connected: boolean }) {
  const { t } = useTranslation();
  const [files, setFiles] = useState<{ name: string; path: string; size: number; updated: number }[]>([]);
  const [selected, setSelected] = useState<string>();
  const [lines, setLines] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [tailing, setTailing] = useState(false);

  const fetchFiles = async () => {
    setLoading(true);
    try {
      const res: any = await proxyGatewayInstance(id, '/logging/files');
      const list = res?.data?.data?.files ?? [];
      setFiles(list);
      const latest = list.find((f: any) => f.most_recently_modified) ?? list[0];
      if (latest) setSelected(latest.path);
    } finally {
      setLoading(false);
    }
  };

  const tail = async () => {
    if (!selected) return;
    setTailing(true);
    try {
      const res: any = await proxyGatewayInstance(id, `/logging/tail?file=${encodeURIComponent(selected)}&lines=200`);
      const body = res?.data;
      if (body?.error) {
        message.error(body.error?.reason ?? body.error);
      } else {
        setLines(body?.data?.lines ?? []);
      }
    } finally {
      setTailing(false);
    }
  };

  useEffect(() => {
    if (connected) fetchFiles();
  }, [id, connected]);

  if (!connected) return <NotConnected t={t} />;
  return (
    <Spin spinning={loading}>
      <Space className="mb-8px w-full" style={{ justifyContent: 'space-between' }}>
        <Select
          className="min-w-280px"
          showSearch
          optionFilterProp="label"
          value={selected}
          onChange={setSelected}
          options={files.map(f => ({
            value: f.path,
            label: `${f.name} (${bytes(f.size)})`
          }))}
          placeholder={t('page.gateway.obs.selectLog')}
        />
        <Button size="small" icon={<ReloadOutlined />} loading={tailing} onClick={tail}>
          {t('page.gateway.obs.tail')}
        </Button>
      </Space>
      {lines.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.obs.tailHint')} />
      ) : (
        <pre className="m-0 max-h-360px overflow-auto rounded-4px bg-[var(--ant-color-fill-tertiary)] p-8px text-12px leading-5">
          {lines.join('\n')}
        </pre>
      )}
    </Spin>
  );
}

// ─── running pipeline tasks ───────────────────────────────────────────────

export function TasksPanel({ id, connected }: { id: string; connected: boolean }) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchTasks = async () => {
    setLoading(true);
    try {
      const res: any = await proxyGatewayInstance(id, '/pipeline/tasks/');
      const map = res?.data?.data ?? {};
      setRows(
        Object.entries(map).map(([taskId, status]: [string, any]) => ({
          key: taskId,
          taskId,
          running: status?.running,
          status
        }))
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (connected) fetchTasks();
  }, [id, connected]);

  if (!connected) return <NotConnected t={t} />;
  return (
    <Spin spinning={loading}>
      <div className="mb-8px text-right">
        <Tooltip title={t('page.gateway.obs.tasksHint')}>
          <Button size="small" icon={<ReloadOutlined />} onClick={fetchTasks} />
        </Tooltip>
      </div>
      <Table
        rowKey="key"
        size="small"
        pagination={false}
        dataSource={rows}
        locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.obs.noTasks')} /> }}
        columns={[
          { title: t('page.gateway.obs.taskName'), dataIndex: 'taskId' },
          {
            title: t('page.gateway.obs.taskState'),
            dataIndex: 'running',
            width: 110,
            render: (v: boolean) => (v ? <Tag color="success">{t('page.gateway.obs.taskRunning')}</Tag> : <Tag>{t('page.gateway.obs.taskIdle')}</Tag>)
          }
        ]}
        expandable={{
          expandedRowRender: r => (
            <pre className="m-0 max-h-280px overflow-auto text-12px">{JSON.stringify(r.status, null, 2)}</pre>
          )
        }}
      />
    </Spin>
  );
}
