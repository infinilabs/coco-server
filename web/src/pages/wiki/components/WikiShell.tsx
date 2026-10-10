import {
  BellOutlined,
  BookOutlined,
  DeleteOutlined,
  EditOutlined,
  FolderAddOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined
} from '@ant-design/icons';
import { Badge, Button, ConfigProvider, Dropdown, Empty, Input, Modal, Tooltip, Tree } from 'antd';
import type { DataNode } from 'antd/es/tree';
import { useEffect, useMemo, useState } from 'react';
import { useAuth } from '@/hooks/business/auth';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { getWikiToc, searchWikiGovernance, searchWikiKbs, searchWikiNotifications, updateWikiToc } from '@/service/api';
import { dropPositionFromAntd, findNode, moveNode, removeNode, toAntdTreeData, tocIdForArticle } from '../shared/toc';
import { CreateKbModal } from './CreateKbModal';
import { NotificationDrawer } from './NotificationDrawer';

/**
 * Persistent left navigation for the wiki app: knowledge bases as roots with their TOC (articles/folders) lazily loaded
 * underneath. State lives at module level so the tree — expansion, loaded TOCs — survives navigation between the wiki
 * pages.
 */
const CACHE_TTL = 5000;

const wikiNav = {
  kbs: [] as Api.Wiki.Kb[],
  kbsAt: 0,
  tocs: {} as Record<string, Api.Wiki.TocNode[]>,
  tocsAt: {} as Record<string, number>,
  /** kbId → (articleId → page_type), served alongside the toc for tree icons */
  pageTypes: {} as Record<string, Record<string, string>>,
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

/** insert a node next to (or inside) a reference node */
function insertBeside(
  nodes: Api.Wiki.TocNode[],
  refId: string | null,
  position: 'after' | 'before' | 'inside',
  newNode: Api.Wiki.TocNode
): Api.Wiki.TocNode[] {
  if (!refId) return [...nodes, newNode];
  const result: Api.Wiki.TocNode[] = [];
  for (const node of nodes) {
    if (node.id === refId) {
      if (position === 'before') result.push(newNode, node);
      else if (position === 'after') result.push(node, newNode);
      else if (node.type === 'folder') result.push({ ...node, children: [...(node.children || []), newNode] });
      else result.push(node);
    } else if (node.children) {
      result.push({ ...node, children: insertBeside(node.children, refId, position, newNode) });
    } else {
      result.push(node);
    }
  }
  return result;
}

interface Props {
  /** currently open KB (kb page / article's ?kb=) for selection highlight */
  readonly kbId?: string;
  /** currently open article id, highlighted via its toc node when loaded */
  readonly articleId?: string;
  readonly children: React.ReactNode;
}

export function WikiShell({ kbId, articleId, children }: Props) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [kbs, setKbs] = useState(wikiNav.kbs);
  const [tocs, setTocs] = useState(wikiNav.tocs);
  const [pageTypes, setPageTypes] = useState(wikiNav.pageTypes);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(wikiNav.expandedKeys);
  const [createOpen, setCreateOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const [unread, setUnread] = useState(0);
  const [openGovernance, setOpenGovernance] = useState(0);
  const [renaming, setRenaming] = useState<{ key: string; value: string } | null>(null);
  const { hasAuth } = useAuth();
  const canCreateKb = hasAuth('coco#wiki_kb/create');
  const canUpdateKb = hasAuth('coco#wiki_kb/update');

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

  const storeToc = (id: string, nodes: Api.Wiki.TocNode[], types: Record<string, string>) => {
    wikiNav.tocs = { ...wikiNav.tocs, [id]: nodes };
    wikiNav.pageTypes = { ...wikiNav.pageTypes, [id]: types };
    wikiNav.tocsAt = { ...wikiNav.tocsAt, [id]: Date.now() };
    setTocs(wikiNav.tocs);
    setPageTypes(wikiNav.pageTypes);
  };

  const fetchToc = (id: string) => {
    if (Date.now() - (wikiNav.tocsAt[id] || 0) < CACHE_TTL && wikiNav.tocs[id]) return;
    getWikiToc(id).then(({ nodes, pageTypes: types }) => storeToc(id, nodes, types));
  };

  const fetchUnread = () => {
    searchWikiNotifications().then(res => {
      const list = ((res as any).data || []) as Api.Wiki.Notification[];
      setUnread(list.filter(n => !n.read).length);
    });
  };

  const fetchGovernanceCount = () => {
    searchWikiGovernance({ status: 'open' }).then(res => {
      setOpenGovernance(((res as any)?.total as number) || 0);
    });
  };

  useEffect(() => {
    fetchKbs();
    fetchUnread();
    fetchGovernanceCount();
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

  /* ---------------- TOC editing (folders, rename, remove) ---------------- */

  const commitToc = (kbOwnerId: string, next: Api.Wiki.TocNode[]) => {
    wikiNav.tocs = { ...wikiNav.tocs, [kbOwnerId]: next };
    wikiNav.tocsAt = { ...wikiNav.tocsAt, [kbOwnerId]: Date.now() };
    setTocs(wikiNav.tocs);
    updateWikiToc(kbOwnerId, next);
  };

  const addFolder = (parentKey: string | null) => {
    if (!kbId || !tocs[kbId]) return;
    const folder: Api.Wiki.TocNode = {
      id: `folder-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`,
      title: t('page.wiki.tree.newFolder'),
      type: 'folder',
      children: []
    };
    const next = insertBeside(tocs[kbId], parentKey, parentKey ? 'inside' : 'after', folder);
    commitToc(kbId, next);
    setExpandedKeys(prev => {
      const merged = Array.from(new Set([...prev, `kb:${kbId}`, ...(parentKey ? [parentKey] : []), folder.id]));
      wikiNav.expandedKeys = merged;
      return merged;
    });
    setRenaming({ key: folder.id, value: folder.title });
  };

  const renameNode = () => {
    if (!renaming) return;
    const { key, value } = renaming;
    const owner = kbIdOfTocNode(key) || kbId;
    if (!owner || !wikiNav.tocs[owner]) return;
    const apply = (nodes: Api.Wiki.TocNode[]): Api.Wiki.TocNode[] =>
      nodes.map(n =>
        n.id === key ? { ...n, title: value || n.title } : n.children ? { ...n, children: apply(n.children) } : n
      );
    commitToc(owner, apply(wikiNav.tocs[owner]));
    setRenaming(null);
  };

  const deleteTocNode = (key: string) => {
    const kbOwnerId = kbIdOfTocNode(key);
    if (!kbOwnerId) return;
    const [next, removed] = removeNode(wikiNav.tocs[kbOwnerId], key);
    if (!removed) return;
    commitToc(kbOwnerId, next);
  };

  const nodeMenu = (key: string, isFolder: boolean): any => ({
    items: [
      { key: 'rename', icon: <EditOutlined />, label: t('page.wiki.tree.rename') },
      isFolder && { key: 'add', icon: <FolderAddOutlined />, label: t('page.wiki.tree.addSubfolder') },
      { key: 'delete', danger: true, icon: <DeleteOutlined />, label: t('page.wiki.tree.delete') }
    ].filter(Boolean),
    onClick: ({ key: action }: any) => {
      if (action === 'rename') {
        const owner = kbIdOfTocNode(key);
        const node = owner ? findNode(wikiNav.tocs[owner], key) : null;
        if (node) setRenaming({ key, value: node.title });
      } else if (action === 'add') {
        addFolder(key);
      } else if (action === 'delete') {
        deleteTocNode(key);
      }
    }
  });

  const treeData: DataNode[] = useMemo(
    () =>
      kbs.map(kb => ({
        key: `kb:${kb.id}`,
        title: kb.name,
        // marks the KB root row so the stylesheet can read it as a section header
        className: 'wiki-toc-kb',
        // custom kb emoji lives in the icon slot next to the folder/article icons
        icon: kb.icon ? <span className='wiki-toc-kb-icon'>{kb.icon}</span> : <BookOutlined />,
        children: tocs[kb.id] ? toAntdTreeData(tocs[kb.id], pageTypes[kb.id]) : undefined
      })),
    [kbs, tocs, pageTypes]
  );

  const titleRender = (node: DataNode) => {
    const key = String(node.key);
    const title = String(node.title ?? '');
    const owner = key.startsWith('kb:') ? null : kbIdOfTocNode(key);
    const nodeObj = owner ? findNode(wikiNav.tocs[owner], key) : null;
    if (!nodeObj)
      return (
        <span className='wiki-toc-title'>
          {key.startsWith('kb:') ? title : <span className='truncate'>{title}</span>}
        </span>
      );
    // actions via right-click on the row or left-click on the "..." affordance —
    // a plain left-click on the row must just select/navigate
    const menu = nodeMenu(key, nodeObj.type === 'folder');
    return (
      <Dropdown
        menu={menu}
        trigger={['contextMenu']}
      >
        <span className='min-w-0 flex flex-1 items-center justify-between gap-4px'>
          <span className='wiki-toc-title'>{title}</span>
          <Dropdown
            menu={menu}
            trigger={['click']}
          >
            <MoreOutlined
              className='text-gray-400 hover:text-gray-600'
              onClick={e => e.stopPropagation()}
            />
          </Dropdown>
        </span>
      </Dropdown>
    );
  };

  const renameDialog = renaming ? (
    <Modal
      open
      cancelText={t('common.cancel')}
      okText={t('common.confirm')}
      title={t('page.wiki.tree.rename')}
      onCancel={() => setRenaming(null)}
      onOk={renameNode}
    >
      <Input
        autoFocus
        value={renaming.value}
        onChange={e => setRenaming({ key: renaming.key, value: e.target.value })}
        onPressEnter={renameNode}
      />
    </Modal>
  ) : null;

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

  // antd Tree requires loadData to always resolve — returning undefined for loaded
  // nodes crashes the tree (`.then` of undefined) on re-expansion
  const onLoadData = (node: { key: React.Key }): Promise<void> => {
    const key = String(node.key);
    if (!key.startsWith('kb:')) return Promise.resolve();
    const id = key.slice(3);
    if (tocs[id]) return Promise.resolve();
    return getWikiToc(id).then(({ nodes, pageTypes: types }) => storeToc(id, nodes, types));
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
    commitToc(dragKb, next);
  };

  const onRefresh = () => {
    wikiNav.tocs = {};
    wikiNav.tocsAt = {};
    wikiNav.pageTypes = {};
    setTocs({});
    setPageTypes({});
    fetchKbs(true);
    fetchUnread();
    fetchGovernanceCount();
    if (kbId) fetchToc(kbId);
  };

  return (
    // themed space: the wiki keeps the app theme untouched (colors stay unified
    // with Coco AI); only the corner radius is tuned for the panel kit
    <ConfigProvider
      theme={{
        token: {
          borderRadius: 8
        }
      }}
    >
      <div className='wiki-page h-full min-h-0 flex flex-col gap-12px p-12px lg:!flex-row'>
        <aside className='wiki-sidebar'>
          <div className='wiki-sidebar-head'>
            <span className='wiki-sidebar-brand'>
              <BookOutlined />
              {t('route.wiki')}
            </span>
            <div className='wiki-sidebar-tools'>
              <Tooltip title={t('page.wiki.governance.title')}>
                <Badge
                  count={openGovernance}
                  offset={[-2, 2]}
                  size='small'
                >
                  <Button
                    icon={<SafetyCertificateOutlined />}
                    size='small'
                    type='text'
                    onClick={() => nav('/wiki/governance')}
                  />
                </Badge>
              </Tooltip>
              <Tooltip title={t('page.wiki.notification.title')}>
                <Badge
                  count={unread}
                  offset={[-2, 2]}
                  size='small'
                >
                  <Button
                    icon={<BellOutlined />}
                    size='small'
                    type='text'
                    onClick={() => setNotifOpen(true)}
                  />
                </Badge>
              </Tooltip>
              <Tooltip title={t('page.wiki.tree.addFolder')}>
                <Button
                  disabled={!kbId || !tocs[kbId] || !canUpdateKb}
                  icon={<FolderAddOutlined />}
                  size='small'
                  type='text'
                  onClick={() => addFolder(null)}
                />
              </Tooltip>
              <Tooltip title={t('common.refresh')}>
                <Button
                  icon={<ReloadOutlined />}
                  size='small'
                  type='text'
                  onClick={onRefresh}
                />
              </Tooltip>
              {canCreateKb && (
                <Tooltip title={t('page.wiki.hub.newKb')}>
                  <Button
                    icon={<PlusOutlined />}
                    size='small'
                    type='text'
                    onClick={() => setCreateOpen(true)}
                  />
                </Tooltip>
              )}
            </div>
          </div>
          <div className='wiki-sidebar-body'>
            {kbs.length === 0 ? (
              <Empty
                description={t('page.wiki.hub.empty')}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            ) : (
              <Tree
                blockNode
                draggable={{ icon: false }}
                // whole rows are the drag surface already — the default HolderOutlined
                // grip on every line only eats width
                expandedKeys={expandedKeys}
                loadData={onLoadData as any}
                selectedKeys={selectedKeys}
                titleRender={titleRender}
                treeData={treeData}
                onDrop={onTreeDrop as any}
                onExpand={onExpand}
                onSelect={onSelect}
                className="wiki-toc-tree"
                // without showIcon antd hides node icons entirely (ant-tree-icon-hide)
                showIcon
              />
            )}
          </div>
        </aside>
        <div className='min-w-0 flex-1 overflow-auto'>
          <div className='mx-auto h-full max-w-1280px w-full 2xl:max-w-1440px'>{children}</div>
        </div>
        {renameDialog}

        <CreateKbModal
          open={createOpen}
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            fetchKbs(true);
          }}
        />
        <NotificationDrawer
          open={notifOpen}
          onClose={() => {
            setNotifOpen(false);
            fetchUnread();
          }}
        />
      </div>
    </ConfigProvider>
  );
}
