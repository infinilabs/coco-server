import { createRoot } from 'react-dom/client';
import Fullscreen from './Fullscreen';

type FullscreenEntryProps = Record<string, any> & {
  container: string | Element | DocumentFragment;
}

export function fullscreen(props: FullscreenEntryProps) {
  const container = typeof props.container === 'string' ? document.querySelector(props.container) : props.container;
  if (!container) {
    throw new Error('fullscreen container not found');
  }
  const root = createRoot(container);
  root.render(<Fullscreen {...props} />);
}