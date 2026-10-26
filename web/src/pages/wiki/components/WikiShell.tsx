import {
  ApartmentOutlined,
  BellOutlined,
  BookOutlined,
  DeleteOutlined,
  EditOutlined,
  FolderAddOutlined,
  HomeOutlined,
  MoreOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
  TeamOutlined
} from '@ant-design/icons';
import { Badge, Button, ConfigProvider, Drawer, Dropdown, Empty, Input, Modal, Tooltip, Tree } from 'antd';
import { BookCheck, Check, ChevronDown, ChevronRight, createLucideIcon, Eye, EyeOff, Search } from 'lucide-react';
import type { DataNode } from 'antd/es/tree';
import { type FC, useEffect, useMemo, useRef, useState } from 'react';
import { useAuth } from '@/hooks/business/auth';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useResponsive } from 'ahooks';
import { getWikiToc, createWikiWorkspace, getWikiTocOrganizeStatus, organizeWikiToc, searchWikiGovernance, searchWikiKbs, searchWikiNotifications, searchWikiWorkspaces, updateWikiToc } from '@/service/api';
import {
  countTocMaturity,
  dropPositionFromAntd,
  findNode,
  moveNode,
  removeNode,
  toAntdTreeData,
  tocIdForArticle
} from '../shared/toc';

/**
 * Persistent shell for the wiki app, laid out like the AI-chat screen: a
 * collapsible left sider (workspace search + knowledge-base TOC tree) and a
 * slim topbar on the right whose left side carries the cross-KB entry
 * switcher (knowledge base / governance) plus the notification bell, with
 * the global "new KB" action on its right edge. State lives at module level
 * so the tree — expansion, loaded TOCs — survives navigation between the
 * wiki pages.
 */
const CACHE_TTL = 5000;

// sidebar toggle glyphs — pixel-identical copies of the chat header's
// LeftSvg/RightSvg (ui-search ChatHeaderLayout) so the wiki topbar toggle
// reads exactly like the AI-chat one: closed → chevron pointing right
// (expand), open → chevron pointing left (collapse)
const SidebarExpandSvg: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none" className={className}>
    <g>
      <path d="M6 9C6 7.34315 7.34315 6 9 6H39C40.6569 6 42 7.34315 42 9V39C42 40.6569 40.6569 42 39 42H9C7.34315 42 6 40.6569 6 39V9Z" fill="none" stroke="currentColor" strokeWidth="4" strokeLinejoin="round"></path><path d="M32 6V42" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M16 20L20 24L16 28" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M26 6H38" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path>
    </g>
  </svg>
);

const SidebarCollapseSvg: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none" className={className}>
    <g>
      <rect x="6" y="6" width="36" height="36" rx="3" fill="none" stroke="currentColor" strokeWidth="4" strokeLinejoin="round"></rect><path d="M18 6V42" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M11 6H36" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M32 20L28 24L32 28" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path>
    </g>
  </svg>
);

// refresh glyph — same lucide definition as the chat sidebar's
// RefreshIcon (ui-search icons/RefreshIcon.tsx) so both sidebars read alike
const RefreshIcon = createLucideIcon('RefreshIcon', [
  ['path', { d: 'M42 8V24', transform: 'scale(0.5)', strokeWidth: '4', key: 'arrow-top' }],
  ['path', { d: 'M6 24L6 40', transform: 'scale(0.5)', strokeWidth: '4', key: 'arrow-bottom' }],
  ['path', { d: 'M6 24C6 33.9411 14.0589 42 24 42C28.8556 42 33.2622 40.0774 36.5 36.9519', transform: 'scale(0.5)', strokeWidth: '4', key: 'arc-bottom' }],
  ['path', { d: 'M42.0007 24C42.0007 14.0589 33.9418 6 24.0007 6C18.9152 6 14.3223 8.10896 11.0488 11.5', transform: 'scale(0.5)', strokeWidth: '4', key: 'arc-top' }]
]);

