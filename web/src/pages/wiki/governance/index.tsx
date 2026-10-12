import {
  CheckOutlined,
  CloseOutlined,
  FileTextOutlined,
  FolderOutlined,
  SafetyCertificateOutlined,
  ThunderboltOutlined
} from '@ant-design/icons';
import { Badge, Button, Empty, List, Popconfirm, Segmented, Space, Tag, Tooltip, Tree } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { type GovernanceProposal, applyWikiGovernanceProposal, searchWikiGovernance, updateWikiGovernanceStatus } from '@/service/api';
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
 * page will carry — claim + source excerpt + link to the cited document. */
function CompileProposalPreview({ evidence }: { evidence: Record<string, any> }) {
  const { t } = useTranslation();
  const aliases = (evidence.aliases || []) as string[];
  const statements = (evidence.statements || []) as { claim?: string; doc_id?: string; excerpt?: string }[];
  return (
    <div
      className='mt-6px rounded-6px border border-solid p-8px'
      style={{ borderColor: 'var(--ant-color-border-secondary)' }}
      onClick={e => e.stopPropagation()}
    >
      <div className='mb-4px text-11.5px color-[var(--wiki-text-3)]'>
        {t('page.wiki.governance.compilePreview', {
          kind: String(evidence.kind || ''),
          count: statements.length
        })}
        {aliases.length > 0 ? ` · ${aliases.join(' / ')}` : ''}
        {evidence.target_folder ? ` · ${t('page.wiki.governance.targetFolder')}: ${String(evidence.target_folder)}` : ''}
      </div>
      <div className='flex max-h-220px flex-col gap-6px overflow-y-auto'>
        {statements.map((s, i) => (
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

  const fetchData = (st: string = status) => {
    setLoading(true);
    searchWikiGovernance({ status: st === 'all' ? undefined : st }).then(res => {
      setProposals(((res as any)?.data || []) as GovernanceProposal[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData(status);
  }, [status]);

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
        // a generated page is the visible outcome — land on it so the fix
        // is inspectable; other fixes stay in the queue
        if (proposal.type === 'knowledge_compile' && res?.article_id) {
          nav(`/wiki/article/${res.article_id}?kb=${proposal.kb_id || ''}`);
          return;
        }
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
          <div className='mb-10px text-12.5px color-[var(--wiki-text-3)]'>{t('page.wiki.governance.subtitle')}</div>
          <List
            className='wiki-row-list'
            dataSource={proposals}
            loading={loading}
            pagination={{ pageSize: 10, hideOnSinglePage: true }}
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
                  className='cursor-pointer'
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
                  onClick={() => goArticle(proposal)}
                >
                  <List.Item.Meta
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
                        <a>{proposal.article_title || proposal.article_id}</a>
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
