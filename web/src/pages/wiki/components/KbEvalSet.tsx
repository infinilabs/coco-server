import { CaretRightOutlined, DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Empty, Input, Popconfirm, Select, Space, Table, Tag, Tooltip } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  deleteWikiEvalCase,
  fetchWikiEvalCases,
  fetchWikiEvalRuns,
  runWikiEval,
  saveWikiEvalCase,
  searchWikiArticles
} from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

const fmtTime = (iso?: string) => {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 16).replace('T', ' ');
};

const pct = (v?: number) => `${((v ?? 0) * 100).toFixed(1)}%`;

/**
 * The KB's validation set (LLM-wiki P3): golden queries with expected
 * pages, replayed against KB-scoped published-page retrieval. Skill
 * proposals landing in the KB fire a replay automatically — the trend in
 * the runs table is what the gate consults.
 */
export function KbEvalSet({ kbId }: { readonly kbId: string }) {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const canUpdateKb = hasAuth('coco#wiki_kb/update');

  const [cases, setCases] = useState<Api.Wiki.EvalCase[]>([]);
  const [runs, setRuns] = useState<Api.Wiki.EvalRun[]>([]);
  const [lastRun, setLastRun] = useState<Api.Wiki.EvalRun | null>(null);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [saving, setSaving] = useState(false);

  // add-case form
  const [query, setQuery] = useState('');
  const [picked, setPicked] = useState<string[]>([]);
  const [note, setNote] = useState('');

  const fetchAll = useCallback(() => {
    if (!kbId) return;
    setLoading(true);
    Promise.all([fetchWikiEvalCases(kbId), fetchWikiEvalRuns(kbId)])
      .then(([c, r]) => {
        setCases(c.data || []);
        setRuns(r.data || []);
      })
      .finally(() => setLoading(false));
  }, [kbId]);

  useEffect(() => {
    fetchAll();
  }, [fetchAll]);

  // article picker options: the KB's pages, client-filtered (golden sets
  // reference curated pages, the corpus list is bounded)
  useEffect(() => {
    if (!kbId) return;
    searchWikiArticles({ kbId, size: 1000 }).then(res => setArticles(res.data || []));
  }, [kbId]);

  const articleOptions = useMemo(
    () =>
      articles.map(a => ({
        value: a.id,
        label: a.title,
        // the picker must not promise pages retrieval can never return
        disabled: a.status !== 'published'
      })),
    [articles]
  );
  const titleOf = useCallback(
    (id: string) => articles.find(a => a.id === id)?.title || id,
    [articles]
  );

  const onRun = () => {
    setRunning(true);
    runWikiEval(kbId)
      .then(run => {
        setLastRun(run);
        window.$message?.success(t('page.wiki.evalSet.runDone', { hit: pct(run?.top4_rate), mrr: run?.mrr?.toFixed(3) }));
        fetchAll();
      })
      .finally(() => setRunning(false));
  };

  const onSaveCase = () => {
    if (!query.trim() || picked.length === 0) return;
    setSaving(true);
    saveWikiEvalCase(kbId, {
      query: query.trim(),
      expected_ids: picked,
      expected_titles: picked.map(titleOf),
      note: note.trim() || undefined
    })
      .then(() => {
        setQuery('');
        setPicked([]);
        setNote('');
        fetchAll();
      })
      .finally(() => setSaving(false));
  };

  const onDeleteCase = (id: string) => {
    deleteWikiEvalCase(kbId, id).then(() => {
      setCases(prev => prev.filter(c => c.id !== id));
    });
  };

  return (
    <div>
      <div className='mb-12px flex items-center gap-8px'>
        <div className='text-13px color-[var(--ant-color-text-tertiary)]'>{t('page.wiki.evalSet.hint')}</div>
        {canUpdateKb && (
          <Button
            className='ml-auto'
            disabled={cases.length === 0}
            icon={<CaretRightOutlined />}
            loading={running}
            onClick={onRun}
          >
            {t('page.wiki.evalSet.run')}
          </Button>
        )}
      </div>

      {cases.length === 0 && !loading ? (
        <Empty description={t('page.wiki.evalSet.empty')} />
      ) : (
        <Table
          columns={[
            { title: t('page.wiki.evalSet.colQuery'), dataIndex: 'query' },
            {
              title: t('page.wiki.evalSet.colExpected'),
              dataIndex: 'expected_ids',
              render: (ids: string[], row: Api.Wiki.EvalCase) =>
                (ids || []).map(id => (
                  <Tag key={id}>{(row.expected_titles || [])[ids.indexOf(id)] || titleOf(id)}</Tag>
                ))
            },
            { title: t('page.wiki.evalSet.colNote'), dataIndex: 'note', render: (v?: string) => v || '—' },
            {
              title: t('page.wiki.evalSet.colCreated'),
              dataIndex: 'created',
              render: (v?: string) => <span className='text-xs text-gray-400'>{fmtTime(v)}</span>
            },
            ...(canUpdateKb
              ? [
                  {
                    title: '',
                    key: 'actions',
                    width: 60,
                    render: (_: unknown, row: Api.Wiki.EvalCase) => (
                      <Popconfirm title={t('page.wiki.evalSet.deleteConfirm')} onConfirm={() => onDeleteCase(row.id)}>
                        <Button danger icon={<DeleteOutlined />} size='small' type='text' />
                      </Popconfirm>
                    )
                  }
                ]
              : [])
          ]}
          dataSource={cases}
          loading={loading}
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          rowKey='id'
          size='small'
        />
      )}

      {canUpdateKb && (
        <div className='mt-12px flex items-center gap-8px'>
          <Input
            allowClear
            className='w-260px'
            placeholder={t('page.wiki.evalSet.queryPlaceholder')}
            value={query}
            onChange={e => setQuery(e.target.value)}
          />
          <Select
            className='min-w-300px flex-1'
            mode='multiple'
            optionFilterProp='label'
            placeholder={t('page.wiki.evalSet.expectedPlaceholder')}
            showSearch
            value={picked}
            onChange={setPicked}
            options={articleOptions}
          />
          <Input
            allowClear
            className='w-180px'
            placeholder={t('page.wiki.evalSet.notePlaceholder')}
            value={note}
            onChange={e => setNote(e.target.value)}
          />
          <Button
            disabled={!query.trim() || picked.length === 0}
            icon={<PlusOutlined />}
            loading={saving}
            onClick={onSaveCase}
          >
            {t('page.wiki.evalSet.add')}
          </Button>
        </div>
      )}

      {lastRun && (
        <div className='mt-16px'>
          <Alert
            message={
              <Space size={4} wrap>
                <span>{t('page.wiki.evalSet.lastRun')}</span>
                <Tag color='green'>
                  hit@4 {pct(lastRun.top4_rate)} ({lastRun.top4_hits}/{lastRun.total_cases})
                </Tag>
                <Tag color='blue'>MRR {lastRun.mrr?.toFixed(3)}</Tag>
                {lastRun.trigger === 'skill_applied' && <Tag color='purple'>{t('page.wiki.evalSet.triggerSkill')}</Tag>}
              </Space>
            }
            type={lastRun.top4_rate >= 0.5 ? 'success' : 'warning'}
          />
          <Table
            className='mt-8px'
            columns={[
              { title: t('page.wiki.evalSet.colQuery'), dataIndex: 'query' },
              {
                title: t('page.wiki.evalSet.colOutcome'),
                dataIndex: 'hit_rank',
                render: (rank: number, row) =>
                  row.error ? (
                    <Tag color='red'>{row.error}</Tag>
                  ) : rank > 0 ? (
                    <Tag color={rank <= 4 ? 'green' : 'orange'}>
                      {t('page.wiki.evalSet.hitAt', { rank })}
                    </Tag>
                  ) : (
                    <Tag>{t('page.wiki.evalSet.miss')}</Tag>
                  )
              },
              {
                title: t('page.wiki.evalSet.colRetrieved'),
                dataIndex: 'top_titles',
                render: (titles?: string[]) => (
                  <Tooltip title={(titles || []).join(' / ')}>
                    <span className='text-xs'>{(titles || []).slice(0, 4).join(' / ') || '—'}</span>
                  </Tooltip>
                )
              }
            ]}
            dataSource={lastRun.cases || []}
            pagination={false}
            rowKey='query'
            size='small'
          />
        </div>
      )}

      {runs.length > 0 && (
        <div className='mt-16px'>
          <div className='mb-8px text-13px font-medium'>{t('page.wiki.evalSet.trend')}</div>
          <Table
            columns={[
              { title: t('page.wiki.evalSet.colTime'), dataIndex: 'created', render: (v?: string) => fmtTime(v) },
              {
                title: t('page.wiki.evalSet.colTrigger'),
                dataIndex: 'trigger',
                render: (v?: string) =>
                  v === 'skill_applied' ? (
                    <Tag color='purple'>{t('page.wiki.evalSet.triggerSkill')}</Tag>
                  ) : (
                    <Tag>{t('page.wiki.evalSet.triggerManual')}</Tag>
                  )
              },
              { title: 'hit@4', dataIndex: 'top4_rate', render: (v: number) => pct(v) },
              { title: 'MRR', dataIndex: 'mrr', render: (v: number) => v?.toFixed(3) },
              { title: t('page.wiki.evalSet.colCases'), dataIndex: 'total_cases' }
            ]}
            dataSource={runs}
            pagination={{ pageSize: 5, hideOnSinglePage: true }}
            rowKey='id'
            size='small'
          />
        </div>
      )}
    </div>
  );
}
