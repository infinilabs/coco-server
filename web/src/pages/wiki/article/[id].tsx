import {
  ArrowLeftOutlined,
  BoldOutlined,
  CheckOutlined,
  ClusterOutlined,
  CopyOutlined,
  DownloadOutlined,
  EditOutlined,
  LineOutlined,
  LinkOutlined,
  UnorderedListOutlined,
  EyeOutlined,
  HistoryOutlined,
  RobotOutlined,
  SendOutlined,
  StarFilled,
  StarOutlined,
  LikeFilled,
  LikeOutlined,
  TagsOutlined
} from '@ant-design/icons';
import {
  Avatar,
  Button,
  Card,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Input,
  List,
  Modal,
  Select,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography
} from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Markdown from '@/components/DocumentDrawer/Markdown';
import {
  createWikiBookmark,
  deleteWikiBookmark,
  getWikiArticle,
  getWikiArticleVersions,
  getWikiEntity,
  createWikiLike,
  deleteWikiLike,
  searchWikiBookmarks,
  searchWikiLikes,
  updateWikiArticle,
  updateWikiArticleStatus
} from '@/service/api';
import { parseStructuredContent, parseWikiLink } from '../shared/content';
import { WikiLinkTag, renderInlineWikiLinks } from '../shared/WikiLinkTag';
import { diffLines } from '../shared/diff';
import { AIEditModal } from '../components/AIEditModal';
import { ArticleComments } from '../components/ArticleComments';
import { recordRecentArticle } from '../shared/recent';
import { ArticleOutline } from '../components/ArticleOutline';
import { selectUserInfo } from '@/store/slice/auth';
import { useAuth } from '@/hooks/business/auth';
import { WikiShell } from '../components/WikiShell';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  reviewed: 'processing',
  published: 'success',
  archived: 'warning'
};

const CHANGE_TYPE_COLOR: Record<string, string> = {
  'ai-generated': 'purple',
  'human-edited': 'blue',
  'auto-updated': 'orange'
};

/** allowed forward transitions of the article status machine */
const NEXT_STATUS: Partial<Record<string, { to: string; labelKey: string }[]>> = {
  draft: [{ to: 'reviewed', labelKey: 'page.wiki.article.submitReview' }],
  reviewed: [{ to: 'published', labelKey: 'page.wiki.article.publish' }],
  published: [{ to: 'archived', labelKey: 'page.wiki.article.archive' }]
};

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className='mb-5'>
      <div className='mb-2 font-medium'>{title}</div>
      {children}
    </div>
  );
}

function RelatedList({ items, color, resolve, onOpen }: { items: string[]; color: string; resolve: (type: string, name: string) => boolean; onOpen: (type: string, name: string) => void }) {
  return (
    <Space size={4} wrap>
      {items.map(entry => {
        const link = parseWikiLink(entry);
        if (link) {
          return (
            <WikiLinkTag
              key={entry}
              type={link.type}
              label={link.label}
              resolved={resolve(link.type, link.label)}
              onClick={() => onOpen(link.type, link.label)}
            />
          );
        }
        return (
          <Tag color={color} key={entry}>
            {entry}
          </Tag>
        );
      })}
    </Space>
  );
}

