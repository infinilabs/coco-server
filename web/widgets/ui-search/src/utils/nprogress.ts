/**
 * A self-contained, scoped progress bar used by the `ui-search` widget.
 *
 * ## Why this exists
 *
 * The host application (and many other consumers) also use the `nprogress` package. Because `nprogress` exports a
 * single shared object (one module instance, one internal `status`, and a hard-coded DOM id `#nprogress`), importing it
 * here would silently couple the widget to the host:
 *
 * - Both would render into / read from the same `#nprogress` element.
 * - Mutating `NProgress.render` / `remove` here would break the host's router guard progress bar.
 * - The widget's CSS (`#nprogress .bar { background: rgb(var(--ui-search--nprogress-color)) }`) would win the cascade
 *   over the host's `#nprogress` rules, and since `--ui-search--nprogress-color` is only defined inside `.ui-search`,
 *   the host's top-of-page bar becomes transparent ("styles lost").
 *
 * To stay fully independent, this module provides a tiny progress bar that:
 *
 * - renders under a _unique_ DOM id (`nprogress-ui-search`) so it never collides with a host's `#nprogress` element or
 *   CSS,
 * - renders at `document.body` (or the widget's shadow root), never inside `.ui-search`, so its `position: fixed` bar
 *   anchors to the _viewport_ — hosts may wrap the widget in a `contain: layout` container (the Coco app shell does),
 *   which re-anchors fixed descendants to that container and would slide the bar down under the host's header,
 * - keeps its own internal status and timers, so it cannot interfere with any other progress bar instance,
 * - reads its color from `--ui-search--nprogress-color` (scoped to the widget), mirroring the widget's `nprogress.css`.
 *
 * The public surface intentionally mimics the subset of the `nprogress` API the widget actually uses (`configure`,
 * `start`, `done`).
 */

export interface ScopedNProgressSettings {
  minimum: number;
  easing: string;
  speed: number;
  trickle: boolean;
  trickleRate: number;
  trickleSpeed: number;
  showSpinner: boolean;
  template: string;
}

export interface ScopedNProgress {
  configure(options: Partial<ScopedNProgressSettings>): ScopedNProgress;
  start(): ScopedNProgress;
  done(force?: boolean): ScopedNProgress;
  set(n: number): ScopedNProgress;
  remove(): void;
}

const BAR_SELECTOR = '[role="bar"]';
const SPINNER_SELECTOR = '[role="spinner"]';

const DEFAULT_SETTINGS: ScopedNProgressSettings = {
  minimum: 0.08,
  easing: 'ease',
  speed: 500,
  trickle: true,
  trickleRate: 0.02,
  trickleSpeed: 800,
  // the top bar alone is the loading indicator; a corner spinner collides with
  // host header controls when the widget is embedded below a host header
  showSpinner: false,
  template:
    '<div class="bar" role="bar"><div class="peg"></div></div>' +
    '<div class="spinner" role="spinner"><div class="spinner-icon"></div></div>'
};

const PROGRESS_ID = 'nprogress-ui-search';

type RootLike = Element | ShadowRoot | Document;

// Where the progress bar should be looked up / appended. The widget's
// <Wrapper> sets this to the shadow root (when used in a shadow DOM) or to
// `document`. Defaults to `document`.
let rootGetter: () => RootLike = () => document;

/**
 * Configure where the scoped progress bar lives. Called once by the widget's root <Wrapper> so that Shadow DOM usage is
 * supported.
 */
export function setNProgressRoot(getter: () => RootLike) {
  rootGetter = getter;
}

const clamp = (n: number, min: number, max: number) => {
  if (n < min) return min;
  if (n > max) return max;
  return n;
};

const toBarPerc = (n: number) => (n - 1) * 100;

function applyStyles(el: HTMLElement | null | undefined, props: Record<string, string>) {
  if (!el) return;
  for (const prop of Object.keys(props)) {
    el.style.setProperty(prop, props[prop]);
  }
}

