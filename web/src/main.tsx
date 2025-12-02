import { createRoot } from 'react-dom/client';
import { ErrorBoundary } from 'react-error-boundary';
import { Provider } from 'react-redux';

import { setupRouter } from '@/router';
import { store } from '@/store';

import FallbackRender from '../ErrorBoundary.tsx';

import App from './App.tsx';
import './plugins/assets';
import { setupI18n } from './locales';
import { setupAppVersionNotification, setupDayjs, setupIconifyOffline, setupLoading, setupNProgress } from './plugins';
import './icons';
import { setupMicro } from './components/micro/index.tsx';

function renderRoot(root) {
  root.render(
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    // @ts-ignore
    <ErrorBoundary fallbackRender={FallbackRender}>
      <Provider store={store}>
        <App />
      </Provider>
    </ErrorBoundary>
  );
}

function setupApp() {
  setupI18n();

  setupLoading();

  setupNProgress();

  setupIconifyOffline();

  setupRouter();

  setupDayjs();
  
  const container = document.getElementById('root');
  if (!container) return;
  const root = createRoot(container);

  if (window.__POWERED_BY_WUJIE__) {
    setupMicro(root, renderRoot)
  } else {
    setupAppVersionNotification();
    renderRoot(root)
  }
}

setupApp();
