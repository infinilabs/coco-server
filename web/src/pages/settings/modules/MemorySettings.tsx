import { useMemo } from 'react';
import { Button, Empty, Modal, Popconfirm, Segmented, Space, Table, Tag, message } from 'antd';
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
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);

  const canManage = hasAuth('coco#memory/update');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter && statusFilter !== 'all') params.status = statusFilter;
      const res: any = await request({ method: 'get', url: '/memory/_mine', params });
      setItems((res?.data?.items as MemoryItem[]) ?? []);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.memory_settings.loadFailed'));
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
        title: t(`page.memory_settings.${action}Title`),
        content:
          action === 'confirm'
            ? t('page.memory_settings.confirmHint')
            : t('page.memory_settings.rejectHint'),
        onOk: async () => {
          try {
            await request({ method: 'put', url: `/memory/${id}/_${action}` });
            message.success(t('page.memory_settings.settled'));
            load();
          } catch (e: any) {
            message.error(e?.response?.data?.message ?? e?.message ?? t('page.memory_settings.loadFailed'));
          }
        }
      });
    },
    [load, t]
  );

  // batch ops run against the CURRENT page's selected rows; pending-only
  // for confirm/reject (settled rows in the selection are silently
  // skipped), all rows for delete
  const runBatch = useCallback(
    (action: 'confirm' | 'reject' | 'delete') => {
      const targets =
        action === 'delete'
          ? items.filter((r) => selectedKeys.includes(r.id))
          : items.filter((r) => selectedKeys.includes(r.id) && r.status === 'pending');
      if (targets.length === 0) {
        message.info(t('page.memory_settings.batchNoPending'));
        return;
      }
      const titleKey =
        action === 'delete' ? 'batchDeleteTitle' : action === 'confirm' ? 'batchConfirmTitle' : 'batchRejectTitle';
      Modal.confirm({
        title: t(`page.memory_settings.${titleKey}`, { n: targets.length }),
        content: t(
          `page.memory_settings.${
            action === 'delete' ? 'batchDeleteHint' : action === 'confirm' ? 'batchConfirmHint' : 'batchRejectHint'
          }`
        ),
        okButtonProps: action === 'delete' ? { danger: true } : undefined,
        onOk: async () => {
          const req =
            action === 'delete'
              ? (id: string) => request({ method: 'delete', url: `/memory/${id}` })
              : (id: string) => request({ method: 'put', url: `/memory/${id}/_${action}` });
          const results = await Promise.allSettled(targets.map((r) => req(r.id)));
          const ok = results.filter((x) => x.status === 'fulfilled').length;
          const fail = targets.length - ok;
          if (fail > 0) {
            message.warning(t('page.memory_settings.batchPartial', { ok, fail }));
          } else {
            message.success(t(action === 'delete' ? 'page.memory_settings.batchDeleted' : 'page.memory_settings.batchSettled', { ok }));
          }
          setSelectedKeys([]);
          load();
        }
      });
    },
    [items, selectedKeys, load, t]
  );

  const remove = useCallback(
    async (id: string) => {
      try {
        await request({ method: 'delete', url: `/memory/${id}` });
        message.success(t('page.memory_settings.deleted'));
        load();
      } catch (e: any) {
        message.error(e?.response?.data?.message ?? e?.message ?? t('page.memory_settings.loadFailed'));
      }
    },
    [load, t]
  );

  const columns = useMemo(
    () => [
      {
        title: t('page.memory_settings.kind'),
        dataIndex: 'kind',
        width: 110,
        render: (v: string) => <Tag color={KIND_COLORS[v] ?? 'default'}>{v}</Tag>
      },
      {
        title: t('page.memory_settings.content'),
        dataIndex: 'content',
        ellipsis: true
      },
      {
        title: t('page.memory_settings.status'),
        dataIndex: 'status',
        width: 100,
        render: (v: string) =>
          v === 'confirmed' ? (
            <Tag color="green">{t('page.memory_settings.confirmed')}</Tag>
          ) : v === 'rejected' ? (
            <Tag color="red">{t('page.memory_settings.rejected')}</Tag>
          ) : v === 'superseded' ? (
            <Tag color="default">{t('page.memory_settings.superseded')}</Tag>
          ) : (
            <Tag color="orange">{t('page.memory_settings.pending')}</Tag>
          )
      },
      {
        title: t('page.memory_settings.updated'),
        dataIndex: 'updated',
        width: 170,
        render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
      },
      {
        title: t('page.memory_settings.actions'),
        key: 'actions',
        width: 200,
        render: (_: unknown, r: MemoryItem) => (
          <Space size={4}>
            {r.status === 'pending' && canManage ? (
              <>
                <Button size="small" type="primary" onClick={() => settle(r.id, 'confirm')}>
                  {t('page.memory_settings.confirm')}
                </Button>
                <Button size="small" onClick={() => settle(r.id, 'reject')}>
                  {t('page.memory_settings.reject')}
                </Button>
              </>
            ) : null}
            {canManage ? (
              <Popconfirm title={t('page.memory_settings.deleteConfirm')} onConfirm={() => remove(r.id)}>
                <Button size="small" danger>
                  {t('page.memory_settings.delete')}
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
    <div>
      <div className="mb-8px flex items-center justify-between gap-8px">
        <Segmented
          value={statusFilter}
          onChange={(v) => {
            setStatusFilter(v as string);
            setSelectedKeys([]);
          }}
          options={[
            { label: t('page.memory_settings.all'), value: 'all' },
            { label: t('page.memory_settings.pending'), value: 'pending' },
            { label: t('page.memory_settings.confirmed'), value: 'confirmed' },
            { label: t('page.memory_settings.rejected'), value: 'rejected' },
            { label: t('page.memory_settings.superseded'), value: 'superseded' }
          ]}
        />
        {canManage && selectedKeys.length > 0 && (
          <Space size={4}>
            <span className="text-size-12px opacity-60">
              {t('page.memory_settings.selected', { n: selectedKeys.length })}
            </span>
            <Button size="small" type="primary" onClick={() => runBatch('confirm')}>
              {t('page.memory_settings.batchConfirm')}
            </Button>
            <Button size="small" onClick={() => runBatch('reject')}>
              {t('page.memory_settings.batchReject')}
            </Button>
            <Button size="small" danger onClick={() => runBatch('delete')}>
              {t('page.memory_settings.batchDelete')}
            </Button>
          </Space>
        )}
      </div>
      <Table<MemoryItem>
        rowKey="id"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={items}
        rowSelection={
          canManage
            ? {
                selectedRowKeys: selectedKeys,
                onChange: (keys) => setSelectedKeys(keys as string[])
              }
            : undefined
        }
        pagination={{ pageSize: 10, hideOnSinglePage: true }}
        locale={{ emptyText: <Empty description={t('page.memory_settings.empty')} /> }}
      />
    </div>
  );
}