function createScopedNProgress(): ScopedNProgress {
  const settings: ScopedNProgressSettings = { ...DEFAULT_SETTINGS };
  let status: number | null = null;
  let trickleTimer: ReturnType<typeof setTimeout> | null = null;

  const getRoot = (): RootLike => rootGetter();

  const isRendered = () => Boolean(getRoot().querySelector(`#${PROGRESS_ID}`));

  /**
   * Resolve the container the progress element should be appended to.
   *
   * The bar is `position: fixed` and must anchor to the viewport top. A host can wrap the widget in a `contain: layout`
   * container (the Coco app shell does, to keep the widget's own fixed header below the shell header) — fixed
   * descendants of such a container re-anchor to the container instead of the viewport. Staying outside the widget
   * (body / shadow root) keeps the viewport anchoring; the shadow root is required there because that is where this
   * widget's CSS lives in shadow-DOM embedding.
   */
  const resolveContainer = (): Node => {
    const root = getRoot();
    if (typeof ShadowRoot !== 'undefined' && root instanceof ShadowRoot) return root;
    return document.body;
  };

  const render = (fromStart: boolean): HTMLElement | null => {
    if (isRendered()) return getRoot().querySelector(`#${PROGRESS_ID}`) as HTMLElement | null;

    const progress = document.createElement('div');
    progress.id = PROGRESS_ID;
    progress.innerHTML = settings.template;

    const bar = progress.querySelector(BAR_SELECTOR) as HTMLElement | null;
    const perc = fromStart ? '-100' : String(toBarPerc(status || 0));
    applyStyles(bar, {
      transition: 'all 0s linear',
      transform: `translate3d(${perc}%,0,0)`
    });

    if (!settings.showSpinner) {
      progress.querySelector(SPINNER_SELECTOR)?.remove();
    }

    // The bar's color custom property is scoped to `.ui-search` by the widget
    // theme setup, but the element renders outside that container — carry the
    // resolved value onto the element so `rgb(var(...))` still applies.
    const widget = getRoot().querySelector('.ui-search') as HTMLElement | null;
    const color = widget ? getComputedStyle(widget).getPropertyValue('--ui-search--nprogress-color').trim() : '';
    if (color) progress.style.setProperty('--ui-search--nprogress-color', color);

    resolveContainer().appendChild(progress);
    return progress;
  };

  const remove = () => {
    getRoot().querySelector(`#${PROGRESS_ID}`)?.remove();
  };

  const stopTrickle = () => {
    if (trickleTimer !== null) {
      clearTimeout(trickleTimer);
      trickleTimer = null;
    }
  };

  const trickle = () => {
    if (typeof status !== 'number') return;
    const amount = (1 - status) * clamp(Math.random() * status, 0.1, 0.95);
    set(clamp(status + amount, 0, 0.994));
  };

  function set(n: number): ScopedNProgress {
    const started = typeof status === 'number';
    n = clamp(n, settings.minimum, 1);
    status = n === 1 ? null : n;

    const progress = render(!started);
    const bar = progress?.querySelector(BAR_SELECTOR) as HTMLElement | null;
    const { speed, easing } = settings;

    // Force a repaint so the transition takes effect.
    progress?.offsetWidth;

    applyStyles(bar, {
      transition: `all ${speed}ms ${easing}`,
      transform: `translate3d(${toBarPerc(n)}%,0,0)`
    });

    if (n === 1) {
      applyStyles(progress, { transition: 'none', opacity: '1' });
      progress?.offsetWidth;
      setTimeout(() => {
        applyStyles(progress, {
          transition: `all ${speed}ms linear`,
          opacity: '0'
        });
        setTimeout(() => {
          remove();
          stopTrickle();
        }, speed);
      }, speed);
    }
    return instance;
  }

  function start(): ScopedNProgress {
    if (typeof status !== 'number') set(0);
    stopTrickle();
    if (settings.trickle) {
      const work = () => {
        if (status === null) return;
        trickle();
        trickleTimer = setTimeout(work, settings.trickleSpeed);
      };
      trickleTimer = setTimeout(work, settings.trickleSpeed);
    }
    return instance;
  }

  function done(force?: boolean): ScopedNProgress {
    if (!force && status === null) return instance;
    stopTrickle();
    return set(1);
  }

  function configure(options: Partial<ScopedNProgressSettings>): ScopedNProgress {
    for (const key of Object.keys(options) as (keyof ScopedNProgressSettings)[]) {
      const value = options[key];
      if (value !== undefined) (settings as any)[key] = value;
    }
    return instance;
  }

  const instance: ScopedNProgress = { configure, start, done, set, remove };
  return instance;
}

// Module-level singleton, scoped exclusively to the ui-search widget.
const scopedNProgress = createScopedNProgress();

export default scopedNProgress;
