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
  /** background image for the current theme: { light, dark } */
  background?: Record<string, any>;
  /** banner slot max height in px (width stays adaptive) */
  logoMaxHeight?: number;
}

const HomeLayout: FC<HomeLayoutProps> = (props) => {
  const {
    loading,
    logo,
    welcome,
    searchbox,
    isMobile,
    theme,
    recommends,
    background,
    logoMaxHeight
  } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';

  const backgroundImage = theme === 'dark' ? background?.dark : background?.light;

  useNProgress(loading);

  return (
      <Layout 
        className={`${styles.uiSearch} relative w-full h-full overflow-x-hidden overflow-y-hidden bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
        style={
          backgroundImage
            ? {
                backgroundImage: `url("${backgroundImage}")`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat'
              }
            : undefined
        }
      >
        <Content className="bg-transparent w-full h-full flex flex-col items-center justify-start absolute top-15% left-0">
          {/* explicit height — the Logo inside is h-full and would collapse to
              0 against an auto-height parent; width adapts to the image ratio,
              capped at 320px and centered */}
          <div
            className='max-w-320px w-full flex items-center justify-center [&_img]:max-w-full'
            style={{ height: `${logoMaxHeight || 64}px` }}
          >
            {logo}
          </div>
          {welcome && (
            <div
              className={`${isMobile ? "w-full px-32px" : "w-627px"} mt-24px`}
            >
              {welcome}
            </div>
          )}
          <div className={`${isMobile ? "w-full px-24px" : "w-720px"} mt-40px`}>
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