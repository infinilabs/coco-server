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
      className="size-full flex-col-stretch"
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
        className={`b-t-1px b-r-1px ${siderCollapse ? 'text-center' : ''}`}
        style={{ borderColor }}
      >
        <MenuToggler />
      </div>
    </DarkModeContainer>
  );
});

export default GlobalSider;
