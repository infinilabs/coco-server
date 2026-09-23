import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Tooltip, Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getWikiToc, searchWikiKbs, updateWikiToc } from '@/service/api';
import { dropPositionFromAntd, findNode, moveNode, toAntdTreeData, tocIdForArticle } from '../shared/toc';
import { CreateKbModal } from './CreateKbModal';

/**
 * Persistent left navigation for the wiki app: knowledge bases as roots with their TOC
 * (articles/folders) lazily loaded underneath. State lives at module level so the tree —
 * expansion, loaded TOCs — survives navigation between the wiki pages.
 */
const CACHE_TTL = 5000;

const wikiNav = {
  kbs: [] as Api.Wiki.Kb[],
  kbsAt: 0,
  tocs: {} as Record<string, Api.Wiki.TocNode[]>,
  tocsAt: {} as Record<string, number>,
  expandedKeys: [] as React.Key[]
};

/** kb:<id> key for the KB that contains a toc node id */
function kbIdOfTocNode(key: string): string | null {
  for (const [kbId, toc] of Object.entries(wikiNav.tocs)) {
    if (findNode(toc, key)) return kbId;
  }
  return null;
}

/** ancestor chain (excluding the node itself) for expanding down to a node */
function pathToNode(nodes: Api.Wiki.TocNode[], id: string): string[] | null {
  for (const node of nodes) {
    if (node.id === id) return [];
    if (node.children) {
      const path = pathToNode(node.children, id);
      if (path) return [node.id, ...path];
    }
  }
  return null;
}

interface Props {
  /** currently open KB (kb page / article's ?kb=) for selection highlight */
  kbId?: string;
  /** currently open article id, highlighted via its toc node when loaded */
  articleId?: string;
  children: React.ReactNode;
}

