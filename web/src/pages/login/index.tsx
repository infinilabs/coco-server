import { getPaletteColorByNumber, mixColor } from '@sa/color';
import { getDarkMode, getThemeSettings } from '@/store/slice/theme';
import Introduce from '../guide/modules/Introduce';
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

    return (
        <div
            className="relative size-full flex-center overflow-hidden bg-layout"
            style={{ backgroundColor: bgThemeColor }}
        >
            <div className="w-1/3 ">
                <Introduce />
            </div>
            <div className="h-100% w-2/3 bg-white">
              <div className="size-full flex flex-col items-left justify-center px-10%">
                <div className="w-440px">
                      <LoginForm />
                  </div>
              </div>
            </div>
        </div>
    );
}
