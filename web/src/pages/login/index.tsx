import { getPaletteColorByNumber, mixColor } from '@sa/color';
import { getDarkMode, getThemeSettings } from '@/store/slice/theme';
import LoginForm from './modules/LoginForm';
import bg from "@/assets/svg-icon/login.svg"
import bgZH from "@/assets/svg-icon/login-zh.svg"
import { getLocale } from '@/store/slice/app';
import { getIsLogin } from '@/store/slice/auth';
import CocoAI from './modules/CocoAI';

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
    const [cocoAIVisible, setCocoAIVisible] = useState(false)

    const backgroundImage = locale === 'zh-CN' ? bgZH : bg

    const isToProvider = useMemo(() => {
      return !!(provider && requestID && product)
    }, [provider, requestID, product])
    
    useEffect(() => {
      if (isLogin && isToProvider) {
        setCocoAIVisible(true)
      }
    }, [isLogin, isToProvider])

    return (
        <div
            className="relative size-full flex-center overflow-hidden bg-layout"
            style={{ backgroundColor: bgThemeColor }}
        >
            <div className="p-10px absolute right-0 top-0">
              <LangSwitch className="px-12px" />
            </div>
            <div className="w-1/3 h-100% bg-top-left sm:bg-center-left md:bg-center-left bg-[size:100%_auto] bg-no-repeat" style={{ backgroundImage: `url(${backgroundImage})` }}>
                
            </div>
            <div className="h-100% w-2/3 bg-white">
              <div className="size-full flex flex-col items-left justify-center px-10% overflow-auto">
                {
                  cocoAIVisible ? (
                    <div className="w-550px">
                      <CocoAI provider={provider} requestID={requestID} />
                    </div>
                  ) : (
                    <div className="w-440px">
                      <LoginForm onProvider={isToProvider ? () => setCocoAIVisible(true) : undefined}/>
                    </div>
                  )
                }
              </div>
            </div>
        </div>
    );
}