export function Component() {
  const { t } = useTranslation();
  const { id } = useParams();
  const nav = useNavigate();
  const [searchParams] = useSearchParams();
  const kbId = searchParams.get('kb') || '';
  const userInfo = useAppSelector(selectUserInfo);
  const canEditArticle = useAuth().hasAuth('coco#wiki_article/update');

  const [article, setArticle] = useState<Api.Wiki.Article | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const contentRef = useRef<any>(null);

  // wrap the selection (or drop a placeholder at the cursor) in the given markers
  const insertAround = (before: string, after: string, placeholder: string) => {
    const el = contentRef.current?.resizableTextArea?.textArea as HTMLTextAreaElement | undefined;
    if (!el) return;
    const start = el.selectionStart ?? draft.content?.length ?? 0;
    const end = el.selectionEnd ?? start;
    const selected = draft.content?.slice(start, end) || '';
    const inserted = `${before}${selected || placeholder}${after}`;
    const next = `${draft.content?.slice(0, start) ?? ''}${inserted}${draft.content?.slice(end) ?? ''}`;
    setDraft(d => ({ ...d, content: next }));
    requestAnimationFrame(() => {
      el.focus();
      const caret = start + before.length + (selected || placeholder).length;
      el.setSelectionRange(caret, caret);
    });
  };

  const onExportMarkdown = () => {
    if (!article) return;
    const blob = new Blob([article.content || ''], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${article.title || 'article'}.md`;
    a.click();
    URL.revokeObjectURL(url);
  };
  const [saving, setSaving] = useState(false);
  const [preview, setPreview] = useState(false);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [versions, setVersions] = useState<Api.Wiki.Version[]>([]);
  const [versionView, setVersionView] = useState<Api.Wiki.Version | null>(null);
  const [diffPair, setDiffPair] = useState<{ older: Api.Wiki.Version; newer: Api.Wiki.Version } | null>(null);
  const [aiEditOpen, setAiEditOpen] = useState(false);
  const [bookmarkId, setBookmarkId] = useState<string | null>(null);
  const [likeId, setLikeId] = useState<string | null>(null);
  const [likeCount, setLikeCount] = useState(0);

  // edit form state (kept flat — the form is a single record)
  const [draft, setDraft] = useState<Partial<Api.Wiki.Article>>({});

  const fetchArticle = () => {
    if (!id) return;
    setLoading(true);
    getWikiArticle(id).then(res => {
      const a = (res as any) as Api.Wiki.Article | null;
      setArticle(a);
      setLoading(false);
    });
  };

  useEffect(fetchArticle, [id]);

  useEffect(() => {
    if (article?.id) {
      recordRecentArticle({ id: article.id, title: article.title, kb_id: kbId || article.kb_id });
    }
  }, [article?.id]);

  useEffect(() => {
    if (!id) return;
    searchWikiBookmarks().then(res => {
      const hit = (((res as any)?.data || []) as any[]).find(b => b.article_id === id);
      setBookmarkId(hit?.id ?? null);
    });
    searchWikiLikes(id).then(res => {
      const likes = ((res as any)?.data || []) as any[];
      setLikeCount(likes.length);
      const mine = likes.find(l => l.user_id && l.user_id === userInfo?.id);
      setLikeId(mine?.id ?? null);
    });
  }, [id]);

  const toggleBookmark = () => {
    if (!id) return;
    if (bookmarkId) {
      deleteWikiBookmark(bookmarkId).then(() => setBookmarkId(null));
    } else {
      createWikiBookmark(id).then(res => setBookmarkId(((res as any)?._id as string) || null));
    }
  };

  // dangling wikilink repair (phase O2): unresolved [[type:name]] links can
  // be turned into proposed entities in one click, then relinked
  const unresolvedLinks = useMemo(
    () => (article?.linked_pages || []).filter(lp => !lp.entity_id),
    [article?.linked_pages]
  );
  const [repairOpen, setRepairOpen] = useState(false);
  const [repairing, setRepairing] = useState(false);

  const runRepair = async () => {
    if (!article?.id || unresolvedLinks.length === 0) return;
    setRepairing(true);
    let okCount = 0;
    try {
      const results = await Promise.all(
        unresolvedLinks.map(lp =>
          createWikiEntity({ name: lp.name, type: lp.type || 'concept' })
            .then(res => ({ name: lp.name, ok: !!(res as any)?._id }))
            .catch(() => ({ name: lp.name, ok: false }))
        )
      );
      okCount = results.filter(r => r.ok).length;
      await relinkWikiArticle(article.id);
    } finally {
      // always release the modal — a failed relink must not wedge the button
      setRepairing(false);
    }
    setRepairOpen(false);
    if (okCount > 0) {
      window.$message?.success(t('page.wiki.repair.success', { count: String(okCount) }));
      fetchArticle();
    } else {
      window.$message?.error(t('page.wiki.repair.failed'));
    }
  };

  const toggleLike = () => {
    if (!id) return;
    if (likeId) {
      deleteWikiLike(likeId).then(() => {
        setLikeId(null);
        setLikeCount(c => Math.max(0, c - 1));
      });
    } else {
      createWikiLike(id).then(res => {
        const newId = ((res as any)?._id as string) || null;
        if (newId) {
          setLikeId(newId);
          setLikeCount(c => c + 1);
        }
      });
    }
  };

  const structured = useMemo(
    () => (article ? parseStructuredContent(article.content) : null),
    [article?.content]
  );

  // wikilink capsules (ontology O4): [[type:name]] in the content resolves
  // against linked_pages saved with the article — resolved links render as
  // colored capsules, unresolved ones as dashed gray with a repair hint
  const linkIndex = useMemo(() => {
    const m = new Map<string, Api.Wiki.LinkedPage>();
    for (const lp of article?.linked_pages || []) {
      m.set(`${lp.type}:${lp.name}`, lp);
    }
    return m;
  }, [article?.linked_pages]);

  const resolveWikiLink = useCallback((type: string, name: string) => !!linkIndex.get(`${type}:${name}`)?.entity_id, [linkIndex]);

  const openWikiLink = useCallback(
    (type: string, name: string) => {
      const lp = linkIndex.get(`${type}:${name}`);
      if (!lp?.entity_id) {
        // unresolved: offer the existing one-click repair flow
        setRepairOpen(true);
        return;
      }
      getWikiEntity(lp.entity_id).then(entity => {
        if (entity?.article_id) {
          nav(`/wiki/article/${entity.article_id}?kb=${kbId || article?.kb_id || ''}`);
        } else {
          const targetKb = kbId || article?.kb_id || '';
          nav(`/wiki/kb/${targetKb}?tab=graph&entity=${lp.entity_id}`);
        }
      });
    },
    [linkIndex, kbId, article?.kb_id, nav]
  );

  // rewrite wikilinks into #wikilink anchors so they survive the markdown
  // pipeline; renderLink turns them back into capsules at render time
  const mainContent = useMemo(() => {
    const raw = structured?.mainContent ?? '';
    return raw.replace(/\[\[([^:\[\]]+):([^\[\]]+)]]/g, (_m, t: string, n: string) => {
      const type = String(t).trim();
      const label = String(n).trim();
      return `[${label}](#wikilink?t=${encodeURIComponent(type)}&n=${encodeURIComponent(label)})`;
    });
  }, [structured?.mainContent]);

  const renderWikiLink = useCallback(
    (href: string) => {
      if (!href.startsWith('#wikilink?')) return undefined;
      const params = new URLSearchParams(href.slice('#wikilink?'.length));
      const type = params.get('t') || '';
      const label = params.get('n') || '';
      return (
        <WikiLinkTag type={type} label={label} resolved={resolveWikiLink(type, label)} onClick={() => openWikiLink(type, label)} />
      );
    },
    [resolveWikiLink, openWikiLink]
  );

  const currentDiff = useMemo(
    () => (diffPair ? diffLines(diffPair.older.content, diffPair.newer.content) : []),
    [diffPair]
  );

  const openVersions = () => {
    if (!id) return;
    getWikiArticleVersions(id).then(res => setVersions(((res as any) || []) as Api.Wiki.Version[]));
    setVersionsOpen(true);
  };

  const startEdit = () => {
    if (!article) return;
    setDraft({ title: article.title, summary: article.summary, tags: article.tags, content: article.content });
    setEditing(true);
    setPreview(false);
  };

  const save = () => {
    if (!id || !draft.title) return;
    setSaving(true);
    updateWikiArticle(id, draft).then(() => {
      setSaving(false);
      setEditing(false);
      window.$message?.success(t('common.updateSuccess'));
      fetchArticle();
    });
  };

  const transitionStatus = (to: string) => {
    if (!id) return;
    // status is protected on PUT /wiki/article/:id — transitions go
    // through the dedicated status-machine endpoint
    updateWikiArticleStatus(id, to as Api.Wiki.ArticleStatus).then(() => {
      window.$message?.success(t('common.updateSuccess'));
      fetchArticle();
    });
  };

  if (!loading && !article) {
    return <Empty description={t('page.wiki.article.notFound')} />;
  }

  const backTo = () => nav(kbId ? `/wiki/kb/${kbId}` : '/wiki/list');

  return (
    <WikiShell kbId={kbId} articleId={id}>
    <div className='wiki-article-page min-h-500px'>
      <Card
        bordered={false}
        className='card-wrapper'
        title={
            <div className='flex items-center gap-3'>
              <Button icon={<ArrowLeftOutlined />} onClick={backTo} />
              {loading ? null : article?.title}
            </div>
        }
        extra={
          !editing &&
          article && (
            <Space>
              {article.status && <Tag color={STATUS_COLOR[article.status]}>{t(`page.wiki.status.${article.status}`)}</Tag>}
              <Tooltip title={likeId ? t('page.wiki.like.remove') : t('page.wiki.like.add')}>
                <Button icon={likeId ? <LikeFilled style={{ color: '#1990ff' }} /> : <LikeOutlined />} onClick={toggleLike}>
                  {likeCount > 0 ? likeCount : ''}
                </Button>
              </Tooltip>
              <Tooltip title={bookmarkId ? t('page.wiki.bookmark.remove') : t('page.wiki.bookmark.add')}>
                <Button
                  icon={bookmarkId ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />}
                  onClick={toggleBookmark}
                />
              </Tooltip>
              {canEditArticle &&
                (NEXT_STATUS[article.status] || []).map(({ to, labelKey }) => (
                  <Tooltip key={to} title={t('page.wiki.article.statusFlowHint')}>
                    <Button icon={<SendOutlined />} onClick={() => transitionStatus(to)}>
                      {t(labelKey)}
                    </Button>
                  </Tooltip>
                ))}
              <Button icon={<HistoryOutlined />} onClick={openVersions}>
                {t('page.wiki.article.versions')}
              </Button>
              <Tooltip title={t('page.wiki.article.copyLink')}>
                <Button
                  icon={<CopyOutlined />}
                  onClick={() => {
                    navigator.clipboard?.writeText(window.location.href);
                    window.$message?.success(t('page.wiki.article.linkCopied'));
                  }}
                />
              </Tooltip>
              <Tooltip title={t('page.wiki.article.exportMd')}>
                <Button icon={<DownloadOutlined />} onClick={onExportMarkdown} />
              </Tooltip>
              {canEditArticle && (
                <Button icon={<RobotOutlined />} onClick={() => setAiEditOpen(true)}>
                  {t('page.wiki.aiEdit.title')}
                </Button>
              )}
{canEditArticle && (
              <Button icon={<EditOutlined />} type="primary" onClick={startEdit}>
                {t('page.wiki.article.edit')}
              </Button>
              )}
            </Space>
          )
        }
      >
        <Spin spinning={loading}>
          {!article || editing || unresolvedLinks.length === 0 ? null : (
            <div className='mb-3 flex items-center justify-between rounded-6px border border-dashed border-gray-300 px-12px py-8px dark:border-gray-600'>
              <span className='text-13px text-gray-500'>
                {t('page.wiki.repair.hint', { count: String(unresolvedLinks.length) })}
                {unresolvedLinks.slice(0, 5).map(lp => (
                  <Tag className='ml-2' key={lp.name}>
                    {lp.type ? `${lp.type}:` : ''}
                    {lp.name}
                  </Tag>
                ))}
              </span>
              <Button loading={repairing} onClick={() => setRepairOpen(true)} size='small' type='primary' ghost>
                {t('page.wiki.repair.action')}
              </Button>
            </div>
          )}
          {!article ? null : editing ? (
            <div className='flex flex-col gap-4'>
              <Input
                placeholder={t('page.wiki.createArticle.title')}
                size='large'
                value={draft.title}
                onChange={e => setDraft(d => ({ ...d, title: e.target.value }))}
              />
              <Input.TextArea
                placeholder={t('page.wiki.createArticle.summary')}
                rows={2}
                value={draft.summary}
                onChange={e => setDraft(d => ({ ...d, summary: e.target.value }))}
              />
              <Select
                mode='tags'
                open={false}
                placeholder={t('page.wiki.kb.columns.tags')}
                prefix={<TagsOutlined className='mr-1 text-gray-400' />}
                style={{ width: '100%' }}
                suffixIcon={null}
                tokenSeparators={[' ', ',']}
                value={draft.tags || []}
                onChange={tags => setDraft(d => ({ ...d, tags }))}
              />
              <div className='flex flex-wrap items-center justify-between gap-8px'>
                <div className='flex items-center gap-4px'>
                  <span className='font-medium'>{t('page.wiki.article.contentEditor')}</span>
                  <Tooltip title={t('page.wiki.editor.heading')}>
                    <Button icon={<LineOutlined />} onClick={() => insertAround('\n## ', '', t('page.wiki.editor.sectionTitle'))} size='small' type='text' />
                  </Tooltip>
                  <Tooltip title={t('page.wiki.editor.bold')}>
                    <Button icon={<BoldOutlined />} onClick={() => insertAround('**', '**', t('page.wiki.editor.text'))} size='small' type='text' />
                  </Tooltip>
                  <Tooltip title={t('page.wiki.editor.list')}>
                    <Button icon={<UnorderedListOutlined />} onClick={() => insertAround('\n- ', '', t('page.wiki.editor.item'))} size='small' type='text' />
                  </Tooltip>
                  <Tooltip title={t('page.wiki.editor.link')}>
                    <Button icon={<LinkOutlined />} onClick={() => insertAround('[', '](url)', t('page.wiki.editor.linkText'))} size='small' type='text' />
                  </Tooltip>
                  <Tooltip title={t('page.wiki.editor.wikilink')}>
                    <Button
                      icon={<ClusterOutlined />}
                      onClick={() => insertAround('[[', ']]', 'type:name')}
                      size='small'
                      type='text'
                    />
                  </Tooltip>
                </div>
                <Button
                  icon={<EyeOutlined />}
                  size='small'
                  type={preview ? 'primary' : 'default'}
                  onClick={() => setPreview(p => !p)}
                >
                  {t('page.wiki.article.preview')}
                </Button>
              </div>
              <div className={preview ? 'flex flex-col gap-8px xl:!flex-row' : ''}>
                <Input.TextArea
                  autoSize={{ minRows: 18, maxRows: 36 }}
                  className={`font-mono ${preview ? 'min-w-0 flex-1' : ''}`}
                  ref={contentRef}
                  value={draft.content}
                  onChange={e => setDraft(d => ({ ...d, content: e.target.value }))}
                />
                {preview && (
                  <div
                    className='min-h-300px min-w-0 flex-1 overflow-auto rounded border border-solid p-4'
                    style={{ borderColor: 'var(--ant-color-border)' }}
                  >
                    <Markdown content={draft.content || ''} />
                  </div>
                )}
              </div>
              <Space>
                <Button icon={<CheckOutlined />} loading={saving} type='primary' onClick={save}>
                  {t('common.confirm')}
                </Button>
                <Button
                  onClick={() => {
                    setEditing(false);
                  }}
                >
                  {t('common.close')}
                </Button>
              </Space>
            </div>
          ) : (
            <div className='flex flex-row gap-4'>
            <div className='wiki-article-body mx-auto min-w-0 max-w-860px flex-1'>
              <div className='mb-4 flex flex-wrap items-center gap-2 text-xs text-gray-400'>
                {article.page_type && <Tag>{t(`page.wiki.pageType.${article.page_type}`)}</Tag>}
                {article.ai_generated && (
                  <Tag color='purple'>
                    {`AI · ${t(`page.wiki.confidence.${article.confidence || 'medium'}`)}`}
                  </Tag>
                )}
                {article.aliases?.map(alias => (
                  <Tag key={alias}>{alias}</Tag>
                ))}
                <span>{`${t('page.wiki.hub.lastUpdated')} ${article.updated_at}`}</span>
                <span className='ml-auto flex items-center gap-1'>
                  {article.contributors.map(c => (
                    <Avatar key={c.id} size='small'>
                      {c.avatar}
                    </Avatar>
                  ))}
                </span>
              </div>

              {structured && (
                <>
                  {structured.definition && (
                    <Section title={t('page.wiki.article.sections.definition')}>
                      <Typography.Paragraph>{renderInlineWikiLinks(structured.definition, resolveWikiLink, openWikiLink)}</Typography.Paragraph>
                    </Section>
                  )}
                  {structured.keyCharacteristics.length > 0 && (
                    <Section title={t('page.wiki.article.sections.characteristics')}>
                      <ul className='ml-5 list-disc'>
                        {structured.keyCharacteristics.map(item => (
                          <li key={item}>{renderInlineWikiLinks(item, resolveWikiLink, openWikiLink)}</li>
                        ))}
                      </ul>
                    </Section>
                  )}
                  {structured.applications.length > 0 && (
                    <Section title={t('page.wiki.article.sections.applications')}>
                      <ul className='ml-5 list-disc'>
                        {structured.applications.map(item => (
                          <li key={item}>{renderInlineWikiLinks(item, resolveWikiLink, openWikiLink)}</li>
                        ))}
                      </ul>
                    </Section>
                  )}
                  {structured.relatedConcepts.length > 0 && (
                    <Section title={t('page.wiki.article.sections.relatedConcepts')}>
                      <RelatedList color='geekblue' items={structured.relatedConcepts} resolve={resolveWikiLink} onOpen={openWikiLink} />
                    </Section>
                  )}
                  {structured.relatedEntities.length > 0 && (
                    <Section title={t('page.wiki.article.sections.relatedEntities')}>
                      <RelatedList color='cyan' items={structured.relatedEntities} resolve={resolveWikiLink} onOpen={openWikiLink} />
                    </Section>
                  )}
                  {structured.mentions.length > 0 && (
                    <Section title={t('page.wiki.article.sections.mentions')}>
                      <div className='flex flex-col gap-2'>
                        {structured.mentions.map(mention => (
                          <blockquote
                            className='ma-0 border-l-4 border-solid pl-3 text-gray-500'
                            key={mention}
                            style={{ borderColor: 'var(--ant-color-border)' }}
                          >
                            {renderInlineWikiLinks(mention, resolveWikiLink, openWikiLink)}
                          </blockquote>
                        ))}
                      </div>
                    </Section>
                  )}
                  {structured.mainContent.trim() && (
                    <>
                      <Divider />
                      <Markdown content={mainContent} renderLink={renderWikiLink} />
                    </>
                  )}
                  {article.sources.length > 0 && (
                    <>
                      <Divider />
                      <Section title={t('page.wiki.article.sections.sources')}>
                        <List
                          dataSource={article.sources}
                          renderItem={src => (
                            <List.Item
                              actions={[
                                src.url ? (
                                  <a href={src.url} key='url' rel='noreferrer' target='_blank'>
                                    {t('page.wiki.article.openSource')}
                                  </a>
                                ) : null
                              ]}
                            >
                              <List.Item.Meta
                                description={
                                  <div>
                                    <div className='text-gray-500'>{src.excerpt}</div>
                                    <Space className='mt-1' size={4}>
                                      <Tag>{src.source_name}</Tag>
                                      {src.locator && <span className='text-xs text-gray-400'>{src.locator}</span>}
                                    </Space>
                                  </div>
                                }
                                title={`${src.title} · doc_id:${src.doc_id}`}
                              />
                            </List.Item>
                          )}
                        />
                      </Section>
                    </>
                  )}
                </>
              )}
              <Divider />
              <ArticleComments
                articleId={article.id}
                currentUserId={userInfo?.id || ''}
                currentUserName={userInfo?.name || ''}
              />
            </div>
            <ArticleOutline content={article.content} />
            </div>
          )}
        </Spin>
      </Card>

      <Modal
        cancelText={t('common.cancel')}
        okText={t('page.wiki.repair.confirm')}
        confirmLoading={repairing}
        onCancel={() => setRepairOpen(false)}
        onOk={runRepair}
        open={repairOpen}
        title={t('page.wiki.repair.title')}
      >
        <p className='mb-2 text-gray-500'>{t('page.wiki.repair.description')}</p>
        {unresolvedLinks.map(lp => (
          <Tag key={`${lp.type}:${lp.name}`}>
            {lp.type ? `${lp.type}:` : ''}
            {lp.name}
          </Tag>
        ))}
      </Modal>

      <Drawer
        open={versionsOpen}
        title={t('page.wiki.article.versions')}
        onClose={() => setVersionsOpen(false)}
        width={480}
      >
        <List
          dataSource={versions}
          renderItem={(v, idx) => (
            <List.Item
              actions={[
                <Button
                  key='view'
                  size='small'
                  type='link'
                  onClick={() => {
                    setVersionView(v);
                  }}
                >
                  {t('page.wiki.article.versionView')}
                </Button>,
                idx < versions.length - 1 ? (
                  <Button
                    key='diff'
                    size='small'
                    type='link'
                    onClick={() => {
                      setDiffPair({ older: versions[idx + 1], newer: v });
                    }}
                  >
                    {t('page.wiki.diff.view')}
                  </Button>
                ) : null
              ]}
            >
              <List.Item.Meta
                description={
                  <Space wrap>
                    <span className='text-xs text-gray-400'>{v.created_at}</span>
                    <span className='text-xs text-gray-400'>{v.created_by}</span>
                  </Space>
                }
                title={
                  <Space>
                    <Tag color={CHANGE_TYPE_COLOR[v.change_type]}>{t(`page.wiki.changeType.${v.change_type}`)}</Tag>
                    <span>{`v${v.version}`}</span>
                  </Space>
                }
              />
              {v.change_summary && <div className='text-xs text-gray-500'>{v.change_summary}</div>}
            </List.Item>
          )}
        />
      </Drawer>

      <Modal
        footer={null}
        open={!!versionView}
        title={versionView ? `v${versionView.version} · ${t(`page.wiki.changeType.${versionView.change_type}`)}` : ''}
        width={720}
        onCancel={() => setVersionView(null)}
      >
        {versionView && <Markdown content={versionView.content} />}
      </Modal>

      <Modal
        footer={null}
        open={!!diffPair}
        title={
          diffPair
            ? t('page.wiki.diff.vsPrev', { older: diffPair.older.version, newer: diffPair.newer.version })
            : ''
        }
        width={720}
        onCancel={() => setDiffPair(null)}
      >
        {diffPair && (
          <div className='font-mono text-xs'>
            <div className='mb-2 flex gap-4 text-gray-400'>
              <span>
                <span className='mr-1 inline-block h-10px w-10px bg-[var(--ant-color-success)]' />
                {t('page.wiki.diff.added', { count: currentDiff.filter(l => l.type === 'add').length })}
              </span>
              <span>
                <span className='mr-1 inline-block h-10px w-10px bg-[var(--ant-color-error)]' />
                {t('page.wiki.diff.removed', { count: currentDiff.filter(l => l.type === 'del').length })}
              </span>
            </div>
            <div className='max-h-480px overflow-auto rounded border border-solid' style={{ borderColor: 'var(--ant-color-border)' }}>
              {currentDiff.map((line, i) => (
                <div
                  className='whitespace-pre-wrap px-2'
                  key={i}
                  style={{
                    background:
                      line.type === 'add'
                        ? 'var(--ant-color-success-bg)'
                        : line.type === 'del'
                          ? 'var(--ant-color-error-bg)'
                          : undefined,
                    color:
                      line.type === 'add'
                        ? 'var(--ant-color-success)'
                        : line.type === 'del'
                          ? 'var(--ant-color-error)'
                          : undefined
                  }}
                >
                  {line.type === 'add' ? '+ ' : line.type === 'del' ? '- ' : '  '}
                  {line.text}
                </div>
              ))}
            </div>
          </div>
        )}
      </Modal>

      <AIEditModal
        articleId={id || ''}
        open={aiEditOpen}
        onClose={() => {
          setAiEditOpen(false);
        }}
        onApplied={fetchArticle}
      />
    </div>
    </WikiShell>
  );
}
