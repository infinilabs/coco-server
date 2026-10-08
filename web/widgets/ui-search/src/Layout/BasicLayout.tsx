import { Layout } from 'antd';
import styles from './index.module.less';
import { DARK_CLASS } from '../theme/shared';
import { type FC, type ReactElement, type ReactNode, cloneElement, useEffect, useLayoutEffect, useRef, useState } from 'react';
import useNProgress from '../hooks/useNProgress';
import CommonDrawer from './CommonDrawer';
import SearchHeaderLayout from './SearchHeaderLayout';
import BackTopButton from './BackTopButton';

const { Content, Sider } = Layout;

// inline-preview geometry: the center column keeps EXACTLY its no-preview
// width (same cap, same padding) — opening a preview must not change the
// result list's width in either direction. The pane takes all the leftover
// width (floored and capped for readability); only when the container is too
// narrow does the center give up width, and the pane narrows to its floor
// before that happens
const PREVIEW_PANE_MIN = 480;
const PREVIEW_PANE_MAX = 1920;
// with the pane open the connector only needs a slim gutter: the column trims
// its right padding from the no-preview 88px down to this and the pane absorbs
// the difference — the padded box (the result cards) keeps its exact
// no-preview width, only the dead space between list and pane shrinks
const PREVIEW_GUTTER_FULL = 88;
const PREVIEW_GUTTER_COMPACT = 32;

const computePreviewGeometry = (available: number, centerKeep: number) => {
  const center = Math.min(centerKeep, Math.max(available - PREVIEW_PANE_MIN, 0));
  const pane = Math.min(available - center, PREVIEW_PANE_MAX);
  return { pane, center };
};

interface BasicLayoutProps {
  readonly initContainer?: (ref: HTMLDivElement | null) => void;
  readonly getContainer?: () => HTMLElement | null;
  readonly loading?: boolean;
  readonly logo?: ReactNode;
  readonly searchbox?: ReactNode;
  readonly tabs?: ReactNode;
  readonly tools?: ReactNode;
  readonly toolbar?: ReactNode;
  readonly aggregations?: ReactNode;
  /** removable chips for the active facet selection, shown above the result header */
  readonly filterChips?: ReactNode;
  readonly resultHeader?: ReactElement;
  readonly aiOverview?: ReactNode;
  readonly resultList?: ReactNode;
  readonly recommends?: ReactNode;
  readonly hasRecommendsData?: boolean;
  readonly isMobile?: boolean;
  readonly theme?: string;
  readonly siderCollapse?: boolean;
  readonly setSiderCollapse?: (v: boolean) => void;
  readonly rightMenuWidth?: number;
  readonly recommendsCollapse?: boolean;
  readonly setRecommendsCollapse?: (v: boolean) => void;
  readonly histogram?: ReactNode;
  /** in-flow preview pane that takes over the right column from recommends */
  readonly preview?: ReactNode;
  readonly previewWidth?: number;
  [key: string]: any;
}

