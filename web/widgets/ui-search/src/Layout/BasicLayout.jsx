import { Drawer, FloatButton, Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { cloneElement, useCallback, useEffect, useRef, useState } from "react";
import GlobalLoading from "../GlobalLoading";
import SearchHeaderLayout from "./SearchHeaderLayout";

const { Content, Sider } = Layout;

const BasicLayout = (props) => {
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
    isMobile,
    theme,
    siderCollapse,
    setSiderCollapse,
    rightMenuWidth,
    recommendsCollapse,
    setRecommendsCollapse
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer();
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

      if (fitsLeftAndRight && aggregations && recommends) {
        targetLeftCollapse = false;
        targetRightCollapse = false;
      } else if (fitsLeftOnly && aggregations) {
        targetLeftCollapse = false;
        targetRightCollapse = true;
      } else {
        targetLeftCollapse = true;
        targetRightCollapse = true;
      }

      // Only update if changed (compare with ref to get current value)
      if (aggregations && siderCollapseRef.current !== targetLeftCollapse) {
        if (targetLeftCollapse === false && userCollapsedLeftRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetLeftCollapse) userCollapsedLeftRef.current = false;
          setSiderCollapse(targetLeftCollapse);
        }
      }
      if (recommends && recommendsCollapseRef.current !== targetRightCollapse) {
        if (targetRightCollapse === false && userCollapsedRightRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetRightCollapse) userCollapsedRightRef.current = false;
          setRecommendsCollapse(targetRightCollapse);
        }
      }
    };

    const observer = new ResizeObserver(handleResize);
    observer.observe(container);

    return () => observer.disconnect();
  }, [scrollContainer, isMobile, aggregations, recommends, setSiderCollapse, setRecommendsCollapse]);

  // Close drawers when collapse state changes to collapsed
  useEffect(() => {
    if (siderCollapse) setLeftDrawerOpen(false);
  }, [siderCollapse]);

  useEffect(() => {
    if (recommendsCollapse) setRightDrawerOpen(false);
  }, [recommendsCollapse]);

  const showLeftSider = aggregations && !siderCollapse && !isMobile;
  const showRightSider = recommends && !recommendsCollapse && !isMobile;

  const siderProps = {
    breakpoint: 'md',
    collapsedWidth: 0,
    trigger: null,
    className: bgClass,
  };

  return (
    <Layout
      ref={initContainer}
      className={`${styles.uiSearch} relative w-full h-100vh overflow-x-hidden overflow-y-auto ${bgClass} ui-search ${themeClass}`}
      style={{
        height: '100vh',
        overflowY: loading ? 'hidden' : 'auto',
      }}
    >
      <GlobalLoading loading={loading} theme={theme} />

      <SearchHeaderLayout
        logo={logo}
        searchbox={searchbox}
        tabs={tabs}
        tools={tools}
        isMobile={isMobile}
        showLeftSider={showLeftSider}
        showRightSider={showRightSider}
        leftWidth={280}
        rightWidth={isMobile ? 0 : 400}
        centerPadding={isMobile ? 'px-16px' : 'pl-72px pr-112px'}
        centerMaxWidth={'max-w-840px'}
        rightMenuWidth={rightMenuWidth}
      />

      {/* Unified Left-Center-Right Layout */}
      <Layout className={bgClass} style={{ minHeight: '100%', paddingTop: '122px' }}>
        {/* Left Column: Logo + Aggregations */}
        {aggregations && (
          isMobile || siderCollapse ? (
            <Drawer
              placement="left"
              open={leftDrawerOpen}
              onClose={() => setLeftDrawerOpen(false)}
              closeIcon={null}
              getContainer={getContainer}
              push={false}
              classNames={{
                wrapper: `!overflow-hidden !left-12px !top-146px !bottom-24px !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`,
                body: '!p-16px !rounded-12px',
                mask: '!bg-transparent !backdrop-filter-none'
              }}
              maskClosable
              width={280}
              autoFocus={false}
            >
              {aggregations}
            </Drawer>
          ) : (
            <Sider width={280} {...siderProps} style={{ overflow: 'visible' }}>
              {/* Content part */}
              <div className="w-full pl-80px pt-32px">{aggregations}</div>
            </Sider>
          )
        )}

        {/* Center Column: Search/Tabs + Results */}
        <Content
          className={`${bgClass} min-w-400px ${showLeftSider && showRightSider ? 'max-w-840px' : !showLeftSider && !showRightSider ? '' : 'max-w-1120px'}`}
          style={{ overflow: 'visible' }}
        >
          {/* Content part */}
          <div className={`pt-32px ${isMobile ? 'px-0px' : 'pl-56px pr-96px'}`}>
            {toolbar && <div className="pl-16px mb-16px">{toolbar}</div>}
            <div className="px-16px mb-16px">
              {resultHeader && cloneElement(resultHeader, {
                hasRecommends: !!recommends,
                userCollapsedLeft: userCollapsedLeftRef.current,
                userCollapsedRight: userCollapsedRightRef.current,
                setSiderCollapse: (v) => { userCollapsedLeftRef.current = !!v; setSiderCollapse(v); },
                setRecommendsCollapse: (v) => { userCollapsedRightRef.current = !!v; setRecommendsCollapse(v); },
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
        {recommends && (
          isMobile || recommendsCollapse ? (
            <Drawer
              placement="right"
              open={rightDrawerOpen}
              onClose={() => setRightDrawerOpen(false)}
              closeIcon={null}
              getContainer={getContainer}
              destroyOnHidden
              push={false}
              classNames={{
                wrapper: `!overflow-hidden !right-12px !top-146px !bottom-24px !rounded-12px !shadow-[0_2px_20px_rgba(0,0,0,0.1)] !dark:shadow-[0_2px_20px_rgba(255,255,255,0.2)]`,
                body: '!p-16px !rounded-12px',
                mask: '!bg-transparent !backdrop-filter-none'
              }}
              maskClosable
              width={400}
            >
              {recommends}
            </Drawer>
          ) : (
            <Sider width={400} {...siderProps} style={{ overflow: 'visible' }}>
              {/* Content part */}
              <div className="flex-1 flex flex-col gap-16px pt-32px">
                {recommends}
              </div>
            </Sider>
          )
        )}
      </Layout>

      {scrollContainer && backTopShow && !loading && (
        <FloatButton.BackTop
          target={() => scrollContainer}
          visibilityHeight={0}
          duration={300}
          onClick={handleBackTopClick}
          style={{
            right: 24,
            bottom: 24,
            zIndex: 9999,
            display: backTopShow ? 'flex' : 'none',
          }}
        />
      )}
    </Layout>
  );
};

export default BasicLayout;