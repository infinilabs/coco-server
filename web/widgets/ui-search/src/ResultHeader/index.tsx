import { PanelLeftOpen, PanelRightOpen } from "lucide-react";
import { useTranslation } from 'react-i18next';
import { type FC } from "react";

interface ResultHeaderProps {
  hits?: { total?: number; took?: number };
  isMobile?: boolean;
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
}

export const ResultHeader: FC<ResultHeaderProps> = (props) => {
  const {
    hits, isMobile,
    hasRecommends,
    siderCollapse, setSiderCollapse,
    recommendsCollapse, setRecommendsCollapse,
    userCollapsedLeft, userCollapsedRight,
    leftDrawerOpen, setLeftDrawerOpen,
    rightDrawerOpen, setRightDrawerOpen
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
    <div className="flex gap-8px items-center w-full text-[#999]">
      <LeftToggleIcon className="text-[#666] w-16px h-16px cursor-pointer" onClick={handleLeftToggle} />
      <div className="text-12px flex-1">
        {t('labels.resultsWithTime', { count: hits?.total || 0, took: hits?.took || 0 })}
      </div>
      {showRightToggle && (
        <RightToggleIcon className="text-[#666] w-16px h-16px cursor-pointer" onClick={handleRightToggle} />
      )}
    </div>
  );
}

export default ResultHeader;
