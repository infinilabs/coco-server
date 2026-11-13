/* TOC tree manipulation — ported from the coco-wiki prototype toc-utils.
 * Antd Tree drop events map onto moveNode via gap (before=-1/inside=0/after=1). */
import type { DataNode } from 'antd/es/tree';
import type { ReactNode } from 'react';
import { CompassOutlined, FileTextOutlined, FolderOpenOutlined, FolderOutlined, IdcardOutlined, LinkOutlined } from '@ant-design/icons';

export type DropPosition = 'before' | 'inside' | 'after';

export function removeNode(nodes: Api.Wiki.TocNode[], id: string): [Api.Wiki.TocNode[], Api.Wiki.TocNode | null] {
  let removed: Api.Wiki.TocNode | null = null;
  const result = nodes.reduce<Api.Wiki.TocNode[]>((acc, node) => {
    if (node.id === id) {
      removed = node;
      return acc;
    }
    if (node.children) {
      const [newChildren, found] = removeNode(node.children, id);
      if (found) {
        removed = found;
        acc.push({ ...node, children: newChildren });
        return acc;
      }
    }
    acc.push(node);
    return acc;
  }, []);
  return [result, removed];
}

export function insertNode(
  nodes: Api.Wiki.TocNode[],
  targetId: string,
  position: DropPosition,
  newNode: Api.Wiki.TocNode
): Api.Wiki.TocNode[] {
  const result: Api.Wiki.TocNode[] = [];
  for (const node of nodes) {
    if (node.id === targetId) {
      if (position === 'before') {
        result.push(newNode, node);
      } else if (position === 'after') {
        result.push(node, newNode);
      } else if (position === 'inside' && node.type === 'folder') {
        result.push({ ...node, children: [...(node.children || []), newNode] });
      } else {
        result.push(node);
      }
    } else if (node.children) {
      const newChildren = insertNode(node.children, targetId, position, newNode);
      result.push(newChildren !== node.children ? { ...node, children: newChildren } : node);
    } else {
      result.push(node);
    }
  }
  return result;
}

export function moveNode(
  tree: Api.Wiki.TocNode[],
  draggedId: string,
  targetId: string,
  position: DropPosition
): Api.Wiki.TocNode[] {
  if (draggedId === targetId) return tree;
  const [treeWithout, removed] = removeNode(tree, draggedId);
  if (!removed) return tree;
  return insertNode(treeWithout, targetId, position, removed);
}

/** antd Tree onDrop info.dropToGap + dropPosition → DropPosition */
export function dropPositionFromAntd(dropToGap: boolean, antdPosition: number, isFolder: boolean): DropPosition {
  if (dropToGap) return antdPosition === -1 ? 'before' : 'after';
  return isFolder ? 'inside' : 'after';
}

/** per-page-type icon so the tree reads like a file explorer: folders vs article kinds.
 * Deliberately uncolored — antd's tree mutes them via .wiki-toc-iconEle rules,
 * color-coded icons read noisy against the flat sidebar (Yuque keeps icons quiet). */
function articleIcon(pageType?: string) {
  switch (pageType) {
    case 'entity':
      return <IdcardOutlined />;
    case 'source':
      return <LinkOutlined />;
    case 'map':
      return <CompassOutlined />;
    default:
      // concept and untyped pages are plain documents
      return <FileTextOutlined />;
  }
}

const STATUS_TAG: Record<string, string> = {
  draft: '草稿',
  archived: '已归档'
};

/** 成熟度排序权重: 阅读面 (published/未标状态的旧数据) 在前, 草稿次之, 已归档垫底 —
 * 用户手排的顺序在各自桶内保持稳定。 */
function maturity(status?: string): number {
  if (status === 'archived') return 2;
  if (status === 'draft') return 1;
  return 0;
}

