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

  const borderColor = 'var(--ant-color-border)'

  return (
    <DarkModeContainer
      className="size-full flex-col-stretch"
      inverted={darkMenu}
    >
      {showLogo && (
        <GlobalLogo
          // showTitle={!siderCollapse}
          showTitle={false}
          style={{ height: `${headerHeight}px`, borderColor }}
          className={`b-b-1px b-r-1px`}
          siderCollapse={siderCollapse}
          darkMode={darkMode}
        />
      )}
      <div
        className={showLogo ? `flex-1-hidden b-r-1px` : 'h-full'}
        style={{ borderColor }}
        id={GLOBAL_SIDER_MENU_ID}
      />
      <div style={{ borderColor }} className={`b-t-1px b-r-1px ${siderCollapse ? 'text-center' : ''}`}>
        <MenuToggler />
      </div>
    </DarkModeContainer>
  );
});

export default GlobalSider;
