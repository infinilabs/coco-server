import { generateRandomString } from '@/utils/common';
import { configResponsive } from 'ahooks';

import { FullscreenPage } from 'ui-search';

const uuid = `integration-${generateRandomString(8)}`

configResponsive({ sm: 640 });
function useSimpleQueryParams(defaultParams = {}) {
  const [params, setParams] = useState({
    from: 0,
    size: 10,
    sort: [],
    filter: {},
    ...defaultParams,
  });

  return [params, setParams];
}
export function Component() {
  return "search"
}
