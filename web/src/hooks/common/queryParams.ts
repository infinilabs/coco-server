import queryString from 'query-string';

export default function useQueryParams(defaultParams: any = {}) {
  
  const getInitialParams = useCallback(() => {
    const currentUrl = new URL(window.location.href);
    if (currentUrl.hash) {
      const hashParts = currentUrl.hash.split('?');
      currentUrl.search = hashParts[1] || '';
    }
    const urlParams = queryString.parse(currentUrl.search, {
      parseBooleans: false,
      types: {
        from: 'number',
        size: 'number',
        sort: 'string',
        filter: 'string[]',
        aggfilter: 'string[]',
      }
    });

    const defaultSortStr = typeof defaultParams.sort === 'string'
      ? defaultParams.sort
      : (defaultParams.sort || []).map(
            ([field, order]: any[]) => `${field}:${order}`
        ).join(',');

    return {
      from: 0,
      size: 10,
      ...defaultParams,
      ...urlParams,
      sort: urlParams.sort || defaultSortStr
    };
  }, [defaultParams]);

  const [searchParams, setSearchParams] = useState<any>(getInitialParams);

  const queryParams = useMemo(() => {
    const filter: Record<string, any[]> = {};
    if (searchParams.filter) {
      if (Array.isArray(searchParams.filter)) {
        searchParams.filter.forEach((item: any) => {
          if (!item) return;
          const arr = item.split(':');
          if (arr.length === 2 && arr[0] && arr[1]) {
            if (Array.isArray(filter[arr[0]])) {
              filter[arr[0]].push(arr[1]);
            } else {
              filter[arr[0]] = [arr[1]];
            }
          }
        });
      } else {
        const arr = searchParams.filter.split(':');
        if (arr.length === 2 && arr[0] && arr[1]) {
          filter[arr[0]] = [arr[1]];
        }
      }
    }
    const aggfilter: Record<string, any[]> = {};
    if (searchParams.aggfilter) {
      if (Array.isArray(searchParams.aggfilter)) {
        searchParams.aggfilter.forEach((item: any) => {
          if (!item) return;
          const arr = item.split(':');
          if (arr.length === 2 && arr[0] && arr[1]) {
            if (Array.isArray(aggfilter[arr[0]])) {
              aggfilter[arr[0]].push(arr[1]);
            } else {
              aggfilter[arr[0]] = [arr[1]];
            }
          }
        });
      } else {
        const arr = searchParams.aggfilter.split(':');
        if (arr.length === 2 && arr[0] && arr[1]) {
          aggfilter[arr[0]] = [arr[1]];
        }
      }
    }
    return {
      ...(searchParams || {}),
      filter,
      aggfilter,
      sort: searchParams.sort || ''
    };
  }, [searchParams]);

  const setQueryParams = useCallback((arg: any) => {
    let newParams: any = {};
    if (typeof arg === 'function') {
      newParams = arg(queryParams);
    } else if (typeof arg === 'object' && arg !== null) {
      Object.entries(arg).forEach(([key, value]) => {
        newParams[key] = value;
      });
    }
    const filter = newParams.filter;
    const filters: any[] = [];
    if (filter) {
      Object.entries(filter).forEach(([key, value]) => {
        if (Array.isArray(value)) {
          value.forEach(item => {
            if (!item) return;
            filters.push(`${key}:${item}`);
          });
        }
      });
    }

    const aggfilter = newParams.aggfilter;
    const aggfilters: any[] = [];
    if (aggfilter) {
      Object.entries(aggfilter).forEach(([key, value]) => {
        if (Array.isArray(value)) {
          value.forEach(item => {
            if (!item) return;
            aggfilters.push(`${key}:${item}`);
          });
        }
      });
    }

    const sort = typeof newParams.sort === 'string' ? newParams.sort : '';
    const newSearchParams = {
      ...newParams,
      filter: filters,
      aggfilter: aggfilters,
      sort
    };
    if (!newSearchParams.sort) {
      delete newSearchParams.sort;
    }
    const currentUrl = new URL(window.location.href);
    if (currentUrl.hash) {
      const hashParts = currentUrl.hash.split('?');
      currentUrl.hash = hashParts[0];
    }
    const { pathname, hash } = currentUrl;
    const newUrl = `${pathname}${hash}?${queryString.stringify(newSearchParams)}`;
    window.history.pushState(null, '', newUrl);
    setSearchParams(newSearchParams);
  }, [queryParams]);

  return [queryParams, setQueryParams];
}
