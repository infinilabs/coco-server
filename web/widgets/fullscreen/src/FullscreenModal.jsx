import { FullscreenModal } from './ui-search';

export default (props) => {

    const [queryParams, setQueryParams] = useState({
        from: 0,
        size: 10,
    });

    return (
        <FullscreenModal {...props} queryParams={queryParams} setQueryParams={setQueryParams}/>
    )
}