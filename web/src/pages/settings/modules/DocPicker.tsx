import { useEffect, useMemo, useState } from 'react';
import { Button, Input, Modal, Table, Typography } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { request } from '@/service/request';

// module-level title cache shared by every picker instance on the page
const titleCache = new Map<string, string>();

const fetchDocs = async (params: { query?: string; from?: number; size?: number; ids?: string[] }) => {
  const { query = '', from = 0, size = 10, ids = [] } = params || {};
  let url = `/document/_search?from=${from}&size=${size}`;
  if (query.trim()) {
    // picker search: exact keyword match + relevance order — the endpoint
    // defaults to created:desc and inherits the global fuzziness, which made
    // Chinese results look unordered and full of look-alike noise
    url += `&query=${encodeURIComponent(query.trim())}&fuzziness=0&sort=_score:desc`;
  } else {
    url += `&sort=created:desc`;
  }
  if (ids.length) url += `&filter=${encodeURIComponent(`id:any(${ids.join(',')})`)}`;
  if (ids.length) url += `&filter=${encodeURIComponent(`id:any(${ids.join(',')})`)}`;
  const res: any = await request({ method: 'get', url });
  const hits = res?.data?.hits?.hits || [];
  const total = res?.data?.hits?.total?.value ?? hits.length;
  const rows = hits.map((h: any) => {
    const id = h._source?.id || h._id;
    const title = h._source?.title || id;
    titleCache.set(id, title);
    return { id, title, created: h._source?.created || '' };
  });
  return { rows, total };
};

// loads titles for ids missing from the cache (one-shot id filter per batch)
const useDocTitles = (ids: string[]) => {
  const [titles, setTitles] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    for (const id of ids) {
      if (titleCache.has(id)) initial[id] = titleCache.get(id)!;
    }
    return initial;
  });
  const missingKey = ids
    .filter((id) => titles[id] === undefined && !titleCache.has(id))
    .join(',');

  useEffect(() => {
    // also adopt ids that landed in the shared cache after mount
    const adopt = ids.filter((id) => titleCache.has(id) && titles[id] === undefined);
    if (adopt.length) {
      setTitles((prev) => {
        const next = { ...prev };
        for (const id of adopt) next[id] = titleCache.get(id)!;
        return next;
      });
    }
    if (!missingKey) return;
    let cancelled = false;
    const missing = missingKey.split(',');
    (async () => {
      const { rows } = await fetchDocs({ ids: missing, size: missing.length });
      if (cancelled) return;
      setTitles((prev) => {
        const next = { ...prev };
        for (const r of rows) next[r.id] = r.title;
        return next;
      });
    })();
    return () => {
      cancelled = true;
    };
  }, [missingKey]);

  return useMemo(() => {
    const merged = { ...titles };
    for (const id of ids) {
      if (merged[id] === undefined && titleCache.has(id)) merged[id] = titleCache.get(id)!;
    }
    return merged;
  }, [titles, ids.join(',')]);
};

interface PickerModalProps {
  open: boolean;
  selected: string[];
  onConfirm: (ids: string[]) => void;
  onClose: () => void;
}