const wikiNav = {
  kbs: [] as Api.Wiki.Kb[],
  kbsAt: 0,
  tocs: {} as Record<string, Api.Wiki.TocNode[]>,
  tocsAt: {} as Record<string, number>,
  /** kbId → (articleId → page_type), served alongside the toc for tree icons */
  pageTypes: {} as Record<string, Record<string, string>>,
  /** kbId → (articleId → status), parked-node rendering (dim + tag + tail sort) */
  statuses: {} as Record<string, Record<string, string>>,
  /** 侧边栏默认只展示成熟内容; 打开后草稿/已归档也以置底淡色呈现 */
  showAllToc: false,
  expandedKeys: [] as React.Key[],
  // workspace selection survives route changes: each wiki page mounts its own
  // WikiShell, so component state would reset to 'all' on every navigation
  workspaceId: 'all' as string
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

/** every node key in the tree — used to expand-all while filtering */
function collectAllKeys(nodes: DataNode[]): React.Key[] {
  const keys: React.Key[] = [];
  const walk = (list: DataNode[]) => {
    list.forEach(n => {
      keys.push(n.key);
      if (n.children) walk(n.children);
    });
  };
  walk(nodes);
  return keys;
}

/** data the shell shares with the page it hosts (hub consumes the workspace filter) */
export interface WikiShellContext {
  workspaceId: string;
  workspaces: { id: string; name: string }[];
  /** bumped whenever the shell refetches KBs — lets hosted pages re-sync */
  kbVersion: number;
  /** force a shell KB refetch (tree + kbVersion bump); hosted pages call it
      after they mutate KBs themselves (e.g. the hub's create card) */
  refreshKbs: () => void;
}

interface Props {
  /** currently open KB (kb page / article's ?kb=) for selection highlight */
  readonly kbId?: string;
  /** currently open article id, highlighted via its toc node when loaded */
  readonly articleId?: string;
  readonly children: React.ReactNode | ((ctx: WikiShellContext) => React.ReactNode);
}

// sider drag-resize bounds (px) — the width persists in localStorage so the
// reader's preferred TOC width survives reloads and navigation
const SIDER_WIDTH_KEY = 'wiki:sider-width';
const SIDER_WIDTH_DEFAULT = 260;
const SIDER_WIDTH_MIN = 200;
const SIDER_WIDTH_MAX = 480;

function loadSiderWidth(): number {
  const saved = Number(localStorage.getItem(SIDER_WIDTH_KEY));
  return saved >= SIDER_WIDTH_MIN && saved <= SIDER_WIDTH_MAX ? saved : SIDER_WIDTH_DEFAULT;
}

export function WikiShell({ kbId, articleId, children }: Props) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const location = useLocation();
  const responsive = useResponsive();
  const isMobile = !responsive.md;
  const [kbs, setKbs] = useState(wikiNav.kbs);
  const [tocs, setTocs] = useState(wikiNav.tocs);
  const [pageTypes, setPageTypes] = useState(wikiNav.pageTypes);
  const [statuses, setStatuses] = useState(wikiNav.statuses);
  const [showAllToc, setShowAllToc] = useState(wikiNav.showAllToc);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>(wikiNav.expandedKeys);
  const [unread, setUnread] = useState(0);
  const [openGovernance, setOpenGovernance] = useState(0);
  const [renaming, setRenaming] = useState<{ key: string; value: string } | null>(null);
  const [collapsed, setCollapsed] = useState(isMobile);
  // TOC sider width — drag the right-edge strip to change, double-click (or
  // arrow keys with it focused) to adjust; persisted on release
  const [siderWidth, setSiderWidth] = useState(loadSiderWidth);
  const [resizing, setResizing] = useState(false);
  const resizeSession = useRef<{ pointerId: number; left: number } | null>(null);
  // refresh icon spins for ~1s after a click, mirroring the chat sidebar
  const [isRefresh, setIsRefresh] = useState(false);
  const [treeFilter, setTreeFilter] = useState('');
  const [workspaces, setWorkspaces] = useState<{ id: string; name: string }[]>([]);
  const [workspaceId, setWorkspaceIdState] = useState<string>(wikiNav.workspaceId);
  const [wsCreateOpen, setWsCreateOpen] = useState(false);
  const [wsName, setWsName] = useState('');
  const [wsOpen, setWsOpen] = useState(false);
  const [wsFilter, setWsFilter] = useState('');
  // bumped after each shell KB refetch so hosted pages (hub card wall) re-sync
  const [kbVersion, setKbVersion] = useState(0);
  // expansion state to restore when the tree filter is cleared
  const savedExpandedKeys = useRef<React.Key[] | null>(null);
  const { hasAuth } = useAuth();
  const canUpdateKb = hasAuth('coco#wiki_kb/update');

  // keep the selection in the module store so it survives the remount each
  // wiki page navigation performs on this shell
  const setWorkspaceId = (v: string) => {
    wikiNav.workspaceId = v;
    setWorkspaceIdState(v);
  };

  /* ---------------- sider drag-resize ---------------- */

  // pointer capture keeps the drag on the strip even when the pointer moves
  // over the tree; width = pointer distance from the sider's left edge
  const onResizerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (collapsed || e.button !== 0) return;
    const sider = (e.currentTarget as HTMLDivElement).parentElement;
    if (!sider) return;
    resizeSession.current = { pointerId: e.pointerId, left: sider.getBoundingClientRect().left };
    e.currentTarget.setPointerCapture(e.pointerId);
    setResizing(true);
  };
  const onResizerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const session = resizeSession.current;
    if (!session || session.pointerId !== e.pointerId) return;
    const next = Math.min(SIDER_WIDTH_MAX, Math.max(SIDER_WIDTH_MIN, e.clientX - session.left));
    setSiderWidth(next);
  };
  const onResizerUp = (e: React.PointerEvent<HTMLDivElement>) => {
    if (resizeSession.current?.pointerId !== e.pointerId) return;
    resizeSession.current = null;
    setResizing(false);
    // read the width through the updater so the very last move (possibly
    // not yet committed when up fires) is what gets persisted
    setSiderWidth(w => {
      localStorage.setItem(SIDER_WIDTH_KEY, String(w));
      return w;
    });
  };
  // double-click on the strip restores the default width
  const onResizerReset = () => {
    if (collapsed) return;
    setSiderWidth(SIDER_WIDTH_DEFAULT);
    localStorage.setItem(SIDER_WIDTH_KEY, String(SIDER_WIDTH_DEFAULT));
  };
  // keyboard resize for the strip (focus + ←/→, 16px steps)
  const onResizerKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    const next = Math.min(
      SIDER_WIDTH_MAX,
      Math.max(SIDER_WIDTH_MIN, siderWidth + (e.key === 'ArrowRight' ? 16 : -16))
    );
    setSiderWidth(next);
    localStorage.setItem(SIDER_WIDTH_KEY, String(next));
  };

  const fetchKbs = (force = false) => {
    if (!force && Date.now() - wikiNav.kbsAt < CACHE_TTL && wikiNav.kbs.length) return;
    searchWikiKbs().then(res => {
      const list = ((res as any).data || []) as Api.Wiki.Kb[];
      wikiNav.kbs = list;
      wikiNav.kbsAt = Date.now();
      setKbs(list);
      setKbVersion(v => v + 1);
    });
  };

  const storeToc = (id: string, nodes: Api.Wiki.TocNode[], types: Record<string, string>, stats: Record<string, string>) => {
    // final guard: an empty result must never blank a tree already on screen
    if ((!nodes || nodes.length === 0) && (wikiNav.tocs[id] || []).length > 0) return;
    wikiNav.tocs = { ...wikiNav.tocs, [id]: nodes };
    wikiNav.pageTypes = { ...wikiNav.pageTypes, [id]: types };
    wikiNav.statuses = { ...wikiNav.statuses, [id]: stats };
    wikiNav.tocsAt = { ...wikiNav.tocsAt, [id]: Date.now() };
    setTocs(wikiNav.tocs);
    setPageTypes(wikiNav.pageTypes);
    setStatuses(wikiNav.statuses);
  };

  const fetchToc = (id: string) => {
    if (Date.now() - (wikiNav.tocsAt[id] || 0) < CACHE_TTL && wikiNav.tocs[id]) return;
    getWikiToc(id).then(({ nodes, pageTypes: types, statuses: stats }) => {
      if (nodes == null) return; // transient failure — keep the tree on screen
      storeToc(id, nodes, types, stats ?? {});
    });
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

  const fetchWorkspaces = () => {
    searchWikiWorkspaces().then(res => setWorkspaces((res as any) || []));
  };

  const onCreateWorkspace = () => {
    if (!wsName.trim()) return;
    createWikiWorkspace(wsName.trim()).then(res => {
      if ((res as any)?._id) {
        setWsName('');
        setWsCreateOpen(false);
        fetchWorkspaces();
      }
    });
  };

  useEffect(() => {
    fetchKbs();
    fetchUnread();
    fetchGovernanceCount();
    fetchWorkspaces();
    if (kbId) fetchToc(kbId);
  }, [kbId]);

  // expand + highlight down to the open article once its KB toc is loaded;
  // on a KB page (no article) still open the KB root so the freshly loaded
  // toc is visible — otherwise the tree reads as "collapsed" on entry
  useEffect(() => {
    if (!kbId || !tocs[kbId]) return;
    if (articleId) {
      const tocId = tocIdForArticle(tocs[kbId], articleId);
      if (!tocId) return;
      const path = pathToNode(tocs[kbId], tocId) || [];
      const merged = Array.from(new Set([...wikiNav.expandedKeys, `kb:${kbId}`, ...path]));
      wikiNav.expandedKeys = merged;
      setExpandedKeys(merged);
      return;
    }
    const merged = Array.from(new Set([...wikiNav.expandedKeys, `kb:${kbId}`]));
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

  const addFolder = (parentKey: string | null, targetKb?: string) => {
    const kb = targetKb || kbId;
    if (!kb || !tocs[kb]) return;
    const folder: Api.Wiki.TocNode = {
      id: `folder-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`,
      title: t('page.wiki.tree.newFolder'),
      type: 'folder',
      children: []
    };
    const next = insertBeside(tocs[kb], parentKey, parentKey ? 'inside' : 'after', folder);
    commitToc(kb, next);
    setExpandedKeys(prev => {
      const merged = Array.from(new Set([...prev, `kb:${kb}`, ...(parentKey ? [parentKey] : []), folder.id]));
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

  const tocMaturity = useMemo(
    () => countTocMaturity(tocs, statuses),
    [tocs, statuses]
  );

  const treeData: DataNode[] = useMemo(
    () =>
      // the topbar workspace switcher scopes the whole app: tree roots follow
      // the same filter as the hub card wall ('all' = unfiltered)
      kbs
        .filter(kb => workspaceId === 'all' || kb.workspace_id === workspaceId)
        .map(kb => ({
          key: `kb:${kb.id}`,
          title: kb.name,
          // marks the KB root row so the stylesheet can read it as a section header
          className: 'wiki-toc-kb',
          // custom kb emoji lives in the icon slot next to the folder/article icons
          icon: kb.icon ? <span className='wiki-toc-kb-icon'>{kb.icon}</span> : <BookOutlined />,
          children: tocs[kb.id]
            ? toAntdTreeData(tocs[kb.id], pageTypes[kb.id], statuses[kb.id], { includeAll: showAllToc })
            : undefined
        })),
    [kbs, tocs, pageTypes, statuses, showAllToc, workspaceId]
  );

  // sidebar search filters the tree in place: keep self-matching rows (a
  // matching KB root keeps its whole toc) plus branches with matching
  // descendants
  const displayTreeData: DataNode[] = useMemo(() => {
    const q = treeFilter.trim().toLowerCase();
    if (!q) return treeData;
    const walk = (nodes: DataNode[]): DataNode[] =>
      nodes.reduce<DataNode[]>((acc, node) => {
        const kids = node.children ? walk(node.children) : undefined;
        const selfMatch = String(node.title ?? '')
          .toLowerCase()
          .includes(q);
        if (selfMatch || (kids && kids.length > 0)) {
          acc.push({ ...node, children: kids && kids.length > 0 ? kids : node.children });
        }
        return acc;
      }, []);
    return walk(treeData);
  }, [treeData, treeFilter]);

  // while filtering, expand everything so matches are visible; restore the
  // pre-filter expansion once the filter is cleared
  useEffect(() => {
    if (treeFilter.trim()) {
      if (savedExpandedKeys.current === null) savedExpandedKeys.current = wikiNav.expandedKeys;
      const keys = collectAllKeys(displayTreeData);
      wikiNav.expandedKeys = keys;
      setExpandedKeys(keys);
    } else if (savedExpandedKeys.current !== null) {
      wikiNav.expandedKeys = savedExpandedKeys.current;
      setExpandedKeys(savedExpandedKeys.current);
      savedExpandedKeys.current = null;
    }
  }, [treeFilter, displayTreeData]);

  const titleRender = (node: DataNode) => {
    const key = String(node.key);
    const title = String(node.title ?? '');
    const owner = key.startsWith('kb:') ? null : kbIdOfTocNode(key);
    const nodeObj = owner ? findNode(wikiNav.tocs[owner], key) : null;
    if (!nodeObj) {
      // KB root row: no toc node behind it — offer "new root folder" via
      // right-click or the "..." affordance (replaces the old sider toolbar
      // button that could only create folders in the *current* KB)
      if (key.startsWith('kb:')) {
        const kbForFolder = key.slice(3);
        const kbMenu: any = {
          items: [
            canUpdateKb && { key: 'addFolder', icon: <FolderAddOutlined />, label: t('page.wiki.tree.addFolder') }
          ].filter(Boolean),
          onClick: ({ key: action }: any) => {
            if (action === 'addFolder') addFolder(null, kbForFolder);
          }
        };
        return (
          <Dropdown
            menu={kbMenu}
            trigger={['contextMenu']}
          >
            <span className='min-w-0 flex flex-1 items-center justify-between gap-4px'>
              <span className='wiki-toc-title'>{title}</span>
              {canUpdateKb && (
                <Dropdown
                  menu={kbMenu}
                  trigger={['click']}
                >
                  <MoreOutlined
                    className='text-gray-400 hover:text-gray-600'
                    onClick={e => e.stopPropagation()}
                  />
                </Dropdown>
              )}
            </span>
          </Dropdown>
        );
      }
      return (
        <span className='wiki-toc-title'>
          <span className='truncate'>{title}</span>
        </span>
      );
    }
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
    return getWikiToc(id).then(({ nodes, pageTypes: types, statuses: stats }) => storeToc(id, nodes ?? [], types, stats ?? {}));
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
    // spin the glyph for ~1s like the chat sidebar's refresh does
    setIsRefresh(true);
    setTimeout(() => setIsRefresh(false), 1000);
    wikiNav.tocs = {};
    wikiNav.tocsAt = {};
    wikiNav.pageTypes = {};
    setTocs({});
    setPageTypes({});
    fetchKbs(true);
    fetchUnread();
    fetchGovernanceCount();
    fetchWorkspaces();
    if (kbId) fetchToc(kbId);
  };

  // AI semantic grouping of the loose root pages: the pass runs in the
  // backend background (slow model × several calls outlives any HTTP
  // timeout): poll the status endpoint until running flips false — the
  // backend state survives page reloads, so a UI opened mid-run (or after a
  // server-triggered pass) picks its spinner back up from the same place
  const [organizing, setOrganizing] = useState(false);
  const organizeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const stopOrganizePolling = () => {
    if (organizeTimer.current) {
      clearTimeout(organizeTimer.current);
      organizeTimer.current = null;
    }
  };
  const pollOrganize = (id: string, startedAt: number) => {
    stopOrganizePolling();
    const poll = async () => {
      try {
        const st = await getWikiTocOrganizeStatus(id);
        if (!st) throw new Error('empty status');
        if (!st.running) {
          setOrganizing(false);
          if (typeof st.loose === 'number') setLooseCount(st.loose);
          // a result timestamped before our trigger belongs to an earlier
          // pass — treat only fresh outcomes as this run's
          const finishedAt = st.last_at ? Date.parse(st.last_at) : 0;
          if (st.last && finishedAt >= startedAt - 60_000) {
            if (st.last_error) {
              window.$message?.error(t('page.wiki.toc.organizeFailed'));
            } else if (st.last.moved > 0) {
              onRefresh();
              window.$message?.success(t('page.wiki.toc.organizeDone', { moved: st.last.moved }));
            } else {
              window.$message?.info(t('page.wiki.toc.organizeEmpty'));
            }
          } else {
            onRefresh(); // run ended without a fresh record — resync anyway
          }
          return;
        }
      } catch {
        // transient poll failure — keep waiting
      }
      if (Date.now() - startedAt > 30 * 60 * 1000) {
        setOrganizing(false);
        window.$message?.info(t('page.wiki.toc.organizeStillRunning'));
        return;
      }
      organizeTimer.current = setTimeout(poll, 6000);
    };
    organizeTimer.current = setTimeout(poll, 6000);
  };
  const onOrganize = () => {
    if (!kbId || organizing) return;
    Modal.confirm({
      title: t('page.wiki.toc.organize'),
      content: t('page.wiki.toc.organizeHint'),
      okText: t('page.wiki.toc.organizeOk'),
      onOk: async () => {
        const startedAt = Date.now();
        setOrganizing(true);
        try {
          const res = await organizeWikiToc(kbId);
          if (!res || res.started === false) {
            if (res?.already_running) {
              window.$message?.info(t('page.wiki.toc.organizeRunning'));
              pollOrganize(kbId, startedAt);
              return;
            }
            setOrganizing(false);
            window.$message?.info(t('page.wiki.toc.organizeEmpty'));
            return;
          }
          pollOrganize(kbId, startedAt);
        } catch (e: any) {
          window.$message?.error(e?.response?.data?.error?.reason ?? e?.message ?? t('page.wiki.toc.organizeFailed'));
          setOrganizing(false);
        }
      }
    });
  };
  // a reload mid-run (or a pass triggered outside the UI) should still show
  // the in-progress state instead of an idle button; the idle branch also
  // carries the loose count for the button tooltip
  const [looseCount, setLooseCount] = useState(-1);
  useEffect(() => {
    if (!kbId) return undefined;
    let cancelled = false;
    setLooseCount(-1);
    getWikiTocOrganizeStatus(kbId)
      .then(st => {
        if (cancelled || !st) return;
        setLooseCount(st.loose ?? -1);
        if (st.running) {
          setOrganizing(true);
          pollOrganize(kbId, Date.now() - 60_000);
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kbId]);
  useEffect(() => stopOrganizePolling, []);

  // the hub card wall is now the sider's "workspace home" entry
  const isHub = location.pathname === '/wiki/list' || location.pathname === '/wiki';

  const siderBody = (
    <>
      {/* search + refresh share one row, mirroring the chat sidebar's
          HistoryList header (input flex-1, bordered square refresh at right) */}
      <div className='wiki-sider-search flex items-center gap-6px'>
        <Input
          allowClear
          className='min-w-0 flex-1'
          prefix={<SearchOutlined className='text-gray-400' />}
          placeholder={t('page.wiki.search.placeholder')}
          value={treeFilter}
          onChange={e => setTreeFilter(e.target.value)}
        />
        <Tooltip title={t('common.refresh')}>
          {/* chat sidebar refresh look: 32px bordered square + spinning glyph */}
          <Button
            type='default'
            className='size-8 p-0 flex items-center justify-center border-[#F0F0F0] dark:border-[#303030] rounded-12px'
            onClick={onRefresh}
          >
            <RefreshIcon strokeWidth={2} className={`size-4 ${isRefresh ? 'animate-spin' : ''}`} />
          </Button>
        </Tooltip>
        <Tooltip
          title={
            tocMaturity.parked > 0
              ? showAllToc
                ? `展示全部 ${tocMaturity.mature + tocMaturity.parked} 篇 (含草稿/已归档)`
                : `仅展示成熟内容 ${tocMaturity.mature} 篇 — 另有 ${tocMaturity.parked} 篇草稿/已归档已收起`
              : '没有草稿或已归档内容'
          }
        >
          {/* 收纳开关: 未成熟内容默认不出现在阅读树里 */}
          <Button
            type='default'
            className={`size-8 p-0 flex items-center justify-center rounded-12px ${
              showAllToc
                ? 'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-900 dark:bg-blue-950 dark:text-blue-300'
                : 'border-[#F0F0F0] text-gray-500 dark:border-[#303030]'
            }`}
            onClick={() => setShowAllToc(v => !v)}
            disabled={tocMaturity.parked === 0}
          >
            {showAllToc ? <Eye className='size-4' /> : <EyeOff className='size-4' />}
          </Button>
        </Tooltip>
        {kbId && canUpdateKb && (
          <Tooltip
            title={
              !organizing && looseCount > 0
                ? t('page.wiki.toc.organizeTooltip', { loose: looseCount })
                : t('page.wiki.toc.organize')
            }
          >
            <Button
              type='default'
              className='size-8 p-0 flex items-center justify-center border-[#F0F0F0] dark:border-[#303030] rounded-12px'
              loading={organizing}
              onClick={onOrganize}
            >
              <ApartmentOutlined className='text-14px' />
            </Button>
          </Tooltip>
        )}
      </div>
      {/* workspace home — the hub card wall lives here now (the topbar
          Segmented only carries governance + notifications) */}
      <div className='wiki-sider-home'>
        <button
          type='button'
          className={`flex w-full cursor-pointer items-center gap-2 rounded-lg border-0 bg-transparent px-2 py-1.5 text-left text-sm transition-colors hover:bg-[#EDEDED] dark:hover:bg-[#353F4D] ${
            isHub ? 'font-medium' : ''
          }`}
          style={{
            backgroundColor: isHub ? 'var(--ant-color-primary-bg)' : undefined,
            color: 'var(--ant-color-text)'
          }}
          onClick={() => nav('/wiki/list')}
        >
          <HomeOutlined className='h-4 w-4' />
          <span className='flex-1 truncate'>{t('page.wiki.hub.home')}</span>
        </button>
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
            treeData={displayTreeData}
            onDrop={onTreeDrop as any}
            onExpand={onExpand}
            onSelect={onSelect}
            className='wiki-toc-tree'
            // without showIcon antd hides node icons entirely (ant-tree-icon-hide)
            showIcon
            // linear expand/collapse chevrons (lucide) instead of antd's
            // filled caret — matches the outlined folder/article icons
            switcherIcon={({ expanded }: { expanded?: boolean }) =>
              expanded ? (
                <ChevronDown className='h-3.5 w-3.5 opacity-70' />
              ) : (
                <ChevronRight className='h-3.5 w-3.5 opacity-70' />
              )
            }
          />
        )}
      </div>
    </>
  );

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
      <div className={`wiki-page wiki-shell h-full min-h-0 ${resizing ? 'wiki-shell-resizing' : ''}`}>
        {!isMobile && (
          <aside
            className={`wiki-sider ${collapsed ? 'wiki-sider-collapsed' : ''}`}
            style={collapsed ? undefined : { width: siderWidth }}
          >
            {siderBody}
            {/* right-edge drag strip: resize the TOC sider, double-click to
                reset — hidden along with the sider when collapsed */}
            <div
              aria-orientation='vertical'
              className='wiki-sider-resizer'
              role='separator'
              tabIndex={collapsed ? -1 : 0}
              onKeyDown={onResizerKeyDown}
              onDoubleClick={onResizerReset}
              onPointerCancel={onResizerUp}
              onPointerDown={onResizerDown}
              onPointerMove={onResizerMove}
              onPointerUp={onResizerUp}
            />
          </aside>
        )}
        <div className='flex h-full min-w-0 flex-1 flex-col'>
          <header className='wiki-topbar'>
            {/* sider toggle — same look as the chat header's history toggle:
                bordered default-size button + panel glyphs */}
            <Button
              icon={collapsed ? <SidebarExpandSvg className='h-4 w-4' /> : <SidebarCollapseSvg className='h-4 w-4' />}
              onClick={() => setCollapsed(v => !v)}
              className='!rounded-12px border-[#F0F0F0] dark:border-[#303030] shrink-0'
            />
            {/* workspace switcher — chat-style trigger + custom panel (same
                look as the assistant switcher): bordered rounded button that
                opens a popover with search + check-marked list; sits at the
                topbar's left edge and filters the hub card wall */}
            <div className='relative'>
              <Button
                className='w-170px !rounded-12px border-[#F0F0F0] !px-8px dark:border-[#303030]'
                onClick={() => {
                  setWsOpen(v => !v);
                  setWsFilter('');
                }}
              >
                <TeamOutlined className='text-gray-400' />
                <span className='flex-1 truncate text-left text-sm'>
                  {workspaceId === 'all'
                    ? t('page.wiki.workspace.all')
                    : workspaces.find(w => w.id === workspaceId)?.name || ''}
                </span>
                <ChevronDown className='h-4 w-4 opacity-60' />
              </Button>
              {wsOpen && (
                <div
                  className='absolute left-0 top-full z-50 mt-2 w-64 rounded-xl border border-solid border-[#F0F0F0] p-3 shadow-lg dark:border-[#303030]'
                  style={{
                    backgroundColor: 'var(--ant-color-bg-elevated)',
                    color: 'var(--ant-color-text)',
                    transformOrigin: 'top center',
                    animation: 'expandDown 0.2s ease-out'
                  }}
                >
                  <div
                    className='mb-2 flex items-center justify-between text-sm font-semibold'
                    style={{ color: 'var(--ant-color-text)' }}
                  >
                    <div className='truncate'>{`${t('page.wiki.workspace.title')}（${workspaces.length}）`}</div>
                    <Tooltip title={t('common.refresh')}>
                      <Button
                        icon={<ReloadOutlined className='text-14px' />}
                        size='small'
                        type='text'
                        onClick={fetchWorkspaces}
                      />
                    </Tooltip>
                  </div>
                  <Input
                    allowClear
                    autoFocus
                    className='h-8 rounded-12px border-[#F0F0F0] dark:border-[#303030]'
                    placeholder={t('page.wiki.workspace.searchPlaceholder')}
                    prefix={<Search className='h-4 w-4 text-[#999] dark:text-[#666]' />}
                    style={{ backgroundColor: 'var(--ant-color-fill-quaternary)' }}
                    value={wsFilter}
                    onChange={e => setWsFilter(e.target.value)}
                  />
                  <div className='custom-scrollbar mt-2 max-h-60 overflow-auto'>
                    <div className='flex flex-col gap-1'>
                      <button
                        className={`flex w-full cursor-pointer items-center gap-2 rounded-lg border-0 bg-transparent px-2 py-1.5 text-left text-sm transition-colors hover:bg-[#EDEDED] dark:hover:bg-[#353F4D] ${
                          workspaceId === 'all' ? 'font-medium' : ''
                        }`}
                        style={{
                          backgroundColor: workspaceId === 'all' ? 'var(--ant-color-primary-bg)' : undefined,
                          color: workspaceId === 'all' ? 'var(--ant-color-primary)' : 'var(--ant-color-text)'
                        }}
                        type='button'
                        onClick={() => {
                          setWorkspaceId('all');
                          setWsOpen(false);
                        }}
                      >
                        <BookOutlined className='h-4 w-4' />
                        <div className='flex-1 truncate'>{t('page.wiki.workspace.all')}</div>
                        {workspaceId === 'all' && (
                          <Check
                            className='ml-auto h-4 w-4'
                            style={{ color: 'var(--ant-color-primary)' }}
                          />
                        )}
                      </button>
                      {workspaces
                        .filter(w => w.name.toLowerCase().includes(wsFilter.trim().toLowerCase()))
                        .map(w => {
                          const isActive = workspaceId === w.id;
                          return (
                            <button
                              className={`flex w-full cursor-pointer items-center gap-2 rounded-lg border-0 bg-transparent px-2 py-1.5 text-left text-sm transition-colors hover:bg-[#EDEDED] dark:hover:bg-[#353F4D] ${
                                isActive ? 'font-medium' : ''
                              }`}
                              key={w.id}
                              style={{
                                backgroundColor: isActive ? 'var(--ant-color-primary-bg)' : undefined,
                                color: isActive ? 'var(--ant-color-primary)' : 'var(--ant-color-text)'
                              }}
                              type='button'
                              onClick={() => {
                                setWorkspaceId(w.id);
                                setWsOpen(false);
                              }}
                            >
                              <TeamOutlined className='h-4 w-4' />
                              <div className='flex-1 truncate'>{w.name}</div>
                              {isActive && (
                                <Check
                                  className='ml-auto h-4 w-4'
                                  style={{ color: 'var(--ant-color-primary)' }}
                                />
                              )}
                            </button>
                          );
                        })}
                      {wsFilter.trim() &&
                        workspaces.filter(w => w.name.toLowerCase().includes(wsFilter.trim().toLowerCase()))
                          .length === 0 && (
                          <div
                            className='py-8 text-center text-sm'
                            style={{ color: 'var(--ant-color-text-quaternary)' }}
                          >
                            {t('common.noData')}
                          </div>
                        )}
                    </div>
                  </div>
                  <div className='mt-2 border-t border-solid border-[#F0F0F0] pt-2 dark:border-[#303030]'>
                    <button
                      className='flex w-full cursor-pointer items-center gap-2 rounded-lg border-0 bg-transparent px-2 py-1.5 text-left text-sm transition-colors hover:bg-[#EDEDED] dark:hover:bg-[#353F4D]'
                      style={{ color: 'var(--ant-color-text)' }}
                      type='button'
                      onClick={() => {
                        setWsOpen(false);
                        setWsCreateOpen(true);
                      }}
                    >
                      <PlusOutlined />
                      {t('page.wiki.workspace.create')}
                    </button>
                  </div>
                </div>
              )}
              {wsOpen && <div className='fixed inset-0 z-40' onClick={() => setWsOpen(false)} />}
            </div>
            {/* governance + notifications as bordered icon buttons (same look
                as the sider toggle), names surface on hover via tooltip */}
            <Tooltip title={t('page.wiki.governance.title')}>
              <Badge
                count={openGovernance}
                size='small'
              >
                <Button
                  icon={<BookCheck className='h-4 w-4' />}
                  onClick={() => nav('/wiki/governance')}
                  className='!rounded-12px border-[#F0F0F0] dark:border-[#303030]'
                />
              </Badge>
            </Tooltip>
            <Tooltip title={t('page.wiki.notification.title')}>
              <Badge
                count={unread}
                size='small'
              >
                <Button
                  icon={<BellOutlined />}
                  onClick={() => nav('/wiki/notifications')}
                  className='!rounded-12px border-[#F0F0F0] dark:border-[#303030]'
                />
              </Badge>
            </Tooltip>
            <div className='flex-1' />
          </header>
          <div className='wiki-content min-h-0 flex-1 overflow-auto'>
            <div className='mx-auto h-full max-w-1280px w-full 2xl:max-w-1440px'>
              {typeof children === 'function'
                ? children({ workspaceId, workspaces, kbVersion, refreshKbs: () => fetchKbs(true) })
                : children}
            </div>
          </div>
        </div>
        {isMobile && (
          <Drawer
            classNames={{ body: '!p-0px' }}
            closable={false}
            onClose={() => setCollapsed(true)}
            open={!collapsed}
            placement='left'
            width={280}
          >
            {siderBody}
          </Drawer>
        )}
        {renameDialog}
        <Modal
          cancelText={t('common.cancel')}
          okText={t('common.create')}
          open={wsCreateOpen}
          title={t('page.wiki.workspace.create')}
          onCancel={() => setWsCreateOpen(false)}
          onOk={onCreateWorkspace}
        >
          <Input
            placeholder={t('page.wiki.workspace.namePlaceholder')}
            value={wsName}
            onChange={e => setWsName(e.target.value)}
          />
        </Modal>
      </div>
    </ConfigProvider>
  );
}
