import { useCallback, useEffect, useRef, useState } from 'react';

interface PreviewConnectorProps {
  /** id of the result row currently shown in the preview pane */
  readonly activeId: string | number;
  /** widget root scoping the row/pane queries; falls back to the document */
  readonly getContainer?: () => HTMLElement | null;
}

interface ConnectorGeometry {
  /** anchor on the selected row's right edge (svg user units = px) */
  x1: number;
  y1: number;
  /** fixed port at the middle of the pane's left edge */
  x2: number;
  y2: number;
}

const cssEscape = (value: string) =>
  typeof CSS !== 'undefined' && typeof CSS.escape === 'function'
    ? CSS.escape(value)
    : value.replace(/["\\]/g, '\\$&');

const findRow = (root: ParentNode, activeId: string) =>
  root.querySelector<HTMLElement>(`[data-result-id="${cssEscape(activeId)}"]`);
const findPane = (root: ParentNode) => root.querySelector<HTMLElement>('[data-preview-pane]');

/**
 * Leader line from the selected result row to the inline preview pane.
 *
 * Both endpoints are located by DOM query (data-result-id on the row wrapper,
 * data-preview-pane on BasicLayout's fixed pane) instead of prop-drilled refs.
 * The pane end is a fixed port at the middle of its left edge; the row end
 * follows the row while the virtualizer keeps it mounted and otherwise falls
 * back to the last known anchor, clamped into the visible band, so scrolling
 * the row out of the virtual window bends the line instead of dropping it.
 * Geometry is viewport-based, re-measured on scroll (capture phase: the
 * widget's root is the scroll container) and on resize of either endpoint.
 * The line hides only when the pane itself is gone (folded to the overlay
 * variant) or no anchor was ever seen for the current selection.
 */
export function PreviewConnector(props: PreviewConnectorProps) {
  const { activeId, getContainer } = props;
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [geometry, setGeometry] = useState<ConnectorGeometry | null>(null);
  // the virtualizer unmounts the selected row once it scrolls out of the
  // virtual window — remember its anchor so the line survives and rides the
  // visible band edge until the row scrolls back in
  const lastAnchor = useRef<{ x1: number; y1Raw: number } | null>(null);

  useEffect(() => {
    lastAnchor.current = null;
  }, [activeId]);

  const measure = useCallback(() => {
    const svg = svgRef.current;
    if (!svg) return;

    const root: ParentNode = getContainer?.() ?? document;
    const pane = findPane(root);
    if (!pane) {
      setGeometry(null);
      return;
    }
    const paneRect = pane.getBoundingClientRect();
    if (paneRect.width === 0) {
      setGeometry(null);
      return;
    }

    // coordinates are computed against the svg's own rect so the overlay stays
    // correct even when an ancestor establishes a non-viewport containing block
    const svgRect = svg.getBoundingClientRect();

    let x1: number;
    let y1Raw: number;
    const row = findRow(root, String(activeId));
    if (row) {
      const rowRect = row.getBoundingClientRect();
      if (rowRect.width > 0) {
        x1 = rowRect.right + 6 - svgRect.left;
        y1Raw = rowRect.top + rowRect.height / 2 - svgRect.top;
        lastAnchor.current = { x1, y1Raw };
      } else if (lastAnchor.current) {
        ({ x1, y1Raw } = lastAnchor.current);
      } else {
        setGeometry(null);
        return;
      }
    } else if (lastAnchor.current) {
      ({ x1, y1Raw } = lastAnchor.current);
    } else {
      setGeometry(null);
      return;
    }

    // the overlay is fixed and escapes the scroll container's overflow clip —
    // clamp the row anchor into the visible band (below the header, above the
    // bottom) whether the row is merely clipped or fully unmounted
    let y1 = y1Raw;
    const clipEl = getContainer?.();
    if (clipEl) {
      const clipRect = clipEl.getBoundingClientRect();
      const clipTop = Math.max(clipRect.top, paneRect.top) + 8;
      const clipBottom = Math.max(clipRect.bottom - 8, clipTop);
      y1 = Math.min(Math.max(y1Raw, clipTop), clipBottom);
    }

    // the pane end is a fixed port: the middle of its left edge
    const x2 = paneRect.left - 7 - svgRect.left;
    const y2 = paneRect.top + paneRect.height / 2 - svgRect.top;

    const next = { x1, y1, x2, y2 };
    setGeometry(prev =>
      prev && prev.x1 === next.x1 && prev.y1 === next.y1 && prev.x2 === next.x2 && prev.y2 === next.y2
        ? prev
        : next
    );
  }, [activeId, getContainer]);

  useEffect(() => {
    let raf = 0;
    const schedule = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(measure);
    };

    // the pane mounts in the same commit as this overlay — measure after paint
    schedule();

    window.addEventListener('scroll', schedule, true);
    window.addEventListener('resize', schedule);

    const observer = new ResizeObserver(schedule);
    const root: ParentNode = getContainer?.() ?? document;
    const row = findRow(root, String(activeId));
    const pane = findPane(root);
    if (row) observer.observe(row);
    if (pane) observer.observe(pane);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('scroll', schedule, true);
      window.removeEventListener('resize', schedule);
      observer.disconnect();
    };
  }, [measure, activeId, getContainer]);

  const { x1, y1, x2, y2 } = geometry ?? { x1: 0, y1: 0, x2: 0, y2: 0 };
  // horizontal control-point extension: proportional to the gap but floored
  // and capped — a compact gutter must not push the curve's shoulders past
  // either endpoint into the row or the pane
  const curve = Math.min(Math.max(6, (x2 - x1) * 0.45), 40);
  const d = `M ${x1} ${y1} C ${x1 + curve} ${y1}, ${x2 - curve} ${y2}, ${x2} ${y2}`;
  const gradientId = `ui-search-pc-${String(activeId).replace(/[^a-zA-Z0-9_-]/g, '')}`;

  return (
    <svg
      aria-hidden="true"
      ref={svgRef}
      style={{ position: 'fixed', top: 0, left: 0, width: '100%', height: '100%', pointerEvents: 'none', zIndex: 20 }}
    >
      <style>{`
        .ui-search-pc-draw {
          stroke-dasharray: 1;
          animation: ui-search-pc-draw 0.45s cubic-bezier(0.22, 1, 0.36, 1) both;
        }
        .ui-search-pc-fade {
          animation: ui-search-pc-fade 0.3s ease-out both;
        }
        @keyframes ui-search-pc-draw {
          /* starting exactly at the dash length lets a zero-length dash with a
             round cap paint a stray dot in some engines — start just past it */
          from { stroke-dashoffset: 1.02; }
          to { stroke-dashoffset: 0; }
        }
        @keyframes ui-search-pc-fade {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @media (prefers-reduced-motion: reduce) {
          .ui-search-pc-draw, .ui-search-pc-fade { animation: none; }
        }
      `}</style>
      {geometry ? (
        <g className="ui-search-pc-fade" key={String(activeId)}>
          <defs>
            <linearGradient gradientUnits="userSpaceOnUse" id={gradientId} x1={x1} x2={x2} y1={0} y2={0}>
              <stop offset={0} style={{ stopColor: 'var(--ant-color-primary)' }} stopOpacity={0.15} />
              <stop offset={0.4} style={{ stopColor: 'var(--ant-color-primary)' }} stopOpacity={0.55} />
              <stop offset={1} style={{ stopColor: 'var(--ant-color-primary)' }} stopOpacity={0.9} />
            </linearGradient>
          </defs>
          {/* soft glow under the line */}
          <path
            className="ui-search-pc-draw"
            d={d}
            fill="none"
            pathLength={1}
            style={{ stroke: 'var(--ant-color-primary)' }}
            strokeOpacity={0.12}
            strokeWidth={5}
            strokeLinecap="round"
          />
          <path
            className="ui-search-pc-draw"
            d={d}
            fill="none"
            pathLength={1}
            stroke={`url(#${gradientId})`}
            strokeWidth={1.5}
            strokeLinecap="round"
          />
          <circle cx={x1} cy={y1} opacity={0.15} r={5} style={{ fill: 'var(--ant-color-primary)' }} />
          <circle cx={x1} cy={y1} opacity={0.9} r={2.5} style={{ fill: 'var(--ant-color-primary)' }} />
          <circle cx={x2} cy={y2} opacity={0.15} r={6} style={{ fill: 'var(--ant-color-primary)' }} />
          <circle cx={x2} cy={y2} r={3} style={{ fill: 'var(--ant-color-primary)' }} />
        </g>
      ) : null}
    </svg>
  );
}

export default PreviewConnector;
