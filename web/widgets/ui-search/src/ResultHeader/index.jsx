import { PanelLeftClose, PanelLeftOpen, PanelRightClose, PanelRightOpen } from "lucide-react";

export function ResultHeader(props) {
  const {
    hits, isMobile,
    hasRecommends,
    siderCollapse, setSiderCollapse,
    recommendsCollapse, setRecommendsCollapse,
    leftDrawerOpen, setLeftDrawerOpen,
    rightDrawerOpen, setRightDrawerOpen
  } = props;

  const handleLeftToggle = () => {
    if (isMobile || siderCollapse) {
      // In drawer mode: toggle drawer open/close
      setLeftDrawerOpen(!leftDrawerOpen);
    } else {
      // In sider mode: collapse it
      setSiderCollapse(true);
    }
  };

  const handleRightToggle = () => {
    if (isMobile || recommendsCollapse) {
      // In drawer mode: toggle drawer open/close
      setRightDrawerOpen(!rightDrawerOpen);
    } else {
      // In sider mode: collapse it
      setRecommendsCollapse(true);
    }
  };

  const showRightToggle = hasRecommends && (isMobile || recommendsCollapse);
  const LeftToggleIcon = (isMobile || siderCollapse) && !leftDrawerOpen ? PanelRightOpen : PanelLeftOpen;
  const RightToggleIcon = showRightToggle && !rightDrawerOpen ? PanelLeftOpen : PanelRightOpen;

  return (
    <div className="flex gap-8px items-center w-full text-[#999]">
      <LeftToggleIcon className="w-16px h-16px cursor-pointer" onClick={handleLeftToggle} />
      <div className="text-12px flex-1">
        Found {hits?.total || 0} records ({hits?.took || 0} millisecond)
      </div>
      {showRightToggle && (
        <RightToggleIcon className="w-16px h-16px cursor-pointer" onClick={handleRightToggle} />
      )}
    </div>
  );
}

export default ResultHeader;
