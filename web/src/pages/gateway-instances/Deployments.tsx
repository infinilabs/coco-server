import { PlusOutlined, SyncOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Button, Form, Input, Modal, Popconfirm, Radio, Select, Space, Switch, Table, Tag, Tooltip, message } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import {
  createPipelineDeployment,
  deletePipelineDeployment,
  fetchGatewayInstances,
  fetchPipelineDeploymentStatus,
  fetchPipelineDeployments,
  updatePipelineDeployment,
  type DeploymentStatus,
  type GatewayInstance,
  type PipelineDeployment
} from '@/service/api/gateway';
import { fetchPipelineList } from '@/service/api/pipeline';
import { formatESSearchResult } from '@/service/request/es';

const stateColor: Record<string, string> = {
  running: 'success',
  pending: 'warning',
  stopped: 'default',
  failed: 'error',
  unknown: 'default'
};

function stateLabel(v: string): string {
  return (v || 'unknown').toLowerCase();
}

export function Deployments({ onRefreshTargets }: { onRefreshTargets?: () => void }) {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    // the UI gates on the pipeline permission family the user profile
    // actually carries (generic#pipeline/*); the Fleet API routes check the
    // system:pipeline keys server-side, which admins satisfy regardless
    create: hasAuth('generic#pipeline/create'),
    update: hasAuth('generic#pipeline/update'),
    remove: hasAuth('generic#pipeline/delete')
  };

  const [data, setData] = useState<PipelineDeployment[]>([]);
  const [instances, setInstances] = useState<GatewayInstance[]>([]);
  const [pipelines, setPipelines] = useState<{ id: string; name: string }[]>([]);
  const [statuses, setStatuses] = useState<Record<string, DeploymentStatus>>({});
  const [loading, setLoading] = useState(false);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<PipelineDeployment>();
  const [saving, setSaving] = useState(false);
  const [targetMode, setTargetMode] = useState<'instances' | 'groups'>('instances');
  const [form] = Form.useForm();

  const [detailStatus, setDetailStatus] = useState<DeploymentStatus>();
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const res: any = await fetchPipelineDeployments();
      const items = res?.data?.items ?? [];
      setData(items);
      // per-deployment reported state (desired vs acknowledged)
      const next: Record<string, DeploymentStatus> = {};
      for (const dep of items) {
        const s: any = await fetchPipelineDeploymentStatus(dep.id);
        if (s?.data) next[dep.id] = s.data;
      }
      setStatuses(next);
    } finally {
      setLoading(false);
    }
  };

  const fetchOptions = async () => {
    const [instRes, pipeRes]: any[] = await Promise.all([
      fetchGatewayInstances(),
      fetchPipelineList({ from: 0, size: 100 })
    ]);
    setInstances(
      (instRes?.data?.items ?? []).filter((i: GatewayInstance) => i.status === 'approved')
    );
    const pipes = formatESSearchResult(pipeRes?.data)?.data ?? [];
    setPipelines(pipes.map((p: any) => ({ id: p.id, name: p.name })));
  };

  useEffect(() => {
    fetchData();
    fetchOptions();
  }, []);

  // deployment targets converge on the reconciler's 15s tick — poll gently
  useEffect(() => {
    const timer = setInterval(fetchData, 20000);
    return () => clearInterval(timer);
  }, []);

  const instanceNameOf = (id: string) => {
    const inst = instances.find(i => i.id === id);
    return inst ? inst.alias || inst.name : id;
  };

  const pipelineNameOf = (id: string) => pipelines.find(p => p.id === id)?.name ?? id;

  const openCreate = () => {
    setEditing(undefined);
    setTargetMode('instances');
    form.resetFields();
    form.setFieldsValue({ strategy: 'all', enabled: true, instances: [], groups: [] });
    setModalOpen(true);
  };

  const openEdit = (dep: PipelineDeployment) => {
    setEditing(dep);
    const groupSelector = dep.placement?.selector?.group;
    if (groupSelector) {
      setTargetMode('groups');
      form.setFieldsValue({
        name: dep.name,
        pipeline_id: dep.pipeline_id,
        strategy: dep.placement?.strategy ?? 'all',
        groups: groupSelector.split(',').map(v => v.trim()).filter(Boolean),
        enabled: dep.enabled
      });
    } else {
      setTargetMode('instances');
      form.setFieldsValue({
        name: dep.name,
        pipeline_id: dep.pipeline_id,
        strategy: dep.placement?.strategy ?? 'all',
        instances: dep.placement?.instances ?? [],
        enabled: dep.enabled
      });
    }
    setModalOpen(true);
  };

  const onSubmit = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const placement: any = { strategy: values.strategy };
      if (targetMode === 'groups') {
        // 分组目标走 placement.selector 的保留键 `group` (逗号分隔任一命中),
        // 成员变化在 reconciler 每轮重估 — 新入组实例自动获得部署
        placement.selector = { ...(editing?.placement?.selector ?? {}) };
        delete placement.selector.group;
        placement.selector.group = (values.groups ?? []).join(',');
        placement.instances = [];
      } else {
        placement.instances = values.instances ?? [];
        placement.selector = editing?.placement?.selector; // 非 group 的自定义标签选择器原样保留
      }
      const body = {
        name: values.name,
        pipeline_id: values.pipeline_id,
        enabled: !!values.enabled,
        placement
      };
      const res: any = editing ? await updatePipelineDeployment(editing.id, body) : await createPipelineDeployment(body);
      if (res?.data) {
        message.success(editing ? t('common.updateSuccess') : t('common.addSuccess'));
        setModalOpen(false);
        fetchData();
        onRefreshTargets?.();
      } else if (res?.error) {
        message.error((res as any).error?.reason ?? (res as any).error);
      }
    } finally {
      setSaving(false);
    }
  };

  const onToggleEnabled = async (dep: PipelineDeployment, enabled: boolean) => {
    const res: any = await updatePipelineDeployment(dep.id, {
      ...dep,
      enabled
    });
    if (res?.data) {
      message.success(t('common.updateSuccess'));
      fetchData();
    }
  };

  const onDelete = async (dep: PipelineDeployment) => {
    const res: any = await deletePipelineDeployment(dep.id);
    if (res?.data) {
      message.success(t('common.deleteSuccess'));
      fetchData();
    }
  };

  const instanceOptions = useMemo(
    () =>
      instances.map(i => ({
        value: i.id,
        label: `${i.alias || i.name} (${i.type || 'instance'})`
      })),
    [instances]
  );

  const pipelineOptions = useMemo(
    () => pipelines.map(p => ({ value: p.id, label: p.name })),
    [pipelines]
  );

  const groupOptions = useMemo(() => {
    const set = new Set<string>();
    for (const i of instances) for (const g of i.groups ?? []) if (g) set.add(g);
    return [...set].sort().map(g => ({ value: g, label: g }));
  }, [instances]);

  const columns: any[] = [
    {
      title: t('page.gateway.deploy.columns.name'),
      dataIndex: 'name',
      render: (v: string) => <span className="font-500">{v}</span>
    },
    {
      title: t('page.gateway.deploy.columns.pipeline'),
      dataIndex: 'pipeline_id',
      render: (v: string) => pipelineNameOf(v)
    },
    {
      title: t('page.gateway.deploy.columns.strategy'),
      dataIndex: ['placement', 'strategy'],
      width: 100,
      render: (v: string) => (
        <Tag>{v === 'singleton' ? t('page.gateway.deploy.singleton') : t('page.gateway.deploy.allInstances')}</Tag>
      )
    },
    {
      title: t('page.gateway.deploy.columns.targets'),
      dataIndex: ['placement', 'instances'],
      render: (ids: string[] = [], record: PipelineDeployment) => {
        const selector = record.placement?.selector;
        if (selector && Object.keys(selector).length > 0) {
          return (
            <span className="font-mono text-12px">
              {Object.entries(selector).map(([k, v]) => `${k}=${v}`).join(',')}
            </span>
          );
        }
        if (!ids?.length) return '-';
        return (
          <Tooltip title={ids.join(', ')}>
            <span>
              {ids.slice(0, 2).map(instanceNameOf).join(', ')}
              {ids.length > 2 ? ` +${ids.length - 2}` : ''}
            </span>
          </Tooltip>
        );
      }
    },
    {
      title: t('page.gateway.deploy.columns.rollout'),
      key: 'rollout',
      width: 110,
      render: (_: any, row: PipelineDeployment) => {
        const st = statuses[row.id];
        if (!st) return '-';
        const ready = st.ready_count ?? 0;
        return (
          <Button
            type="link"
            className="!px-0"
            onClick={() => {
              setDetailStatus(st);
              setDetailOpen(true);
            }}
          >
            <Tag color={ready >= st.desired_count && st.desired_count > 0 ? 'success' : 'processing'}>
              {ready}/{st.desired_count}
            </Tag>
          </Button>
        );
      }
    },
    {
      title: t('page.gateway.deploy.columns.version'),
      dataIndex: 'version',
      width: 80,
      render: (v: number) => `v${v}`
    },
    {
      title: t('page.gateway.deploy.columns.enabled'),
      dataIndex: 'enabled',
      width: 90,
      hidden: !permissions.update,
      render: (v: boolean, row: PipelineDeployment) => (
        <Switch size="small" checked={!!v} disabled={!permissions.update} onChange={val => onToggleEnabled(row, val)} />
      )
    },
    {
      title: t('common.operation'),
      key: 'actions',
      width: 130,
      hidden: !permissions.update && !permissions.remove,
      render: (_: any, row: PipelineDeployment) => (
        <Space>
          {permissions.update && (
            <Button type="link" size="small" className="!px-0" onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          )}
          {permissions.remove && (
            <Popconfirm title={t('page.gateway.deploy.deleteConfirm', { name: row.name })} onConfirm={() => onDelete(row)}>
              <Button type="link" size="small" danger className="!px-0">
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  return (
    <>
      <div className="mb-12px flex items-center justify-between">
        <span className="text-13px text-[var(--ant-color-text-tertiary)]">{t('page.gateway.deploy.hint')}</span>
        <Space>
          <span className="text-13px text-[var(--ant-color-text-tertiary)]">{t('common.totalItems', { total: data.length })}</span>
          <Button icon={<SyncOutlined />} onClick={fetchData} />
          {permissions.create && (
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              {t('page.gateway.deploy.create')}
            </Button>
          )}
        </Space>
      </div>
      <Table rowKey="id" loading={loading} columns={columns} dataSource={data} pagination={false} />

      <Modal
        destroyOnClose
        open={modalOpen}
        title={editing ? t('page.gateway.deploy.editTitle') : t('page.gateway.deploy.createTitle')}
        confirmLoading={saving}
        onOk={onSubmit}
        onCancel={() => setModalOpen(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item label={t('page.gateway.deploy.columns.name')} name="name" rules={[{ required: true }]}>
            <Input placeholder={t('page.gateway.deploy.namePlaceholder')} />
          </Form.Item>
          <Form.Item label={t('page.gateway.deploy.columns.pipeline')} name="pipeline_id" rules={[{ required: true }]}>
            <Select showSearch optionFilterProp="label" options={pipelineOptions} placeholder={t('page.gateway.deploy.pipelinePlaceholder')} />
          </Form.Item>
          <Form.Item label={t('page.gateway.deploy.targetLabel')}>
            <Radio.Group
              value={targetMode}
              onChange={e => setTargetMode(e.target.value)}
              optionType="button"
              buttonStyle="solid"
            >
              <Radio.Button value="instances">{t('page.gateway.deploy.byInstances', '按实例')}</Radio.Button>
              <Radio.Button value="groups">{t('page.gateway.deploy.byGroups', '按分组')}</Radio.Button>
            </Radio.Group>
          </Form.Item>
          {targetMode === 'instances' ? (
            <Form.Item
              label={t('page.gateway.deploy.targetLabel')}
              name="instances"
              rules={[{ required: targetMode === 'instances', message: t('page.gateway.deploy.targetPlaceholder') }]}
            >
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                options={instanceOptions}
                placeholder={t('page.gateway.deploy.targetPlaceholder')}
              />
            </Form.Item>
          ) : (
            <Form.Item
              label={t('page.gateway.deploy.byGroups', '按分组')}
              name="groups"
              rules={[{ required: targetMode === 'groups', message: t('page.gateway.deploy.groupsPlaceholder', '选择要部署到哪些分组') }]}
              extra={t('page.gateway.deploy.groupsExtra', '分组内的实例变化会自动跟随部署 (reconciler 每轮重估成员)')}
            >
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                options={groupOptions}
                placeholder={t('page.gateway.deploy.groupsPlaceholder', '选择要部署到哪些分组')}
              />
            </Form.Item>
          )}
          <Form.Item label={t('page.gateway.deploy.columns.strategy')} name="strategy">
            <Radio.Group>
              <Radio.Button value="all">{t('page.gateway.deploy.allInstances')}</Radio.Button>
              <Radio.Button value="singleton">{t('page.gateway.deploy.singleton')}</Radio.Button>
            </Radio.Group>
          </Form.Item>
          <Form.Item label={t('page.gateway.deploy.columns.enabled')} name="enabled" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        destroyOnClose
        open={detailOpen}
        footer={null}
        title={t('page.gateway.deploy.statusTitle')}
        onCancel={() => setDetailOpen(false)}
      >
        {detailStatus && (
          <>
            <div className="mb-8px">
              <Tag color="processing">
                {t('page.gateway.deploy.readyCount', { ready: detailStatus.ready_count, desired: detailStatus.desired_count })}
              </Tag>
            </div>
            <Table
              rowKey="instance_id"
              size="small"
              pagination={false}
              dataSource={detailStatus.instances ?? []}
              columns={[
                {
                  title: t('page.gateway.columns.name'),
                  dataIndex: 'instance_id',
                  render: (v: string) => instanceNameOf(v)
                },
                {
                  title: t('page.gateway.deploy.reportState'),
                  dataIndex: 'state',
                  width: 100,
                  render: (v: string, r: any) => (
                    <Space size={4}>
                      <Tag color={stateColor[stateLabel(v)] ?? 'default'}>{stateLabel(v)}</Tag>
                      {r.stale && <Tag color="warning">{t('page.gateway.deploy.stale')}</Tag>}
                    </Space>
                  )
                },
                { title: t('page.gateway.deploy.columns.version'), dataIndex: 'version', width: 70, render: (v: number) => `v${v}` },
                {
                  title: t('page.gateway.deploy.lastSeen'),
                  dataIndex: 'last_seen',
                  width: 170,
                  render: (v: string) => (v ? new Date(v).toLocaleString() : '-')
                },
                {
                  title: t('page.gateway.deploy.reason'),
                  dataIndex: 'reason',
                  ellipsis: true,
                  render: (v: string) => v || '-'
                }
              ]}
            />
          </>
        )}
      </Modal>
    </>
  );
}
