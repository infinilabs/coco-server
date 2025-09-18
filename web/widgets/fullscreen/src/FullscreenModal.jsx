import { useState } from 'react';
import { FullscreenModal } from './ui-search';

export default (props) => {

    const [queryParamsState, setQueryParamsState] = useState({
        from: 0,
        size: 10,
    });

    return (
        <FullscreenModal {...props} queryParams={queryParamsState} setQueryParams={setQueryParamsState}/>
    )
}