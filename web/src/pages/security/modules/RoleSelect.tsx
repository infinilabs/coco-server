import DropdownList from '@/common/src/DropdownList';
import { fetchRoles } from '@/service/api/security';
import { formatESSearchResult } from '@/service/request/es';
import { useRequest } from '@sa/hooks';

const RoleSelect = (props) => {
    const {
        value,
        onChange,
        className,
        mode,
        width = '100%',
        allowClear,
        placeholder,
    } = props;

    const { hasAuth } = useAuth();

    const permissions = {
      search: hasAuth('generic#security:role/search')
    }

    const { data, loading, run } = useRequest(fetchRoles, { manual: true });

    const [queryParams, setQueryParams] = useState({
        query: '',
        from: 0,
        size: 10,
        sort: 'created:desc'
    });

    useEffect(() => {
        if (permissions.search) {
            run(queryParams);
        }
    }, [queryParams, permissions.search]);

    const result = useMemo(() => {
        return formatESSearchResult(data);
    }, [data]);

    return (
        <DropdownList
            value={value}
            onChange={(item) => onChange && onChange(item)}
            className={className}
            mode={mode}
            loading={loading}
            width={width}
            allowClear={allowClear}
            data={result.data}
            placeholder={placeholder}
            renderItem={(item: any) => <span>{item.name}</span>}
            renderLabel={(item: any) => item?.name}
            rowKey='name'
            pagination={{
                currentPage: result.total ? Math.floor(queryParams.from / queryParams.size) + 1 : 0,
                total: result.total,
                onChange: page => {
                setQueryParams(params => ({ ...params, from: (page - 1) * params.size }));
                }
            }}
            onSearchChange={(query: string) => {
                setQueryParams(params => ({ ...params, query, from: 0 }));
            }}
        />
    )

}

export default RoleSelect