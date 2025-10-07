import LangSwitch from '@/components/stateful/LangSwitch';
import ThemeSchemaSwitch from '@/components/stateful/ThemeSchemaSwitch';
import DarkModeContainer from '@/components/stateless/common/DarkModeContainer';
import { GLOBAL_HEADER_MENU_ID } from '@/constants/app';

import GlobalBreadcrumb from '../global-breadcrumb';
import GlobalLogo from '../global-logo';

import UserAvatar from './components/UserAvatar';
import { localStg } from '@/utils/storage';
import { getProviderInfo } from '@/store/slice/server';

interface Props {
  isMobile: boolean;
  mode: UnionKey.ThemeLayoutMode;
  reverse?: boolean;
  siderWidth: number;
}

const HEADER_PROPS_CONFIG: Record<UnionKey.ThemeLayoutMode, App.Global.HeaderProps> = {
  horizontal: {
    showLogo: true,
    showMenu: true,
    showMenuToggler: false
  },
  'horizontal-mix': {
    showLogo: true,
    showMenu: true,
    showMenuToggler: false
  },
  vertical: {
    showLogo: false,
    showMenu: true,
    showMenuToggler: false
  },
  'vertical-mix': {
    showLogo: false,
    showMenu: false,
    showMenuToggler: false
  }
};

const GlobalHeader: FC<Props> = memo(({ isMobile, mode, reverse, siderWidth }) => {
  const { showLogo, showMenu, showMenuToggler } = HEADER_PROPS_CONFIG[mode];

  const showToggler = reverse ? true : showMenuToggler;

  const borderColor = 'var(--ant-color-border)';

  const nav = useNavigate();
  const { t } = useTranslation();
  const providerInfo = useAppSelector(getProviderInfo);
  
  const { search_settings } = providerInfo || {}

  return (
    <DarkModeContainer
      className="h-full flex-y-center b-b-1px px-12px"
      style={{ borderColor }}
    >
      {showLogo && (
        <GlobalLogo
          className="h-full"
          style={{ width: `${siderWidth}px` }}
        />
      )}
      <div>{reverse ? true : showMenuToggler}</div>

      {showToggler && <MenuToggler />}

      <div
        className="h-full flex-y-center flex-1-hidden"
        id={GLOBAL_HEADER_MENU_ID}
      >
        {!isMobile && !showMenu && <GlobalBreadcrumb className="ml-12px" />}
      </div>

      <div className="h-full flex-y-center justify-end">
        {
          search_settings?.enabled && (
            <ButtonIcon
              className="px-12px"
              tooltipContent={t('common.search')}
              onClick={() => nav(`/search`)}
            >
              <IconUilSearch />
            </ButtonIcon>
          )
        }
        {/* <GlobalSearch /> */}
        {/* {!isMobile && (
          <FullScreen
            className="px-12px"
            full={isFullscreen}
            toggleFullscreen={toggleFullscreen}
          />
        )} */}
        <LangSwitch className="px-12px" />
        <ThemeSchemaSwitch className="px-12px" />
        {/* <ThemeButton /> */}
        <UserAvatar />
      </div>
    </DarkModeContainer>
  );
});

export default GlobalHeader;
