import { ExperimentOutlined, ReloadOutlined } from '@ant-design/icons';
import { Alert, Button, Empty, Popconfirm, Space, Table, Tag, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { distillWikiKb, getWikiEvolutionLog, listWikiExperience } from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

/* raw enums → locale key suffixes; unknown values fall back to the raw string */
const KIND_COLOR: Record<string, string> = {
  proposal_applied: 'green',
  proposal_dismissed: 'default',
  pattern_distilled: 'purple',
  skill_proposed: 'geekblue',
  gate_verdict: 'cyan',
  compile_agent: 'orange'
};
const KIND_LABEL: Record<string, string> = {
  proposal_applied: 'kindApplied',
  proposal_dismissed: 'kindDismissed',
  pattern_distilled: 'kindDistilled',
  skill_proposed: 'kindSkill',
  gate_verdict: 'kindGate',
  compile_agent: 'kindAgent'
};
const EXP_KIND_LABEL: Record<string, string> = { chat: 'expChat', correction: 'expCorrection', gap: 'expGap' };
const FEEDBACK_LABEL: Record<string, string> = { positive: 'fbPositive', negative: 'fbNegative' };

const fmtTime = (iso?: string) => {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 16).replace('T', ' ');
};

/**
 * The KB's LLM-wiki loop at a glance: captured experiences (chat votes,
 * corrections, gaps) with a manual distill entry point, and the append-only
 * evolution log every actor (human or auto-approve) writes to.
 */
export function KbEvolution({ kb }: { readonly kb: Api.Wiki.Kb }) {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const canUpdateKb = hasAuth('coco#wiki_kb/update');

  const [experiences, setExperiences] = useState<Api.Wiki.Experience[]>([]);
  const [expTotal, setExpTotal] = useState(0);
  const [log, setLog] = useState<Api.Wiki.EvolutionEntry[]>([]);
  const [logTotal, setLogTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [distilling, setDistilling] = useState(false);

  const label = (map: Record<string, string>, v?: string) => (v && map[v] ? t(`page.wiki.evolution.${map[v]}`) : v || '—');

  const fetchAll = useCallback(() => {
    setLoading(true);
    Promise.all([listWikiExperience(kb.id), getWikiEvolutionLog(kb.id)])
      .then(([exp, logRes]) => {
        setExperiences(exp.entries);
        setExpTotal(exp.total);
        setLog(logRes.entries);
        setLogTotal(logRes.total);
      })
      .finally(() => setLoading(false));
  }, [kb.id]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  const onDistill = () => {
    setDistilling(true);
    distillWikiKb(kb.id)
      .then(run => {
        message.success(
          t('page.wiki.evolution.distillDone', { sampled: run?.sampled ?? 0, proposals: run?.proposals ?? 0 })
        );
        fetchAll();
      })
      .finally(() => setDistilling(false));
  };

  const expColumns = [
    { title: t('page.wiki.evolution.time'), dataIndex: 'created', width: 130, render: (v: string) => fmtTime(v) || '—' },
    {
      title: t('page.wiki.evolution.kind'),
      dataIndex: 'kind',
      width: 90,
      render: (v: string) => <Tag>{label(EXP_KIND_LABEL, v)}</Tag>
    },
    {
      title: t('page.wiki.evolution.feedback'),
      dataIndex: 'feedback',
      width: 80,
      render: (v: string) =>
        v ? <Tag color={v === 'positive' ? 'green' : 'red'}>{label(FEEDBACK_LABEL, v)}</Tag> : '—'
    },
    {
      title: t('page.wiki.evolution.question'),
      dataIndex: 'question',
      ellipsis: true,
      render: (v: string) => v || '—'
    },
    {
      title: t('page.wiki.evolution.note'),
      dataIndex: 'note',
      width: '30%',
      ellipsis: true,
      render: (v: string) => v || '—'
    }
  ];

  const logColumns = [
    { title: t('page.wiki.evolution.time'), dataIndex: 'created', width: 130, render: (v: string) => fmtTime(v) || '—' },
    {
      title: t('page.wiki.evolution.kind'),
      dataIndex: 'kind',
      width: 110,
      render: (v: string) => <Tag color={KIND_COLOR[v]}>{label(KIND_LABEL, v)}</Tag>
    },
    {
      title: t('page.wiki.evolution.actor'),
      dataIndex: 'actor',
      width: 140,
      ellipsis: true,
      render: (v: string) => v || '—'
    },
    {
      title: t('page.wiki.evolution.summary'),
      dataIndex: 'summary',
      ellipsis: true,
      render: (v: string) => v || '—'
    }
  ];

  return (
    <div className='flex w-full flex-col gap-14px'>
      <section className='wiki-panel'>
        <div className='wiki-panel-head'>
          <span className='wiki-panel-title'>
            <ExperimentOutlined className='color-[var(--wiki-accent)]' />
            {t('page.wiki.evolution.experienceTitle')}
            {expTotal > 0 && <span className='ml-8px text-12px color-[var(--wiki-text-3)]'>{expTotal}</span>}
          </span>
          <Space>
            {canUpdateKb && (
              <Popconfirm title={t('page.wiki.evolution.distillConfirm')} onConfirm={onDistill}>
                <Button loading={distilling} type='primary'>
                  {t('page.wiki.evolution.distill')}
                </Button>
              </Popconfirm>
            )}
            <Button icon={<ReloadOutlined />} onClick={fetchAll} />
          </Space>
        </div>
        <div className='p-16px pt-8px'>
          {!kb.experience_capture && <Alert className='mb-12px' showIcon type='info' message={t('page.wiki.evolution.captureOff')} />}
          <Table
            columns={expColumns}
            dataSource={experiences}
            loading={loading}
            locale={{ emptyText: <Empty description={t('page.wiki.evolution.expEmpty')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
            pagination={{ pageSize: 8, hideOnSinglePage: true, size: 'small' }}
            rowKey='id'
            size='small'
          />
        </div>
      </section>

      <section className='wiki-panel'>
        <div className='wiki-panel-head'>
          <span className='wiki-panel-title'>
            {t('page.wiki.evolution.title')}
            {logTotal > 0 && <span className='ml-8px text-12px color-[var(--wiki-text-3)]'>{logTotal}</span>}
          </span>
        </div>
        <div className='p-16px pt-8px'>
          <Table
            columns={logColumns}
            dataSource={log}
            loading={loading}
            locale={{ emptyText: <Empty description={t('page.wiki.evolution.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
            pagination={{ pageSize: 8, hideOnSinglePage: true, size: 'small' }}
            rowKey='id'
            size='small'
          />
        </div>
      </section>
    </div>
  );
}
