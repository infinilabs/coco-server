import { useCallback, useEffect, useRef, useState } from "react";

const BOTTOM_THRESHOLD = 50;
const TOP_THRESHOLD = 5;

export interface ScrollManagerOptions {
  /** Called when user scrolls to the top. Use for loading older messages. */
  onScrollToTop?: () => void;
}

/**
 * Bidirectional scroll state machine:
 * - Detects user scrolling up → locks auto-scroll-to-bottom.
 * - Re-enables when user scrolls back to bottom or programmatic scrollToBottom is called.
 * - Auto-scrolls on DOM mutations (streaming) when not locked.
 * - Detects scroll-to-top for triggering history pagination.
 * - Provides scroll anchoring for seamless prepending of older messages.
 */
export function useScrollManager(
  scrollContainerRef: React.RefObject<HTMLDivElement | null>,
  options?: ScrollManagerOptions
) {
  const isLockedRef = useRef(false);
  const programmaticScrollRef = useRef(false);
  const rafRef = useRef<number>();
  const onScrollToTopRef = useRef(options?.onScrollToTop);
  onScrollToTopRef.current = options?.onScrollToTop;

  const [isAtBottom, setIsAtBottom] = useState(true);

  const checkIsAtBottom = useCallback((container: HTMLElement) => {
    const { scrollTop, scrollHeight, clientHeight } = container;
    return scrollHeight - scrollTop - clientHeight < BOTTOM_THRESHOLD;
  }, []);

  // Scroll event → lock / unlock + scroll-to-top detection
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      if (programmaticScrollRef.current) {
        programmaticScrollRef.current = false;
        return;
      }

      const atBottom = checkIsAtBottom(container);
      setIsAtBottom(atBottom);

      if (atBottom) {
        // User scrolled back to bottom → unlock immediately
        isLockedRef.current = false;
      } else {
        // User scrolled away from bottom → lock immediately to prevent snap-back
        isLockedRef.current = true;
      }

      // Scroll-to-top detection for loading older history
      if (container.scrollTop <= TOP_THRESHOLD) {
        onScrollToTopRef.current?.();
      }
    };

    container.addEventListener("scroll", handleScroll, { passive: true });
    return () => {
      container.removeEventListener("scroll", handleScroll);
    };
  }, [scrollContainerRef, checkIsAtBottom]);

  // MutationObserver → auto-scroll when unlocked
  // Uses throttled rAF (max once per frame ≈ 16ms at 60Hz) with instant
  // pixel-level scrollTop assignment (no smooth animation) to prevent
  // jitter from overlapping smooth scroll animations during streaming.
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    let ticking = false;

    const observer = new MutationObserver(() => {
      if (isLockedRef.current) return;
      if (ticking) return;
      ticking = true;
      rafRef.current = requestAnimationFrame(() => {
        ticking = false;
        if (!container || isLockedRef.current) return;
        programmaticScrollRef.current = true;
        // Instant sync (behavior: 'auto') — never use smooth during streaming
        container.scrollTop = container.scrollHeight;
        setIsAtBottom(true);
      });
    });

    observer.observe(container, {
      childList: true,
      subtree: true,
      characterData: true,
    });

    return () => {
      observer.disconnect();
      if (rafRef.current) cancelAnimationFrame(rafRef.current);
    };
  }, [scrollContainerRef]);

  const scrollToBottom = useCallback(
    (force?: boolean) => {
      const container = scrollContainerRef.current;
      if (!container) return;
      if (force || !isLockedRef.current) {
        programmaticScrollRef.current = true;
        container.scrollTop = container.scrollHeight;
        setIsAtBottom(true);
        isLockedRef.current = false;
      }
    },
    [scrollContainerRef]
  );

  const resetScrollState = useCallback(() => {
    isLockedRef.current = false;
    setIsAtBottom(true);
  }, []);

  // ─── Scroll Anchoring (Strategy 3) ──────────────────────────────────────
  // Save/restore the visual position of the first visible message
  // when prepending older messages at the top.

  const anchorRef = useRef<{ id: string; offsetTop: number } | null>(null);

  /**
   * Call BEFORE prepending new messages to the DOM.
   * Records the first visible message's position relative to the scroll container.
   */
  const saveScrollAnchor = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const messages = container.querySelectorAll("[data-message-id]");
    const containerRect = container.getBoundingClientRect();
    for (const msg of messages) {
      const rect = msg.getBoundingClientRect();
      if (rect.bottom > containerRect.top) {
        const id = msg.getAttribute("data-message-id") || "";
        anchorRef.current = {
          id,
          offsetTop: rect.top - containerRect.top,
        };
        break;
      }
    }
  }, [scrollContainerRef]);

  /**
   * Call AFTER new messages have been rendered in the DOM.
   * Adjusts scrollTop so the anchored message stays in the same visual position.
   */
  const restoreScrollAnchor = useCallback(() => {
    const container = scrollContainerRef.current;
    const anchor = anchorRef.current;
    if (!container || !anchor) return;

    const el = container.querySelector(`[data-message-id="${anchor.id}"]`);
    if (!el) {
      anchorRef.current = null;
      return;
    }

    const containerRect = container.getBoundingClientRect();
    const rect = el.getBoundingClientRect();
    const currentOffset = rect.top - containerRect.top;
    const delta = currentOffset - anchor.offsetTop;

    if (Math.abs(delta) > 1) {
      programmaticScrollRef.current = true;
      container.scrollTop += delta;
    }

    anchorRef.current = null;
  }, [scrollContainerRef]);

  return {
    isAtBottom,
    scrollToBottom,
    resetScrollState,
    saveScrollAnchor,
    restoreScrollAnchor,
  };
}
