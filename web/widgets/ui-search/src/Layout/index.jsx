import { FloatButton, Layout, Spin } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import { useCallback, useEffect, useState } from "react";

const { Header, Content } = Layout;

const BasicLayout = (props) => {
  const {
    initContainer,
    getContainer,
    isFirst,
    loading, 
    logo,
    welcome,
    searchbox,
    rightMenuWidth,
    aggregations,
    resultHeader,
    aiOverview,
    resultList,
    widgets,
    isMobile,
    theme,
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const scrollContainer = getContainer();
  const [backTopShow, setBackTopShow] = useState(false);

  const handleContainerScroll = useCallback(() => {
    if (!scrollContainer || isFirst || loading) return;

    setBackTopShow(scrollContainer.scrollTop > 400);
  }, [scrollContainer, isFirst, loading]);

  useEffect(() => {
    if (!scrollContainer || isFirst) {
      setBackTopShow(false);
      return;
    }

    scrollContainer.addEventListener("scroll", handleContainerScroll);
    handleContainerScroll();

    return () => {
      scrollContainer.removeEventListener("scroll", handleContainerScroll);
    };
  }, [scrollContainer, handleContainerScroll, isFirst]);

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
          overflowY: loading ? 'hidden' : (isFirst ? 'hidden' : 'auto'),
        }}
      >
        {loading && (
          <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: theme === 'dark' ? 'rgba(0, 0, 0, 0.85)' : 'rgba(255, 255, 255, 0.85)',
            zIndex: 99999,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            pointerEvents: 'auto',
            backdropFilter: 'blur(2px)',
          }}>
            <Spin size="large" tip={theme === 'dark' ? "加载中..." : "Loading..."} />
          </div>
        )}
        {isFirst ? (
          <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] min-h-[calc(100vh)] flex flex-col items-center justify-center">
            <div className={`mb-24px max-w-320px max-h-320px`}>{logo}</div>
            {welcome && (
              <div
                className={`${isMobile ? "w-full px-32px" : "w-627px"} mb-60px`}
              >
                {welcome}
              </div>
            )}
            <div className={`${isMobile ? "w-full px-24px" : "w-664px"}`}>
              {searchbox}
            </div>
          </Content>
        ) : (
          <>
            <Header
              className={`bg-[rgb(var(--ui-search--container-bg-color))] position-sticky top-0 z-1 w-full h-72px shadow-md px-0`}
            >
              <div
                className={`px-12px m-auto h-full flex items-center justify-${isMobile ? "left" : "center"}`}
              >
                <div
                  className={`h-full ${isMobile ? "w-40px mr-8px" : "w-200px"}`}
                >
                  {logo}
                </div>
                <div
                  style={
                    isMobile && rightMenuWidth
                      ? { paddingRight: rightMenuWidth }
                      : {}
                  }
                  className={`h-full ${isMobile ? "w-[calc(100%-48px)]" : "w-[calc(100%-500px)] max-w-724px px-40px"}`}
                >
                  {searchbox}
                </div>
                {!isMobile && <div className="w-300px h-full"></div>}
              </div>
            </Header>
            <Content 
              className="bg-[rgb(var(--ui-search--layout-bg-color))] h-[calc(100vh-72px)]"
              style={{ overflow: 'visible' }}
            >
              <div
                className={`px-12px py-24px w-full m-auto ${isMobile ? "" : "flex justify-center"}`}
              >
                {!isMobile && <div className="w-200px">{aggregations}</div>}
                <div
                  className={`${isMobile ? "mb-40px w-full" : "w-[calc(100%-500px)] max-w-724px px-40px"}`}
                >
                  <div className="mb-12px">{resultHeader}</div>
                  {aiOverview && <div className="mb-28px">{aiOverview}</div>}
                  <div className="mb-24px">{resultList}</div>
                </div>
                <div
                  className={`${isMobile ? "w-full" : "w-300px mt-28px"} flex flex-col gap-16px`}
                >
                  {widgets}
                </div>
              </div>
            </Content>
          </>
        )}
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