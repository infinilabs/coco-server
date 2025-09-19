import { useState } from 'react';
import useQueryParams from './hooks/queryParams';
import { FullscreenPage } from './ui-search';

export default (props) => {

    const { enableQueryParams = true } = props; 

    const [queryParams, setQueryParams] = useQueryParams();

    const [queryParamsState, setQueryParamsState] = useState({
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
        <FullscreenPage {...props} {...queryParamsProps}/>
    )
}