export function toAntdTreeData(
  nodes: Api.Wiki.TocNode[],
  pageTypes?: Record<string, string>,
  statuses?: Record<string, string>,
  opts?: { includeAll?: boolean }
): DataNode[] {
  const includeAll = opts?.includeAll ?? false;

  // 默认只展示成熟内容: 未成熟节点直接剔除, 因此空了的文件夹也递归剪掉
  if (statuses && !includeAll) {
    const keep = (list: Api.Wiki.TocNode[]): Api.Wiki.TocNode[] => {
      const out: Api.Wiki.TocNode[] = [];
      for (const n of list) {
        if (n.type === 'article') {
          if (maturity(statuses[n.article_id || '']) === 0) out.push(n);
          continue;
        }
        const children = n.children ? keep(n.children) : [];
        if (children.length > 0) out.push({ ...n, children });
      }
      return out;
    };
    nodes = keep(nodes);
  }

  // 稳定排序: 文件夹与正式内容留在用户排的位置, 未成熟内容统一沉到组尾
  // (includeAll 时才可见; 排序对剩余的成熟内容是恒等的)
  const sorted = nodes
    .map((node, i) => ({ node, i, rank: maturity(node.type === 'article' ? statuses?.[node.article_id || ''] : undefined) }))
    .sort((a, b) => a.rank - b.rank || a.i - b.i)
    .map(x => x.node);

  return sorted.map(node => {
    const status = node.type === 'article' ? statuses?.[node.article_id || ''] : undefined;
    return {
      key: node.id,
      // plain string always — titleRender adds the parked dimming/tag from
      // the TocNode (a JSX title here stringifies to "[object Object]" in
      // titleRender and defeats sidebar search)
      title: node.title,
      // only folders expand (lazy children); articles are leaves with no caret —
      // without isLeaf every node shows an expand arrow because loadData is set
      isLeaf: node.type !== 'folder',
      icon:
        node.type === 'folder'
          ? ({ expanded }: { expanded?: boolean }) => (expanded ? <FolderOpenOutlined /> : <FolderOutlined />)
          : articleIcon(pageTypes?.[node.article_id || '']),
      children: node.children ? toAntdTreeData(node.children, pageTypes, statuses, opts) : undefined
    };
  });
}

/** sidebar row title: parked pages (draft/archived) render dimmed with a
 * status suffix — titleRender renders this from the TocNode so treeData
 * titles stay plain strings. */
export function tocNodeTitle(node: Api.Wiki.TocNode, status?: string): ReactNode {
  if (maturity(status) === 0) return node.title;
  return (
    <span className='opacity-55'>
      {node.title}
      <span className='ml-1 text-[10px] text-muted-foreground/70'>{STATUS_TAG[status || ''] || status}</span>
    </span>
  );
}

/** 成熟度盘点: (正式内容, 未成熟) 文章数 — 侧边栏开关的提示文案用。 */
export function countTocMaturity(
  tocs: Record<string, Api.Wiki.TocNode[]>,
  statuses: Record<string, Record<string, string>>
): { mature: number; parked: number } {
  let mature = 0;
  let parked = 0;
  const walk = (list: Api.Wiki.TocNode[], stats?: Record<string, string>) => {
    for (const n of list) {
      if (n.type === 'article') {
        if (maturity(stats?.[n.article_id || '']) > 0) parked++;
        else mature++;
      }
      if (n.children) walk(n.children, stats);
    }
  };
  for (const [kbId, nodes] of Object.entries(tocs)) walk(nodes, statuses[kbId]);
  return { mature, parked };
}

/** find a TocNode (and its parent chain) by node id */
export function findNode(nodes: Api.Wiki.TocNode[], id: string): Api.Wiki.TocNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    if (node.children) {
      const found = findNode(node.children, id);
      if (found) return found;
    }
  }
  return null;
}

/** find the toc node id for an article id (first match) */
export function tocIdForArticle(nodes: Api.Wiki.TocNode[], articleId: string): string | null {
  for (const node of nodes) {
    if (node.article_id === articleId) return node.id;
    if (node.children) {
      const found = tocIdForArticle(node.children, articleId);
      if (found) return found;
    }
  }
  return null;
}