// searchable multi-select document picker: paged table over
// GET /document/_search (permission-filtered), checkboxes preserved across
// pages, confirm merges the selection back as doc ids
export const DocPickerModal = ({ open, selected, onConfirm, onClose }: PickerModalProps) => {
  const { t } = useTranslation();
  const [text, setText] = useState('');
  const [from, setFrom] = useState(0);
  const [rows, setRows] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [selKeys, setSelKeys] = useState<string[]>([]);
  const pageSize = 10;

  useEffect(() => {
    if (!open) return;
    setSelKeys([...selected]);
    setText('');
    setFrom(0);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setLoading(true);
    fetchDocs({ query: text, from, size: pageSize })
      .then(({ rows, total }) => {
        if (cancelled) return;
        setRows(rows);
        setTotal(total);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, text, from]);

  return (
    <Modal
      open={open}
      title={t('page.settings.search_settings.labels.doc_picker_title')}
      width={680}
      onCancel={onClose}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t('common.cancel')}
        </Button>,
        <Button
          key="ok"
          type="primary"
          onClick={() => {
            onConfirm(selKeys);
            onClose();
          }}
        >
          {t('common.confirm')}
        </Button>
      ]}
    >
      <Input
        allowClear
        placeholder={t('page.settings.search_settings.labels.doc_picker_search')}
        style={{ marginBottom: 12 }}
        onChange={(e) => {
          setText(e.target.value);
          setFrom(0);
        }}
      />
      <Table
        size="small"
        loading={loading}
        dataSource={rows}
        rowKey="id"
        pagination={{
          current: from / pageSize + 1,
          pageSize,
          total,
          showSizeChanger: false,
          onChange: (page) => setFrom((page - 1) * pageSize)
        }}
        rowSelection={{
          selectedRowKeys: selKeys,
          onChange: (keys) => setSelKeys(keys as string[]),
          preserveSelectedRowKeys: true
        }}
        columns={[
          { title: t('page.settings.search_settings.labels.doc_col_title'), dataIndex: 'title', ellipsis: true },
          {
            title: t('page.settings.search_settings.labels.doc_col_id'),
            dataIndex: 'id',
            width: 170,
            render: (id: string) => (
              <Typography.Text
                copyable={{ text: id, tooltips: [id, id] }}
                ellipsis
                className="font-mono"
                style={{ fontSize: 11, maxWidth: 130 }}
              >
                {id}
              </Typography.Text>
            )
          },
          {
            title: t('page.settings.search_settings.labels.doc_col_created'),
            dataIndex: 'created',
            width: 100,
            render: (v: string) => (v ? String(v).slice(0, 10) : '—')
          }
        ]}
      />
    </Modal>
  );
};

interface TableSelectProps {
  value?: string[];
  onChange?: (ids: string[]) => void;
  className?: string;
}

// document-list management field (blocked docs + pinned-rule docs): a small
// table of the selected docs (title / id / remove icon) plus an "add" button
// that opens DocPickerModal
export const DocTableSelect = ({ value = [], onChange, className }: TableSelectProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const titles = useDocTitles(value);

  const rows = value.map((id) => ({ id, title: titles[id] || id }));

  return (
    <div className={className}>
      {rows.length > 0 && (
        <Table
          size="small"
          dataSource={rows}
          rowKey="id"
          pagination={false}
          columns={[
            { title: t('page.settings.search_settings.labels.doc_col_title'), dataIndex: 'title', ellipsis: true },
            {
              title: t('page.settings.search_settings.labels.doc_col_id'),
              dataIndex: 'id',
              width: 170,
              render: (id: string) => (
                <Typography.Text
                  copyable={{ text: id, tooltips: [id, id] }}
                  className="font-mono"
                  style={{ fontSize: 11 }}
                >
                  {id.slice(0, 10)}…
                </Typography.Text>
              )
            },
            {
              title: '',
              width: 48,
              render: (_: any, record: any) => (
                <Button
                  icon={<DeleteOutlined />}
                  onClick={() => onChange?.(value.filter((id) => id !== record.id))}
                  size="small"
                  type="text"
                />
              )
            }
          ]}
        />
      )}
      {/* add button sits below the existing rows, matching the pinned-rule layout */}
      <Button
        size="small"
        icon={<PlusOutlined />}
        style={rows.length ? { marginTop: 8 } : undefined}
        onClick={() => setOpen(true)}
      >
        {t('page.settings.search_settings.labels.doc_add_btn')}
      </Button>
      <DocPickerModal
        open={open}
        selected={value}
        onClose={() => setOpen(false)}
        onConfirm={(ids) => onChange?.(ids)}
      />
    </div>
  );
};
