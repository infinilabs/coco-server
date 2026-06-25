import { Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { cloneElement, useEffect, useRef, useState, type ReactNode, type FC, type ReactElement } from "react";
import useNProgress from "../hooks/useNProgress";
import SearchHeaderLayout from "./SearchHeaderLayout";
import CommonDrawer from "./CommonDrawer";
import BackTopButton from "./BackTopButton";
import { AnimatePresence, motion } from "motion/react";

const { Content, Sider } = Layout;

interface MediaLayoutProps {
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
  resultList?: ReactNode;
  isMobile?: boolean;
  theme?: string;
  siderCollapse?: boolean;
  setSiderCollapse?: (v: boolean) => void;
  detailCollapse?: boolean;
  rightMenuWidth?: number;
  histogram: ReactNode;
  [key: string]: any;
}

const MediaLayout: FC<MediaLayoutProps> = (props) => {
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
    rightMenuWidth,
    histogram
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer?.() ?? null;
  const [leftDrawerOpen, setLeftDrawerOpen] = useState(false);
  const [headerScrolled, setHeaderScrolled] = useState(false);
  const userCollapsedLeftRef = useRef(false);
  const siderCollapseRef = useRef(siderCollapse);

  useEffect(() => { siderCollapseRef.current = siderCollapse; }, [siderCollapse]);

  // Collapse left sider on mount
  useEffect(() => {
    userCollapsedLeftRef.current = true;
    setSiderCollapse?.(true);
  }, []);

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
          setSiderCollapse?.(targetLeftCollapse);
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

  const headerHeight = '122px';

  useNProgress(loading);

  return (
    <Layout
      ref={initContainer}
      className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-auto bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      onScroll={(event) => setHeaderScrolled(event.currentTarget.scrollTop > 0)}
    >
      <SearchHeaderLayout
        logo={logo}
        searchbox={searchbox}
        tabs={tabs}
        tools={tools}
        isMobile={isMobile}
        showLeftSider={!isMobile && !siderCollapse && !!aggregations}
        showRightSider={false}
        leftWidth={280}
        rightWidth={isMobile ? 0 : (rightMenuWidth || 0)}
        centerPadding={isMobile ? 'px-16px' : 'pl-72px pr-112px'}
        centerMaxWidth="max-w-840px"
        rightMenuWidth={rightMenuWidth}
        scrolled={headerScrolled}
      />
      <Content style={{ paddingTop: headerHeight }}>
        <Layout>
          {aggregations && (
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
              <Sider width={280} className="bg-[rgb(var(--ui-search--layout-bg-color))]" breakpoint="md" collapsedWidth={0} trigger={null}>
                <div className="w-full pl-80px pt-32px">{aggregations}</div>
              </Sider>
            )
          )}
          <Content className={`bg-[rgb(var(--ui-search--layout-bg-color))] min-w-400px ${aggregations && !(isMobile || siderCollapse) ? 'w-[calc(100%-280px)]' : 'w-[calc(100%)]'}`} style={{ overflow: 'visible' }}>
            <div className={`py-32px transition-[width] duration-300 ease-in-out ${isMobile ? 'px-16px' : siderCollapse ? 'pl-24px' : 'pl-72px'} pr-24px ${detailCollapse || (isMobile || siderCollapse) ? 'w-full' : 'w-[calc(100%-820px)]'}`}>
              <div className={`mb-16px`}>
                {resultHeader && cloneElement(resultHeader, {
                  userCollapsedLeft: userCollapsedLeftRef.current,
                  setSiderCollapse: (v: boolean) => { userCollapsedLeftRef.current = !!v; setSiderCollapse?.(v); },
                  leftDrawerOpen,
                  setLeftDrawerOpen
                })}
              </div>
              <AnimatePresence initial={false}>
                {histogram ? (
                  <motion.div
                    key="histogram"
                    initial={{ opacity: 0, height: 0, y: -8 }}
                    animate={{ opacity: 1, height: 'auto', y: 0 }}
                    exit={{ opacity: 0, height: 0, y: -8 }}
                    transition={{ duration: 0.16, ease: 'easeOut' }}
                  >
                    <div className="mb-16px">
                      {histogram}
                    </div>
                  </motion.div>
                ) : null}
              </AnimatePresence>
              <div>{resultList}</div>
            </div>
          </Content>
        </Layout>
      </Content>
      <BackTopButton getContainer={getContainer} loading={loading} zIndex={9999} />
    </Layout>
  );
};

export default MediaLayout;