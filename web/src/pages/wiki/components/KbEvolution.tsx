import { ExperimentOutlined, ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Alert, Button, Empty, Popconfirm, Space, Table, Tag, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { distillWikiKb, getWikiEvolutionLog, listWikiExperience, proposeWikiSkills } from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

/* raw enums → locale key suffixes; unknown values fall back to the raw string */
const KIND_COLOR: Record<string, string> = {
  proposal_applied: 'green',
  proposal_dismissed: 'default',
  pattern_distilled: 'purple',
  skill_proposed: 'geekblue',
  gate_verdict: 'cyan',
  compile_agent: 'orange',
  freshness: 'gold'
};
const KIND_LABEL: Record<string, string> = {
  proposal_applied: 'kindApplied',
  proposal_dismissed: 'kindDismissed',
  pattern_distilled: 'kindDistilled',
  skill_proposed: 'kindSkill',
  gate_verdict: 'kindGate',
  compile_agent: 'kindAgent',
  freshness: 'kindFreshness'
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
  const [distillInfo, setDistillInfo] = useState<Api.Wiki.DistillInfo | null>(null);
  const [log, setLog] = useState<Api.Wiki.EvolutionEntry[]>([]);
  const [logTotal, setLogTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [distilling, setDistilling] = useState(false);
  const [proposing, setProposing] = useState(false);

  const label = (map: Record<string, string>, v?: string) => (v && map[v] ? t(`page.wiki.evolution.${map[v]}`) : v || '—');

  const fetchAll = useCallback(() => {
    setLoading(true);
    Promise.all([listWikiExperience(kb.id), getWikiEvolutionLog(kb.id)])
      .then(([exp, logRes]) => {
        setExperiences(exp.entries);
        setExpTotal(exp.total);
        setDistillInfo(exp.distill ?? null);
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

  // manual skill proposer: turn recent experiences into skill-page
  // proposals (drafts land in the article list, the run logs here)
  const onPropose = () => {
    setProposing(true);
    proposeWikiSkills(kb.id)
      .then(run => {
        if (run?.no_action) {
          message.info(t('page.wiki.evolution.proposeNoAction'));
        } else {
          message.success(
            t('page.wiki.evolution.proposeDone', {
              sampled: run?.sampled ?? 0,
              proposals: run?.proposals ?? 0,
              created: run?.created ?? 0,
              updated: run?.updated ?? 0
            })
          );
        }
        fetchAll();
      })
      .finally(() => setProposing(false));
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
    },
    {
      title: t('page.wiki.evolution.expiry'),
      dataIndex: 'retention',
      width: 110,
      // expired traces are swept by the distiller — a red tag says why a row
      // the operator just read may be gone tomorrow
      render: (v: string) => {
        if (!v) return '—';
        const expired = new Date(v).getTime() < Date.now();
        return (
          <Tag color={expired ? 'red' : 'default'}>
            {fmtTime(v).slice(0, 10)}
            {expired ? ` · ${t('page.wiki.evolution.expired')}` : ''}
          </Tag>
        );
      }
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
      // one hop from log line to the artifact it talks about: applied or
      // dismissed proposals link into the governance queue (focus mode),
      // entries carrying an article_id link straight to the article
      render: (v: string, record: any) => {
        const detail = (record?.detail || {}) as { article_id?: string; article_title?: string };
        const govId = record?.target_type === 'governance' ? record?.target_id : undefined;
        if (!detail.article_id && !govId) return v || '—';
        return (
          <Space size={4} wrap>
            <span>{v || '—'}</span>
            {detail.article_id && (
              <a href={`#/wiki/article/${detail.article_id}?kb=${kb.id}`}>
                {detail.article_title || t('page.wiki.evolution.openArticle')}
              </a>
            )}
            {govId && <a href={`#/wiki/governance?focus=${govId}`}>{t('page.wiki.evolution.openProposal')}</a>}
          </Space>
        );
      }
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
            {canUpdateKb && (
              <Popconfirm title={t('page.wiki.evolution.proposeConfirm')} onConfirm={onPropose}>
                <Button icon={<ThunderboltOutlined />} loading={proposing}>
                  {t('page.wiki.evolution.propose')}
                </Button>
              </Popconfirm>
            )}
            <Button icon={<ReloadOutlined />} onClick={fetchAll} />
          </Space>
        </div>
        <div className='p-16px pt-8px'>
          {!kb.experience_capture && <Alert className='mb-12px' showIcon type='info' message={t('page.wiki.evolution.captureOff')} />}
          {distillInfo && (
            <Alert
              className='mb-12px'
              showIcon
              type='info'
              message={
                ['off', 'disabled', '0'].includes(distillInfo.interval.toLowerCase())
                  ? t('page.wiki.evolution.distillOff')
                  : t('page.wiki.evolution.distillSchedule', {
                      interval: distillInfo.interval,
                      last: distillInfo.last_run ? fmtTime(distillInfo.last_run) : t('page.wiki.evolution.distillNeverRan')
                    })
              }
            />
          )}
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
