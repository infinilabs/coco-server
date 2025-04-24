import DarkModeContainer from '@/components/stateless/common/DarkModeContainer.tsx';
import { GLOBAL_SIDER_MENU_ID } from '@/constants/app';
import { getDarkMode } from '@/store/slice/theme';

import GlobalLogo from '../global-logo';

interface Props {
  headerHeight: number;
  inverted: boolean;
  isHorizontalMix: boolean;
  isVerticalMix: boolean;
  siderCollapse: boolean;
}

const GlobalSider: FC<Props> = memo(({ headerHeight, inverted, isHorizontalMix, isVerticalMix, siderCollapse }) => {
  const darkMode = useAppSelector(getDarkMode);

  const showLogo = !isVerticalMix && !isHorizontalMix;

  const darkMenu = !darkMode && !isHorizontalMix && inverted;

  const borderColor = 'var(--ant-color-border)';

  return (
    <DarkModeContainer
      className="size-full flex-col-stretch css-var-r0 ant-menu-css-var"
      inverted={darkMenu}
    >
      {showLogo && (
        <GlobalLogo
          // showTitle={!siderCollapse}
          className="b-b-1px b-r-1px"
          darkMode={darkMode}
          showTitle={false}
          siderCollapse={siderCollapse}
          style={{ borderColor, height: `${headerHeight}px` }}
        />
      )}
      <div
        className={showLogo ? `flex-1-hidden b-r-1px` : 'h-full'}
        id={GLOBAL_SIDER_MENU_ID}
        style={{ borderColor }}
      />
      <div
        className={`b-r-1px ${siderCollapse ? 'text-center' : ''}`}
        style={{ 
          borderColor,
        }}
      >
        <LicenseTrigger className="w-[calc(100%-16px)] mx-8px my-4px" />
      </div>
      <div
        className={`b-t-1px b-r-1px ${siderCollapse ? 'text-center' : ''}`}
        style={{ borderColor }}
      >
        <MenuToggler className="w-[calc(100%-16px)] mx-8px my-4px" />
      </div>
    </DarkModeContainer>
  );
});

export default GlobalSider;
