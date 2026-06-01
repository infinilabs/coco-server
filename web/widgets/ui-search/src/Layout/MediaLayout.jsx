import { Drawer, FloatButton, Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { cloneElement, useCallback, useEffect, useRef, useState } from "react";
import GlobalLoading from "../GlobalLoading";
import SearchHeaderLayout from "./SearchHeaderLayout";

const { Content, Sider } = Layout;

const MediaLayout = (props) => {
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
    resultList,
    isMobile,
    theme,
    siderCollapse,
    setSiderCollapse,
    detailCollapse,
    rightMenuWidth
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer();
  const [backTopShow, setBackTopShow] = useState(false);
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const userCollapsedLeftRef = useRef(false);
  const siderCollapseRef = useRef(siderCollapse);

  useEffect(() => { siderCollapseRef.current = siderCollapse; }, [siderCollapse]);

  // Auto-collapse/expand left sider based on container width
  useEffect(() => {
    const container = scrollContainer;
    if (!container || isMobile) return;

    const LEFT_WIDTH = 280;
    const MIN_CENTER = 450;

    const handleResize = () => {
      const totalWidth = container.clientWidth;
      const fitsLeft = totalWidth - LEFT_WIDTH >= MIN_CENTER;
      const targetLeftCollapse = !fitsLeft || !aggregations;

      if (aggregations && siderCollapseRef.current !== targetLeftCollapse) {
        if (targetLeftCollapse === false && userCollapsedLeftRef.current) {
          // Don't auto-expand if user manually collapsed
        } else {
          if (targetLeftCollapse) userCollapsedLeftRef.current = false;
          setSiderCollapse(targetLeftCollapse);
        }
      }
    };

    const observer = new ResizeObserver(handleResize);
    observer.observe(container);

    return () => observer.disconnect();
  }, [scrollContainer, isMobile, aggregations, setSiderCollapse]);

  // Close drawer when collapse state changes to collapsed
  useEffect(() => {
    if (siderCollapse) setLeftDrawerOpen(false);
  }, [siderCollapse]);

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

  const headerHeight = '122px';

  return (
    <Layout
      ref={initContainer}
      className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-auto bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      style={{
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
        showLeftSider={!isMobile && !siderCollapse && !!aggregations}
        showRightSider={!isMobile}
        leftWidth={280}
        rightWidth={0}
        centerPadding={isMobile ? 'px-16px' : 'pl-40px pr-112px'}
        centerMaxWidth="max-w-840px"
        rightMenuWidth={rightMenuWidth}
      />
      <Content style={{ paddingTop: headerHeight }}>
        <Layout>
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
              <Sider width={280} className="bg-[rgb(var(--ui-search--layout-bg-color))]" breakpoint="md" collapsedWidth={0} trigger={null}>
                <div className="w-full pl-80px pt-32px">{aggregations}</div>
              </Sider>
            )
          )}
          <Content className={`bg-[rgb(var(--ui-search--layout-bg-color))] min-w-400px ${aggregations && !(isMobile || siderCollapse) ? 'w-[calc(100%-280px)]' : 'w-[calc(100%)]'}`} style={{ overflow: 'visible' }}>
            <div className={`pt-32px ${isMobile ? 'px-16px' : 'pl-40px pr-24px'} ${detailCollapse || (isMobile || siderCollapse) ? 'w-full' : 'w-[calc(100%-820px)]'}`}>
              {toolbar && <div className="pl-24px mb-16px">{toolbar}</div>}
              <div className={`mb-16px`}>
                {resultHeader && cloneElement(resultHeader, {
                  userCollapsedLeft: userCollapsedLeftRef.current,
                  setSiderCollapse: (v) => { userCollapsedLeftRef.current = !!v; setSiderCollapse(v); },
                  leftDrawerOpen,
                  setLeftDrawerOpen
                })}
              </div>
              <div className="mb-24px">{resultList}</div>
            </div>
          </Content>
        </Layout>
      </Content>
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
            display: backTopShow ? "flex" : "none",
          }}
        />
      )}
    </Layout>
  );
};

export default MediaLayout;