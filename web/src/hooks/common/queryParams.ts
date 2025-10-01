import { useSearchParams } from 'react-router-dom';
import queryString from 'query-string';

export default function useQueryParams(defaultParams) {
    const [searchParams, setSearchParams] = useSearchParams();
    
    const params = queryString.parse(searchParams.toString(), {
		parseBooleans: false,
		types: {
				query: 'string',
				from: 'number',
				size: 'number',
				sort: 'string',
                filter: 'string[]'
		},
    });

    const queryParams = useMemo(() => {
        const filter = {}
        if (params.filter) {
            if (Array.isArray(params.filter)) {
                params.filter.forEach((item) => {
                    if (!item) return;
                    const arr = item.split(':')
                    if (arr.length === 2 && arr[0] && arr[1]) {
                        if (Array.isArray(filter[arr[0]])) {
                            filter[arr[0]].push(arr[1]);
                        } else {
                            filter[arr[0]] = [arr[1]];
                        }
                    }
                })
            } else {
                const arr = params.filter.split(':')
                if (arr.length === 2 && arr[0] && arr[1]) {
                    filter[arr[0]] = [arr[1]];
                }
            }
        }
        return {
            from: 0,
            size: 10,
            sort: 'created:desc',
            ...(defaultParams || {}),
            ...(params || {}),
            filter,
        }
    }, [params, defaultParams])

    const setQueryParams = useCallback((arg) => {
        let params:any = {} 
        if (typeof arg === 'function') {
            params = arg(queryParams)
        } else if (typeof arg === 'object' && arg !== null) {
            Object.entries(arg).forEach(([key, value]) => {
                params[key] = value;
            });
        }
        const filter = params.filter;
        const filters = [];
        if (filter) {
            Object.entries(filter).forEach(([key, value]) => {
                if (Array.isArray(value)) {
                    value.forEach((item) => {
                        if (!item) return;
                        filters.push(`${key}:${item}`)
                    })
                }
            })
        }
        setSearchParams({
            ...params,
            filter: filters
        });
    }, [queryParams])

    return [queryParams, setQueryParams];
}