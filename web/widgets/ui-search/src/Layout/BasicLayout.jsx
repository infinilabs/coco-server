import { FloatButton, Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { useCallback, useEffect, useState } from "react";
import GlobalLoading from "../GlobalLoading";

const { Header, Content, Sider } = Layout;

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
    siderCollapse
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer();
  const [backTopShow, setBackTopShow] = useState(false);

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

  return (
    <Layout
      ref={initContainer}
      className={`${styles.uiSearch} relative w-full h-100vh overflow-x-hidden overflow-y-auto bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      style={{
        height: '100vh',
        overflowY: loading ? 'hidden' : 'auto',
        paddingTop: '122px',
        boxSizing: 'border-box'
      }}
    >
      <GlobalLoading loading={loading} theme={theme} />
      <Header className="fixed top-0 left-0 right-0 z-1001 !p-0 !h-auto bg-[rgb(var(--ui-search--layout-bg-color))] border-b border-[var(--ui-search-antd-color-border-secondary)]">
        <Layout className="bg-[rgb(var(--ui-search--layout-bg-color))]">
          <Sider width={280} breakpoint="md" collapsedWidth={0} trigger={null}>
            <div className={`position-sticky top-0 z-10 bg-[rgb(var(--ui-search--layout-bg-color))] pt-16px h-122px w-full pl-80px`}>
              <div className={`h-48px w-full`}>
                {logo}
              </div>
            </div>
          </Sider>
          <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] min-w-400px max-w-840px" style={{ overflow: 'visible' }}>
            <div className={`position-sticky top-0 z-10 bg-[rgb(var(--ui-search--layout-bg-color))] pt-16px h-122px ${isMobile ? 'px-16px' : 'pl-56px pr-96px'}`}>
              <div className={`flex gap-8px items-center`}>
                {isMobile && (
                  <div className={`h-40px w-40px`}>
                    {logo}
                  </div>
                )}
                <div className={`flex-1 ${isMobile ? '' : 'px-24px'}`}>
                  {searchbox}
                </div>
              </div>
              {
                tabs && (
                  <div className={`w-full pt-12px ${isMobile ? '' : 'px-24px'} flex items-center justify-between`}>
                    <div>
                      {tabs}
                    </div>
                    <div>
                      {tools}
                    </div>
                  </div>
                )
              }
            </div>
          </Content>
          <Sider width={400} className="bg-[rgb(var(--ui-search--layout-bg-color))]" breakpoint="md" collapsedWidth={0} trigger={null}>
            <div className="position-sticky top-0 z-10 bg-[rgb(var(--ui-search--layout-bg-color))] pt-16px h-122px"></div>
          </Sider>
        </Layout>
      </Header>
      <Content className="bg-[rgb(var(--ui-search--layout-bg-color))]">
        <Layout className="bg-[rgb(var(--ui-search--layout-bg-color))]">
          {
            aggregations && !siderCollapse && (
              <Sider width={280} className="bg-[rgb(var(--ui-search--layout-bg-color))]" breakpoint="md" collapsedWidth={0} trigger={null}>
                <div className="w-full pl-80px pt-32px">{aggregations}</div>
              </Sider>
            )
          }
          <Content className={`bg-[rgb(var(--ui-search--layout-bg-color))] min-w-400px ${aggregations && !siderCollapse ? 'max-w-840px' : 'max-w-1120px'}`} style={{ overflow: 'visible' }}>
            <div className={`pt-32px ${isMobile ? 'px-16px' : 'pl-56px pr-96px'}`}>
              {toolbar && <div className="pl-24px mb-16px">{toolbar}</div>}
              <div className={`${isMobile ? '' : 'pl-24px'} mb-16px`}>{resultHeader}</div>
              {aiOverview && <div className="mb-12px">{aiOverview}</div>}
              <div className="mb-24px">{resultList}</div>
            </div>
          </Content>
          <Sider width={400} className="bg-[rgb(var(--ui-search--layout-bg-color))]" breakpoint="md" collapsedWidth={0} trigger={null}>
            <div className={`${isMobile ? "w-full" : "flex-1"} flex flex-col gap-16px pt-32px`}>
              {recommends}
            </div>
          </Sider>
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

export default BasicLayout;