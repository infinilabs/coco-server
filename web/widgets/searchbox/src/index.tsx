import { createRoot } from 'react-dom/client';

import { DocSearch } from './DocSearch';
import type { DocSearchProps } from './DocSearch';
import './styles/index.css';

export function searchbox(props: DocSearchProps) {
  const container =
    typeof props.container === 'string'
      ? ((props.environment || window) as Window).document.querySelector(props.container)
      : props.container;
  if (!container) return;
  const root = createRoot(container);
  root.render(<DocSearch {...props} />);
}

export default searchbox;
