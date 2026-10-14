import { useMemo } from 'react';
import { Button, Card, Empty, Modal, Popconfirm, Segmented, Space, Table, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { request } from '@/service/request';

/**
 * Long-term memory management (W6): every memory starts PENDING and stays
 * out of all prompts until its owner confirms it here — the human gate
 * for the most personal layer of the system. Rejects keep the record for
 * audit (suppressed), deletes remove it outright.
 */

interface MemoryItem {
  id: string;
  kind: string;
  status: string;
  content: string;
  updated?: string;
  context?: Record<string, unknown>;
}

const KIND_COLORS: Record<string, string> = {
  profile: 'geekblue',
  preference: 'purple',
  fact: 'cyan',
  task: 'gold',
  interest: 'green'
};

export default function MemorySettings() {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const [items, setItems] = useState<MemoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [statusFilter, setStatusFilter] = useState<string>('pending');

  const canManage = hasAuth('coco#memory/update');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter && statusFilter !== 'all') params.status = statusFilter;
      const res: any = await request({ method: 'get', url: '/memory/_mine', params });
      setItems((res?.data?.items as MemoryItem[]) ?? []);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.settings.memory_settings.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [statusFilter, t]);

  useEffect(() => {
    if (hasAuth('coco#memory/read')) load();
  }, [load]);

  const settle = useCallback(
    (id: string, action: 'confirm' | 'reject') => {
      Modal.confirm({
        title: t(`page.settings.memory_settings.${action}Title`),
        content:
          action === 'confirm'
            ? t('page.settings.memory_settings.confirmHint')
            : t('page.settings.memory_settings.rejectHint'),
        onOk: async () => {
          try {
            await request({ method: 'put', url: `/memory/${id}/_${action}` });
            message.success(t('page.settings.memory_settings.settled'));
            load();
          } catch (e: any) {
            message.error(e?.response?.data?.message ?? e?.message ?? t('page.settings.memory_settings.loadFailed'));
          }
        }
      });
    },
    [load, t]
  );

  const remove = useCallback(
    async (id: string) => {
      try {
        await request({ method: 'delete', url: `/memory/${id}` });
        message.success(t('page.settings.memory_settings.deleted'));
        load();
      } catch (e: any) {
        message.error(e?.response?.data?.message ?? e?.message ?? t('page.settings.memory_settings.loadFailed'));
      }
    },
    [load, t]
  );

  const columns = useMemo(
    () => [
      {
        title: t('page.settings.memory_settings.kind'),
        dataIndex: 'kind',
        width: 110,
        render: (v: string) => <Tag color={KIND_COLORS[v] ?? 'default'}>{v}</Tag>
      },
      {
        title: t('page.settings.memory_settings.content'),
        dataIndex: 'content',
        ellipsis: true
      },
      {
        title: t('page.settings.memory_settings.status'),
        dataIndex: 'status',
        width: 100,
        render: (v: string) =>
          v === 'confirmed' ? (
            <Tag color="green">{t('page.settings.memory_settings.confirmed')}</Tag>
          ) : v === 'rejected' ? (
            <Tag color="red">{t('page.settings.memory_settings.rejected')}</Tag>
          ) : (
            <Tag color="orange">{t('page.settings.memory_settings.pending')}</Tag>
          )
      },
      {
        title: t('page.settings.memory_settings.updated'),
        dataIndex: 'updated',
        width: 170,
        render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
      },
      {
        title: t('page.settings.memory_settings.actions'),
        key: 'actions',
        width: 200,
        render: (_: unknown, r: MemoryItem) => (
          <Space size={4}>
            {r.status === 'pending' && canManage ? (
              <>
                <Button size="small" type="primary" onClick={() => settle(r.id, 'confirm')}>
                  {t('page.settings.memory_settings.confirm')}
                </Button>
                <Button size="small" onClick={() => settle(r.id, 'reject')}>
                  {t('page.settings.memory_settings.reject')}
                </Button>
              </>
            ) : null}
            {canManage ? (
              <Popconfirm title={t('page.settings.memory_settings.deleteConfirm')} onConfirm={() => remove(r.id)}>
                <Button size="small" danger>
                  {t('page.settings.memory_settings.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        )
      }
    ],
    [t, canManage, settle, remove]
  );

  return (
    <Card size="small" title={t('page.settings.memory_settings.title')}>
      <Typography.Text type="secondary" className="text-12px">
        {t('page.settings.memory_settings.hint')}
      </Typography.Text>
      <div className="mt-8px mb-8px">
        <Segmented
          value={statusFilter}
          onChange={(v) => setStatusFilter(v as string)}
          options={[
            { label: t('page.settings.memory_settings.pending'), value: 'pending' },
            { label: t('page.settings.memory_settings.confirmed'), value: 'confirmed' },
            { label: t('page.settings.memory_settings.rejected'), value: 'rejected' },
            { label: t('page.settings.memory_settings.all'), value: 'all' }
          ]}
        />
      </div>
      <Table<MemoryItem>
        rowKey="id"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={items}
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        locale={{ emptyText: <Empty description={t('page.settings.memory_settings.empty')} /> }}
      />
    </Card>
  );
}
