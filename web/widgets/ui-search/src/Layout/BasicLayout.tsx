import { FloatButton, Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { cloneElement, useCallback, useEffect, useRef, useState, type ReactNode, type FC, type ReactElement } from "react";
import useNProgress from "../hooks/useNProgress";
import CommonDrawer from "./CommonDrawer";
import SearchHeaderLayout from "./SearchHeaderLayout";

const { Content, Sider } = Layout;

interface BasicLayoutProps {
  initContainer?: (ref: HTMLDivElement | null) => void;
  getContainer?: () => HTMLElement | null;
  loading?: boolean;
  logo?: ReactNode;
  searchbox?: ReactNode;
  tabs?: ReactNode;
  tools?: ReactNode;
  toolbar?: ReactNode;
  aggregations?: ReactNode;
  resultHeader?: ReactElement;
  aiOverview?: ReactNode;
  resultList?: ReactNode;
  recommends?: ReactNode;
  hasRecommendsData?: boolean;
  isMobile?: boolean;
  theme?: string;
  siderCollapse?: boolean;
  setSiderCollapse?: (v: boolean) => void;
  rightMenuWidth?: number;
  recommendsCollapse?: boolean;
  setRecommendsCollapse?: (v: boolean) => void;
  [key: string]: any;
}

const BasicLayout: FC<BasicLayoutProps> = (props) => {
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
    setRecommendsCollapse
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer?.() ?? null;
  const [backTopShow, setBackTopShow] = useState(false);
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const [rightDrawerOpen, setRightDrawerOpen] = useState(false);

  const handleContainerScroll = useCallback(() => {
    if (!scrollContainer || loading) return;

    setBackTopShow(scrollContainer.scrollTop > 400);
  }, [scrollContainer, loading]);

  useEffect(() => {
    if (!scrollContainer) {
      setBackTopShow(false);
      return;
    }

    scrollContainer.addEventListener("scroll", handleContainerScroll);
    handleContainerScroll();

    return () => {
      scrollContainer.removeEventListener("scroll", handleContainerScroll);
    };
  }, [scrollContainer, handleContainerScroll]);

  const handleBackTopClick = useCallback(() => {
    if (!scrollContainer || loading) return;
    scrollContainer.scrollTo({
      top: 0,
      behavior: "smooth",
    });
  }, [scrollContainer, loading]);

  useNProgress(loading);

  const bgClass = 'bg-[rgb(var(--ui-search--layout-bg-color))]';

  // Use refs to track current collapse state without re-creating observer
  const siderCollapseRef = useRef(siderCollapse);
  const recommendsCollapseRef = useRef(recommendsCollapse);
  const userCollapsedLeftRef = useRef(false);
  const userCollapsedRightRef = useRef(false);

  useEffect(() => { siderCollapseRef.current = siderCollapse; }, [siderCollapse]);
  useEffect(() => { recommendsCollapseRef.current = recommendsCollapse; }, [recommendsCollapse]);

  // Auto-collapse/expand based on container width
  // Priority: collapse right first, then left; expand left first, then right
  useEffect(() => {
    const container = scrollContainer;
    if (!container || isMobile) return;

    const LEFT_WIDTH = 280;
    const RIGHT_WIDTH = 400;
    const MIN_CENTER = 450;

    const handleResize = () => {
      const totalWidth = container.clientWidth;

      // Calculate what fits
      const fitsLeftAndRight = totalWidth - LEFT_WIDTH - RIGHT_WIDTH >= MIN_CENTER;
      const fitsLeftOnly = totalWidth - LEFT_WIDTH >= MIN_CENTER;

      let targetLeftCollapse, targetRightCollapse;

      if (fitsLeftAndRight && recommends && hasRecommendsData) {
        targetLeftCollapse = false;
        targetRightCollapse = false;
      } else if (fitsLeftOnly ) {
        targetLeftCollapse = false;
        targetRightCollapse = true;
      } else {
        targetLeftCollapse = true;
        targetRightCollapse = true;
      }

      // Only update if changed (compare with ref to get current value)
      if (siderCollapseRef.current !== targetLeftCollapse) {
        if (targetLeftCollapse === false && userCollapsedLeftRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetLeftCollapse) userCollapsedLeftRef.current = false;
          setSiderCollapse?.(targetLeftCollapse);
        }
      }
      if (recommends && hasRecommendsData && recommendsCollapseRef.current !== targetRightCollapse) {
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

    return () => observer.disconnect();
  }, [scrollContainer, isMobile, recommends, hasRecommendsData, setSiderCollapse, setRecommendsCollapse]);

  // Close drawers when collapse state changes to collapsed
  useEffect(() => {
    if (siderCollapse) setLeftDrawerOpen(false);
  }, [siderCollapse]);

  useEffect(() => {
    if (recommendsCollapse) setRightDrawerOpen(false);
  }, [recommendsCollapse]);

  const showLeftSider = !siderCollapse && !isMobile;
  const showRightSider = !!(recommends && hasRecommendsData && !recommendsCollapse && !isMobile);

  const siderProps = {
    breakpoint: 'md' as const,
    collapsedWidth: 0,
    trigger: null,
    className: bgClass,
  };

  return (
    <Layout
      ref={initContainer}
      className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-auto ${bgClass} ui-search ${themeClass}`}
    >

      <SearchHeaderLayout
        logo={logo}
        searchbox={searchbox}
        tabs={tabs}
        tools={tools}
        isMobile={isMobile}
        showLeftSider={showLeftSider}
        showRightSider={showRightSider}
        leftWidth={280}
        rightWidth={showRightSider ? 400 : (isMobile ? 0 : (rightMenuWidth || 0))}
        centerPadding={isMobile ? 'px-16px' : 'pl-72px pr-112px'}
        centerMaxWidth={'max-w-840px'}
        rightMenuWidth={rightMenuWidth}
      />

      {/* Unified Left-Center-Right Layout */}
      <Layout className={bgClass} style={{ minHeight: '100%', paddingTop: '122px' }}>
        {/* Left Column: Logo + Aggregations */}
        {(
          isMobile || siderCollapse ? (
            <CommonDrawer
              placement="left"
              open={leftDrawerOpen}
              onClose={() => setLeftDrawerOpen(false)}
              getContainer={getContainer}
              classNames={{
                wrapper: '!left-0px !top-122px !bottom-0px',
                body: '!p-16px',
              }}
              size={280}
            >
              {aggregations}
            </CommonDrawer>
          ) : (
            <Sider width={280} {...siderProps} style={{ overflow: 'visible' }}>
              {/* Content part */}
              <div className="w-full pl-80px pt-32px">{aggregations}</div>
            </Sider>
          )
        )}

        {/* Center Column: Search/Tabs + Results */}
        <Content
          className={`${bgClass} min-w-400px ${showLeftSider && showRightSider ? 'max-w-840px' : !showLeftSider && !showRightSider ? '' : 'max-w-840px'}`}
          style={{ overflow: 'visible' }}
        >
          {/* Content part */}
          <div className={`py-32px ${isMobile ? 'px-0px' : 'pl-56px pr-96px'}`}>
            {toolbar && <div className="pl-16px mb-16px">{toolbar}</div>}
            <div className="px-16px mb-16px">
              {resultHeader && cloneElement(resultHeader, {
                hasRecommends: !!(recommends && hasRecommendsData),
                userCollapsedLeft: userCollapsedLeftRef.current,
                userCollapsedRight: userCollapsedRightRef.current,
                setSiderCollapse: (v: boolean) => { userCollapsedLeftRef.current = !!v; setSiderCollapse?.(v); },
                setRecommendsCollapse: (v: boolean) => { userCollapsedRightRef.current = !!v; setRecommendsCollapse?.(v); },
                leftDrawerOpen,
                setLeftDrawerOpen,
                rightDrawerOpen,
                setRightDrawerOpen
              })}
            </div>
            {aiOverview}
            <div className="mb-24px">{resultList}</div>
          </div>
        </Content>

        {/* Right Column: Spacer + Recommends */}
        {recommends && hasRecommendsData && (
          isMobile || recommendsCollapse ? (
            <CommonDrawer
              placement="right"
              open={rightDrawerOpen}
              onClose={() => setRightDrawerOpen(false)}
              getContainer={getContainer}
              classNames={{
                wrapper: '!right-0px !top-122px !bottom-0px',
                body: '!p-16px',
              }}
              size={400}
            >
              {recommends}
            </CommonDrawer>
          ) : (
            <Sider width={400} {...siderProps} style={{ overflow: 'visible' }}>
              {/* Content part */}
              <div className="flex-1 flex flex-col gap-16px pt-32px">
                {recommends}
              </div>
            </Sider>
          )
        )}
        {/* Mount recommends hidden for data fetching when no data yet */}
        {recommends && !hasRecommendsData && <div className="hidden">{recommends}</div>}
      </Layout>

      {scrollContainer && backTopShow && !loading && (
        <FloatButton.BackTop
          target={() => scrollContainer}
          visibilityHeight={0}
          duration={300}
          onClick={handleBackTopClick}
          className="!border-[#F0F0F0] !dark:border-[#303030] "
          style={{
            right: 24,
            bottom: 24,
            zIndex: 999,
            display: backTopShow ? 'flex' : 'none',
          }}
        />
      )}
    </Layout>
  );
};

export default BasicLayout;