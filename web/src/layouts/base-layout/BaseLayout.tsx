import { AdminLayout, LAYOUT_SCROLL_EL_ID } from '@sa/materials';
import type { LayoutMode } from '@sa/materials';
import { configResponsive } from 'ahooks';
import './index.scss';

import {
  getContentXScrollable,
  getFullContent,
  getMixSiderFixed,
  getSiderCollapse,
  setIsMobile,
  setSiderCollapse
} from '@/store/slice/app';
import { getThemeSettings, setLayoutMode } from '@/store/slice/theme';

import GlobalBreadcrumb from '../modules/global-breadcrumb';
import GlobalContent from '../modules/global-content';
import GlobalFooter from '../modules/global-footer';
import GlobalHeader from '../modules/global-header';
import GlobalMenu from '../modules/global-menu';
import GlobalSider from '../modules/global-sider';
import GlobalTab from '../modules/global-tab';
import { logout } from '@/service/api';
import { fetchProviderInfo, fetchSettings } from '@/service/api/server';
import { getDefaultModelTips, setDefaultModel, setServer } from '@/store/slice/server';
import DefaultModel from '@/pages/guide/modules/DefaultModel';
import { isEmpty } from 'lodash';
import TopBanner, { TopBannerHeight } from '../modules/top-banner';
import { localStg } from '@/utils/storage';

const ThemeDrawer = lazy(() => import('../modules/theme-drawer'));

const LAYOUT_MODE_VERTICAL: LayoutMode = 'vertical';
const LAYOUT_MODE_HORIZONTAL: LayoutMode = 'horizontal';
const LAYOUT_MODE_VERTICAL_MIX = 'vertical-mix';
const LAYOUT_MODE_HORIZONTAL_MIX = 'horizontal-mix';

configResponsive({ sm: 640 });

