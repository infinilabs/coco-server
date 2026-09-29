/* TOC tree manipulation — ported from the coco-wiki prototype toc-utils.
 * Antd Tree drop events map onto moveNode via gap (before=-1/inside=0/after=1). */
import type { DataNode } from 'antd/es/tree';

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

export function toAntdTreeData(nodes: Api.Wiki.TocNode[]): DataNode[] {
  return nodes.map(node => ({
    key: node.id,
    title: node.title,
    children: node.children ? toAntdTreeData(node.children) : undefined
  }));
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
