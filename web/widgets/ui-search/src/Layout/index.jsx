import { Layout, Spin } from "antd";

const { Header, Content } = Layout;

import styles from "./index.module.less"
import { DARK_CLASS } from "../theme/shared";

const BasicLayout = (props) => {
  const {
    rootID,
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

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light'

  return (
    <Spin spinning={loading}>
      <Layout id={rootID} className={`overflow-auto ${styles.uiSearch} h-full bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}>
        {isFirst ? (
          <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] min-h-[calc(100vh)] flex flex-col items-center justify-center">
            <div className={`mb-24px w-210px h-68px`}>{logo}</div>
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
            <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] h-[calc(100vh-72px)]">
              <div
                className={`px-12px py-24px w-full m-auto ${isMobile ? "" : "flex justify-center"}`}
              >
                {!isMobile && <div className="w-200px">{aggregations}</div>}
                <div
                  className={`${isMobile ? "mb-40px w-full" : "w-[calc(100%-500px)] max-w-724px px-40px"}`}
                >
                  <div className="mb-12px">{resultHeader}</div>
                  {aiOverview && <div className="mb-16px">{aiOverview}</div>}
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
      </Layout>
    </Spin>
  );
};

export default BasicLayout;
