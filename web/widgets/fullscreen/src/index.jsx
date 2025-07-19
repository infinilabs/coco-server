import { createRoot } from 'react-dom/client';
import Fullscreen from './Fullscreen';

export function fullscreen(props) {
  const container = typeof props.container === 'string' ? document.querySelector(props.container) : props.container;
  const root = createRoot(container);
  root.render(<Fullscreen {...props} />);
}