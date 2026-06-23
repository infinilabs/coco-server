import { useState } from 'react';

import useQueryParams from './hooks/queryParams';
import { FullscreenPage } from 'ui-search';

type QueryParams = Record<string, any>;
type FullscreenPageWrapperProps = Record<string, any> & {
    enableQueryParams?: boolean;
}

export default function FullscreenPageWrapper(props: FullscreenPageWrapperProps) {

    const { enableQueryParams = true } = props;

    const [queryParams, setQueryParams] = useQueryParams();

    const [queryParamsState, setQueryParamsState] = useState<QueryParams>({
        from: 0,
        size: 10,
    });

    const queryParamsProps = enableQueryParams ? {
        queryParams,
        setQueryParams,
    } : {
        queryParams: queryParamsState,
        setQueryParams: setQueryParamsState
    }

    return (
        <FullscreenPage 
            {...props} 
            {...queryParamsProps}
            onLogoClick={() => {
                if (enableQueryParams) {
                    const hashWithoutParams = window.location.hash.split('?')[0] || '';
                    const newUrl = window.location.origin + window.location.pathname + hashWithoutParams;
                    history.replaceState(null, '', newUrl);
                } else {
                    setQueryParamsState({
                        from: 0,
                        size: 10,
                    })
                }
            }}
        />
    )
}
