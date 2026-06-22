import { useState } from 'react';
import { FullscreenModal } from 'ui-search';

type QueryParams = Record<string, any>;
type FullscreenModalWrapperProps = Record<string, any>;

export default function FullscreenModalWrapper(props: FullscreenModalWrapperProps) {

    const [queryParamsState, setQueryParamsState] = useState<QueryParams>({
        from: 0,
        size: 10,
    });

    return (
        <FullscreenModal 
            {...props} 
            queryParams={queryParamsState} 
            setQueryParams={setQueryParamsState}
            onLogoClick={() => {
                setQueryParamsState({
                    from: 0,
                    size: 10,
                })
            }}
        />
    )
}