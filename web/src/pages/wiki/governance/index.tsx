import {
  CheckOutlined,
  CloseOutlined,
  FileTextOutlined,
  FolderOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined
} from '@ant-design/icons';
import { Badge, Button, Checkbox, Empty, List, Popconfirm, Segmented, Space, Tag, Tooltip, Tree } from 'antd';
import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { type GovernanceProposal, applyWikiGovernanceProposal, batchUpdateWikiGovernanceStatus, getBatchApplyProgress, searchWikiGovernance, startBatchApply, updateWikiGovernanceStatus } from '@/service/api';
import { WikiShell } from '../components/WikiShell';

const TYPE_COLOR: Record<string, string> = {
  stale: 'warning',
  duplicate: 'purple',
  conflict: 'error',
  low_quality: 'default',
  orphan: 'cyan',
  knowledge_gap: 'volcano',
  correction: 'geekblue',
  entity_conflict: 'error',
  entity_duplicate: 'purple',
  knowledge_compile: 'geekblue',
  tree_organize: 'cyan'
};

// proposal types whose fix the queue can execute itself — these render the
// primary 执行修复 button; the rest keep the record-only resolve/dismiss
const APPLICABLE = new Set([
  'orphan',
  'duplicate',
  'stale',
  'article_refresh',
  'entity_conflict',
  'knowledge_compile',
  'tree_organize'
]);

interface RawTocNode {
  id?: string;
  title?: string;
  type?: string;
  children?: RawTocNode[];
}

/** Review-before-apply for knowledge_compile: the statements the generated
 * page will carry — claim + source excerpt + link to the cited document.
 * Topic proposals (evidence.sections) preview the outline instead: each
 * planned section with its drafted prose and provenance count. */
