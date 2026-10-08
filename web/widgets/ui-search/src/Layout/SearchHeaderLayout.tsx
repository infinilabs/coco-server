import { Layout } from "antd";
import { useState, type FC, type ReactNode } from "react";

const { Content, Sider } = Layout;

const BG_CLASS = 'bg-[rgb(var(--ui-search--layout-bg-color))]';

interface SearchHeaderLayoutProps {
  logo?: ReactNode;
  searchbox?: ReactNode;
  tabs?: ReactNode;
  tools?: ReactNode;
  isMobile?: boolean;
  showLeftSider?: boolean;
  showRightSider?: boolean;
  leftWidth?: number;
  rightWidth?: number;
  centerPadding?: string;
  centerMaxWidth?: string;
  /** dynamic max-width for the center column (inline preview geometry) — an
   * inline style so a measured pixel value can win over the static class */
  centerMaxWidthPx?: number;
  rightMenuWidth?: number;
  scrolled?: boolean;
}

const SearchHeaderLayout: FC<SearchHeaderLayoutProps> = ({
  logo,
  searchbox,
  tabs,
  tools,
  isMobile,
  showLeftSider,
  showRightSider,
  leftWidth = 240,
  rightWidth = 400,
  centerPadding,
  centerMaxWidth,
  centerMaxWidthPx,
  rightMenuWidth,
  scrolled,
}) => {
  const defaultCenterPadding = isMobile ? 'px-16px' : 'pl-56px pr-96px';
  const padding = centerPadding || defaultCenterPadding;
  const [showLogoInCenter, setShowLogoInCenter] = useState(false);

  return (
    <div className={`fixed top-0 left-0 right-0 z-88 !p-0 h-auto ${BG_CLASS} border-b border-solid border-[var(--ant-color-border-secondary)] transition-shadow ${scrolled ? 'shadow-[0_2px_6px_rgba(0,0,0,0.1)] dark:shadow-[0_2px_6px_rgba(255,255,255,0.1)]' : ''}`}>
      {/* hasSider: without a Sider child antd Layout stacks vertically, which
          turns the right spacer's `flex: 0 0 <width>px` into a block that tall —
          the fixed header box then extends ~480px over the page and swallows
          every pointer event below its visible 64px (histogram brush, result
          clicks) whenever the left column is absent */}
      <Layout hasSider className={BG_CLASS}>
        {/* keep the header's left column in lockstep with the body sider, so the
            search box stays aligned with the result list when the sider collapses.
            Mount/unmount instead of animating the width — Sider's width transition
            can stall mid-frame in embedded webviews and leave the column at 0. */}
        {showLeftSider !== false && (
          <Sider onBreakpoint={(broken) => setShowLogoInCenter(broken)} width={leftWidth} breakpoint="md" collapsedWidth={0} trigger={null} className={BG_CLASS}>
            <div
              className={`w-full pl-64px ${BG_CLASS} ${
                isMobile ? "pt-16px h-122px" : "h-64px flex items-center"
              }`}
            >
              <div className={isMobile ? "h-48px w-full" : "h-48px w-48px"}>{logo}</div>
            </div>
          </Sider>
        )}
        <Content
          className={`${BG_CLASS} ${isMobile ? 'min-w-0' : 'min-w-400px'} ${centerMaxWidth || ''}`}
          style={{ overflow: 'visible', maxWidth: centerMaxWidthPx }}
        >
          {isMobile ? (
            <div className={`pt-16px h-122px ${padding}`}>
              <div className="flex gap-8px items-center">
                {showLogoInCenter && (
                  <div className="h-40px w-40px">{logo}</div>
                )}
                <div
                  className={`flex-1 box-border`}
                  style={isMobile && rightMenuWidth ? { paddingRight: rightMenuWidth } : undefined}
                >
                  {searchbox}
                </div>
              </div>
              {tabs && (
                <div className="w-full pt-12px flex items-center justify-between">
                  <div>{tabs}</div>
                  <div>{tools}</div>
                </div>
              )}
            </div>
          ) : (
            // one compact row: search box + category tabs + tool toggles — the
            // old two-row header (search box above tabs) spent 122px of
            // always-visible height; this spends 64px
            <div className={`flex h-64px items-center gap-16px ${padding}`}>
              {showLogoInCenter && (
                <div className="h-48px w-48px flex-none">{logo}</div>
              )}
              <div className="min-w-0 flex-1 box-border">{searchbox}</div>
              {tabs}
              {tools}
            </div>
          )}
        </Content>
        {/* same mount/unmount rationale as the left column; a plain spacer
          instead of Sider — the preview pane resizes this column dynamically
          and Sider's width transition stalls in embedded webviews */}
        {showRightSider ? (
          <div style={{ flex: `0 0 ${rightWidth}px`, width: rightWidth }} className={BG_CLASS}>
            <div className={`${isMobile ? "pt-16px h-122px" : "h-64px"} ${BG_CLASS}`} />
          </div>
        ) : rightMenuWidth ? (
          <div style={{ flex: `0 0 ${rightMenuWidth}px`, width: rightMenuWidth }} className={BG_CLASS} />
        ) : null}
      </Layout>
    </div>
  );
};

export default SearchHeaderLayout;
