import {
  memo,
  useRef,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react";

/**
 * Node Consolidation (骨灰级方案):
 *
 * Messages are split into chunks. When a chunk scrolls far off-screen,
 * all its child DOM nodes are destroyed and replaced by a single spacer div
 * preserving the measured total height. When the user scrolls back, the chunk
 * is "unfrozen" and all messages are re-rendered.
 *
 * Result: DOM node count stays bounded (< hundreds) regardless of message count.
 */

const DEFAULT_CHUNK_SIZE = 5;
const DEFAULT_VIEWPORT_BUFFER = 1000;

// ─── Types ───────────────────────────────────────────────────────────────────

export interface VMsgItem {
  _id: string;
  [key: string]: any;
}

export interface VMsgListProps {
  /** Array of message objects. Each must have a unique `_id`. */
  messages: VMsgItem[];
  /** The scrollable container element (for IntersectionObserver root). */
  scrollRoot: HTMLElement | null;
  /** How many messages per consolidation group. Default: 50 */
  chunkSize?: number;
  /** Px buffer around viewport before freeze/unfreeze triggers. Default: 2000 */
  viewportBuffer?: number;
  /** Render function for each message. Receives the message and its global index. */
  renderMessage: (message: VMsgItem, index: number) => ReactNode;
}

// ─── Component ───────────────────────────────────────────────────────────────

interface ChunkState {
  frozen: boolean;
  height: number;
}

export const VMsgList = memo(function VMsgList({
  messages,
  scrollRoot,
  chunkSize = DEFAULT_CHUNK_SIZE,
  viewportBuffer = DEFAULT_VIEWPORT_BUFFER,
  renderMessage,
}: VMsgListProps) {
  const heightMapRef = useRef<Record<string, number>>({});
  const [chunkStates, setChunkStates] = useState<Map<number, ChunkState>>(
    new Map()
  );
  const chunkRefs = useRef<Map<number, HTMLDivElement>>(new Map());
  const observerRef = useRef<IntersectionObserver | null>(null);
  // Track scrollRoot readiness to handle initial null ref
  const [resolvedRoot, setResolvedRoot] = useState<HTMLElement | null>(scrollRoot);

  useEffect(() => {
    if (scrollRoot && scrollRoot !== resolvedRoot) {
      setResolvedRoot(scrollRoot);
    }
  }, [scrollRoot, resolvedRoot]);

  // Split into chunks
  const chunks: VMsgItem[][] = [];
  for (let i = 0; i < messages.length; i += chunkSize) {
    chunks.push(messages.slice(i, i + chunkSize));
  }

  // Height measurement callback
  const measureRef = useCallback(
    (el: HTMLDivElement | null, messageId: string) => {
      if (!el) return;
      const h = el.getBoundingClientRect().height;
      if (h > 0) {
        heightMapRef.current[messageId] = h;
      }
    },
    []
  );

  // IntersectionObserver for freeze/unfreeze
  useEffect(() => {
    if (!resolvedRoot) return;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        setChunkStates((prev) => {
          let changed = false;
          const next = new Map(prev);

          for (const entry of entries) {
            const el = entry.target as HTMLDivElement;
            const chunkIndex = Number(el.dataset.chunkIndex);
            if (isNaN(chunkIndex)) continue;

            const current = next.get(chunkIndex);

            if (entry.isIntersecting) {
              // Near viewport → unfreeze
              if (current?.frozen) {
                next.set(chunkIndex, { frozen: false, height: 0 });
                changed = true;
              }
            } else {
              // Far from viewport → freeze if fully measured
              if (!current?.frozen) {
                const start = chunkIndex * chunkSize;
                const end = Math.min(start + chunkSize, messages.length);
                let totalHeight = 0;
                let allMeasured = true;

                for (let i = start; i < end; i++) {
                  const h = heightMapRef.current[messages[i]?._id];
                  if (h == null || h === 0) {
                    allMeasured = false;
                    break;
                  }
                  totalHeight += h;
                }

                if (allMeasured && totalHeight > 0) {
                  next.set(chunkIndex, { frozen: true, height: totalHeight });
                  changed = true;
                }
              }
            }
          }

          return changed ? next : prev;
        });
      },
      {
        root: resolvedRoot,
        rootMargin: `${viewportBuffer}px 0px ${viewportBuffer}px 0px`,
        threshold: 0,
      }
    );

    chunkRefs.current.forEach((el) => {
      observerRef.current?.observe(el);
    });

    return () => {
      observerRef.current?.disconnect();
      observerRef.current = null;
    };
  }, [resolvedRoot, viewportBuffer, messages.length, chunkSize]);

  const setChunkRef = useCallback(
    (index: number, el: HTMLDivElement | null) => {
      if (el) {
        chunkRefs.current.set(index, el);
        observerRef.current?.observe(el);
      } else {
        const prev = chunkRefs.current.get(index);
        if (prev) observerRef.current?.unobserve(prev);
        chunkRefs.current.delete(index);
      }
    },
    []
  );

  return (
    <>
      {chunks.map((chunk, chunkIndex) => {
        const state = chunkStates.get(chunkIndex);
        const isFrozen = state?.frozen === true;

        return (
          <div
            key={`chunk-${chunkIndex}`}
            ref={(el) => setChunkRef(chunkIndex, el)}
            data-chunk-index={chunkIndex}
            data-frozen={isFrozen}
            style={
              isFrozen
                ? { height: state.height, overflow: "hidden", contain: "strict" }
                : { overflowAnchor: "auto" }
            }
          >
            {isFrozen
              ? null
              : chunk.map((message, i) => {
                  const globalIndex = chunkIndex * chunkSize + i;
                  return (
                    <MsgWrapper
                      key={message._id}
                      messageId={message._id}
                      onMeasure={measureRef}
                    >
                      {renderMessage(message, globalIndex)}
                    </MsgWrapper>
                  );
                })}
          </div>
        );
      })}
    </>
  );
});

// ─── Internal: Message height measurement wrapper ────────────────────────────

interface MsgWrapperProps {
  messageId: string;
  onMeasure: (el: HTMLDivElement | null, messageId: string) => void;
  children: ReactNode;
}

const MsgWrapper = memo(function MsgWrapper({
  messageId,
  onMeasure,
  children,
}: MsgWrapperProps) {
  const ref = useRef<HTMLDivElement>(null);
  const roRef = useRef<ResizeObserver | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    onMeasure(el, messageId);

    roRef.current = new ResizeObserver(() => {
      onMeasure(el, messageId);
    });
    roRef.current.observe(el);

    return () => {
      roRef.current?.disconnect();
      roRef.current = null;
    };
  }, [messageId, onMeasure]);

  return (
    <div ref={ref} data-message-id={messageId}>
      {children}
    </div>
  );
});
