import loadingIcon from '@/assets/svg-icon/file-loading.svg?raw';

// keep the boot splash hidden briefly so fast loads never flash it on screen;
// mirrors ICON_DELAY_MS in GlobalLoading
const ICON_DELAY_MS = 300;

export function setupLoading() {
  // the icon self-animates (stroke-drawing loop) via its embedded <style>
  const icon = loadingIcon.replace('width="160" height="160"', 'width="48" height="48"');

  const loading = `
<div style="position:${window.__POWERED_BY_WUJIE__ ? 'absolute' : 'fixed'};left:0;top:0;width:100%;height:100%;display:flex;align-items:center;justify-content:center;">
  <style>
    @keyframes coco-boot-in {
      from { opacity: 0; }
      to { opacity: 1; }
    }
  </style>
  <div style="width:48px;height:48px;opacity:0;animation:coco-boot-in .2s ease ${ICON_DELAY_MS}ms forwards;">${icon}</div>
</div>`;

  const app = document.getElementById('root');

  if (app) {
    app.innerHTML = loading;
  }
}
