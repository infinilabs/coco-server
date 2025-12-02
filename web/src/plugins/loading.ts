// @unocss-include
import { getRgb } from '@sa/color';

import systemLogo from '@/assets/svg-icon/logo.svg?raw';
import { $t } from '@/locales';
import { localStg } from '@/utils/storage';

export function setupLoading() {
  const themeColor = localStg.get('themeColor') || '#0087FF';

  const { b, g, r } = getRgb(themeColor);

  const primaryColor = `--primary-color: ${r} ${g} ${b}`;

  const loadingClasses = [
    'left-0 top-0',
    'left-0 bottom-0 animate-delay-500',
    'right-0 top-0 animate-delay-1000',
    'right-0 bottom-0 animate-delay-1500'
  ];

  const logoWithClass = systemLogo.replace('<svg', `<svg class="w-320px h-128px text-primary"`);

  const dot = loadingClasses
    .map(item => {
      return `<div class="absolute w-16px h-16px bg-primary rounded-8px animate-pulse ${item}"></div>`;
    })
    .join('\n');

  const loading = `
<div class="fixed-center flex-col ${window.__POWERED_BY_WUJIE__ ? "absolute" : "" }" style="${primaryColor}">
  ${logoWithClass}
  <div class="w-48px h-48px my-24px">
    <div class="relative h-full animate-spin">
      ${dot}
    </div>
  </div>
</div>`;

  const app = document.getElementById('root');

  if (app) {
    app.innerHTML = loading;
  }
}
