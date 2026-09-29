// Recent search queries, kept client-side in localStorage. The list is
// newest-first, deduped (re-searching moves the query to the front) and
// capped so it can never grow unbounded. All access is try/catch-guarded
// because localStorage can be unavailable (private mode, quota, embedders).

const STORAGE_KEY = "ui-search-recent-queries";
const MAX_RECENT_SEARCHES = 10;

function readRaw(): string[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === "string" && !!item.trim());
  } catch {
    return [];
  }
}

function writeRaw(items: string[]) {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(items.slice(0, MAX_RECENT_SEARCHES)));
  } catch {
    // storage unavailable — recent searches are best-effort, never fatal
  }
}

export function loadRecentSearches(): string[] {
  return readRaw();
}

export function pushRecentSearch(query: string) {
  const normalized = (query || "").trim();
  if (!normalized) return;
  const next = [normalized, ...readRaw().filter((item) => item !== normalized)];
  writeRaw(next);
}

export function removeRecentSearch(query: string) {
  const normalized = (query || "").trim();
  writeRaw(readRaw().filter((item) => item !== normalized));
}
