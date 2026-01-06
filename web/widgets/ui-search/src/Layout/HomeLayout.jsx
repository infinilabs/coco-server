import { Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import GlobalLoading from "../GlobalLoading";

const { Content } = Layout;

const HomeLayout = (props) => {
  const {
    loading, 
    logo,
    welcome,
    searchbox,
    isMobile,
    theme,
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';

  return (
      <Layout 
        className={`${styles.uiSearch} relative w-full h-100vh overflow-x-hidden overflow-y-auto bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
        style={{ 
          height: '100vh',
          overflowY: 'hidden',
        }}
      >
        <GlobalLoading loading={loading} theme={theme} />
        <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] h-[calc(100vh)] flex flex-col items-center justify-start pt-[calc(100vh/4)]">
          <div className={`max-w-320px max-h-320px`}>{logo}</div>
          {welcome && (
            <div
              className={`${isMobile ? "w-full px-32px" : "w-627px"} mt-24px`}
            >
              {welcome}
            </div>
          )}
          <div className={`${isMobile ? "w-full px-24px" : "w-720px"} mt-80px`}>
            {searchbox}
          </div>
        </Content>
      </Layout>
  );
};

export default HomeLayout;