function CompileProposalPreview({ evidence }: { evidence: Record<string, any> }) {
  const { t } = useTranslation();
  const aliases = (evidence.aliases || []) as string[];
  const statements = (evidence.statements || []) as { claim?: string; doc_id?: string; excerpt?: string }[];
  const sections = (evidence.sections || []) as { heading?: string; body?: string; doc_ids?: string[] }[];
  const keyEntities = (evidence.key_entities || []) as string[];
  const isTopic = sections.length > 0;
  return (
    <div
      className='mt-6px rounded-6px border border-solid p-8px'
      style={{ borderColor: 'var(--ant-color-border-secondary)' }}
      onClick={e => e.stopPropagation()}
    >
      <div className='mb-4px text-11.5px color-[var(--wiki-text-3)]'>
        {isTopic
          ? t('page.wiki.governance.topicPreview', { count: sections.length, claims: statements.length })
          : t('page.wiki.governance.compilePreview', {
              kind: String(evidence.kind || ''),
              count: statements.length
            })}
        {aliases.length > 0 ? ` · ${aliases.join(' / ')}` : ''}
        {evidence.update_article_id || evidence.update_title ? ` · ${t('page.wiki.governance.updateTarget')}` : ''}
        {Array.isArray(evidence.absorb_ids) && evidence.absorb_ids.length > 0
          ? ` · ${t('page.wiki.governance.absorb', { count: evidence.absorb_ids.length })}`
          : ''}
        {evidence.target_folder ? ` · ${t('page.wiki.governance.targetFolder')}: ${String(evidence.target_folder)}` : ''}
      </div>
      {isTopic && keyEntities.length > 0 && (
        <div className='mb-6px flex flex-wrap gap-4px'>
          {keyEntities.map(name => (
            <span
              className='rounded-4px px-6px text-11px'
              key={name}
              style={{ background: 'var(--wiki-accent-softer)' }}
            >
              {name}
            </span>
          ))}
        </div>
      )}
      <div className='flex max-h-220px flex-col gap-6px overflow-y-auto'>
        {isTopic
          ? sections.map((sec, i) => (
              <div
                className='rounded-4px px-8px py-6px'
                key={i}
                style={{ background: 'var(--wiki-accent-softer)' }}
              >
                <div className='text-12.5px font-500'>
                  {sec.heading}
                  {!!(sec.doc_ids || []).length && (
                    <span className='ml-6px text-11px font-400 color-[var(--wiki-text-3)]'>
                      {t('page.wiki.governance.sectionSources', { count: sec.doc_ids!.length })}
                    </span>
                  )}
                </div>
                {sec.body && (
                  <div
                    className='mt-2px max-h-40px overflow-hidden text-11.5px color-[var(--wiki-text-3)]'
                    style={{ display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}
                  >
                    {sec.body}
                  </div>
                )}
              </div>
            ))
          : statements.map((s, i) => (
              <div
                className='rounded-4px px-8px py-6px'
                key={i}
                style={{ background: 'var(--wiki-accent-softer)' }}
              >
                <div className='text-12.5px'>{s.claim}</div>
                {s.excerpt && (
                  <div
                    className='mt-2px max-h-28px overflow-hidden text-11.5px color-[var(--wiki-text-3)]'
                    style={{ display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}
                  >
                    {s.excerpt}
                  </div>
                )}
                {s.doc_id && (
                  <a
                    className='mt-2px inline-block text-11px text-[var(--ant-color-link)]'
                    href={`#/preview/document/${s.doc_id}`}
                    onClick={e => e.stopPropagation()}
                  >
                    {t('page.wiki.article.openSource')}
                  </a>
                )}
              </div>
            ))}
      </div>
    </div>
  );
}

/** Review-before-apply for entity_conflict: the two competing types, with
 * the mentioning document one click away. */
function ConflictProposalPreview({ evidence }: { evidence: Record<string, any> }) {
  const { t } = useTranslation();
  return (
    <div
      className='mt-6px flex flex-wrap items-center gap-6px'
      onClick={e => e.stopPropagation()}
    >
      <Tag color='gold'>
        {t('page.wiki.governance.existingType')}: {String(evidence.existing_type || '?')}
      </Tag>
      <span className='text-11px color-[var(--wiki-text-3)]'>vs</span>
      <Tag color='purple'>
        {t('page.wiki.governance.mentionedAs')}: {String(evidence.extracted_type || '?')}
      </Tag>
      {evidence.doc_id && (
        <a
          className='text-11px text-[var(--ant-color-link)]'
          href={`#/preview/document/${evidence.doc_id}`}
          onClick={e => e.stopPropagation()}
        >
          {t('page.wiki.article.openSource')}
        </a>
      )}
    </div>
  );
}

/** Review-before-apply for tree_organize: renders the proposed TOC straight
 * from evidence.tree — folders with article counts, articles as leaves. */
function TreeProposalPreview({ tree }: { tree: RawTocNode[] }) {
  const { t } = useTranslation();
  const countArticles = (n: RawTocNode): number =>
    n.type === 'article' ? 1 : (n.children || []).reduce((s, c) => s + countArticles(c), 0);
  const total = (nodes: RawTocNode[]): number => nodes.reduce((s, n) => s + countArticles(n), 0);
  const toTreeData = (nodes: RawTocNode[], prefix: string): any[] =>
    nodes.map((n, i) => ({
      key: `${prefix}-${i}`,
      title:
        n.type === 'article' ? n.title : `${n.title || '?'} (${countArticles(n)})`,
      icon:
        n.type === 'article' ? (
          <FileTextOutlined className='color-[var(--wiki-text-3)]' />
        ) : (
          <FolderOutlined className='color-[var(--wiki-accent)]' />
        ),
      children: n.children?.length ? toTreeData(n.children, `${prefix}-${i}`) : undefined
    }));
  return (
    <div
      className='mt-6px rounded-6px border border-solid p-8px'
      style={{ borderColor: 'var(--ant-color-border-secondary)' }}
      onClick={e => e.stopPropagation()}
    >
      <div className='mb-4px text-11.5px color-[var(--wiki-text-3)]'>
        {t('page.wiki.governance.treePreview', { count: total(tree), folders: tree.length })}
      </div>
      {tree.length > 0 ? (
        <Tree
          defaultExpandedKeys={tree.map((_, i) => `r-${i}`)}
          height={260}
          showIcon
          treeData={toTreeData(tree, 'r')}
        />
      ) : null}
    </div>
  );
}

/**
 * Knowledge-governance queue: the background scanner files proposals (stale pages, duplicates, conflicts, low quality,
 * orphans) and this page is the human gate — every fix happens on the article face, marking resolved or dismissing only
 * records the decision (D1).
 */
export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [status, setStatus] = useState<string>('open');
  const [proposals, setProposals] = useState<GovernanceProposal[]>([]);
  const [loading, setLoading] = useState(true);
  // batch review: the checked ids and the in-flight flag for the toolbar
  const [selected, setSelected] = useState<string[]>([]);
  const [batchLoading, setBatchLoading] = useState(false);
  // controlled pagination: a literal pageSize in the config is treated as
  // controlled and snaps back to 10 on every refresh
  const [pageCurrent, setPageCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const fetchData = (st: string = status) => {
    setLoading(true);
    searchWikiGovernance({ status: st === 'all' ? undefined : st }).then(res => {
      setProposals(((res as any)?.data || []) as GovernanceProposal[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData(status);
    // a status switch resets the checked set — selections belong to one
    // filtered view only
    setSelected([]);
    setPageCurrent(1);
  }, [status]);

  const toggleSelected = (id: string, checked: boolean) => {
    setSelected(prev => (checked ? [...prev, id] : prev.filter(x => x !== id)));
  };

  const runBatch = (to: string) => {
    if (!selected.length) return;
    setBatchLoading(true);
    batchUpdateWikiGovernanceStatus(selected, to)
      .then((res: any) => {
        const updated = res?.updated ?? 0;
        const skipped = res?.skipped ?? 0;
        if (skipped > 0) {
          window.$message?.warning(t('page.wiki.governance.batchPartial', { updated, skipped }));
        } else {
          window.$message?.success(t('page.wiki.governance.batchDone', { count: updated }));
        }
        setSelected([]);
        fetchData();
      })
      .catch(() => {
        window.$message?.error(t('page.wiki.governance.batchFailed'));
      })
      .finally(() => setBatchLoading(false));
  };

  // batch FIX: the worker runs server-side (a compile apply is an LLM call
  // per page); the UI only starts it and polls progress — leaving the page
  // never stops the worker, and a fresh mount re-attaches to a running one
  const [applyProgress, setApplyProgress] = useState<{ done: number; total: number; current?: string } | null>(null);
  const applyPollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // select-all on a big queue exceeds the per-batch cap (200) — the run is
  // split into chunks and the poller starts the next one as each finishes
  const applyChunksRef = useRef<string[][]>([]);
  const applyFinishedRef = useRef(0);
  const applyTotalRef = useRef(0);

  const stopApplyPolling = () => {
    if (applyPollRef.current) {
      clearInterval(applyPollRef.current);
      applyPollRef.current = null;
    }
  };

  const pollApplyProgress = () => {
    getBatchApplyProgress().then((p: any) => {
      if (!p || p.status === 'idle') {
        stopApplyPolling();
        setApplyProgress(null);
        setBatchLoading(false);
        return;
      }
      if (p.status === 'running') {
        setBatchLoading(true);
        setApplyProgress({
          done: applyFinishedRef.current + p.done + p.failed + p.skipped,
          total: applyTotalRef.current || p.total,
          current: p.current
        });
        return;
      }
      // chunk finished — count it, start the next one, or wrap up
      applyFinishedRef.current += p.done + p.failed + p.skipped
      if (applyChunksRef.current.length > 0) {
        const next = applyChunksRef.current.shift();
        if (next && next.length) {
          startBatchApply(next)
            .then(() => { /* polling continues */ })
            .catch(() => {
              stopApplyPolling();
              setApplyProgress(null);
              setBatchLoading(false);
              window.$message?.error(t('page.wiki.governance.batchFailed'));
            });
          return;
        }
      }
      stopApplyPolling();
      setApplyProgress(null);
      setBatchLoading(false);
      setSelected([]);
      fetchData();
      const failedShown = p.failed;
      if (failedShown > 0 || p.skipped > 0) {
        window.$message?.warning(t('page.wiki.governance.batchApplyDone', { done: applyFinishedRef.current + p.done, failed: failedShown, skipped: p.skipped }));
      } else {
        window.$message?.success(t('page.wiki.governance.batchApplyAllDone', { count: applyFinishedRef.current + p.done }));
      }
    });
  };

  const startApplyPolling = () => {
    stopApplyPolling();
    applyPollRef.current = setInterval(pollApplyProgress, 2500);
  };

  useEffect(() => {
    // a running batch survives navigation — re-attach on mount (chunks
    // queued client-side do not survive; the running one finishes)
    getBatchApplyProgress().then((p: any) => {
      if (p?.status === 'running') {
        setBatchLoading(true);
        applyTotalRef.current = p.total;
        setApplyProgress({ done: p.done + p.failed + p.skipped, total: p.total, current: p.current });
        startApplyPolling();
      }
    });
    return stopApplyPolling;
  }, []);

  const runBatchApply = () => {
    if (!selected.length) return;
    const chunks: string[][] = [];
    for (let i = 0; i < selected.length; i += 200) {
      chunks.push(selected.slice(i, i + 200));
    }
    applyChunksRef.current = chunks.slice(1);
    applyTotalRef.current = selected.length;
    applyFinishedRef.current = 0;
    setBatchLoading(true);
    startBatchApply(chunks[0])
      .then(() => startApplyPolling())
      .catch((err: any) => {
        setBatchLoading(false);
        if (String(err)?.includes('already running')) {
          window.$message?.warning(t('page.wiki.governance.batchApplyRunning'));
        } else {
          window.$message?.error(t('page.wiki.governance.batchFailed'));
        }
      });
  };

  const act = (proposal: GovernanceProposal, to: string) => {
    updateWikiGovernanceStatus(proposal.id, to).then(() => {
      window.$message?.success(t('common.updateSuccess'));
      fetchData();
    });
  };

  const [applying, setApplying] = useState<string>('');
  const apply = (proposal: GovernanceProposal) => {
    setApplying(proposal.id);
    applyWikiGovernanceProposal(proposal.id)
      .then((res: any) => {
        const action = res?.applied || '';
        window.$message?.success(t('page.wiki.governance.applied', { action }));
        // stay on the queue — the review work is batch work; the outcome is
        // visible via the refreshed list and the generated page via its row
        fetchData();
      })
      .catch(() => {
        window.$message?.error(t('page.wiki.governance.applyFailed'));
      })
      .finally(() => setApplying(''));
  };

  const goArticle = (proposal: GovernanceProposal) => {
    // entity-dimension proposals have no article page — land on the KB;
    // work-side signals (knowledge_gap/correction) anchor on a query or a
    // chat message, not an article either
    if (proposal.type.startsWith('entity_') || proposal.type === 'knowledge_gap' || proposal.type === 'correction') {
      nav(proposal.kb_id ? `/wiki/kb/${proposal.kb_id}` : '/wiki/list');
      return;
    }
    nav(`/wiki/article/${proposal.article_id}?kb=${proposal.kb_id}`);
  };

  const openCount = proposals.length;

  return (
    <WikiShell>
      <section className='wiki-panel'>
        <div className='wiki-panel-head'>
          <span className='wiki-panel-title'>
            <SafetyCertificateOutlined className='color-[var(--wiki-accent)]' />
            {t('page.wiki.governance.title')}
          </span>
          <Segmented
            value={status}
            options={[
              { value: 'open', label: t('page.wiki.governance.statusOpen') },
              { value: 'resolved', label: t('page.wiki.governance.statusResolved') },
              { value: 'dismissed', label: t('page.wiki.governance.statusDismissed') },
              { value: 'all', label: t('page.wiki.governance.statusAll') }
            ]}
            onChange={v => setStatus(v as string)}
          />
        </div>
        <div className='px-16px pb-16px pt-4px'>
          <div className='mb-10px flex items-center gap-8px text-12.5px color-[var(--wiki-text-3)]'>
            {/* select-all over the whole filtered queue (the fetched window),
                not just the visible page — batch actions are the point */}
            <Checkbox
              checked={proposals.length > 0 && selected.length >= proposals.length}
              indeterminate={selected.length > 0 && selected.length < proposals.length}
              onChange={e => setSelected(e.target.checked ? proposals.map(p2 => p2.id) : [])}
            />
            <span>{t('page.wiki.governance.selectAll')}</span>
            <span>{t('page.wiki.governance.subtitle')}</span>
          </div>
          {/* batch review bar: only when something is checked */}
          {selected.length > 0 && (
            <div
              className='mb-10px flex items-center gap-10px rounded-6px border border-solid px-10px py-6px'
              style={{ borderColor: 'var(--ant-color-primary-border)', background: 'var(--ant-color-primary-bg)' }}
            >
              <span className='text-12.5px'>
                {t('page.wiki.governance.batchSelected', { count: selected.length })}
              </span>
              <Popconfirm
                onConfirm={() => runBatch('resolved')}
                title={t('page.wiki.governance.batchResolveConfirm', { count: selected.length })}
              >
                <Button
                  icon={<CheckOutlined />}
                  loading={batchLoading}
                  size='small'
                  type='primary'
                >
                  {t('page.wiki.governance.batchResolve')}
                </Button>
              </Popconfirm>
              <Popconfirm
                onConfirm={() => runBatch('dismissed')}
                title={t('page.wiki.governance.batchDismissConfirm', { count: selected.length })}
              >
                <Button
                  icon={<CloseOutlined />}
                  loading={batchLoading}
                  size='small'
                >
                  {t('page.wiki.governance.batchDismiss')}
                </Button>
              </Popconfirm>
              <Popconfirm
                onConfirm={() => runBatchApply()}
                title={t('page.wiki.governance.batchApplyConfirm', { count: selected.length })}
              >
                <Button
                  disabled={batchLoading || !!applyProgress}
                  icon={<ThunderboltOutlined />}
                  size='small'
                  type='primary'
                >
                  {t('page.wiki.governance.batchApply')}
                </Button>
              </Popconfirm>
              {applyProgress ? (
                <span className='text-12px color-[var(--ant-color-warning)]'>
                  {t('page.wiki.governance.batchRunning', {
                    done: applyProgress.done,
                    total: applyProgress.total,
                    current: applyProgress.current || ''
                  })}
                </span>
              ) : null}
              <a
                className='text-12px'
                onClick={() => setSelected([])}
              >
                {t('page.wiki.governance.batchClear')}
              </a>
            </div>
          )}
          <List
            className='wiki-row-list'
            dataSource={proposals}
            loading={loading}
            pagination={{
              current: pageCurrent,
              pageSize,
              showSizeChanger: true,
              pageSizeOptions: [10, 20, 50, 100],
              hideOnSinglePage: true,
              onChange: (p: number, ps: number) => {
                setPageCurrent(p);
                setPageSize(ps);
              }
            }}
            locale={{
              emptyText: (
                <Empty
                  description={t('page.wiki.governance.empty')}
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              )
            }}
            renderItem={proposal => {
              const dupOf = proposal.evidence?.duplicate_of;
              return (
                <List.Item
                  actions={[
                    APPLICABLE.has(proposal.type) && (
                      <Popconfirm
                        key='apply'
                        title={t('page.wiki.governance.applyConfirm')}
                        onConfirm={() => apply(proposal)}
                      >
                        <Button
                          icon={<ThunderboltOutlined />}
                          onClick={e => e.stopPropagation()}
                          loading={applying === proposal.id}
                          size='small'
                          type='primary'
                        >
                          {t('page.wiki.governance.apply')}
                        </Button>
                      </Popconfirm>
                    ),
                    <Popconfirm
                      key='resolve'
                      title={t('page.wiki.governance.resolveConfirm')}
                      onConfirm={() => act(proposal, 'resolved')}
                    >
                      <Button
                        ghost
                        icon={<CheckOutlined />}
                        onClick={e => e.stopPropagation()}
                        size='small'
                        type='primary'
                      >
                        {t('page.wiki.governance.resolve')}
                      </Button>
                    </Popconfirm>,
                    <Popconfirm
                      key='dismiss'
                      title={t('page.wiki.governance.dismissConfirm')}
                      onConfirm={() => act(proposal, 'dismissed')}
                    >
                      <Button
                        icon={<CloseOutlined />}
                        onClick={e => e.stopPropagation()}
                        size='small'
                      >
                        {t('page.wiki.governance.dismiss')}
                      </Button>
                    </Popconfirm>
                  ].filter(Boolean)}
                >
                  <List.Item.Meta
                    avatar={
                      <Checkbox
                        checked={selected.includes(proposal.id)}
                        onClick={e => e.stopPropagation()}
                        onChange={e => toggleSelected(proposal.id, e.target.checked)}
                      />
                    }
                    description={
                      <Space
                        direction='vertical'
                        size={2}
                      >
                        <span>{proposal.reason}</span>
                        {proposal.type === 'tree_organize' && Array.isArray(proposal.evidence?.tree) && (
                          <TreeProposalPreview tree={proposal.evidence.tree as RawTocNode[]} />
                        )}
                        {proposal.type === 'knowledge_compile' && proposal.evidence?.statements && (
                          <CompileProposalPreview evidence={proposal.evidence} />
                        )}
                        {proposal.type === 'entity_conflict' && proposal.evidence?.existing_type && (
                          <ConflictProposalPreview evidence={proposal.evidence} />
                        )}
                        {proposal.type === 'article_refresh' && proposal.evidence?.doc_id && (
                          <a
                            className='text-11.5px text-[var(--ant-color-link)]'
                            href={`#/preview/document/${proposal.evidence.doc_id}`}
                            onClick={e => e.stopPropagation()}
                          >
                            {t('page.wiki.governance.refreshDoc')}
                          </a>
                        )}
                        {dupOf && (
                          <Tooltip title={t('page.wiki.governance.duplicateOf')}>
                            <Tag
                              className='cursor-pointer'
                              color='purple'
                              onClick={e => {
                                e.stopPropagation();
                                nav(`/wiki/article/${dupOf.id}?kb=${proposal.kb_id}`);
                              }}
                            >
                              {String(dupOf.title || dupOf.id)}
                            </Tag>
                          </Tooltip>
                        )}
                        {proposal.evidence?.pending_version ? (
                          <span className='text-xs color-[var(--wiki-text-3)]'>
                            {t('page.wiki.governance.pendingVersion', {
                              version: String(proposal.evidence.pending_version)
                            })}
                          </span>
                        ) : null}
                      </Space>
                    }
                    title={
                      <Space>
                        <Badge
                          count={status === 'open' ? '!' : 0}
                          offset={[10, 0]}
                        >
                          <Tag color={TYPE_COLOR[proposal.type] || 'default'}>
                            {t(`page.wiki.governance.type.${proposal.type}`)}
                          </Tag>
                        </Badge>
                        {/* the only row-jump affordance: an explicit title link —
                            confirm/dismiss actions must never navigate away */}
                        <a
                          className='cursor-pointer'
                          onClick={e => {
                            e.stopPropagation();
                            goArticle(proposal);
                          }}
                        >
                          {proposal.article_title || proposal.article_id}
                        </a>
                      </Space>
                    }
                  />
                </List.Item>
              );
            }}
          />
          {status === 'open' && openCount > 0 && (
            <div className='mt-2 text-xs color-[var(--wiki-text-3)]'>
              {t('page.wiki.governance.hint', { count: String(openCount) })}
            </div>
          )}
        </div>
      </section>
    </WikiShell>
  );
}
