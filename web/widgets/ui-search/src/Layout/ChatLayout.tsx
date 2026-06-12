import { Layout } from 'antd';
import { type FC, type ReactNode } from 'react';

import { DARK_CLASS } from '../theme/shared';
import useNProgress from '../hooks/useNProgress';
import logoTextDark from '../icons/logo-text-dark.svg';
import logoTextLight from '../icons/logo-text-light.svg';

import styles from './index.module.less';
import ChatIcon from '../icons/ChatIcon';

const { Content, Sider } = Layout;

interface ChatLayoutProps {
  loading?: boolean;
  theme?: 'light' | 'dark';
  isMobile?: boolean;
  logo?: {
    light?: string;
    light_mobile?: string;
    dark?: string;
    dark_mobile?: string;
  };
  handleLogoClick?: () => void;
  sidebar?: ReactNode;
  sidebarCollapsed?: boolean;
  header?: ReactNode;
  content?: ReactNode;
  input?: ReactNode;
  initContainer?: ((ref: HTMLDivElement | null) => void) | React.Ref<HTMLDivElement>;
}

const ChatLayout: FC<ChatLayoutProps> = (props) => {
  const { loading, theme, isMobile, logo, handleLogoClick, sidebar, sidebarCollapsed, header, content, input, initContainer } = props;

  const themeClass = theme === 'dark' ? DARK_CLASS : 'light';

  useNProgress(loading);

  const logoNode = (
    <div className='flex items-center gap-16px'>
      <div className='flex items-center cursor-pointer' onClick={() => handleLogoClick?.()}>
        <img
          alt='Coco'
          className='block h-10 w-auto dark:hidden'
          src={logo?.light || logoTextLight}
        />
        <img
          alt='Coco'
          className='hidden h-10 w-auto dark:block'
          src={logo?.dark || logoTextDark}
        />
      </div>
      <div className='flex items-center gap-1 rounded-9px px-2 py-1.5 bg-[#FFF] dark:bg-[#000]'>
        <ChatIcon
          className='h-4 w-4 text-[#7C3AED]'
          fill='currentColor'
        />
        <span className='text-12px text-#333 dark:text-#ccc'>Chat</span>
      </div>
    </div>
  );

  return (
    <Layout
      className={`${styles.uiSearch} relative w-full h-full bg-[rgb(var(--ui-search--layout-bg-color))] ui-search ${themeClass}`}
      ref={initContainer}
      style={{ overflow: 'hidden' }}
    >

      {/* Sidebar - Hidden on mobile, animated collapse/expand */}
      {!isMobile && (
        <Sider
          breakpoint='md'
          className='h-full border-r border-solid border-[var(--ant-color-border-secondary)] bg-[rgb(var(--ui-search--layout-bg-color))] !transition-all !duration-300 !ease-in-out overflow-hidden'
          collapsed={sidebarCollapsed}
          collapsedWidth={0}
          trigger={null}
          width={260}
        >
          <div className='h-full flex flex-col w-[260px]'>
            <div className='h-16 flex shrink-0 items-center bg-[#F3F4F6] px-14px dark:bg-[#1F2937]'>{logoNode}</div>
            <div className='flex-1 overflow-y-hidden bg-[#F3F4F6] dark:bg-[#1F2937]'>{sidebar}</div>
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
        <Content className='relative w-full flex-1 overflow-hidden'>
          <div className='h-full w-full'>
            {content}
          </div>
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

export default ChatLayout;
