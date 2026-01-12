import { Layout } from 'antd';
import PropTypes from 'prop-types';
import styles from './index.module.less';
import { DARK_CLASS } from '../theme/shared';
import GlobalLoading from '../GlobalLoading';
import logoTextDark from '../assets/images/logo-text-dark.svg';
import logoTextLight from '../assets/images/logo-text-light.svg';
import { Sparkles } from 'lucide-react';

const { Content, Sider } = Layout;

const ChatLayout = props => {
  const { loading, theme, isMobile, logo, sidebar, sidebarCollapsed, header, content, input, initContainer } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';
  const logoNode = logo ?? (
    <div className='flex items-center gap-2'>
      <div className='flex items-center'>
        <img
          alt='Coco'
          className='block h-10 w-auto dark:hidden'
          src={logoTextLight}
        />
        <img
          alt='Coco'
          className='hidden h-10 w-auto dark:block'
          src={logoTextDark}
        />
      </div>
      <div className='bg---ui-search-antd-color-bg-container ml-4 flex items-center gap-1 border border-[#bbb] rounded-full px-2 py-1.5'>
        <Sparkles
          className='h-4 w-4 text-[#7C3AED]'
          fill='currentColor'
        />
        <span className='text-xs text-[#999]'>Chat</span>
      </div>
    </div>
  );

  return (
    <Layout
      className={`${styles.uiSearch} relative w-full h-100vh bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      ref={initContainer}
      style={{ height: '100vh', overflow: 'hidden' }}
    >
      <GlobalLoading
        loading={loading}
        theme={theme}
      />

      {/* Sidebar - Hidden on mobile, or handled via Drawer by parent if needed */}
      {!isMobile && !sidebarCollapsed && (
        <Sider
          breakpoint='md'
          className='h-full border-r border-[#ebebeb] bg-[rgb(var(--ui-search--layout-bg-color))]'
          collapsedWidth='0'
          trigger={null}
          width={260}
        >
          <div className='h-full flex flex-col'>
            <div className='h-16 flex shrink-0 items-center bg-[#F3F4F6] px-4 dark:bg-[#1F2937]'>{logoNode}</div>
            <div className='flex-1 overflow-y-auto'>{sidebar}</div>
          </div>
        </Sider>
      )}

      {/* Main Content Area */}
      <Layout className='relative h-full flex flex-col bg-[rgb(var(--ui-search--layout-bg-color))]'>
        {/* Header */}
        {header && (
          <div className='z-10 h-16 flex shrink-0 items-center bg-[rgb(var(--ui-search--layout-bg-color))]'>
            {header}
          </div>
        )}

        {/* Chat Messages Area */}
        <Content className='relative w-full flex-1 overflow-y-auto'>
          <div className='mx-auto h-full max-w-4xl w-full px-4 py-4'>{content}</div>
        </Content>

        {/* Input Area - Fixed at bottom of the main layout column */}
        {input && (
          <div className='w-full shrink-0 bg-[rgb(var(--ui-search--layout-bg-color))] px-4 pb-6 pt-2'>
            <div className='mx-auto max-w-4xl'>{input}</div>
          </div>
        )}
      </Layout>
    </Layout>
  );
};

ChatLayout.propTypes = {
  loading: PropTypes.bool,
  theme: PropTypes.oneOf(['light', 'dark']),
  isMobile: PropTypes.bool,
  logo: PropTypes.node,
  sidebar: PropTypes.node,
  sidebarCollapsed: PropTypes.bool,
  header: PropTypes.node,
  content: PropTypes.node,
  input: PropTypes.node,
  initContainer: PropTypes.oneOfType([PropTypes.func, PropTypes.object])
};

export default ChatLayout;
