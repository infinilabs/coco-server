import { useCallback, useEffect, useRef } from "react";
import { debounce } from "lodash";

export function useChatScroll(scrollContainerRef: React.RefObject<HTMLDivElement>) {
  const userScrollingRef = useRef(false);
  const scrollTimeoutRef = useRef<ReturnType<typeof setTimeout>>();
  const lastScrollHeightRef = useRef<number>(0);
  const programmaticScrollRef = useRef(false);

  const isNearBottom = (container: HTMLElement) => {
    const { scrollTop, scrollHeight, clientHeight } = container;
    return Math.abs(scrollHeight - scrollTop - clientHeight) < 150;
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const scrollToBottom = useCallback(
    debounce((force?: boolean) => {
      const container = scrollContainerRef.current;
      if (!container) return;

      const contentChanged = lastScrollHeightRef.current !== container.scrollHeight;
      lastScrollHeightRef.current = container.scrollHeight;

      if (force || !userScrollingRef.current || (contentChanged && isNearBottom(container))) {
        programmaticScrollRef.current = true;
        container.scrollTo({
          top: container.scrollHeight,
          behavior: "smooth",
        });
      }
    }, 50),
    [scrollContainerRef]
  );

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    lastScrollHeightRef.current = container.scrollHeight;

    const observer = new MutationObserver(() => {
      if (!userScrollingRef.current) {
        scrollToBottom();
      }
    });

    observer.observe(container, {
      childList: true,
      subtree: true,
      characterData: true,
    });

    return () => {
      observer.disconnect();
    };
  }, [scrollContainerRef, scrollToBottom]);

  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      if (programmaticScrollRef.current) {
        programmaticScrollRef.current = false;
        return;
      }

      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }

      const near = isNearBottom(container);
      if (!near) {
        userScrollingRef.current = true;
      }

      scrollTimeoutRef.current = setTimeout(() => {
        if (isNearBottom(container)) {
          userScrollingRef.current = false;
        }
      }, 300);
    };

    container.addEventListener("scroll", handleScroll);
    return () => {
      container.removeEventListener("scroll", handleScroll);
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, [scrollContainerRef]);

  const resetUserScrolling = useCallback(() => {
    userScrollingRef.current = false;
  }, []);

  return {
    resetUserScrolling,
    scrollToBottom
  };
}