const BasicLayout: FC<BasicLayoutProps> = props => {
  const {
    initContainer,
    getContainer,
    loading,
    logo,
    searchbox,
    tabs,
    tools,
    toolbar,
    aggregations,
    filterChips,
    resultHeader,
    aiOverview,
    resultList,
    recommends,
    hasRecommendsData = true,
    isMobile,
    theme,
    siderCollapse,
    setSiderCollapse,
    rightMenuWidth,
    recommendsCollapse,
    setRecommendsCollapse,
    histogram,
    preview,
    previewWidth
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer?.() ?? null;
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const [rightDrawerOpen, setRightDrawerOpen] = useState(false);
  const [headerScrolled, setHeaderScrolled] = useState(false);
  const previewRequested = Boolean(preview);
  // the inline pane folds to a full-page preview (below the header) when the
  // container is too narrow to keep it beside the result list; mobile always
  // gets the full-page variant — the geometry code below skips it entirely
  const [previewCollapsed, setPreviewCollapsed] = useState(false);
  // measured pane/center widths for the open preview (see computePreviewGeometry)
  const [previewGeometry, setPreviewGeometry] = useState<{ paneWidth: number; centerWidth: number } | null>(null);
  const hasPreview = previewRequested && !isMobile && !previewCollapsed;
  const previewFullScreen = previewRequested && !hasPreview;
  const previewPaneWidth = previewWidth ?? previewGeometry?.paneWidth ?? 720;

  useNProgress(loading);

  const bgClass = 'bg-[rgb(var(--ui-search--layout-bg-color))]';

  // Use refs to track current collapse state without re-creating observer
  const siderCollapseRef = useRef(siderCollapse);
  const recommendsCollapseRef = useRef(recommendsCollapse);
  const previewCollapsedRef = useRef(false);
  const userCollapsedLeftRef = useRef(false);
  const userCollapsedRightRef = useRef(false);

  useEffect(() => {
    siderCollapseRef.current = siderCollapse;
  }, [siderCollapse]);
  useEffect(() => {
    recommendsCollapseRef.current = recommendsCollapse;
  }, [recommendsCollapse]);
  useEffect(() => {
    previewCollapsedRef.current = previewCollapsed;
  }, [previewCollapsed]);

  // Auto-collapse/expand based on container width
  // Priority: collapse right first, then left; expand left first, then right
  // Layout effect (not useEffect): the geometry must settle BEFORE the browser
  // paints the frame where the pane mounts — a passive effect would paint one
  // intermediate frame with the fallback pane width and the un-trimmed column
  // cap, which reads as the page flashing when a preview opens or closes
  useLayoutEffect(() => {
    const container = scrollContainer;
    if (!container || isMobile) return;

    const LEFT_WIDTH = 240;
    const RIGHT_WIDTH = 400;
    // a center column narrower than this reads as squeezed between the side
    // columns — fold a side column away instead. Fold priority as width
    // shrinks: the preview pane narrows first (down to PREVIEW_PANE_MIN),
    // then the facet rail folds, then the pane gives up to the overlay
    // fallback; the center column only shrinks once it is the last column
    // standing
    const MIN_CENTER = 640;

    const handleResize = () => {
      const totalWidth = container.clientWidth;
      const facetRailWidth = aggregations ? LEFT_WIDTH : 0;
      // the no-preview column cap for the current sidebar configuration — the
      // center must keep exactly this width when the pane opens
      const centerKeep =
        (aggregations && !siderCollapseRef.current) ||
        (recommends && hasRecommendsData && !recommendsCollapseRef.current)
          ? 840
          : 1120;

      // inline geometry budgets the compact gutter, not the full one — the
      // pane takes the width the trimmed right padding frees up
      const inlineKeep = centerKeep - (PREVIEW_GUTTER_FULL - PREVIEW_GUTTER_COMPACT);
      const withFacet = computePreviewGeometry(totalWidth - facetRailWidth, inlineKeep);
      const withoutFacet = computePreviewGeometry(totalWidth, inlineKeep);

      let previewInline = false;
      let foldFacetForPreview = false;
      if (previewRequested) {
        if (withFacet.center >= MIN_CENTER) {
          previewInline = true;
        } else if (aggregations && withoutFacet.center >= MIN_CENTER) {
          previewInline = true;
          foldFacetForPreview = true;
        }
      }
      const nextPreviewCollapsed = previewRequested && !previewInline;

      // Calculate what fits
      const fitsLeftAndRight = totalWidth - LEFT_WIDTH - RIGHT_WIDTH >= MIN_CENTER;
      const fitsLeftOnly = totalWidth - LEFT_WIDTH >= MIN_CENTER;
      const fitsRightOnly = totalWidth - RIGHT_WIDTH >= MIN_CENTER;
      const hasRecommendsPane = !previewInline && Boolean(recommends && hasRecommendsData);

      // no facet data → the left column has nothing to show; collapse it so
      // the center column (search box + empty state) gets the full width
      let targetLeftCollapse;
      let targetRightCollapse;

      if (previewInline) {
        // the pane owns the right column while open; the facet rail folds only
        // when the center cannot stay readable beside the pane
        targetLeftCollapse = foldFacetForPreview;
        targetRightCollapse = true;
      } else if (fitsLeftAndRight && aggregations && hasRecommendsPane) {
        targetLeftCollapse = false;
        targetRightCollapse = false;
      } else if (fitsLeftOnly && aggregations) {
        targetLeftCollapse = false;
        targetRightCollapse = true;
      } else if (fitsRightOnly && hasRecommendsPane) {
        targetLeftCollapse = true;
        targetRightCollapse = false;
      } else {
        targetLeftCollapse = true;
        targetRightCollapse = true;
      }

      // Only update if changed (compare with ref to get current value)
      if (previewCollapsedRef.current !== nextPreviewCollapsed) {
        setPreviewCollapsed(nextPreviewCollapsed);
      }
      setPreviewGeometry(prev => {
        const next = previewInline ? (foldFacetForPreview ? withoutFacet : withFacet) : null;
        const geometry = next ? { paneWidth: next.pane, centerWidth: next.center } : null;
        return prev?.paneWidth === geometry?.paneWidth && prev?.centerWidth === geometry?.centerWidth
          ? prev
          : geometry;
      });
      if (siderCollapseRef.current !== targetLeftCollapse) {
        if (targetLeftCollapse === false && userCollapsedLeftRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetLeftCollapse) userCollapsedLeftRef.current = false;
          setSiderCollapse?.(targetLeftCollapse);
        }
      }
      if (hasRecommendsPane && recommendsCollapseRef.current !== targetRightCollapse) {
        if (targetRightCollapse === false && userCollapsedRightRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetRightCollapse) userCollapsedRightRef.current = false;
          setRecommendsCollapse?.(targetRightCollapse);
        }
      }
    };

    const observer = new ResizeObserver(handleResize);
    observer.observe(container);
    // the scroll container's ResizeObserver entry can go undelivered in
    // embedded webviews even though the container reflows — the window resize
    // event is a second, independent notification path for the same change
    window.addEventListener('resize', handleResize);
    // the observer's initial delivery is not guaranteed to arrive (or may fire
    // before the facet/recommend data lands, e.g. triggered by a scrollbar
    // appearing) — converge synchronously on every effect run so the columns
    // settle even when the container never resizes again
    handleResize();

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', handleResize);
    };
  }, [
    scrollContainer,
    isMobile,
    aggregations,
    recommends,
    hasRecommendsData,
    setSiderCollapse,
    setRecommendsCollapse,
    previewRequested,
    previewWidth
  ]);

  // Close drawers when collapse state changes to collapsed
  useEffect(() => {
    if (siderCollapse) setLeftDrawerOpen(false);
  }, [siderCollapse]);

  useEffect(() => {
    if (recommendsCollapse) setRightDrawerOpen(false);
  }, [recommendsCollapse]);

  const showLeftSider = !siderCollapse && !isMobile && Boolean(aggregations);
  const showRightSider = Boolean(recommends && hasRecommendsData && !recommendsCollapse && !isMobile);

  // center column width/padding: the content box keeps EXACTLY its no-preview
  // width (same card width, same left alignment) — with the pane open only the
  // right gutter trims to PREVIEW_GUTTER_COMPACT, so the pane sits close to the
  // list and absorbs the freed width instead of dead space. The header tracks
  // the same geometry to stay aligned with the result rows.
  let contentMaxWidth = 'max-w-840px';
  if (!showLeftSider && !showRightSider) contentMaxWidth = 'max-w-1120px';
  // with the pane open the measured geometry caps the center at the width it
  // keeps beside the pane (its no-preview width minus the gutter trim) — an
  // inline style so the dynamic pixel value wins over the static tailwind cap
  // above; the header tracks the same value
  const previewCenterMaxWidth = hasPreview && previewGeometry ? previewGeometry.centerWidth : undefined;
  let bodyPadding = 'pl-48px pr-88px';
  let headerPadding = 'pl-64px pr-104px';
  if (hasPreview) {
    bodyPadding = 'pl-48px pr-32px';
    headerPadding = 'pl-64px pr-48px';
  }
  if (isMobile) {
    bodyPadding = 'px-0px';
    headerPadding = 'px-16px';
  }

  const siderProps = {
    breakpoint: 'md' as const,
    collapsedWidth: 0,
    trigger: null,
    className: bgClass
  };

  return (
    <Layout
      className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-auto ${bgClass} ui-search ${themeClass}`}
      ref={initContainer}
      onScroll={event => setHeaderScrolled(event.currentTarget.scrollTop > 0)}
    >
      <SearchHeaderLayout
        centerMaxWidth={contentMaxWidth}
        centerMaxWidthPx={previewCenterMaxWidth}
        centerPadding={headerPadding}
        isMobile={isMobile}
        leftWidth={240}
        logo={logo}
        rightMenuWidth={rightMenuWidth}
        rightWidth={hasPreview ? previewPaneWidth : showRightSider ? 400 : isMobile ? 0 : rightMenuWidth || 0}
        scrolled={headerScrolled}
        searchbox={searchbox}
        showLeftSider={showLeftSider}
        showRightSider={showRightSider || hasPreview}
        tabs={tabs}
        tools={tools}
      />

      {/* Unified Left-Center-Right Layout */}
      <Layout
        className={bgClass}
        style={{ minHeight: '100%', paddingTop: isMobile ? '122px' : '64px' }}
      >
        {/* Left Column: Logo + Aggregations — entirely absent when there are no facets */}
        {aggregations ? (
          isMobile || siderCollapse ? (
            <CommonDrawer
              getContainer={getContainer}
              open={leftDrawerOpen}
              placement='left'
              size={280}
              classNames={{
                wrapper: `!left-0px !bottom-0px ${isMobile ? '!top-122px' : '!top-64px'}`,
                body: '!p-16px'
              }}
              onClose={() => setLeftDrawerOpen(false)}
            >
              {aggregations}
            </CommonDrawer>
          ) : (
            <Sider
              width={240}
              {...siderProps}
              style={{ overflow: 'visible' }}
            >
              {/* 64px aligns the facet rail with the result cards' text edge
                  (48px column padding + 16px card padding) */}
              <div className='w-full pl-64px pt-24px'>{aggregations}</div>
            </Sider>
          )
        ) : null}

        {/* Center Column: Search/Tabs + Results */}
        <Content
          className={`${bgClass} ${isMobile ? 'min-w-0' : 'min-w-400px'} ${contentMaxWidth}`}
          style={{ overflow: 'visible', maxWidth: previewCenterMaxWidth }}
        >
          {/* Content part */}
          <div className={`py-24px ${bodyPadding}`}>
            {filterChips ? <div className='mb-12px px-16px'>{filterChips}</div> : null}
            <div className='mb-16px px-16px'>
              {resultHeader &&
                cloneElement(resultHeader as ReactElement<Record<string, any>>, {
                  hasRecommends: Boolean(recommends && hasRecommendsData),
                  userCollapsedLeft: userCollapsedLeftRef.current,
                  userCollapsedRight: userCollapsedRightRef.current,
                  setSiderCollapse: (v: boolean) => {
                    userCollapsedLeftRef.current = Boolean(v);
                    setSiderCollapse?.(v);
                  },
                  setRecommendsCollapse: (v: boolean) => {
                    userCollapsedRightRef.current = Boolean(v);
                    setRecommendsCollapse?.(v);
                  },
                  leftDrawerOpen,
                  setLeftDrawerOpen,
                  rightDrawerOpen,
                  setRightDrawerOpen
                })}
            </div>
            {/* No enter/exit animation here on purpose: a motion.div animating
                height 0 -> 'auto' can stall at height 0, which takes the
                histogram out of layout flow — the result list then renders on
                top of it and swallows every pointer event, breaking brush
                selection. A plain conditional keeps the geometry correct in
                every state. */}
            {histogram ? <div className='mb-16px px-16px'>{histogram}</div> : null}
            {aiOverview}
            <div className={isMobile ? 'px-16px' : ''}>{resultList}</div>
          </div>
        </Content>

        {/* Right Column: preview pane, or Spacer + Recommends. An open preview
            takes the column over from recommends — the pane is pinned with
            fixed positioning to the full height below the 64px widget header
            (it anchors against the same containing block as the header, so it
            works both embedded and standalone) and scrolls internally, so the
            result list beside it stays operable and the pane never participates
            in the page scroll. */}
        {hasPreview ? (
          // a plain flex spacer, not antd Sider — Sider animates width changes
          // and that transition stalls mid-frame in embedded webviews, leaving
          // the column at its previous width while the pane has already moved
          <div
            className={`${bgClass} flex-none`}
            style={{ width: previewPaneWidth }}
          >
            <div
              className='right-0 bottom-0 fixed z-10 border-l border-solid border-[var(--ant-color-border-secondary)]'
              data-preview-pane=''
              style={{ width: previewPaneWidth, top: 64 }}
            >
              {preview}
            </div>
          </div>
        ) : previewFullScreen ? (
          // container too narrow for the inline pane (or mobile viewport) —
          // the preview takes over the whole page below the header instead of
          // floating over the list as a card/drawer; closing it returns to the
          // plain result list. No data-preview-pane marker on purpose: a
          // leader line has no pane edge to point at and hides itself
          <div
            className={`left-0 right-0 bottom-0 fixed z-20 overflow-hidden ${bgClass}`}
            style={{ top: isMobile ? 122 : 64 }}
          >
            {preview}
          </div>
        ) : (
          recommends &&
          hasRecommendsData &&
          (isMobile || recommendsCollapse ? (
            <CommonDrawer
              getContainer={getContainer}
              open={rightDrawerOpen}
              placement='right'
              size={400}
              classNames={{
                wrapper: `!right-0px !bottom-0px ${isMobile ? '!top-122px' : '!top-64px'}`,
                body: '!p-16px'
              }}
              onClose={() => setRightDrawerOpen(false)}
            >
              {recommends}
            </CommonDrawer>
          ) : (
            <Sider
              width={400}
              {...siderProps}
              style={{ overflow: 'visible' }}
            >
              {/* Content part */}
              <div className='flex flex-col flex-1 gap-16px pt-32px'>{recommends}</div>
            </Sider>
          ))
        )}
        {/* Mount recommends hidden for data fetching when no data yet, or while
            the preview pane occupies the column — closing the preview restores
            the pane instantly instead of refetching */}
        {recommends && (!hasRecommendsData || hasPreview) && <div className='hidden'>{recommends}</div>}
      </Layout>

      <BackTopButton
        getContainer={getContainer}
        loading={loading}
      />
    </Layout>
  );
};

export default BasicLayout;
