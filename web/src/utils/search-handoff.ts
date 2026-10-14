/**
 * One-shot handoff for searches started outside the search page.
 *
 * The app shell header embeds the ui-search SearchBox so every page can search.
 * Search-mode submits cross the route boundary through the URL (query /
 * search_type / filter), but chat-mode submits carry in-memory state the URL
 * cannot hold (uploaded attachments, the chosen assistant). Those are parked
 * here right before navigating to `#/chat?mode=chat`, and the search page
 * consumes them exactly once on mount to seed its chat mode.
 */

export interface SearchHandoff {
  query?: string;
  attachments?: any[];
  assistant_id?: string;
}

let pending: SearchHandoff | null = null;

export function setPendingSearch(handoff: SearchHandoff) {
  pending = handoff;
}

export function consumePendingSearch(): SearchHandoff | null {
  const value = pending;
  pending = null;
  return value;
}
