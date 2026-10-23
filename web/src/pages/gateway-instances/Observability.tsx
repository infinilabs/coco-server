import { ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Button, Empty, Input, Modal, Select, Space, Spin, Table, Tag, Tooltip, message } from 'antd';
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

// pretty-print a queue message body when it parses as JSON, raw otherwise
function prettyMsg(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

// ─── queue message browser (modal) ────────────────────────────────────────

function QueueMessagesModal({
  id,
  queue,
  open,
  onClose
}: {
  id: string;
  queue: { type: string; name: string; displayName?: string } | undefined;
  open: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [msgs, setMsgs] = useState<any[]>([]);
  const [empty, setEmpty] = useState(false);
  const [nextOffset, setNextOffset] = useState<string | undefined>(undefined);
  const [history, setHistory] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [size, setSize] = useState(5);

  // offset format on the wire: "segment,position[,version]" (e.g. "0,1445,0")
  const fetchPage = async (offset?: string) => {
    if (!queue) return;
    setLoading(true);
    try {
      const path = `/queue/${encodeURIComponent(queue.name)}/_scroll?size=${size}${offset ? `&offset=${encodeURIComponent(offset)}` : ''}`;
      const res: any = await proxyGatewayInstance(id, path);
      const body = res?.data?.data ?? {};
      if (body.error) {
        message.error(String(body.error));
        return;
      }
      setMsgs(body.messages ?? []);
      setEmpty(body.empty === true);
      const n = body?.context?.next_offset;
      setNextOffset(n ? `${n.segment},${n.position},${n.version ?? 0}` : undefined);
      // push the page's starting offset so 上一页 can replay it
      setHistory(h => (offset ? [...h, offset] : h));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open && queue) {
      setMsgs([]);
      setEmpty(false);
      setNextOffset(undefined);
      setHistory([]);
      fetchPage();
    }
  }, [open, queue?.name, size]);

  if (!queue) return null;
  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      title={`${t('page.gateway.obs.msgTitle')} · ${queue.displayName ?? queue.name}`}
    >
      <Spin spinning={loading}>
        <Space className="mb-8px w-full" style={{ justifyContent: 'space-between' }}>
          <Select
            size="small"
            value={size}
            onChange={setSize}
            options={[5, 10, 20].map(n => ({ value: n, label: `${n} / ${t('page.gateway.obs.page')}` }))}
            style={{ width: 120 }}
          />
          <Space>
            <Button
              size="small"
              disabled={history.length === 0}
              onClick={() => {
                // history holds each visited page's STARTING offset; the
                // current page's start is the last entry — the previous
                // page starts one earlier (undefined = first page)
                const prev = history[history.length - 2];
                setHistory(h => h.slice(0, -1));
                fetchPage(prev);
              }}
            >
              {t('page.gateway.obs.prevPage')}
            </Button>
            <Button size="small" disabled={!nextOffset || msgs.length < size} onClick={() => fetchPage(nextOffset)}>
              {t('page.gateway.obs.nextPage')}
            </Button>
            <Tooltip title={t('page.gateway.obs.refreshPage')}>
              <Button size="small" icon={<ReloadOutlined />} onClick={() => fetchPage(history[history.length - 1])} />
            </Tooltip>
          </Space>
        </Space>
        {empty ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.gateway.obs.msgEmpty')} />
        ) : (
          <div className="max-h-480px overflow-auto">
            {msgs.map((m: any, i: number) => (
              <div key={String(m.offset ?? i)} className="mb-8px rounded-4px bg-[var(--ant-color-fill-tertiary)] p-8px">
                <Space size={4} className="mb-4px">
                  <Tag className="m-0 font-mono">{String(m.offset)}</Tag>
                  <span className="text-12px text-[var(--ant-color-text-tertiary)]">{bytes(m.size)}</span>
                  {m.truncated ? (
                    <Tag className="m-0" color="warning">
                      {t('page.gateway.obs.truncated')}
                    </Tag>
                  ) : null}
                </Space>
                <pre className="m-0 max-h-200px overflow-auto text-12px leading-5">{prettyMsg(String(m.message ?? ''))}</pre>
              </div>
            ))}
          </div>
        )}
      </Spin>
    </Modal>
  );
}

// ─── queues ───────────────────────────────────────────────────────────────

export function QueuePanel({ id, connected }: { id: string; connected: boolean }) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [browsing, setBrowsing] = useState<{ type: string; name: string; displayName?: string } | undefined>(undefined);

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
            // stats keys are internal queue IDs — show the human name when present
            displayName: stats?.metadata?.name ?? name,
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
          { title: t('page.gateway.obs.queueName'), dataIndex: 'displayName' },
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
          },
          {
            title: t('page.gateway.obs.actions'),
            key: 'actions',
            width: 100,
            render: (_: any, r: any) =>
              r.error ? null : (
                <a onClick={() => setBrowsing({ type: r.type, name: r.name, displayName: r.displayName })}>
                  {t('page.gateway.obs.browseMessages')}
                </a>
              )
          }
        ]}
        expandable={{
          rowExpandable: r => !r.error,
          expandedRowRender: r => (
            <pre className="m-0 max-h-240px overflow-auto text-12px">{JSON.stringify(r.raw, null, 2)}</pre>
          )
        }}
      />
      <QueueMessagesModal id={id} queue={browsing} open={!!browsing} onClose={() => setBrowsing(undefined)} />
    </Spin>
  );
}

// ─── logs ─────────────────────────────────────────────────────────────────

export function LogPanel({ id, connected }: { id: string; connected: boolean }) {
  const { t } = useTranslation();
  const [files, setFiles] = useState<{ name: string; path: string; size: number; updated: number }[]>([]);
  const [selected, setSelected] = useState<string>();
  const [keyword, setKeyword] = useState('');
  const [lines, setLines] = useState<string[]>([]);
  const [truncated, setTruncated] = useState(false);
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
      let path = `/logging/tail?file=${encodeURIComponent(selected)}&lines=200`;
      if (keyword.trim()) path += `&keyword=${encodeURIComponent(keyword.trim())}`;
      const res: any = await proxyGatewayInstance(id, path);
      const body = res?.data;
      if (body?.error) {
        message.error(body.error?.reason ?? body.error);
      } else {
        setLines(body?.data?.lines ?? []);
        setTruncated(body?.data?.truncated === true);
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
      <Space.Compact className="mb-8px w-full">
        <Select
          className="min-w-260px"
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
        <Input
          allowClear
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          onPressEnter={tail}
          placeholder={t('page.gateway.obs.keyword')}
          prefix={<SearchOutlined />}
        />
        <Button icon={<ReloadOutlined />} loading={tailing} onClick={tail}>
          {t('page.gateway.obs.tail')}
        </Button>
      </Space.Compact>
      {truncated ? (
        <div className="mb-4px text-12px text-[var(--ant-color-text-tertiary)]">{t('page.gateway.obs.tailTruncated')}</div>
      ) : null}
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
