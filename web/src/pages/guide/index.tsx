import { getPaletteColorByNumber, mixColor } from '@sa/color';
import { getDarkMode, getThemeSettings } from '@/store/slice/theme';
import Guide from './modules/Guide';
import bg from "@/assets/svg-icon/guide.svg"
import bgZH from "@/assets/svg-icon/guide-zh.svg"
import { getLocale } from '@/store/slice/app';

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

    const backgroundImage = locale === 'zh-CN' ? bgZH : bg

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
                <Guide />
            </div>
        </div>
    );
}
