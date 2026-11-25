import NProgress from 'nprogress';

/** Setup plugin NProgress */
export function setupNProgress() {
  // the top bar alone is the loading indicator — a fixed corner spinner lands
  // inside the app shell header and collides with its controls
  NProgress.configure({ easing: 'ease', speed: 500, showSpinner: false });

  // mount on window
  window.NProgress = NProgress;
}
