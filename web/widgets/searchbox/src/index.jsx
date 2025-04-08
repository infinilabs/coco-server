import { createRoot } from 'react-dom/client';

import { DocSearch } from './DocSearch';

export function searchbox(props) {
  const container =
    typeof props.container === 'string'
      ? (props.environment || window).document.querySelector(props.container)
      : props.container;
  const root = createRoot(container);
  root.render(<DocSearch {...props} />);
}

export default searchbox;
