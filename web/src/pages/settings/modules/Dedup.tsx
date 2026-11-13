import { CheckOutlined, DeleteOutlined, EyeInvisibleOutlined, RedoOutlined, ScanOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Col, Popconfirm, Progress, Row, Space, Spin, Statistic, Table, Tag, Tooltip, message } from 'antd';
import { memo, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import '../index.scss';
import {
  actDedupGroup,
  confirmDedupGroup,
  dismissDedupGroup,
  fetchDedupReport,
  removeConfirmedGroup
} from '@/service/api/server';
import { useAuth } from '@/hooks/business/auth';

interface DedupMember {
  id: string;
  title: string;
  source_id: string;
  source_name: string;
  type?: string;
  size?: number;
  updated?: number;
  disabled?: boolean;
}

interface DedupGroup {
  key: string;
  tier: number;
  tier_label: string;
  similarity: number;
  keep_id: string;
  members: DedupMember[];
}

interface DedupReport {
  scanned: number;
  fingerprinted: number;
  group_count: number;
  dismissed_pairs: number;
  generated_at: number;
  groups: DedupGroup[];
  /** live scan overlay — the report itself may be stale while scanning */
  scanning?: boolean;
  stage?: string; // fetch | group
  progress_scanned?: number;
  total?: number;
  error?: string;
}

const tierColor: Record<number, string> = { 1: 'red', 2: 'volcano', 3: 'orange', 4: 'gold' };

const formatTime = (ms?: number) => (ms ? new Date(ms).toLocaleString() : '-');

const Dedup = memo(() => {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();

  const permissions = {
    view: hasAuth('coco#document/read'),
    update: hasAuth('coco#document/update'),
    del: hasAuth('coco#document/delete')
  };
  const canManage = permissions.update;

  const [report, setReport] = useState<DedupReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [acting, setActing] = useState<string | null>(null);
  const [confirmedKeys, setConfirmedKeys] = useState<Set<string>>(new Set());
  const [scan, setScan] = useState<{ running: boolean; done: number; total: number; stage?: string }>({
    running: false,
    done: 0,
    total: 0
  });
  const pollRef = useRef<number>();

  const load = async (refresh = false) => {
    // the Spin only blankets the very first load; afterwards the progress
    // row narrates background scans and polls keep the page interactive
    setLoading((prev) => prev || refresh);
    const res: any = await fetchDedupReport(refresh);
    if (res?.data && !res.error) {
      const data = res.data as DedupReport;
      setReport(data);
      if (data.scanning) {
        setScan({ running: true, done: data.progress_scanned ?? 0, total: data.total ?? 0, stage: data.stage });
        window.clearTimeout(pollRef.current);
        pollRef.current = window.setTimeout(() => load(false), 900);
      } else {
        setScan({ running: false, done: 0, total: 0 });
      }
    } else {
      setScan((s) => ({ ...s, running: false }));
    }
    setLoading(false);
  };

  useEffect(() => {
    if (permissions.view) {
      load();
    }
    return () => window.clearTimeout(pollRef.current);
  }, []);

  const confirmGroup = async (group: DedupGroup) => {
    setActing(`${group.key}:confirm`);
    const res: any = await confirmDedupGroup(
      group.members.map((m) => m.id),
      group.tier_label
    );
    if (res?.data && !res.error) {
      message.success(t('page.settings.dedup.groupConfirmed'));
      setConfirmedKeys((prev) => new Set(prev).add(group.key));
    }
    setActing(null);
  };

  const unconfirmGroup = async (group: DedupGroup) => {
    setActing(`${group.key}:unconfirm`);
    const res: any = await removeConfirmedGroup(group.key);
    if (res?.data && !res.error) {
      message.success(t('page.settings.dedup.groupUnconfirmed'));
      setConfirmedKeys((prev) => {
        const next = new Set(prev);
        next.delete(group.key);
        return next;
      });
    }
    setActing(null);
  };

  const memberAction = (group: DedupGroup, member: DedupMember, action: 'exclude' | 'delete') => {
    return async () => {
      if (action === 'delete' && !permissions.del) {
        message.warning(t('page.settings.dedup.noDeletePermission'));
        return;
      }
      if (!permissions.update) {
        message.warning(t('page.settings.dedup.noUpdatePermission'));
        return;
      }
      setActing(`${group.key}:${member.id}:${action}`);
      const res: any = await actDedupGroup([member.id], action);
      if (res?.data && !res.error) {
        message.success(t('page.settings.dedup.acted'));
        // the backend patches its cached report — re-read, no forced re-scan
        load(false);
      }
      setActing(null);
    };
  };

  const dismissGroup = async (group: DedupGroup) => {
    setActing(`${group.key}:dismiss`);
    const res: any = await dismissDedupGroup(group.members.map((m) => m.id));
    if (res?.data && !res.error) {
      message.success(t('page.settings.dedup.dismissed'));
      // the backend patches its cached report — re-read, no forced re-scan
      load(false);
    }
    setActing(null);
  };

  const tierLabel = (g: DedupGroup) => t(`page.settings.dedup.tiers.${g.tier_label}`);

  return (
    <div>
      <Spin spinning={loading}>
        <Row gutter={16} className="m-b-16px">
          <Col span={5}>
            <Card size="small">
              <Statistic title={t('page.settings.dedup.stats.scanned')} value={report?.scanned ?? 0} />
            </Card>
          </Col>
          <Col span={5}>
            <Card size="small">
              <Statistic title={t('page.settings.dedup.stats.fingerprinted')} value={report?.fingerprinted ?? 0} />
            </Card>
          </Col>
          <Col span={5}>
            <Card size="small">
              <Statistic title={t('page.settings.dedup.stats.groups')} value={report?.group_count ?? 0} />
            </Card>
          </Col>
          <Col span={5}>
            <Card size="small">
              <Statistic title={t('page.settings.dedup.stats.dismissed')} value={report?.dismissed_pairs ?? 0} />
            </Card>
          </Col>
          <Col span={4}>
            <Card size="small" styles={{ body: { padding: '12px' } }}>
              <Space direction="vertical" size={4}>
                <Button
                  icon={<ScanOutlined />}
                  loading={loading || scan.running}
                  onClick={() => load(true)}
                  size="small"
                  type="primary"
                >
                  {t('page.settings.dedup.scan')}
                </Button>
                <span className="text-12px color-[var(--ant-color-text-tertiary)]">
                  {scan.running
                    ? t('page.settings.dedup.scanning')
                    : report
                      ? formatTime(report.generated_at)
                      : ''}
                </span>
              </Space>
            </Card>
          </Col>
        </Row>

        {scan.running && (
          <Card className="m-b-16px" size="small">
            <div className="flex items-center gap-12px">
              <Progress
                className="m-b-0 flex-1"
                percent={scan.total > 0 ? Math.min(100, Math.round((scan.done / scan.total) * 100)) : 0}
                showInfo={scan.total > 0}
                size="small"
                status="active"
              />
              <span className="text-12px whitespace-nowrap color-[var(--ant-color-text-tertiary)]">
                {t(`page.settings.dedup.progress.${scan.stage || 'fetch'}`)}
                {scan.total > 0 ? ` ${scan.done}/${scan.total}` : ''}
              </span>
            </div>
          </Card>
        )}

        {!scan.running && report?.error && (
          <Alert className="m-b-16px" message={t('page.settings.dedup.scanError')} showIcon type="error" />
        )}

        {(report?.groups ?? []).map((g) => (
          <Card
            key={g.key}
            size="small"
            className="m-b-12px"
            title={
              <Space>
                <Tag color={tierColor[g.tier]}>{tierLabel(g)}</Tag>
                <span>
                  {t('page.settings.dedup.similarity')}: {(g.similarity * 100).toFixed(0)}%
                </span>
                <span className="color-[var(--ant-color-text-tertiary)]">
                  {t('page.settings.dedup.keep')}: {g.members.find((m) => m.id === g.keep_id)?.title ?? g.keep_id}
                </span>
              </Space>
            }
            extra={
              <Space>
                {g.tier !== 1 && canManage ? (
                  confirmedKeys.has(g.key) ? (
                    <Button
                      loading={acting === `${g.key}:unconfirm`}
                      onClick={() => unconfirmGroup(g)}
                      size="small"
                      type="text"
                    >
                      {t('page.settings.dedup.unconfirm')}
                    </Button>
                  ) : (
                    <Tooltip title={t('page.settings.dedup.confirmTip')}>
                      <Button
                        icon={<CheckOutlined />}
                        loading={acting === `${g.key}:confirm`}
                        onClick={() => confirmGroup(g)}
                        size="small"
                        type="text"
                      >
                        {t('page.settings.dedup.confirm')}
                      </Button>
                    </Tooltip>
                  )
                ) : null}
                <Tooltip title={t('page.settings.dedup.dismissTip')}>
                  <Button
                    loading={acting === `${g.key}:dismiss`}
                    onClick={() => dismissGroup(g)}
                    size="small"
                    type="text"
                  >
                    {t('page.settings.dedup.dismiss')}
                  </Button>
                </Tooltip>
              </Space>
            }
          >
            <Table
              columns={[
                {
                  title: t('page.settings.dedup.columns.title'),
                  dataIndex: 'title',
                  render: (v: string, r: DedupMember) => (
                    <Space>
                      {r.id === g.keep_id && <Tag color="green">{t('page.settings.dedup.keepTag')}</Tag>}
                      <span>{v || r.id}</span>
                    </Space>
                  )
                },
                { title: t('page.settings.dedup.columns.source'), dataIndex: 'source_name' },
                {
                  title: t('page.settings.dedup.columns.size'),
                  dataIndex: 'size',
                  render: (v?: number) => (v ? `${(v / 1024).toFixed(1)} KB` : '-')
                },
                {
                  title: t('page.settings.dedup.columns.updated'),
                  dataIndex: 'updated',
                  render: (v?: number) => formatTime(v)
                },
                {
                  title: t('page.settings.dedup.columns.actions'),
                  render: (_: any, r: DedupMember) => (
                    <Space>
                      {r.disabled && <Tag>{t('page.settings.dedup.excluded')}</Tag>}
                      {r.id !== g.keep_id && (
                        <>
                          <Button
                            icon={<EyeInvisibleOutlined />}
                            loading={acting === `${g.key}:${r.id}:exclude`}
                            onClick={memberAction(g, r, 'exclude')}
                            size="small"
                            type="text"
                          >
                            {t('page.settings.dedup.exclude')}
                          </Button>
                          <Popconfirm
                            title={t('page.settings.dedup.deleteConfirm')}
                            onConfirm={memberAction(g, r, 'delete')}
                          >
                            <Button
                              danger
                              icon={<DeleteOutlined />}
                              loading={acting === `${g.key}:${r.id}:delete`}
                              size="small"
                              type="text"
                            >
                              {t('page.settings.dedup.delete')}
                            </Button>
                          </Popconfirm>
                        </>
                      )}
                    </Space>
                  )
                }
              ]}
              dataSource={g.members}
              loading={loading}
              pagination={false}
              rowKey="id"
              size="small"
            />
          </Card>
        ))}

        {report && report.group_count === 0 && (
          <div className="color-[var(--ant-color-text-tertiary)]">{t('page.settings.dedup.clean')}</div>
        )}
      </Spin>
    </div>
  );
});

export default Dedup;
