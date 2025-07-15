import { useSearchParams } from 'react-router-dom';
import queryString from 'query-string';

export default function useQueryParams() {
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
        return {
            from: 0,
            size: 10,
            sort: 'created:desc',
            ...(params || {})
        }
    }, [params])

    const setQueryParams = useCallback((arg) => {
        let params:any = {} 
        if (typeof arg === 'function') {
            params = arg(queryParams)
        } else if (typeof arg === 'object' && arg !== null) {
            Object.entries(arg).forEach(([key, value]) => {
                params[key] = value;
            });
        }
        setSearchParams(params);
    }, [queryParams])

    return [queryParams, setQueryParams];
}