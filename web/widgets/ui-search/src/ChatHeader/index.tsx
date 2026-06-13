import { type FC, type ReactNode } from "react";
import { Button, Tooltip } from "antd";
import { Search } from "lucide-react";
import { useTranslation } from 'react-i18next';
import logoMobileSvg from '../icons/coco.svg';

const LeftSvg: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none" className={className}>
    <g>
    <path d="M6 9C6 7.34315 7.34315 6 9 6H39C40.6569 6 42 7.34315 42 9V39C42 40.6569 40.6569 42 39 42H9C7.34315 42 6 40.6569 6 39V9Z" fill="none" stroke="currentColor" strokeWidth="4" strokeLinejoin="round"></path><path d="M32 6V42" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M16 20L20 24L16 28" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M26 6H38" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M26 42H38" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path>
    </g>
  </svg>
)

const RightSvg: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 48 48" width="1em" height="1em" filter="none" className={className}>
    <g>
    <rect x="6" y="6" width="36" height="36" rx="3" fill="none" stroke="currentColor" strokeWidth="4" strokeLinejoin="round"></rect><path d="M18 6V42" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M11 6H36" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M11 42H36" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path><path d="M32 20L28 24L32 28" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round" fill="none"></path>
    </g>
  </svg>
)

const NewChatSvg: FC<{ className?: string }> = ({ className }) => (
  <svg viewBox="0 0 32 32" width="1em" height="1em" filter="none" className={className}>
    <g>
    <path d="M18.667 4v2.667h-13.333v17.847l2.351-1.847h18.983v-9.333h2.667v10.667c0 0.736-0.597 1.333-1.333 1.333v0h-19.393l-5.94 4.667v-24.667c0-0.736 0.597-1.333 1.333-1.333v0h14.667zM25.333 4v-4h2.667v4h4v2.667h-4v4h-2.667v-4h-4v-2.667h4z" fill="rgba(3,135,255,1)"></path>
    </g>
  </svg>
)

interface ChatHeaderProps {
  isMobile?: boolean;
  theme?: 'light' | 'dark';
  logo?: {
    light_mobile?: string;
    dark_mobile?: string;
  };
  handleLogoClick?: () => void;
  isHistoryOpen?: boolean;
  onToggleHistory?: () => void;
  onNewChat?: () => void;
  onBackToSearch?: () => void;
  AssistantList?: ReactNode;
  rightMenuWidth?: number;
}

const ChatHeader: FC<ChatHeaderProps> = (props) => {
  const { isMobile, rightMenuWidth, theme, logo, handleLogoClick, isHistoryOpen, onToggleHistory, onNewChat, onBackToSearch, AssistantList } = props;
  const { t } = useTranslation();

  return (
    <div style={rightMenuWidth ? { paddingRight: rightMenuWidth + 12 } : undefined} className="h-full w-full flex items-center justify-between px-4 border-b border-solid border-[var(--ant-color-border-secondary)] box-border">
      <div className="w-full flex items-center gap-2">
        {isMobile && (
          <div className='flex items-center cursor-pointer shrink-0' onClick={() => handleLogoClick?.()}>
            <img
              src={(theme === 'dark' ? logo?.dark_mobile : logo?.light_mobile) || logoMobileSvg}
              width={40}
              height={40}
            />
          </div>
        )}
        <Button
          icon={isHistoryOpen ? <RightSvg className="h-4 w-4" /> : <LeftSvg className="h-4 w-4" />}
          onClick={onToggleHistory}
          className="!rounded-12px border-[#F0F0F0] dark:border-[#303030] shrink-0"
        />

        <div className="min-w-0 max-w-full flex-shrink flex-grow-0 basis-auto relative">
          {AssistantList}
        </div>

        <Tooltip title={t('labels.newChat')}>
          <Button
            icon={<NewChatSvg className="h-4 w-4 !text-[var(--ant-color-primary)]" />}
            onClick={onNewChat}
            className="!rounded-12px border-[#F0F0F0] dark:border-[#303030] shrink-0"
          />
        </Tooltip>

        {
          isMobile ? (
            <Tooltip title={t('labels.backToSearch')}>
              <Button
                icon={<Search className="h-4 w-4 !text-[var(--ant-color-primary)]" />}
                onClick={onBackToSearch}
                className="!rounded-12px border-[#F0F0F0] dark:border-[#303030] shrink-0"
              />
            </Tooltip>
          ) : (
            <Button
              shape="round"
              icon={<Search className="h-4 w-4 !text-[var(--ant-color-primary)]" />}
              onClick={onBackToSearch}
              className="text-[#999] !rounded-12px border-[#F0F0F0] dark:border-[#303030] !px-8px shrink-0"
            >
              {t('labels.backToSearch')}
            </Button>
          )
        }
      </div>

    </div>
  );
};

export default ChatHeader;
