import { getPaletteColorByNumber, mixColor } from '@sa/color';

import bgZH from '@/assets/svg-icon/guide-zh.svg';
import bg from '@/assets/svg-icon/guide.svg';
import { getLocale } from '@/store/slice/app';
import { getDarkMode, getThemeSettings } from '@/store/slice/theme';

import Guide from './modules/Guide';

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
  const darkMode = useAppSelector(getDarkMode);

  const backgroundImage = locale === 'zh-CN' ? bgZH : bg;

  return (
    <div
      className="relative size-full flex-center overflow-hidden bg-layout"
      style={{ backgroundColor: darkMode ? 'rgb(var(--layout-bg-color))' : '#fff' }}
    >
      <div className="absolute right-0 top-0 p-10px">
        <div className="flex-y-center justify-end">
          <LangSwitch className="px-12px" />
          <ThemeSchemaSwitch className="px-12px" />
        </div>
      </div>
      <div
        className="h-100% w-1/3 bg-[size:100%_auto] bg-top-left bg-no-repeat md:bg-center-left sm:bg-center-left"
        style={{ backgroundImage: `url(${backgroundImage})` }}
      />
      <div className="h-100% w-2/3">
        <Guide />
      </div>
    </div>
  );
}
