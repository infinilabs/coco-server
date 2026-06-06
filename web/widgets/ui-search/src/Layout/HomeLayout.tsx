import { Layout } from "antd";
import styles from "./index.module.less";
import { DARK_CLASS } from "../theme/shared";
import useNProgress from "../hooks/useNProgress";
import { type FC, type ReactNode } from "react";

const { Content } = Layout;

interface HomeLayoutProps {
  loading?: boolean;
  logo?: ReactNode;
  welcome?: ReactNode;
  searchbox?: ReactNode;
  isMobile?: boolean;
  theme?: string;
  recommends?: ReactNode;
}

const HomeLayout: FC<HomeLayoutProps> = (props) => {
  const {
    loading, 
    logo,
    welcome,
    searchbox,
    isMobile,
    theme,
    recommends
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';

  useNProgress(loading);

  return (
      <Layout 
        className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-hidden bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      >
        <Content className="bg-[rgb(var(--ui-search--layout-bg-color))] w-full h-full flex flex-col items-center justify-start absolute top-15% left-0">
          <div className={`max-w-320px max-h-56px`}>{logo}</div>
          {welcome && (
            <div
              className={`${isMobile ? "w-full px-32px" : "w-627px"} mt-24px`}
            >
              {welcome}
            </div>
          )}
          <div className={`${isMobile ? "w-full px-24px" : "w-720px"} mt-80px`}>
            {searchbox}
            <div className={`w-full mt-40px`}>
              {recommends}
            </div>
          </div>
        </Content>
      </Layout>
  );
};

export default HomeLayout;