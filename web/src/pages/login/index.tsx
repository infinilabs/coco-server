import { getPaletteColorByNumber, mixColor } from '@sa/color';

import bgZH from '@/assets/svg-icon/login-zh.svg';
import bg from '@/assets/svg-icon/login.svg';
import { getLocale } from '@/store/slice/app';
import { getIsLogin } from '@/store/slice/auth';
import { getDarkMode, getThemeSettings } from '@/store/slice/theme';

import CocoAI from './modules/CocoAI';
import LoginForm from './modules/LoginForm';

const COLOR_WHITE = '#ffffff';

function useBgColor() {
  const darkMode = useAppSelector(getDarkMode);
  const { themeColor } = useAppSelector(getThemeSettings);

  const bgThemeColor = darkMode ? getPaletteColorByNumber(themeColor, 600) : themeColor;
  const ratio = darkMode ? 0.5 : 0.2;
  const bgColor = mixColor(COLOR_WHITE, themeColor, ratio);

  return {
    bgColor,
    bgThemeColor
  };
}

export function Component() {
  const { bgThemeColor } = useBgColor();
  const locale = useAppSelector(getLocale);
  const [searchParams] = useSearchParams();
  const isLogin = useAppSelector(getIsLogin);
  const provider = searchParams.get('provider');
  const requestID = searchParams.get('request_id');
  const product = searchParams.get('product');
  const [cocoAIVisible, setCocoAIVisible] = useState(() => {
    if (Boolean(provider && requestID && product) && isLogin) {
      return true
    } else {
      return false
    }
  });
  const darkMode = useAppSelector(getDarkMode);

  const backgroundImage = locale === 'zh-CN' ? bgZH : bg;

  const isToProvider = useMemo(() => {
    return Boolean(provider && requestID && product);
  }, [provider, requestID, product]);

  return (
    <div
      className="relative size-full flex-center overflow-hidden bg-layout"
      style={{
        backgroundColor: darkMode ? 'rgb(var(--layout-bg-color))' : '#fff'
      }}
    >
      {
        !window.__POWERED_BY_WUJIE__ && (
          <div className="absolute right-0 top-0 p-10px">
            <div className="flex-y-center justify-end">
              <LangSwitch className="px-12px" />
              <ThemeSchemaSwitch className="px-12px" />
            </div>
          </div>
        )
      }
      <div
        className="h-100% w-1/3 bg-[size:contain] bg-top-left bg-no-repeat md:bg-center-left sm:bg-center-left"
        style={{ backgroundImage: `url(${backgroundImage})` }}
      />
      <div className="h-100% w-2/3">
        <div className="items-left size-full flex flex-col justify-center overflow-auto px-10%">
          {cocoAIVisible ? (
            <div className="w-550px">
              <CocoAI
                provider={provider}
                requestID={requestID}
              />
            </div>
          ) : (
            <div className="w-440px">
              <LoginForm onProvider={isToProvider ? () => setCocoAIVisible(true) : undefined} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
