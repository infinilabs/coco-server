import { Layout } from "antd";

const { Content, Sider } = Layout;

const BG_CLASS = 'bg-[rgb(var(--ui-search--layout-bg-color))]';

const SearchHeaderLayout = ({
  logo,
  searchbox,
  tabs,
  tools,
  isMobile,
  showLeftSider,
  showRightSider,
  leftWidth = 280,
  rightWidth = 400,
  centerPadding,
  centerMaxWidth,
  rightMenuWidth,
}) => {
  const defaultCenterPadding = isMobile ? 'px-16px' : 'pl-56px pr-96px';
  const padding = centerPadding || defaultCenterPadding;
  const [showLogoInCenter, setShowLogoInCenter] = useState(false);

  return (
    <div className={`fixed top-0 left-0 right-0 z-1001 !p-0 h-auto ${BG_CLASS} border-b border-[var(--ant-color-border-secondary)]`}>
      <Layout className={BG_CLASS}>
        <Sider onBreakpoint={(broken) => setShowLogoInCenter(broken)} width={leftWidth} breakpoint="md" collapsedWidth={0} trigger={null} className={BG_CLASS}>
          <div className={`pt-16px h-122px w-full pl-80px ${BG_CLASS}`}>
            <div className="h-48px w-full">{logo}</div>
          </div>
        </Sider>
        <Content
          className={`${BG_CLASS} min-w-400px ${centerMaxWidth || ''}`}
          style={{ overflow: 'visible' }}
        >
          <div className={`pt-16px h-122px ${padding}`}>
            <div className="flex gap-8px items-center">
              {showLogoInCenter && (
                <div className={isMobile ? 'h-40px w-40px' : 'h-48px w-48px'}>{logo}</div>
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
        </Content>
        <Sider width={rightWidth} className={BG_CLASS} breakpoint="md" collapsedWidth={0} trigger={null}>
          <div className={`pt-16px h-122px ${BG_CLASS}`} />
        </Sider>
      </Layout>
    </div>
  );
};

export default SearchHeaderLayout;
