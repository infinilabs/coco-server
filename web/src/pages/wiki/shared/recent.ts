/** Recently-viewed articles, kept locally (no backend round-trip needed). */
export interface RecentArticle {
  id: string;
  title: string;
  kb_id?: string;
  at: number;
}

const KEY = 'wiki-recent-articles';
const LIMIT = 8;

export function getRecentArticles(): RecentArticle[] {
  try {
    const raw = localStorage.getItem(KEY);
    const list = raw ? (JSON.parse(raw) as RecentArticle[]) : [];
    return Array.isArray(list) ? list.filter(x => x?.id && x?.title) : [];
  } catch {
    return [];
  }
}

export function recordRecentArticle(item: Omit<RecentArticle, 'at'>) {
  if (!item.id) return;
  const next = [{ ...item, at: Date.now() }, ...getRecentArticles().filter(x => x.id !== item.id)].slice(0, LIMIT);
  try {
    localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // storage unavailable (private mode) — recents are best-effort
  }
}
