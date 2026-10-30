import type { ArgsProps, MessageInstance } from 'antd/es/message/interface';

/**
 * Wrapper around the antd message instance that keeps error toasts readable:
 *
 * - identical type+content repeats merge into ONE toast (antd replaces same-key messages instead of stacking) — a burst
 *   of request timeouts shows once;
 * - at most MAX_ERROR_TOASTS distinct error toasts are visible, further ones are dropped while the cap holds (noise
 *   reduction).
 */

const MAX_ERROR_TOASTS = 3;
const DEFAULT_DURATION = 3;

type ToastType = 'error' | 'info' | 'loading' | 'success' | 'warning';

function contentKey(content: unknown): string | null {
  if (typeof content === 'string') return content.trim();
  if (typeof content === 'number') return String(content);
  // React nodes can't be keyed reliably — they stay undeduplicated
  return null;
}

export function createQuietMessage(message: MessageInstance): MessageInstance {
  const activeErrors = new Set<string>();
  const errorTimers = new Map<string, ReturnType<typeof setTimeout>>();

  // release the cap slot when the toast's duration is up (with slack for the
  // close animation); re-showing the same key resets the timer
  const trackError = (key: string, duration: number) => {
    const prev = errorTimers.get(key);
    if (prev) clearTimeout(prev);
    errorTimers.set(
      key,
      setTimeout(
        () => {
          errorTimers.delete(key);
          activeErrors.delete(key);
        },
        duration * 1000 + 1000
      )
    );
  };

  const show = (type: ToastType, args: unknown[]) => {
    const [first, second, third] = args;
    const isConfig = Boolean(first) && typeof first === 'object';
    const config = (isConfig ? first : {}) as ArgsProps;
    const content = (isConfig ? config.content : first) as ArgsProps['content'];
    const duration = (isConfig ? config.duration : (second as number | undefined)) ?? DEFAULT_DURATION;
    const onClose = (isConfig ? config.onClose : third) as (() => void) | undefined;
    const userKey = isConfig && config.key !== null && config.key !== undefined ? String(config.key) : undefined;

    const dedupKey = contentKey(content);
    const key = userKey ?? (dedupKey ? `${type}:${dedupKey}` : `${type}:${Date.now()}-${Math.random()}`);

    if (type === 'error') {
      if (!activeErrors.has(key) && activeErrors.size >= MAX_ERROR_TOASTS) {
        // cap reached and this is yet another distinct error — drop it
        return;
      }
      activeErrors.add(key);
      trackError(key, duration);
      message.open({ type, content, duration, key, onClose });
      return;
    }

    message.open({ type, content, duration, key, onClose });
  };

  return {
    success: (...args: unknown[]) => show('success', args),
    error: (...args: unknown[]) => show('error', args),
    info: (...args: unknown[]) => show('info', args),
    warning: (...args: unknown[]) => show('warning', args),
    loading: (...args: unknown[]) => show('loading', args),
    open: (cfg: ArgsProps) => show((cfg.type ?? 'info') as ToastType, [cfg]),
    destroy: () => message.destroy()
  } as MessageInstance;
}
