import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { BookOpen, Pencil } from 'lucide-react';
import { Button, Card, Drawer, Empty, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  batchUpdateWikiEntityStatus,
  createWikiEntity,
  mergeWikiEntities,
  getOntologySchema,
  getWikiEntity,
  searchWikiEntityPage,
  updateWikiEntity,
  type OntologyEntityTypeDef,
  type OntologyPropertyDef,
  type OntologyRelationDef
} from '@/service/api';
import { wikiTypeColor } from '../shared/WikiLinkTag';
import { EntityArticleSections } from './EntityArticleSections';

const STATUS_FLOW: Record<string, string | undefined> = {
  proposed: 'reviewed',
  reviewed: 'published'
};

const STATUS_TAG_COLOR: Record<string, string> = {
  published: 'green',
  reviewed: 'blue',
  proposed: 'orange',
  retired: 'default'
};

// raw enum → locale key suffix; statuses the schema doesn't know fall back to raw
const STATUS_LABEL_KEY: Record<string, string> = {
  proposed: 'statusProposed',
  reviewed: 'statusReviewed',
  published: 'statusPublished',
  retired: 'statusRetired'
};

/* ---------------- property form controls (schema-driven) ---------------- */

function PropertyControl({ def, value, onChange }: { def: OntologyPropertyDef; value: unknown; onChange: (v: unknown) => void }) {
  switch (def.type) {
    case 'number':
      return <InputNumber className="w-full" value={typeof value === 'number' ? value : undefined} onChange={v => onChange(v ?? undefined)} />;
    case 'boolean':
      return <Switch size="small" checked={value === true} onChange={v => onChange(v)} />;
    case 'enum':
      return (
        <Select
          className="w-full"
          allowClear
          value={typeof value === 'string' ? value : undefined}
          options={(def.enum || []).map(e => ({ value: e, label: e }))}
          onChange={v => onChange(v ?? undefined)}
        />
      );
    case 'date':
      return <Input className="w-full" value={typeof value === 'string' ? value : ''} placeholder="YYYY-MM-DD" onChange={e => onChange(e.target.value || undefined)} />;
    default:
      return <Input className="w-full" value={value == null ? '' : String(value)} onChange={e => onChange(e.target.value || undefined)} />;
  }
}

/* ---------------- relations editor ---------------- */

interface RelationRow {
  relation: string;
  target_id: string;
  target_name?: string;
}

function RelationsEditor({
  allowed,
  rows,
  setRows,
  onPickTarget
}: {
  allowed: OntologyRelationDef[];
  rows: RelationRow[];
  setRows: (r: RelationRow[]) => void;
  onPickTarget: (relation: OntologyRelationDef) => void;
}) {
  const { t } = useTranslation();
  const relationLabel = (name: string) => allowed.find(a => a.name === name)?.label || name;

  return (
    <div className="flex flex-col gap-8px">
      {rows.map((row, i) => (
        <div className="flex items-center gap-8px" key={`${row.relation}-${row.target_id}-${i}`}>
          <Tag className="m-0">{relationLabel(row.relation)}</Tag>
          <span className="flex-1 truncate">{row.target_name || row.target_id}</span>
          <Button size="small" type="text" danger onClick={() => setRows(rows.filter((_, x) => x !== i))}>
            {t('common.delete')}
          </Button>
        </div>
      ))}
      <Select
        className="w-full"
        size="small"
        value=""
        placeholder={t('page.wiki.entityManager.addRelation')}
        options={allowed.map(a => ({ value: a.name, label: a.label ? `${a.label} (${a.name})` : a.name }))}
        onChange={name => {
          const def = allowed.find(a => a.name === name);
          if (def) onPickTarget(def);
        }}
      />
    </div>
  );
}

/* ---------------- the manager ---------------- */