const BaseLayout = () => {
  const siderCollapse = useAppSelector(getSiderCollapse);
  const themeSettings = useAppSelector(getThemeSettings);
  const fullContent = useAppSelector(getFullContent);
  const dispatch = useAppDispatch();
  const responsive = useResponsive();
  const nav = useNavigate()
  const { childLevelMenus, isActiveFirstLevelMenuHasChildren } = useMixMenuContext();
  const { hasAuth } = useAuth()
  const permissions = {
    settings: hasAuth('coco#system/read'),
    updateSettings: hasAuth('coco#system/update'),
    updateModelProvider: hasAuth('coco#model_provider/update'),
  }

  const contentXScrollable = useAppSelector(getContentXScrollable);
  const mixSiderFixed = useAppSelector(getMixSiderFixed);
  const defaultModelTips = useAppSelector(getDefaultModelTips);

  const layoutMode = themeSettings.layout.mode.includes(LAYOUT_MODE_VERTICAL)
    ? LAYOUT_MODE_VERTICAL
    : LAYOUT_MODE_HORIZONTAL;

  const isMobile = !responsive.sm;

  const siderVisible = themeSettings.layout.mode !== LAYOUT_MODE_HORIZONTAL;

  const isVerticalMix = themeSettings.layout.mode === LAYOUT_MODE_VERTICAL_MIX;

  const isHorizontalMix = themeSettings.layout.mode === LAYOUT_MODE_HORIZONTAL_MIX;

  function getSiderWidth() {
    const { reverseHorizontalMix } = themeSettings.layout;

    const { mixChildMenuWidth, mixWidth, width } = themeSettings.sider;

    if (isHorizontalMix && reverseHorizontalMix) {
      return isActiveFirstLevelMenuHasChildren ? width : 0;
    }

    let w = isVerticalMix || isHorizontalMix ? mixWidth : width;

    if (isVerticalMix && mixSiderFixed && childLevelMenus.length) {
      w += mixChildMenuWidth;
    }

    return w;
  }

  function getSiderCollapsedWidth() {
    const { reverseHorizontalMix } = themeSettings.layout;
    const { collapsedWidth, mixChildMenuWidth, mixCollapsedWidth } = themeSettings.sider;

    if (isHorizontalMix && reverseHorizontalMix) {
      return isActiveFirstLevelMenuHasChildren ? collapsedWidth : 0;
    }

    let w = isVerticalMix || isHorizontalMix ? mixCollapsedWidth : collapsedWidth;

    if (isVerticalMix && mixSiderFixed && childLevelMenus.length) {
      w += mixChildMenuWidth;
    }

    return w;
  }
  const siderWidth = getSiderWidth();
  const siderCollapsedWidth = getSiderCollapsedWidth();

  useEffect(() => {
    async function updateDefaultModel() {
      const res = await fetchSettings()
      dispatch(setDefaultModel(res.data.default_model || {}))
      const isEmptyDefaultModel = isEmpty(res.data.default_model)
      const defaultModelGuide = localStg.get('defaultModelGuide')
      if (!defaultModelGuide) {
        localStg.set('defaultModelGuide', isEmptyDefaultModel.toString())
      }
    }
    if (permissions.settings) {
      updateDefaultModel()
    }
  }, []);

  useEffect(() => {
    async function updateServerEndpoint() {
      const res = await fetchProviderInfo()
      if (res.data?.endpoint) {
        dispatch(setServer(res.data.endpoint))
      }
    }
    updateServerEndpoint()
  }, [])

  useLayoutEffect(() => {
    dispatch(setIsMobile(isMobile));
    if (isMobile) {
      dispatch(setLayoutMode('vertical'));
      dispatch(setSiderCollapse(true))
    }
  }, [isMobile, dispatch]);

  const isMicro = window.__POWERED_BY_WUJIE__;

  useEffect(() => {
    if (window.$wujie?.props?.onMicroMounted) {
      window.$wujie?.props?.onMicroMounted({
        nav,
        logout: async () => await logout({ ignoreError: true })
      })
    }
  }, [isMicro])

  return (
    <AdminLayout
      Breadcrumb={isMicro ? null : <GlobalBreadcrumb className="px-16px p-t-16px" />}
      contentClass={contentXScrollable ? 'overflow-x-hidden' : ''}
      fixedFooter={themeSettings.footer.fixed}
      fixedTop={themeSettings.fixedHeaderAndTab}
      Footer={isMicro ? null : <GlobalFooter />}
      footerHeight={themeSettings.footer.height}
      footerVisible={themeSettings.footer.visible}
      fullContent={fullContent}
      headerHeight={themeSettings.header.height}
      isMobile={isMobile}
      mode={layoutMode}
      rightFooter={themeSettings.footer.right}
      scrollElId={LAYOUT_SCROLL_EL_ID}
      scrollMode={themeSettings.layout.scrollMode}
      siderCollapse={siderCollapse}
      siderCollapsedWidth={siderCollapsedWidth}
      siderVisible={siderVisible}
      siderWidth={siderWidth}
      Tab={isMicro ? null : <GlobalTab />}
      tabHeight={themeSettings.tab.height}
      tabVisible={themeSettings.tab.visible}
      updateSiderCollapse={() => dispatch(setSiderCollapse(true))}
      Header={
        isMicro ? null : (
          <GlobalHeader
            isMobile={isMobile}
            mode={themeSettings.layout.mode}
            reverse={themeSettings.layout.reverseHorizontalMix}
            siderWidth={themeSettings.sider.width}
          />
        )
      }
      Sider={
        isMicro ? null : (
          <GlobalSider
            headerHeight={themeSettings.header.height}
            inverted={themeSettings.sider.inverted}
            isHorizontalMix={isHorizontalMix}
            isVerticalMix={isVerticalMix}
            siderCollapse={siderCollapse}
          />
        )
      }
      topBanner={isMicro || !defaultModelTips || localStg.get('ignoreDefaultModelTips') === 'true' ? null : (
        <TopBanner />
      )}
      topBannerHeight={TopBannerHeight}
    >
      <GlobalContent closePadding={isMicro} />

      <GlobalMenu
        mode={themeSettings.layout.mode}
        reverse={themeSettings.layout.reverseHorizontalMix}
      />
      {
        permissions.updateSettings && permissions.updateModelProvider && localStg.get('defaultModelGuide') === 'true' && <DefaultModel />
      }
      {/* <Suspense fallback={null}>
        <ThemeDrawer />
      </Suspense> */}
    </AdminLayout>
  );
};

export default BaseLayout;