export function WikiShell({ kbId, articleId, children }: Props) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [kbs, setKbs] = useState(wikiNav.kbs);
  const [tocs, setTocs] = useState(wikiNav.tocs);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(wikiNav.expandedKeys);
  const [createOpen, setCreateOpen] = useState(false);

  const persist = (next: Partial<typeof wikiNav>) => {
    Object.assign(wikiNav, next);
  };

  const fetchKbs = (force = false) => {
    if (!force && Date.now() - wikiNav.kbsAt < CACHE_TTL && wikiNav.kbs.length) return;
    searchWikiKbs().then(res => {
      const list = ((res as any).data || []) as Api.Wiki.Kb[];
      wikiNav.kbs = list;
      wikiNav.kbsAt = Date.now();
      setKbs(list);
    });
  };

  const fetchToc = (id: string) => {
    if (Date.now() - (wikiNav.tocsAt[id] || 0) < CACHE_TTL && wikiNav.tocs[id]) return;
    getWikiToc(id).then(res => {
      const toc = ((res as any) || []) as Api.Wiki.TocNode[];
      wikiNav.tocs = { ...wikiNav.tocs, [id]: toc };
      wikiNav.tocsAt = { ...wikiNav.tocsAt, [id]: Date.now() };
      setTocs(wikiNav.tocs);
    });
  };

  useEffect(() => {
    fetchKbs();
    if (kbId) fetchToc(kbId);
  }, [kbId]);

  // expand + highlight down to the open article once its KB toc is loaded
  useEffect(() => {
    if (!articleId || !kbId || !tocs[kbId]) return;
    const tocId = tocIdForArticle(tocs[kbId], articleId);
    if (!tocId) return;
    const path = pathToNode(tocs[kbId], tocId) || [];
    const keys = [`kb:${kbId}`, ...path];
    const merged = Array.from(new Set([...wikiNav.expandedKeys, ...keys]));
    wikiNav.expandedKeys = merged;
    setExpandedKeys(merged);
  }, [articleId, kbId, tocs]);

  const treeData: DataNode[] = useMemo(
    () =>
      kbs.map(kb => ({
        key: `kb:${kb.id}`,
        title: `${kb.icon || '📚'} ${kb.name}`,
        children: tocs[kb.id] ? toAntdTreeData(tocs[kb.id]) : undefined
      })),
    [kbs, tocs]
  );

  const selectedKeys = useMemo(() => {
    if (articleId && kbId && tocs[kbId]) {
      const tocId = tocIdForArticle(tocs[kbId], articleId);
      if (tocId) return [tocId];
    }
    return kbId ? [`kb:${kbId}`] : [];
  }, [kbId, articleId, tocs]);

  const onSelect = (keys: React.Key[]) => {
    const key = String(keys[0] || '');
    if (!key) return;
    if (key.startsWith('kb:')) {
      nav(`/wiki/kb/${key.slice(3)}`);
      return;
    }
    const kbOwnerId = kbIdOfTocNode(key);
    const node = kbOwnerId ? findNode(tocs[kbOwnerId], key) : null;
    if (node?.type === 'article' && node.article_id) {
      nav(`/wiki/article/${node.article_id}?kb=${kbOwnerId}`);
    }
  };

  const onLoadData = (node: { key: React.Key }): Promise<void> | undefined => {
    const key = String(node.key);
    if (!key.startsWith('kb:')) return undefined;
    const id = key.slice(3);
    if (tocs[id]) return undefined;
    return getWikiToc(id).then(res => {
      const toc = ((res as any) || []) as Api.Wiki.TocNode[];
      wikiNav.tocs = { ...wikiNav.tocs, [id]: toc };
      wikiNav.tocsAt = { ...wikiNav.tocsAt, [id]: Date.now() };
      setTocs(wikiNav.tocs);
    });
  };

  const onExpand = (keys: React.Key[]) => {
    wikiNav.expandedKeys = keys;
    setExpandedKeys(keys);
  };

  // reorder within one KB only — a drag across KB roots is ignored
  const onTreeDrop = (info: { node: DataNode; dragNode: DataNode; dropToGap: boolean; dropPosition: number }) => {
    const draggedId = String(info.dragNode.key);
    const targetId = String(info.node.key);
    if (draggedId.startsWith('kb:') || targetId.startsWith('kb:')) return;
    const dragKb = kbIdOfTocNode(draggedId);
    const targetKb = kbIdOfTocNode(targetId);
    if (!dragKb || dragKb !== targetKb) return;
    const toc = tocs[dragKb] || [];
    const target = findNode(toc, targetId);
    const position = dropPositionFromAntd(info.dropToGap, info.dropPosition, target?.type === 'folder');
    const next = moveNode(toc, draggedId, targetId, position);
    wikiNav.tocs = { ...wikiNav.tocs, [dragKb]: next };
    wikiNav.tocsAt = { ...wikiNav.tocsAt, [dragKb]: Date.now() };
    setTocs(wikiNav.tocs);
    updateWikiToc(dragKb, next);
  };

  const onRefresh = () => {
    wikiNav.tocs = {};
    wikiNav.tocsAt = {};
    setTocs({});
    fetchKbs(true);
    if (kbId) {
      getWikiToc(kbId).then(res => {
        const toc = ((res as any) || []) as Api.Wiki.TocNode[];
        wikiNav.tocs = { ...wikiNav.tocs, [kbId]: toc };
        wikiNav.tocsAt = { ...wikiNav.tocsAt, [kbId]: Date.now() };
        setTocs(wikiNav.tocs);
      });
    }
  };

  return (
    <div className='flex h-full min-h-0 flex-col gap-12px p-12px lg:flex-row'>
      <Card
        className='w-full shrink-0 lg:w-264px'
        size='small'
        title={t('route.wiki')}
        extra={
          <div className='flex items-center gap-4px'>
            <Tooltip title={t('common.refresh')}>
              <Button icon={<ReloadOutlined />} onClick={onRefresh} size='small' type='text' />
            </Tooltip>
            <Tooltip title={t('page.wiki.hub.newKb')}>
              <Button icon={<PlusOutlined />} onClick={() => setCreateOpen(true)} size='small' type='text' />
            </Tooltip>
          </div>
        }
        styles={{ body: { maxHeight: 'calc(100vh - 140px)', overflow: 'auto', minHeight: 120 } }}
      >
        {kbs.length === 0 ? (
          <Empty description={t('page.wiki.hub.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Tree
            blockNode
            draggable
            expandedKeys={expandedKeys}
            loadData={onLoadData as any}
            selectedKeys={selectedKeys}
            treeData={treeData}
            onDrop={onTreeDrop as any}
            onExpand={onExpand}
            onSelect={onSelect}
          />
        )}
      </Card>
      <div className='min-w-0 flex-1 overflow-auto'>{children}</div>
    </div>
  );
}
