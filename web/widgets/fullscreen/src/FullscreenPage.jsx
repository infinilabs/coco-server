import useQueryParams from './hooks/queryParams';
import { FullscreenPage } from './ui-search';

export default (props) => {

    const [queryParams, setQueryParams] = useQueryParams();

    return (
        <FullscreenPage {...props} queryParams={queryParams} setQueryParams={setQueryParams}/>
    )
}