export function WikiEntityManager({ kbId }: { kbId: string }) {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    create: hasAuth('coco#wiki_entity/create'),
    update: hasAuth('coco#wiki_entity/update')
  };

  const statusLabel = (s: string) => (STATUS_LABEL_KEY[s] ? t(`page.wiki.entityManager.${STATUS_LABEL_KEY[s]}`) : s);
  const statusOptions = ['proposed', 'reviewed', 'published'].map(s => ({ value: s, label: statusLabel(s) }));

  const [selectedIds, setSelectedIds] = useState<React.Key[]>([]);
  const [batchStatus, setBatchStatus] = useState<string>('reviewed');
  const [batchLoading, setBatchLoading] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [typeFilter, setTypeFilter] = useState<string | undefined>();
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [data, setData] = useState<Api.Wiki.EntityInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(false);
  const nav = useNavigate();

  const [schemaTypes, setSchemaTypes] = useState<OntologyEntityTypeDef[]>([]);

  // edit drawer state
  const [editing, setEditing] = useState<Api.Wiki.EntityInfo | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  // detail drawer state (click a row/name to inspect an entity)
  const [detail, setDetail] = useState<Api.Wiki.EntityInfo | null>(null);
  const [form] = Form.useForm();
  const [propValues, setPropValues] = useState<Record<string, unknown>>({});
  const [relRows, setRelRows] = useState<RelationRow[]>([]);
  const [saving, setSaving] = useState(false);
  const [targetPicker, setTargetPicker] = useState<{ relation: OntologyRelationDef } | null>(null);
  const [targetOptions, setTargetOptions] = useState<Api.Wiki.EntityInfo[]>([]);
  const [targetSearch, setTargetSearch] = useState('');

  const pageSize = 20;

  const fetchData = useCallback(
    (kw = keyword, tf = typeFilter, sf = statusFilter, pg = page) => {
      setLoading(true);
      searchWikiEntityPage({ query: kw || undefined, type: tf, status: sf, kbId, from: pg * pageSize, size: pageSize })
        .then(res => {
          setData(res.data);
          setTotal(res.total);
        })
        .finally(() => setLoading(false));
    },
    [keyword, typeFilter, statusFilter, page]
  );

  useEffect(() => {
    fetchData();
  }, [typeFilter, statusFilter, page]);

  useEffect(() => {
    getOntologySchema(kbId || undefined).then(res => setSchemaTypes(res.entity_types || []));
  }, [kbId]);

  const typeDef = useMemo(() => {
    const t = form.getFieldValue('type') as string | undefined;
    return schemaTypes.find(x => x.name === t);
  }, [form, schemaTypes, propValues, drawerOpen]);

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ status: 'proposed' });
    setPropValues({});
    setRelRows([]);
    setDrawerOpen(true);
  };

  const openEdit = (entity: Api.Wiki.EntityInfo) => {
    setEditing(entity);
    // load the full record — list rows may carry partial fields
    getWikiEntity(entity.id).then(full => {
      const e = full || entity;
      form.setFieldsValue({ name: e.name, type: e.type, subtype: e.subtype, status: e.status, aliases: e.aliases });
      setPropValues({ ...(e.properties || {}) });
      setRelRows(
        (e.relations || []).map(r => ({
          relation: r.relation,
          target_id: r.target_id,
          target_name: (r as any).target_name
        }))
      );
      setDrawerOpen(true);
    });
  };

  // target picker: search entities of the relation's target_type by keyword
  useEffect(() => {
    if (!targetPicker) return;
    const kw = targetSearch.trim();
    searchWikiEntityPage({ query: kw || undefined, type: targetPicker.relation.target_type === '*' ? undefined : targetPicker.relation.target_type, kbId, size: 20 }).then(
      res => setTargetOptions(res.data)
    );
  }, [targetPicker, targetSearch]);

  const save = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = {
        kb_id: kbId,
        name: values.name,
        type: values.type || '',
        subtype: values.subtype || '',
        status: values.status || 'proposed',
        aliases: values.aliases || [],
        properties: propValues,
        relations: relRows.map(r => ({ relation: r.relation, target_id: r.target_id }))
      };
      if (editing) {
        // partial-update mode validates against the delta, so send the full
        // merged record (the drawer holds it all) — required properties and
        // the relation vocabulary then see the complete object
        await updateWikiEntity(editing.id, payload);
      } else {
        await createWikiEntity(payload);
      }
      message.success(t('common.updateSuccess'));
      setDrawerOpen(false);
      fetchData();
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('common.updateFailed'));
    } finally {
      setSaving(false);
    }
  };

  const advanceStatus = (entity: Api.Wiki.EntityInfo) => {
    const next = STATUS_FLOW[entity.status || ''];
    if (!next) return;
    updateWikiEntity(entity.id, { status: next, kb_id: kbId }).then(() => {
      message.success(t('common.updateSuccess'));
      fetchData();
    });
  };

  const runMerge = () => {
    if (selectedIds.length < 2) return;
    setBatchLoading(true);
    mergeWikiEntities(selectedIds.map(String))
      .then((res: any) => {
        message.success(t('page.wiki.entityManager.merged', { name: res?.name || '' }));
        setSelectedIds([]);
        fetchData();
      })
      .catch(() => message.error(t('page.wiki.entityManager.batchFailed')))
      .finally(() => setBatchLoading(false));
  };

  const runBatchStatus = () => {
    if (!selectedIds.length) return;
    setBatchLoading(true);
    batchUpdateWikiEntityStatus(selectedIds.map(String), batchStatus)
      .then((res: any) => {
        const updated = res?.updated ?? 0;
        const skipped = res?.skipped ?? 0;
        if (skipped > 0) {
          message.warning(t('page.wiki.entityManager.batchPartial', { updated, skipped }));
        } else {
          message.success(t('page.wiki.entityManager.batchDone', { count: updated }));
        }
        setSelectedIds([]);
        fetchData();
      })
      .catch(() => message.error(t('page.wiki.entityManager.batchFailed')))
      .finally(() => setBatchLoading(false));
  };

  const columns: any[] = [
    {
      title: t('page.wiki.entityManager.columns.name'),
      dataIndex: 'name',
      render: (_: any, row: Api.Wiki.EntityInfo) => (
        <Space size={4}>
          <a onClick={() => setDetail(row)}>{row.name}</a>
          {(row.aliases || []).length > 0 && (
            <span className="text-12px text-[var(--ant-color-text-tertiary)]" title={(row.aliases || []).join(', ')}>
              +{row.aliases!.length}
            </span>
          )}
        </Space>
      )
    },
    {
      title: t('page.wiki.entityManager.columns.type'),
      dataIndex: 'type',
      width: 140,
      render: (type: string) =>
        type ? (
          <span className="font-mono text-12px" style={{ color: wikiTypeColor(type) }}>
            {schemaTypes.find(x => x.name === type)?.label || type}
          </span>
        ) : (
          '—'
        )
    },
    {
      title: t('page.wiki.entityManager.columns.status'),
      dataIndex: 'status',
      width: 110,
      render: (status: string) => (status ? <Tag color={STATUS_TAG_COLOR[status]}>{statusLabel(status)}</Tag> : '—')
    },
    {
      title: t('page.wiki.entityManager.columns.relations'),
      dataIndex: 'relations',
      width: 90,
      render: (rels: { target_id: string }[] | undefined) => rels?.length ?? 0
    },
    {
      title: t('common.operation'),
      key: 'actions',
      width: 170,
      render: (_: any, row: Api.Wiki.EntityInfo) => (
        <Space>
          {permissions.update && (
            <Button type="text" size="small" icon={<Pencil />} onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          )}
          {permissions.update && STATUS_FLOW[row.status || ''] && (
            <Popconfirm title={t('page.wiki.entityManager.advanceConfirm', { status: statusLabel(STATUS_FLOW[row.status || '']!) })} onConfirm={() => advanceStatus(row)}>
              <Button type="link" size="small" className="!px-0">
                {t('page.wiki.entityManager.advance', { status: statusLabel(STATUS_FLOW[row.status || '']!) })}
              </Button>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  return (
    <Card className="card-wrapper" title={
      <span className="flex items-center gap-8px">
        <BookOpen />
        {t('page.wiki.entityManager.title')}
        <span className="text-13px font-normal text-[var(--ant-color-text-tertiary)]">{t('common.totalItems', { total })}</span>
      </span>
    } extra={
      permissions.create && (
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          {t('common.add')}
        </Button>
      )
    }>
      {selectedIds.length > 0 && permissions.update && (
        <div
          className="mb-12px flex flex-wrap items-center gap-10px rounded-6px border border-solid px-10px py-6px"
          style={{ borderColor: 'var(--ant-color-primary-border)', background: 'var(--ant-color-primary-bg)' }}
        >
          <span className="text-12.5px">{t('page.wiki.entityManager.batchSelected', { count: selectedIds.length })}</span>
          <span className="text-12.5px">{t('page.wiki.entityManager.batchSetStatus')}</span>
          <Select
            className="w-36"
            size="small"
            value={batchStatus}
            options={[
              { value: 'reviewed', label: t('page.wiki.entityManager.statusReviewed') },
              { value: 'published', label: t('page.wiki.entityManager.statusPublished') },
              { value: 'retired', label: t('page.wiki.entityManager.statusRetired') },
              { value: 'proposed', label: t('page.wiki.entityManager.statusProposed') }
            ]}
            onChange={setBatchStatus}
          />
          <Popconfirm
            title={t('page.wiki.entityManager.batchConfirm', { count: selectedIds.length, status: batchStatus })}
            onConfirm={runBatchStatus}
          >
            <Button loading={batchLoading} size="small" type="primary">
              {t('page.wiki.entityManager.batchApply')}
            </Button>
          </Popconfirm>
          {selectedIds.length >= 2 && (
            <Popconfirm
              title={t('page.wiki.entityManager.mergeConfirm', { count: selectedIds.length })}
              onConfirm={runMerge}
            >
              <Button loading={batchLoading} size="small" type="primary" ghost>
                {t('page.wiki.entityManager.merge')}
              </Button>
            </Popconfirm>
          )}
          <a className="text-12px" onClick={() => setSelectedIds([])}>
            {t('page.wiki.entityManager.batchClear')}
          </a>
        </div>
      )}
      <Space className="mb-12px" wrap>
        <Input.Search
          className="w-56"
          allowClear
          prefix={<SearchOutlined />}
          placeholder={t('page.wiki.entityManager.searchPlaceholder')}
          onSearch={kw => {
            setKeyword(kw);
            setPage(0);
            fetchData(kw, typeFilter, statusFilter, 0);
          }}
        />
        <Select
          className="w-44"
          allowClear
          placeholder={t('page.wiki.entityManager.typeFilter')}
          value={typeFilter}
          options={schemaTypes.map(x => ({ value: x.name, label: x.label || x.name }))}
          onChange={v => {
            setTypeFilter(v);
            setPage(0);
          }}
        />
        <Select
          className="w-36"
          allowClear
          placeholder={t('page.wiki.entityManager.statusFilter')}
          value={statusFilter}
          options={statusOptions}
          onChange={v => {
            setStatusFilter(v);
            setPage(0);
          }}
        />
      </Space>
      <Table
        rowKey="id"
        rowClassName="cursor-pointer"
        onRow={record => ({
          onClick: e => {
            // clicking a row opens the details; interactive cells (selection checkbox, action buttons) opt out
            if ((e.target as HTMLElement).closest('button, a, .ant-table-selection-column')) return;
            setDetail(record);
          }
        })}
        rowSelection={permissions.update ? {
          selectedRowKeys: selectedIds,
          onChange: (keys: React.Key[]) => setSelectedIds(keys)
        } : undefined}
        loading={loading}
        columns={columns}
        dataSource={data}
        pagination={{
          current: page + 1,
          pageSize,
          total,
          onChange: p => setPage(p - 1),
          showSizeChanger: false
        }}
      />

      <Drawer
        title={editing ? t('page.wiki.entityManager.editTitle', { name: editing.name }) : t('page.wiki.entityManager.newTitle')}
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={saving} onClick={save}>
              {t('common.save')}
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" className="flex flex-col gap-4px">
          <Form.Item name="name" label={t('page.wiki.entityManager.columns.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label={t('page.wiki.entityManager.columns.type')}>
            <Select
              allowClear
              options={schemaTypes.map(x => ({ value: x.name, label: x.label ? `${x.label} (${x.name})` : x.name }))}
              onChange={() => setPropValues({})}
            />
          </Form.Item>
          <Form.Item name="subtype" label={t('page.wiki.entityManager.subtype')}>
            <Input />
          </Form.Item>
          <Form.Item name="aliases" label={t('page.wiki.entityManager.aliases')}>
            <Select mode="tags" open={false} tokenSeparators={[',']} />
          </Form.Item>
          <Form.Item name="status" label={t('page.wiki.entityManager.columns.status')}>
            <Select options={statusOptions} />
          </Form.Item>

          {(typeDef?.properties || []).length > 0 && (
            <div className="mb-4">
              <div className="mb-8px text-13px font-medium">{t('page.wiki.entityManager.properties')}</div>
              <div className="flex flex-col gap-8px">
                {typeDef!.properties!.map(def => (
                  <div className="flex items-center gap-8px" key={def.key}>
                    <span className="w-32 shrink-0 text-13px text-[var(--ant-color-text-tertiary)]" title={def.key}>
                      {def.label || def.key}
                      {def.required && <span className="text-[var(--ant-color-error)]"> *</span>}
                    </span>
                    <PropertyControl def={def} value={propValues[def.key]} onChange={v => setPropValues(p => ({ ...p, [def.key]: v }))} />
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="mb-4">
            <div className="mb-8px text-13px font-medium">{t('page.wiki.entityManager.relations')}</div>
            <RelationsEditor
              allowed={typeDef?.relations || []}
              rows={relRows}
              setRows={setRelRows}
              onPickTarget={def => setTargetPicker({ relation: def })}
            />
          </div>
        </Form>
      </Drawer>

      <Drawer
        title={detail ? t('page.wiki.entityManager.detailTitle', { name: detail.name }) : ''}
        width={560}
        open={!!detail}
        onClose={() => setDetail(null)}
        extra={
          detail && (
            <Space>
              {detail.article_id && (
                <Button size="small" onClick={() => nav(`/wiki/article/${detail.article_id}?kb=${kbId}`)}>
                  {t('page.wiki.entityManager.viewArticle')}
                </Button>
              )}
              <Button size="small" onClick={() => nav(`/wiki/kb/${kbId}?tab=graph&entity=${detail.id}`)}>
                {t('page.wiki.entityManager.viewInGraph')}
              </Button>
              {permissions.update && (
                <Button
                  size="small"
                  type="primary"
                  icon={<Pencil />}
                  onClick={() => {
                    const target = detail;
                    setDetail(null);
                    openEdit(target);
                  }}
                >
                  {t('common.edit')}
                </Button>
              )}
            </Space>
          )
        }
      >
        {detail && <EntityArticleSections entityId={detail.id} kbId={kbId} />}
      </Drawer>

      <Modal title={t('page.wiki.entityManager.pickTarget')} open={!!targetPicker} onCancel={() => setTargetPicker(null)} onOk={() => setTargetPicker(null)} footer={null}>
        {targetPicker && (
          <>
            <Input.Search
              className="mb-8px"
              allowClear
              placeholder={t('page.wiki.entityManager.searchPlaceholder')}
              onSearch={setTargetSearch}
              onChange={e => setTargetSearch(e.target.value)}
            />
            {targetOptions.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.wiki.entityManager.noTargets')} />
            ) : (
              <div className="flex flex-col gap-4px max-h-360px overflow-auto">
                {targetOptions.map(e => (
                  <div
                    key={e.id}
                    className="flex items-center gap-8px rounded-6px border border-solid border-[var(--ant-color-border-secondary)] px-8px py-4px cursor-pointer hover:bg-[var(--ant-color-fill-tertiary)]"
                    onClick={() => {
                      setRelRows(rows => [...rows, { relation: targetPicker.relation.name, target_id: e.id, target_name: e.name }]);
                      setTargetPicker(null);
                      setTargetSearch('');
                    }}
                  >
                    <span className="font-mono text-11px" style={{ color: wikiTypeColor(e.type) }}>
                      {e.type}
                    </span>
                    <span>{e.name}</span>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </Modal>
    </Card>
  );
}
