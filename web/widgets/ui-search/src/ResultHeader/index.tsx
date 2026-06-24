import { PanelLeftOpen, PanelRightOpen } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useTranslation } from 'react-i18next';
import { type FC, type ReactNode } from "react";

interface ResultHeaderProps {
  hits?: { total?: number; took?: number };
  isMobile?: boolean;
  hasAggregations?: boolean;
  hasRecommends?: boolean;
  siderCollapse?: boolean;
  setSiderCollapse?: (v: boolean) => void;
  recommendsCollapse?: boolean;
  setRecommendsCollapse?: (v: boolean) => void;
  userCollapsedLeft?: boolean;
  userCollapsedRight?: boolean;
  leftDrawerOpen?: boolean;
  setLeftDrawerOpen?: (v: boolean) => void;
  rightDrawerOpen?: boolean;
  setRightDrawerOpen?: (v: boolean) => void;
  toolbar?: ReactNode;
}

export const ResultHeader: FC<ResultHeaderProps> = (props) => {
  const {
    hits, isMobile,
    hasAggregations,
    hasRecommends,
    siderCollapse, setSiderCollapse,
    recommendsCollapse, setRecommendsCollapse,
    userCollapsedLeft, userCollapsedRight,
    leftDrawerOpen, setLeftDrawerOpen,
    rightDrawerOpen, setRightDrawerOpen,
    toolbar
  } = props;

  const { t } = useTranslation();

  const handleLeftToggle = () => {
    if (isMobile) {
      setLeftDrawerOpen?.(!leftDrawerOpen);
    } else if (siderCollapse && userCollapsedLeft) {
      // User manually collapsed: re-expand as sider
      setSiderCollapse?.(false);
    } else if (siderCollapse) {
      // Auto-collapsed (not enough space): use drawer
      setLeftDrawerOpen?.(!leftDrawerOpen);
    } else {
      // Sider showing: collapse it
      setSiderCollapse?.(true);
    }
  };

  const handleRightToggle = () => {
    if (isMobile) {
      setRightDrawerOpen?.(!rightDrawerOpen);
    } else if (recommendsCollapse && userCollapsedRight) {
      // User manually collapsed: re-expand as sider
      setRecommendsCollapse?.(false);
    } else if (recommendsCollapse) {
      // Auto-collapsed (not enough space): use drawer
      setRightDrawerOpen?.(!rightDrawerOpen);
    } else {
      // Sider showing: collapse it
      setRecommendsCollapse?.(true);
    }
  };

  const showRightToggle = hasRecommends && (isMobile || recommendsCollapse);
  const LeftToggleIcon = (isMobile || siderCollapse) && !leftDrawerOpen ? PanelRightOpen : PanelLeftOpen;
  const RightToggleIcon = showRightToggle && !rightDrawerOpen ? PanelLeftOpen : PanelRightOpen;

  return (
    <div className="flex gap-8px items-start w-full text-[#999] dark:text-[#666]">
      {
        hasAggregations && (
          <span className="h-18px flex flex-none items-center">
            <LeftToggleIcon className="text-[#666] dark:text-white/80 w-16px h-16px cursor-pointer" onClick={handleLeftToggle} />
          </span>
        )
      }
      <div className={`text-12px flex-1 overflow-hidden`}>
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={toolbar ? 'toolbar' : 'summary'}
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 4 }}
            transition={{ duration: 0.16, ease: 'easeOut' }}
          >
            {toolbar || t('labels.resultsWithTime', { count: hits?.total || 0, took: hits?.took || 0 })}
          </motion.div>
        </AnimatePresence>
      </div>
      {showRightToggle && (
        <span className="h-18px flex flex-none items-center">
          <RightToggleIcon className="text-[#666] dark:text-white/80 w-16px h-16px cursor-pointer" onClick={handleRightToggle} />
        </span>
      )}
    </div>
  );
}

export default ResultHeader;
