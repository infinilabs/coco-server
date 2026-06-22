import { useCallback, useRef, useState } from "react";

const DEFAULT_PAGE_SIZE = 20;

export interface UseLoadHistoryOptions {
  /** Number of messages to load per page */
  pageSize?: number;
  /** Fetch function: given chatId + cursor params, returns a page of messages (oldest first) */
  fetchPage: (params: {
    chatId: string;
    from: number;
    size: number;
  }) => Promise<{ hits: Array<{ _id: string; [key: string]: any }>; total?: number } | null>;
}

export interface UseLoadHistoryReturn {
  /** Whether older messages are currently being loaded */
  isLoadingMore: boolean;
  /** Whether there are more older messages to load */
  hasMore: boolean;
  /** Load the next (older) page of messages. Call when user scrolls to top. */
  loadMore: (chatId: string) => Promise<Array<{ _id: string; [key: string]: any }> | null>;
  /** Reset pagination state (call on chat switch) */
  resetPagination: () => void;
}

/**
 * Hook for cursor-based pagination of chat history.
 *
 * - Tracks `from` offset for the next page to load.
 * - Returns loaded messages for prepending to the list.
 * - Exposes `hasMore` to know when all history is fetched.
 */
export function useLoadHistory({
  pageSize = DEFAULT_PAGE_SIZE,
  fetchPage,
}: UseLoadHistoryOptions): UseLoadHistoryReturn {
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const hasMoreRef = useRef(true);
  const [hasMore, setHasMore] = useState(true);
  const fromRef = useRef(0);
  const loadingRef = useRef(false);

  const loadMore = useCallback(
    async (chatId: string) => {
      if (loadingRef.current || !hasMoreRef.current) return null;
      loadingRef.current = true;
      setIsLoadingMore(true);

      try {
        const result = await fetchPage({
          chatId,
          from: fromRef.current,
          size: pageSize,
        });

        if (!result || result.hits.length === 0) {
          hasMoreRef.current = false;
          setHasMore(false);
          return null;
        }

        // Update cursor
        fromRef.current += result.hits.length;

        // Check if we've loaded all messages
        if (result.hits.length < pageSize) {
          hasMoreRef.current = false;
          setHasMore(false);
        } else if (result.total != null && fromRef.current >= result.total) {
          hasMoreRef.current = false;
          setHasMore(false);
        }

        return result.hits;
      } finally {
        loadingRef.current = false;
        setIsLoadingMore(false);
      }
    },
    [fetchPage, pageSize]
  );

  const resetPagination = useCallback(() => {
    fromRef.current = 0;
    hasMoreRef.current = true;
    setHasMore(true);
    loadingRef.current = false;
    setIsLoadingMore(false);
  }, []);

  return { isLoadingMore, hasMore, loadMore, resetPagination };
}
